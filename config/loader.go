package config

import (
	"log/slog"

	"github.com/joho/godotenv"
)

// Load loads configuration from a .env file if present.
// It logs a warning if the file is not found, but does not return an error,
// as the absence of a .env file is common in production environments.
func Load() {
	if err := godotenv.Load(); err != nil {
		// Slog is used here as this is part of the initial app bootstrap,
		// before the full logr logger may be configured.
		slog.Warn("no .env file found or error loading it", "error", err)
	}
}
