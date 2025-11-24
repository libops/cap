package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/libops/cap/config"
	"github.com/libops/cap/scraper"
)

func main() {
	cfg, err := config.LoadFromEnv()
	if err != nil {
		slog.Error("Configuration error", "err", err)
		os.Exit(1)
	}

	s, err := scraper.NewScraper(cfg, slog.Default(), os.Stderr)
	if err != nil {
		slog.Error("Failed to initialize scraper", "err", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		slog.Info("Received shutdown signal", "signal", sig)
		cancel()
	}()

	s.Run(ctx)
	slog.Info("Scraper stopped gracefully")
}
