package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// Config holds application runtime configuration loaded from the environment.
type Config struct {
	AppEnv       string
	Host         string
	Port         int
	DatabasePath string
	SeedDemo     bool
	LogLevel     string
}

// Load reads configuration from environment variables (optionally from a .env file).
// Missing values fall back to sensible defaults so the app runs without a .env file.
func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		AppEnv:       getEnv("APP_ENV", "development"),
		Host:         getEnv("APP_HOST", "127.0.0.1"),
		Port:         getEnvAsInt("APP_PORT", 8080),
		DatabasePath: getEnv("DATABASE_PATH", "data/ledger.db"),
		SeedDemo:     getEnvAsBool("SEED_DEMO", false),
		LogLevel:     strings.ToLower(getEnv("LOG_LEVEL", "info")),
	}

	if cfg.Port < 1 || cfg.Port > 65535 {
		return nil, fmt.Errorf("invalid APP_PORT: %d", cfg.Port)
	}
	if strings.TrimSpace(cfg.DatabasePath) == "" {
		return nil, fmt.Errorf("DATABASE_PATH must not be empty")
	}

	return cfg, nil
}

// Addr returns the HTTP listen address.
func (c *Config) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && strings.TrimSpace(v) != "" {
		return strings.TrimSpace(v)
	}
	return fallback
}

func getEnvAsInt(key string, fallback int) int {
	raw, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(raw) == "" {
		return fallback
	}
	n, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return fallback
	}
	return n
}

func getEnvAsBool(key string, fallback bool) bool {
	raw, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(raw) == "" {
		return fallback
	}
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}
