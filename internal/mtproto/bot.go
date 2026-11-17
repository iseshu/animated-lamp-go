package mtproto

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/message"
	"github.com/gotd/td/telegram/message/markup"
	"github.com/gotd/td/telegram/message/styling"
	"github.com/gotd/td/telegram/uploader"
	"github.com/gotd/td/tg"

	"animated-lamp-go/internal/config"
	"animated-lamp-go/internal/ffmpeg"
	"animated-lamp-go/internal/storage"
	"animated-lamp-go/internal/worker"
)

type MTProtoBot struct {
	client   *telegram.Client
	cfg      *config.Config
	store    storage.Storage
	pool     *worker.Pool
	streamer *Streamer
	sender   *message.Sender
	uploader *uploader.Uploader
	states   sync.Map // int64 -> *MediaState
}

type MediaState struct {
	StreamURL string
	Duration  int
	Doc       *tg.Document
}

func NewBot(cfg *config.Config, store storage.Storage, pool *worker.Pool) (*MTProtoBot, error) {
	dispatcher := tg.NewUpdateDispatcher()

	opts := telegram.Options{
		UpdateHandler: dispatcher,
	}

	client := telegram.NewClient(cfg.APIID, cfg.APIHash, opts)
	up := uploader.NewUploader(client.API())
	sender := message.NewSender(client.API()).WithUploader(up)

	bot := &MTProtoBot{
		client:   client,
		cfg:      cfg,
		store:    store,
		pool:     pool,
		sender:   sender,
		uploader: up,
	}

	dispatcher.OnNewMessage(bot.handleNewMessage)
	dispatcher.OnNewChannelMessage(bot.handleNewChannelMessage)
	dispatcher.OnBotCallbackQuery(bot.handleCallbackQuery)

	return bot, nil
}

func (b *MTProtoBot) Start(ctx context.Context) error {
	streamer, err := NewStreamer(b.client)
	if err != nil {
		return fmt.Errorf("failed to start streamer: %w", err)
	}
	b.streamer = streamer

	return b.client.Run(ctx, func(ctx context.Context) error {
		status, err := b.client.Auth().Status(ctx)
		if err != nil {
			return err
		}
		if !status.Authorized {
			log.Println("Authorizing bot with MTProto token...")
			if _, err := b.client.Auth().Bot(ctx, b.cfg.BotToken); err != nil {
				return fmt.Errorf("MTProto bot auth failed: %w", err)
			}
		}

		self, err := b.client.Self(ctx)
		if err == nil {
			log.Printf("🤖 MTProto Bot running as @%s (ID: %d)", self.Username, self.ID)
		}

		<-ctx.Done()
		_ = b.streamer.Close()
		return ctx.Err()
	})
}

func (b *MTProtoBot) handleNewChannelMessage(ctx context.Context, e tg.Entities, update *tg.UpdateNewChannelMessage) error {
	msg, ok := update.Message.(*tg.Message)
	if !ok || msg.Out {
		return nil
	}
	return b.processMessage(ctx, msg)
}

func (b *MTProtoBot) handleNewMessage(ctx context.Context, e tg.Entities, update *tg.UpdateNewMessage) error {
	msg, ok := update.Message.(*tg.Message)
	if !ok || msg.Out {
		return nil
	}
	return b.processMessage(ctx, msg)
}

func (b *MTProtoBot) processMessage(ctx context.Context, msg *tg.Message) error {
	senderUser, ok := msg.FromID.(*tg.PeerUser)
	if !ok {
		return nil
	}
	userID := senderUser.UserID
	peer := &tg.InputPeerUser{UserID: userID}

	text := strings.TrimSpace(msg.Message)

	// Command routing
	if strings.HasPrefix(text, "/start") {
		_, _ = b.store.GetOrCreateUser(userID)
		_, err := b.sender.To(peer).Text(ctx, "👋 Hello! Send me any direct video link or video file up to 2GB to generate screenshots!")
		return err
	}

	if strings.HasPrefix(text, "/status") {
		if !b.cfg.IsAdmin(userID) {
			_, err := b.sender.To(peer).Text(ctx, "⛔ Admin only.")
			return err
		}
		totalUsers, _ := b.store.TotalUsersCount()
		_, err := b.sender.To(peer).Text(ctx, fmt.Sprintf("📊 Total Users: %d\n⚡ Active Tasks: %d/%d", totalUsers, b.pool.ActiveWorkers(), b.cfg.MaxWorkers))
		return err
	}

	// Media inspection (Video / Document)
	if msg.Media != nil {
		if docMedia, ok := msg.Media.(*tg.MessageMediaDocument); ok {
			if doc, ok := docMedia.Document.(*tg.Document); ok && (strings.HasPrefix(doc.MimeType, "video") || doc.MimeType == "application/octet-stream") {
				streamURL := b.streamer.RegisterDocument(doc)
				return b.probeAndPresent(ctx, peer, userID, streamURL, doc)
			}
		}
	}

	// URL inspection
	if strings.HasPrefix(text, "http://") || strings.HasPrefix(text, "https://") {
		if _, err := url.ParseRequestURI(text); err == nil {
			return b.probeAndPresent(ctx, peer, userID, text, nil)
		}
	}

	return nil
}

func (b *MTProtoBot) probeAndPresent(ctx context.Context, peer *tg.InputPeerUser, userID int64, streamURL string, doc *tg.Document) error {
	probeCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	duration, err := ffmpeg.GetDuration(probeCtx, streamURL, b.cfg.IAMHeader)
	if err != nil {
		_, _ = b.sender.To(peer).Text(ctx, "😟 Sorry, I could not probe or stream this video file.")
		return nil
	}

	b.states.Store(userID, &MediaState{
		StreamURL: streamURL,
		Duration:  duration,
		Doc:       doc,
	})

	row1 := markup.InlineButtonRow(
		markup.Callback("2", []byte("scht_2")),
		markup.Callback("4", []byte("scht_4")),
		markup.Callback("6", []byte("scht_6")),
		markup.Callback("8", []byte("scht_8")),
		markup.Callback("10", []byte("scht_10")),
	)
	row2 := markup.InlineButtonRow(
		markup.Callback("ℹ️ MediaInfo", []byte("act_mi")),
	)

	text := fmt.Sprintf("🎬 **Video Detected (MTProto Stream)**\n\n⏱ Duration: `%ds`\n\nSelect number of screenshots to generate:", duration)
	_, err = b.sender.To(peer).Markup(markup.InlineKeyboard(row1, row2)).Text(ctx, text)
	return err
}

func (b *MTProtoBot) handleCallbackQuery(ctx context.Context, e tg.Entities, update *tg.UpdateBotCallbackQuery) error {
	data := string(update.Data)
	userID := update.UserID
	peer := &tg.InputPeerUser{UserID: userID}

	val, ok := b.states.Load(userID)
	if !ok {
		_, _ = b.sender.To(peer).Text(ctx, "⚠️ Session expired. Please send the video or link again.")
		return nil
	}
	state := val.(*MediaState)

	if strings.HasPrefix(data, "scht_") {
		count, _ := strconv.Atoi(strings.TrimPrefix(data, "scht_"))
		if count < 2 || count > 10 {
			count = 4
		}

		u, _ := b.store.GetOrCreateUser(userID)

		_, _ = b.sender.To(peer).Text(ctx, fmt.Sprintf("⚡ Generating %d in-memory screenshots via MTProto...", count))

		return b.pool.Submit(userID, func(jobCtx context.Context) error {
			w, height, _ := ffmpeg.GetDimensions(jobCtx, state.StreamURL, b.cfg.IAMHeader)
			filter := ffmpeg.BuildWatermarkFilter(u.WatermarkText, config.Colors[u.WatermarkColor], u.WatermarkPosition, u.FontSize, w, height)

			interval := state.Duration / (count + 1)
			if interval <= 0 {
				interval = 1
			}

			var albumElements []message.MultiMediaOption
			for i := 1; i <= count; i++ {
				ts := interval * i
				if u.ScreenshotMode == 1 {
					ts = rand.Intn(state.Duration)
				}

				frameBytes, err := ffmpeg.ExtractFrameToMemory(jobCtx, state.StreamURL, b.cfg.IAMHeader, ts, filter)
				if err != nil {
					continue
				}

				uploadFile, err := b.uploader.FromBytes(jobCtx, fmt.Sprintf("frame_%d.jpg", i), frameBytes)
				if err != nil {
					continue
				}

				caption := fmt.Sprintf("⏱ Frame %d/%d at %ds", i, count, ts)
				albumElements = append(albumElements, message.UploadedPhoto(uploadFile, styling.Plain(caption)))
			}

			if len(albumElements) == 0 {
				_, _ = b.sender.To(peer).Text(jobCtx, "❌ Failed to capture frames from stream.")
				return nil
			}

			_, err := b.sender.To(peer).Album(jobCtx, albumElements[0], albumElements[1:]...)
			return err
		})
	}

	if data == "act_mi" {
		infoBytes, err := ffmpeg.GetMediaInfoJSON(ctx, state.StreamURL, b.cfg.IAMHeader)
		if err != nil {
			_, _ = b.sender.To(peer).Text(ctx, "❌ Failed to read media info.")
			return nil
		}
		formatted, err := ffmpeg.FormatMediaInfo(infoBytes)
		if err != nil {
			formatted = string(infoBytes)
		}
		_, _ = b.sender.To(peer).Text(ctx, formatted)
		return nil
	}

	return nil
}
