package config

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Address     string        `yaml:"address"      env:"FRONTEND_ADDRESS" env-default:":8090"`
	APIAddress  string        `yaml:"api_address"  env:"API_ADDRESS"      env-required:"true"`
	LogLevel    string        `yaml:"log_level"    env:"LOG_LEVEL"        env-default:"INFO"`
	SessionTTL  time.Duration `yaml:"session_ttl"  env:"SESSION_TTL"      env-default:"24h"`
	SearchLimit int           `yaml:"search_limit" env:"SEARCH_LIMIT"     env-default:"5"`
}

func Load(path string) (Config, error) {
	var cfg Config

	if err := cleanenv.ReadConfig(path, &cfg); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			slog.Warn("yaml config not found, trying env", "err", err)
		} else {
			return Config{}, fmt.Errorf("failed to parse yaml file: %w", err)
		}
		if err = cleanenv.ReadEnv(&cfg); err != nil {
			return Config{}, fmt.Errorf("failed to parse env: %w", err)
		}
	}

	return cfg, nil
}
