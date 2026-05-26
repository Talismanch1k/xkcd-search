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
	Address         string        `yaml:"search_address" env:"SEARCH_ADDRESS" env-required:"true"`
	WordsAddress    string        `yaml:"words_address" env:"WORDS_ADDRESS" env-required:"true"`
	DBAddress       string        `yaml:"db_address" env:"DB_ADDRESS" env-default:"localhost:82"`
	LogLevel        string        `yaml:"log_level" env:"LOG_LEVEL" env-default:"INFO"`
	ISearchInterval time.Duration `yaml:"isearch_interval" env:"ISEARCH_INTERVAL" env-default:"24h"`
	BrokerAddress   string        `yaml:"broker_address" env:"BROKER_ADDRESS" env-default:"nats://localhost:4222"`
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
