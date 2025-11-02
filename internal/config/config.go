package config

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Addr          string
	AdminPassword string
	DBPath        string
	LogLevel      slog.Level
	AdminCookie   string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		Addr:          getString("MEMORIES_ADDR", ":8080"),
		AdminPassword: strings.TrimSpace(os.Getenv("ADMIN_PASSWORD")),
		DBPath:        getString("MEMORIES_DB_PATH", "data/memories.db"),
		LogLevel:      getLogLevel("MEMORIES_LOG_LEVEL", slog.LevelInfo),
		AdminCookie:   getString("MEMORIES_ADMIN_COOKIE", "memories_admin"),
	}

	if cfg.AdminPassword == "" {
		return nil, fmt.Errorf("ADMIN_PASSWORD must be set")
	}

	return cfg, nil
}

func getString(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func getLogLevel(key string, fallback slog.Level) slog.Level {
	value := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	switch value {
	case "":
		return fallback
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return fallback
	}
}
