package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"animated-lamp-go/internal/bot"
	"animated-lamp-go/internal/config"
	"animated-lamp-go/internal/storage"
	"animated-lamp-go/internal/worker"
)

func main() {
	cfg := config.Load()
	if cfg.BotToken == "" {
		log.Fatal("BOT_TOKEN environment variable is required")
	}

	log.Printf("Starting Screenshot Bot (Workers: %d, MaxUserTasks: %d)...",
		cfg.MaxWorkers, cfg.MaxProcessesPerUser)

	store, err := storage.NewBoltStorage(cfg.DBPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	pool := worker.NewPool(cfg.MaxWorkers, cfg.MaxProcessesPerUser, cfg.ProcessTimeout)

	server, err := bot.New(cfg, store, pool)
	if err != nil {
		log.Fatalf("Failed to initialize bot: %v", err)
	}

	// Graceful shutdown handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Shutting down gracefully...")
		server.Stop()
		os.Exit(0)
	}()

	server.Start()
}
