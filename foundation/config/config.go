package config

import (
	"os"
	"path/filepath"
	"time"
)

// AppConfig holds all configurable values for the application.
type AppConfig struct {
	MaxHistory     int
	ThemeColorPath string
	PollInterval   time.Duration
}

// Default returns an AppConfig populated with sensible defaults.
func Default() AppConfig {
	home := os.Getenv("HOME")
	return AppConfig{
		MaxHistory:     50,
		ThemeColorPath: filepath.Join(home, ".config/omarchy/current/theme/colors.toml"),
		PollInterval:   500 * time.Millisecond,
	}
}
