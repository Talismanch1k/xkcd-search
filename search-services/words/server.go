package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"
	wordspb "yadro.com/course/proto/words"
	"yadro.com/course/pkg/interceptors"
	"yadro.com/course/words/words"
)

type serverConfig struct {
	Host string `yaml:"host" env:"WORDS_GRPC_HOST" env-default:""`
	Port string `yaml:"port" env:"WORDS_GRPC_PORT" env-required:"true"`
}

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, nil)))

	cfgPath := flag.String("config", "config.yaml", "path to config file")
	flag.Parse()

	if err := run(*cfgPath); err != nil {
		slog.Error("server failed", "err", err)
		os.Exit(1)
	}
}

func run(cfgPath string) error {
	cfg, err := loadConfig(cfgPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	address := net.JoinHostPort(cfg.Host, cfg.Port)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	s := grpc.NewServer(
		grpc.ConnectionTimeout(5*time.Second),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle:     30 * time.Second,
			MaxConnectionAge:      1 * time.Hour,
			MaxConnectionAgeGrace: 20 * time.Second,
			Time:                  10 * time.Second,
			Timeout:               5 * time.Second,
		}),
		grpc.UnaryInterceptor(interceptors.Logging()),
	)
	wordspb.RegisterWordsServer(s, &server{stemmer: words.WordsStemmer{}})
	reflection.Register(s)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		<-ctx.Done()
		slog.Info("Gracefully shutting down server")

		// for slow connections
		timer := time.AfterFunc(25*time.Second, func() {
			slog.Warn("force stop due timeout")
			s.Stop()
		})
		defer timer.Stop()

		s.GracefulStop()
	}()

	slog.Info("server listening", "address", address)
	if err := s.Serve(listener); err != nil {
		return fmt.Errorf("failed to serve: %w", err)
	}

	return nil
}

func loadConfig(path string) (serverConfig, error) {
	var cfg serverConfig

	if err := cleanenv.ReadConfig(path, &cfg); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			slog.Warn("yaml config not found, trying env", "err", err)
		} else {
			return cfg, fmt.Errorf("failed to parse yaml file: %w", err)
		}

		if err = cleanenv.ReadEnv(&cfg); err != nil {
			return cfg, fmt.Errorf("reading config: %w", err)
		}
	}

	return cfg, nil
}
