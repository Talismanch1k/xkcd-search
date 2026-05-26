package config

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type XKCD struct {
	URL         string        `yaml:"url" env:"XKCD_URL" env-default:"xkcd.com"`
	Concurrency int           `yaml:"concurrency" env:"XKCD_CONCURRENCY" env-default:"50"`
	Timeout     time.Duration `yaml:"timeout" env:"XKCD_TIMEOUT" env-default:"10s"`
	CheckPeriod time.Duration `yaml:"check_period" env:"XKCD_CHECK_PERIOD" env-default:"1h"`
}

type Config struct {
	LogLevel      string `yaml:"log_level" env:"LOG_LEVEL" env-default:"DEBUG"`
	Address       string `yaml:"update_address" env:"UPDATE_ADDRESS" env-default:"localhost:80"`
	XKCD          XKCD   `yaml:"xkcd"`
	DBAddress     string `yaml:"db_address" env:"DB_ADDRESS" env-default:"localhost:82"`
	WordsAddress  string `yaml:"words_address" env:"WORDS_ADDRESS" env-default:"localhost:81"`
	BrokerAddress string `yaml:"broker_address" env:"BROKER_ADDRESS" env-default:"nats://localhost:4222"`
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
