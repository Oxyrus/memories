package main

import (
	"log/slog"
	"os"

	"github.com/Oxyrus/memories/internal/config"
	"github.com/Oxyrus/memories/internal/logging"
	"github.com/Oxyrus/memories/internal/router"
	"github.com/Oxyrus/memories/internal/storage/sqlite"
)

func main() {
	bootstrapLogger := logging.New(slog.LevelInfo)

	cfg, err := config.Load()
	if err != nil {
		bootstrapLogger.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	logger := logging.New(cfg.LogLevel)

	store, err := sqlite.Open(cfg.DBPath)
	if err != nil {
		logger.Error("failed to open sqlite database", "path", cfg.DBPath, "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := store.Close(); err != nil {
			logger.Error("failed to close sqlite database", "error", err)
		}
	}()

	logger.Info("starting server", "addr", cfg.Addr)

	r := router.New(cfg, logger, store)

	if err := r.Run(cfg.Addr); err != nil {
		logger.Error("server stopped", "error", err)
		os.Exit(1)
	}
}
