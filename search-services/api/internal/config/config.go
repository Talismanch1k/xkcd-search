package config

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type HTTPServerConfig struct {
	Address string `yaml:"address" env:"API_ADDRESS" env-required:"true"`
	// complex timeouts instead HTTP_SERVER_TIMEOUT
	Timeout      time.Duration `yaml:"timeout" env:"HTTP_SERVER_TIMEOUT" env-default:"15s"`
	ReadTimeout  time.Duration `yaml:"read_timeout" env:"HTTP_SERVER_READ_TIMEOUT" env-default:"5m"`
	WriteTimeout time.Duration `yaml:"write_timeout" env:"HTTP_SERVER_WRITE_TIMEOUT" env-default:"5m"`
	IdleTimeout  time.Duration `yaml:"idle_timeout" env:"HTTP_SERVER_IDLE_TIMEOUT" env-default:"5m"`
}

type Config struct {
	HTTPServer        HTTPServerConfig `yaml:"http_server" env-required:"true"`
	WordsAddress      string           `yaml:"words_address" env:"WORDS_ADDRESS" env-required:"true"`
	UpdateAddress     string           `yaml:"update_address" env:"UPDATE_ADDRESS" env-required:"true"`
	SearchAddress     string           `yaml:"search_address" env:"SEARCH_ADDRESS" env-required:"true"`
	LogLevel          string           `yaml:"log_level" env:"LOG_LEVEL" env-default:"INFO"`
	TokenTTL          time.Duration    `yaml:"token_ttl" env:"TOKEN_TTL" env-default:"2m"`
	SearchConcurrency int              `yaml:"search_concurrency" env:"SEARCH_CONCURRENCY" env-default:"10"`
	SearchRate        int              `yaml:"search_rate" env:"SEARCH_RATE" env-default:"100"`
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
