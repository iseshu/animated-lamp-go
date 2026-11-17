package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"animated-lamp-go/internal/bot"
	"animated-lamp-go/internal/config"
	"animated-lamp-go/internal/mtproto"
	"animated-lamp-go/internal/storage"
	"animated-lamp-go/internal/worker"
)

func main() {
	cfg := config.Load()
	if cfg.BotToken == "" {
		log.Fatal("BOT_TOKEN environment variable is required")
	}

	store, err := storage.NewBoltStorage(cfg.DBPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	pool := worker.NewPool(cfg.MaxWorkers, cfg.MaxProcessesPerUser, cfg.ProcessTimeout)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// If API_ID and API_HASH are provided, run the native MTProto engine
	if cfg.APIID != 0 && cfg.APIHash != "" {
		log.Printf("🚀 Starting in native MTProto mode (API_ID: %d)...", cfg.APIID)
		mtBot, err := mtproto.NewBot(cfg, store, pool)
		if err != nil {
			log.Fatalf("Failed to initialize MTProto bot: %v", err)
		}
		if err := mtBot.Start(ctx); err != nil && err != context.Canceled {
			log.Fatalf("MTProto bot error: %v", err)
		}
		return
	}

	// Otherwise, run in standard Bot API mode
	log.Println("🤖 Starting in Telegram Bot API mode...")
	server, err := bot.New(cfg, store, pool)
	if err != nil {
		log.Fatalf("Failed to initialize bot: %v", err)
	}

	go func() {
		<-ctx.Done()
		log.Println("Shutting down gracefully...")
		server.Stop()
	}()

	server.Start()
}
