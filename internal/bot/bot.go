package bot

import (
	"log"
	"time"

	tele "gopkg.in/telebot.v3"

	"animated-lamp-go/internal/config"
	"animated-lamp-go/internal/storage"
	"animated-lamp-go/internal/worker"
)

type BotServer struct {
	bot      *tele.Bot
	cfg      *config.Config
	storage  storage.Storage
	pool     *worker.Pool
	handlers *Handlers
}

func New(cfg *config.Config, store storage.Storage, pool *worker.Pool) (*BotServer, error) {
	pref := tele.Settings{
		Token:  cfg.BotToken,
		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
	}

	b, err := tele.NewBot(pref)
	if err != nil {
		return nil, err
	}

	kb := NewKeyboards()
	handlers := NewHandlers(b, cfg, store, pool, kb)

	server := &BotServer{
		bot:      b,
		cfg:      cfg,
		storage:  store,
		pool:     pool,
		handlers: handlers,
	}

	server.registerRoutes(kb)
	return server, nil
}

func (s *BotServer) registerRoutes(kb *Keyboards) {
	s.bot.Handle("/start", s.handlers.HandleStart)
	s.bot.Handle("/help", s.handlers.HandleHelp)
	s.bot.Handle("/settings", s.handlers.HandleSettings)
	s.bot.Handle("/set_watermark", s.handlers.HandleSetWatermark)
	s.bot.Handle("/status", s.handlers.HandleStatus)

	s.bot.Handle(tele.OnText, s.handlers.HandleText)

	// Screenshot callbacks
	s.bot.Handle(&kb.BtnScreenshot2, func(c tele.Context) error { return s.handlers.HandleScreenshotCount(c, 2) })
	s.bot.Handle(&kb.BtnScreenshot4, func(c tele.Context) error { return s.handlers.HandleScreenshotCount(c, 4) })
	s.bot.Handle(&kb.BtnScreenshot6, func(c tele.Context) error { return s.handlers.HandleScreenshotCount(c, 6) })
	s.bot.Handle(&kb.BtnScreenshot8, func(c tele.Context) error { return s.handlers.HandleScreenshotCount(c, 8) })
	s.bot.Handle(&kb.BtnScreenshot10, func(c tele.Context) error { return s.handlers.HandleScreenshotCount(c, 10) })

	s.bot.Handle(&kb.BtnMediaInfo, s.handlers.HandleMediaInfoCallback)

	s.bot.Handle(&kb.BtnManualScreenshot, func(c tele.Context) error {
		_ = c.Respond()
		if raw, ok := s.handlers.states.Load(c.Sender().ID); ok {
			state := raw.(*UserState)
			state.AwaitingManualScht = true
		}
		return c.Send("🎯 Send the timestamps separated by commas (e.g. `10, 45, 120` in seconds).", tele.ModeMarkdown)
	})

	s.bot.Handle(&kb.BtnTrimVideo, func(c tele.Context) error {
		_ = c.Respond()
		if raw, ok := s.handlers.states.Load(c.Sender().ID); ok {
			state := raw.(*UserState)
			state.AwaitingManualTrim = true
		}
		return c.Send("✂️ Send the start and end time in `start:end` format (e.g. `15:45` in seconds).", tele.ModeMarkdown)
	})

	// Settings callbacks
	s.bot.Handle(&kb.BtnToggleUploadMode, func(c tele.Context) error { return s.handlers.HandleSettingsCallback(c, "upload_mode") })
	s.bot.Handle(&kb.BtnToggleMode, func(c tele.Context) error { return s.handlers.HandleSettingsCallback(c, "scht_mode") })
	s.bot.Handle(&kb.BtnWatermarkColor, func(c tele.Context) error { return s.handlers.HandleSettingsCallback(c, "wm_color") })
	s.bot.Handle(&kb.BtnWatermarkPos, func(c tele.Context) error { return s.handlers.HandleSettingsCallback(c, "wm_pos") })
	s.bot.Handle(&kb.BtnWatermarkFont, func(c tele.Context) error { return s.handlers.HandleSettingsCallback(c, "wm_font") })
	s.bot.Handle(&kb.BtnSampleDuration, func(c tele.Context) error { return s.handlers.HandleSettingsCallback(c, "smpl_dur") })
	s.bot.Handle(&kb.BtnWatermarkText, func(c tele.Context) error { return s.handlers.HandleSettingsCallback(c, "wm_text") })
	s.bot.Handle(&kb.BtnCloseSettings, func(c tele.Context) error { return s.handlers.HandleSettingsCallback(c, "close") })
}

func (s *BotServer) Start() {
	log.Printf("🤖 Bot started successfully on @%s", s.bot.Me.Username)
	s.bot.Start()
}

func (s *BotServer) Stop() {
	s.bot.Stop()
	s.pool.Stop()
	_ = s.storage.Close()
	log.Println("👋 Bot stopped cleanly.")
}
