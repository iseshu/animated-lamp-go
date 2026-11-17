package bot

import (
	"bytes"
	"context"
	"fmt"
	"math/rand"
	"net/url"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	tele "gopkg.in/telebot.v3"

	"animated-lamp-go/internal/config"
	"animated-lamp-go/internal/ffmpeg"
	"animated-lamp-go/internal/storage"
	"animated-lamp-go/internal/worker"
)

type UserState struct {
	AwaitingManualTrim bool
	AwaitingManualScht bool
	TargetURL          string
	TargetDuration     int
}

type Handlers struct {
	b         *tele.Bot
	cfg       *config.Config
	store     storage.Storage
	pool      *worker.Pool
	kb        *Keyboards
	states    sync.Map // int64 -> *UserState
	startTime time.Time
}

func NewHandlers(b *tele.Bot, cfg *config.Config, store storage.Storage, pool *worker.Pool, kb *Keyboards) *Handlers {
	return &Handlers{
		b:         b,
		cfg:       cfg,
		store:     store,
		pool:      pool,
		kb:        kb,
		startTime: time.Now(),
	}
}

func (h *Handlers) HandleStart(c tele.Context) error {
	u, err := h.store.GetOrCreateUser(c.Sender().ID)
	if err != nil {
		return c.Send("❌ Internal error loading user profile.")
	}

	if u.IsBanned {
		return c.Send(fmt.Sprintf("🚫 You are banned from using this bot.\nReason: %s", u.BanReason))
	}

	welcome := fmt.Sprintf("👋 Hello **%s**!\n\n"+
		"I am an ultra-fast **Screenshot & Video Processing Bot**.\n\n"+
		"⚡ **Features:**\n"+
		"• Extract 2–10 high-resolution screenshots in seconds\n"+
		"• Manual timestamp screenshots\n"+
		"• Generate preview video samples & trim clips\n"+
		"• Detailed `/mediainfo` inspection\n"+
		"• Custom watermarking (colors, sizes, positions)\n\n"+
		"Send me any direct video link (HTTP/HTTPS) or video file to get started!",
		c.Sender().FirstName)

	return c.Send(welcome, tele.ModeMarkdown)
}

func (h *Handlers) HandleHelp(c tele.Context) error {
	helpText := "📖 **Commands Guide**\n\n" +
		"• `/start` - Check bot status and welcome\n" +
		"• `/settings` - Configure upload format, watermark, & durations\n" +
		"• `/set_watermark <text>` - Set custom watermark text (max 30 chars)\n" +
		"• `/mediainfo` - Inspect video metadata\n\n" +
		"🚀 Simply send or forward any video link or video file to use!"
	return c.Send(helpText, tele.ModeMarkdown)
}

func (h *Handlers) HandleSettings(c tele.Context) error {
	u, err := h.store.GetOrCreateUser(c.Sender().ID)
	if err != nil {
		return c.Send("❌ Failed to load settings.")
	}

	menu, text := h.kb.SettingsMenu(u)
	return c.Send(text, menu, tele.ModeMarkdown)
}

func (h *Handlers) HandleSetWatermark(c tele.Context) error {
	args := c.Args()
	if len(args) == 0 {
		return c.Send("Usage: `/set_watermark <text>` (Max 30 characters)\nTo remove watermark, use `/settings`.", tele.ModeMarkdown)
	}

	text := strings.Join(args, " ")
	if len(text) > 30 {
		return c.Send(fmt.Sprintf("⚠️ Text is `%d` characters long. Maximum allowed is 30 characters.", len(text)), tele.ModeMarkdown)
	}

	u, err := h.store.GetOrCreateUser(c.Sender().ID)
	if err != nil {
		return c.Send("❌ Failed to update watermark.")
	}

	u.WatermarkText = text
	if err := h.store.SaveUser(u); err != nil {
		return c.Send("❌ Failed to save watermark.")
	}

	return c.Send(fmt.Sprintf("✅ Watermark updated to: `%s`", text), tele.ModeMarkdown)
}

func (h *Handlers) HandleStatus(c tele.Context) error {
	if !h.cfg.IsAdmin(c.Sender().ID) {
		return c.Send("⛔ Admin only command.")
	}

	totalUsers, _ := h.store.TotalUsersCount()
	activeWorkers := h.pool.ActiveWorkers()

	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	allocMB := float64(m.Alloc) / (1024 * 1024)
	sysMB := float64(m.Sys) / (1024 * 1024)

	status := fmt.Sprintf("📊 **System Status**\n\n"+
		"• **Total Registered Users:** `%d`\n"+
		"• **Active Worker Tasks:** `%d / %d`\n"+
		"• **Uptime:** `%s`\n"+
		"• **Go Alloc RAM:** `%.2f MB`\n"+
		"• **System Sys RAM:** `%.2f MB`\n"+
		"• **Num Goroutines:** `%d`",
		totalUsers, activeWorkers, h.cfg.MaxWorkers,
		time.Since(h.startTime).Truncate(time.Second),
		allocMB, sysMB, runtime.NumGoroutine())

	return c.Send(status, tele.ModeMarkdown)
}

func (h *Handlers) HandleText(c tele.Context) error {
	text := strings.TrimSpace(c.Text())
	userID := c.Sender().ID

	if raw, ok := h.states.Load(userID); ok {
		state := raw.(*UserState)

		if state.AwaitingManualTrim {
			h.states.Delete(userID)
			return h.processManualTrim(c, state.TargetURL, text, state.TargetDuration)
		}

		if state.AwaitingManualScht {
			h.states.Delete(userID)
			return h.processManualScreenshots(c, state.TargetURL, text, state.TargetDuration)
		}
	}

	if strings.HasPrefix(text, "http://") || strings.HasPrefix(text, "https://") {
		return h.handleVideoURL(c, text)
	}

	return nil
}

func (h *Handlers) handleVideoURL(c tele.Context, rawURL string) error {
	u, err := h.store.GetOrCreateUser(c.Sender().ID)
	if err != nil || u.IsBanned {
		return c.Send("🚫 Access restricted.")
	}

	_, err = url.ParseRequestURI(rawURL)
	if err != nil {
		return c.Send("❌ Invalid URL format.")
	}

	waitMsg, err := h.b.Send(c.Recipient(), "🔍 Probing video metadata, please wait...")
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	duration, err := ffmpeg.GetDuration(ctx, rawURL, h.cfg.IAMHeader)
	if err != nil {
		_ = h.b.Delete(waitMsg)
		return c.Send("😟 Sorry, I could not open or stream this video file.")
	}

	h.states.Store(c.Sender().ID, &UserState{
		TargetURL:      rawURL,
		TargetDuration: duration,
	})

	menu := h.kb.ActionsMenu(duration)
	msgText := fmt.Sprintf("🎬 **Video Detected**\n\n⏱ **Duration:** `%s` (`%ds`)\n\nChoose an action below:",
		(time.Duration(duration) * time.Second).String(), duration)

	_, err = h.b.Edit(waitMsg, msgText, menu, tele.ModeMarkdown)
	return err
}

func (h *Handlers) HandleScreenshotCount(c tele.Context, count int) error {
	_ = c.Respond()

	userID := c.Sender().ID
	raw, ok := h.states.Load(userID)
	if !ok {
		return c.Send("⚠️ Session expired, please send the video link again.")
	}
	state := raw.(*UserState)

	u, _ := h.store.GetOrCreateUser(userID)

	progressMsg, _ := h.b.Send(c.Recipient(), fmt.Sprintf("⏳ Queued request for %d screenshots...", count))

	err := h.pool.Submit(userID, func(ctx context.Context) error {
		_, _ = h.b.Edit(progressMsg, fmt.Sprintf("⚡ Generating %d screenshots in RAM...", count))

		w, height, _ := ffmpeg.GetDimensions(ctx, state.TargetURL, h.cfg.IAMHeader)
		filter := ffmpeg.BuildWatermarkFilter(u.WatermarkText, config.Colors[u.WatermarkColor], u.WatermarkPosition, u.FontSize, w, height)

		timestamps := make([]int, count)
		reduced := state.TargetDuration - int(float64(state.TargetDuration)*0.02)
		if reduced <= 0 {
			reduced = state.TargetDuration
		}

		if u.ScreenshotMode == 1 {
			for i := range timestamps {
				timestamps[i] = rand.Intn(reduced)
			}
		} else {
			interval := reduced / (count + 1)
			for i := range timestamps {
				timestamps[i] = interval * (i + 1)
			}
		}

		album := tele.Album{}
		for i, ts := range timestamps {
			frameBytes, err := ffmpeg.ExtractFrameToMemory(ctx, state.TargetURL, h.cfg.IAMHeader, ts, filter)
			if err != nil {
				continue
			}

			caption := fmt.Sprintf("⏱ Frame %d/%d at %s", i+1, count, (time.Duration(ts) * time.Second).String())
			reader := bytes.NewReader(frameBytes)

			if u.AsFile {
				doc := &tele.Document{
					File:     tele.FromReader(reader),
					Caption:  caption,
					FileName: fmt.Sprintf("screenshot_%d.jpg", i+1),
				}
				album = append(album, doc)
			} else {
				photo := &tele.Photo{
					File:    tele.FromReader(reader),
					Caption: caption,
				}
				album = append(album, photo)
			}
		}

		if len(album) == 0 {
			_, _ = h.b.Edit(progressMsg, "❌ Failed to generate screenshots.")
			return nil
		}

		_, _ = h.b.Edit(progressMsg, "📤 Uploading screenshots to Telegram...")
		_, _ = h.b.SendAlbum(c.Recipient(), album)
		_ = h.b.Delete(progressMsg)
		return nil
	})

	if err != nil {
		_, _ = h.b.Edit(progressMsg, fmt.Sprintf("⚠️ %v", err))
	}
	return nil
}

func (h *Handlers) HandleMediaInfoCallback(c tele.Context) error {
	_ = c.Respond()
	userID := c.Sender().ID
	raw, ok := h.states.Load(userID)
	if !ok {
		return c.Send("⚠️ Session expired, please send the video link again.")
	}
	state := raw.(*UserState)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	data, err := ffmpeg.GetMediaInfoJSON(ctx, state.TargetURL, h.cfg.IAMHeader)
	if err != nil {
		return c.Send("❌ Failed to extract media info.")
	}

	formatted, err := ffmpeg.FormatMediaInfo(data)
	if err != nil {
		return c.Send("❌ Failed to parse media info.")
	}

	return c.Send(formatted, tele.ModeMarkdown)
}

func (h *Handlers) HandleSettingsCallback(c tele.Context, action string) error {
	_ = c.Respond()

	u, err := h.store.GetOrCreateUser(c.Sender().ID)
	if err != nil {
		return nil
	}

	switch action {
	case "upload_mode":
		u.AsFile = !u.AsFile
	case "scht_mode":
		u.ScreenshotMode = (u.ScreenshotMode + 1) % 2
	case "wm_color":
		u.WatermarkColor = (u.WatermarkColor + 1) % len(config.Colors)
	case "wm_pos":
		u.WatermarkPosition = (u.WatermarkPosition + 1) % len(config.Positions)
	case "wm_font":
		u.FontSize = (u.FontSize + 1) % len(config.FontSizes)
	case "smpl_dur":
		durations := []int{30, 60, 90, 120}
		for i, d := range durations {
			if d == u.SampleDuration {
				u.SampleDuration = durations[(i+1)%len(durations)]
				break
			}
		}
		if u.SampleDuration == 0 {
			u.SampleDuration = 30
		}
	case "wm_text":
		return c.Send("To set watermark text, send `/set_watermark <your text>`.\nTo clear, set text to blank or disable.", tele.ModeMarkdown)
	case "close":
		return c.Delete()
	}

	_ = h.store.SaveUser(u)
	menu, text := h.kb.SettingsMenu(u)
	return c.Edit(text, menu, tele.ModeMarkdown)
}

func (h *Handlers) processManualScreenshots(c tele.Context, videoURL, rawInput string, duration int) error {
	timestampsStr := strings.Split(rawInput, ",")
	var timestamps []int

	for _, s := range timestampsStr {
		sec, err := strconv.Atoi(strings.TrimSpace(s))
		if err != nil || sec < 0 || sec > duration {
			continue
		}
		timestamps = append(timestamps, sec)
	}

	if len(timestamps) == 0 {
		return c.Send(fmt.Sprintf("❌ No valid timestamps provided. Must be numbers between 0 and %d.", duration))
	}
	if len(timestamps) > 10 {
		timestamps = timestamps[:10]
	}

	u, _ := h.store.GetOrCreateUser(c.Sender().ID)
	progressMsg, _ := h.b.Send(c.Recipient(), fmt.Sprintf("⏳ Processing %d manual screenshots...", len(timestamps)))

	return h.pool.Submit(c.Sender().ID, func(ctx context.Context) error {
		w, height, _ := ffmpeg.GetDimensions(ctx, videoURL, h.cfg.IAMHeader)
		filter := ffmpeg.BuildWatermarkFilter(u.WatermarkText, config.Colors[u.WatermarkColor], u.WatermarkPosition, u.FontSize, w, height)

		album := tele.Album{}
		for i, ts := range timestamps {
			frameBytes, err := ffmpeg.ExtractFrameToMemory(ctx, videoURL, h.cfg.IAMHeader, ts, filter)
			if err != nil {
				continue
			}

			caption := fmt.Sprintf("🎯 Manual frame %d/%d at %s", i+1, len(timestamps), (time.Duration(ts) * time.Second).String())
			reader := bytes.NewReader(frameBytes)

			if u.AsFile {
				album = append(album, &tele.Document{File: tele.FromReader(reader), Caption: caption, FileName: fmt.Sprintf("manual_%d.jpg", i+1)})
			} else {
				album = append(album, &tele.Photo{File: tele.FromReader(reader), Caption: caption})
			}
		}

		if len(album) == 0 {
			_, _ = h.b.Edit(progressMsg, "❌ Failed to capture manual frames.")
			return nil
		}

		_, _ = h.b.SendAlbum(c.Recipient(), album)
		_ = h.b.Delete(progressMsg)
		return nil
	})
}

func (h *Handlers) processManualTrim(c tele.Context, videoURL, rawInput string, maxDur int) error {
	parts := strings.Split(rawInput, ":")
	if len(parts) != 2 {
		return c.Send("❌ Invalid format. Use `start:end` in seconds (e.g. `10:45`).", tele.ModeMarkdown)
	}

	start, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
	end, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err1 != nil || err2 != nil || start < 0 || end <= start || end > maxDur {
		return c.Send(fmt.Sprintf("❌ Invalid boundaries. Must be between `0` and `%d` with `end > start`.", maxDur), tele.ModeMarkdown)
	}

	if (end - start) > h.cfg.MaxTrimDuration {
		return c.Send(fmt.Sprintf("⚠️ Trim duration cannot exceed %d seconds.", h.cfg.MaxTrimDuration))
	}

	progressMsg, _ := h.b.Send(c.Recipient(), "✂️ Trimming clip, please wait...")

	return h.pool.Submit(c.Sender().ID, func(ctx context.Context) error {
		clipFile, cleanup, err := ffmpeg.TrimVideo(ctx, videoURL, h.cfg.IAMHeader, start, end)
		if err != nil {
			_, _ = h.b.Edit(progressMsg, fmt.Sprintf("❌ Trim error: %v", err))
			return nil
		}
		defer cleanup()

		video := &tele.Video{
			File:    tele.FromDisk(clipFile),
			Caption: fmt.Sprintf("✂️ Trimmed: %s to %s", (time.Duration(start) * time.Second).String(), (time.Duration(end) * time.Second).String()),
		}

		_, _ = h.b.Send(c.Recipient(), video)
		_ = h.b.Delete(progressMsg)
		return nil
	})
}
