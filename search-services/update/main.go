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
	"sync"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"
	"yadro.com/course/pkg/closer"
	"yadro.com/course/pkg/interceptors"
	"yadro.com/course/pkg/logger"
	"yadro.com/course/update/adapters/db"
	"yadro.com/course/update/adapters/publisher"
	"yadro.com/course/update/adapters/words"
	"yadro.com/course/update/adapters/xkcd"
	"yadro.com/course/update/core"
	"yadro.com/course/update/internal/config"

	updatepb "yadro.com/course/proto/update"
	updategrpc "yadro.com/course/update/adapters/grpc"
)

func main() {
	// initialize global logger with info level
	logger.Setup()

	var configPath string
	flag.StringVar(&configPath, "config", "config.yaml", "server configuration file")
	flag.Parse()

	cfg, err := config.Load(configPath)
	if err != nil {
		slog.Error("failed to load config", "err", err)
		os.Exit(1)
	}

	// change level after log
	logger.ChangeLevel(cfg.LogLevel)

	if err := run(cfg); err != nil {
		slog.Error("server failed", "err", err)
		os.Exit(1)
	}
}

func run(cfg config.Config) error {
	slog.Info("starting server")

	// database adapter
	storage, err := db.New(cfg.DBAddress)
	if err != nil {
		return fmt.Errorf("connect to db: %w", err)
	}
	defer closer.CloseOrLog(storage)

	if err := storage.Migrate(); err != nil {
		return fmt.Errorf("migrate db: %w", err)
	}

	// xkcd adapter
	xkcd, err := xkcd.NewClient(cfg.XKCD.URL, cfg.XKCD.Timeout)
	if err != nil {
		return fmt.Errorf("create XKCD client: %w", err)
	}

	// words adapter
	words, err := words.NewClient(cfg.WordsAddress)
	if err != nil {
		return fmt.Errorf("create words client: %w", err)
	}
	defer closer.CloseOrLog(words)

	// notifier adapter
	notifier, err := publisher.New(cfg.BrokerAddress)
	if err != nil {
		return fmt.Errorf("create broker: %w", err)
	}
	defer closer.CloseOrLog(notifier)

	// service
	updater, err := core.NewService(storage, xkcd, words, notifier, cfg.XKCD.Concurrency)
	if err != nil {
		return fmt.Errorf("create update service: %w", err)
	}

	// grpc server
	listener, err := net.Listen("tcp", cfg.Address)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}

	s := grpc.NewServer(grpc.ConnectionTimeout(5*time.Second),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle:     2 * time.Minute,
			MaxConnectionAge:      1 * time.Hour,
			MaxConnectionAgeGrace: 5 * time.Minute,
			Time:                  2 * time.Minute,
			Timeout:               30 * time.Second,
		}),
		grpc.UnaryInterceptor(interceptors.Logging()),
	)
	updatepb.RegisterUpdateServer(s, updategrpc.NewServer(updater))
	reflection.Register(s)

	slog.Info("server listening", "address", cfg.Address)
	errCh := make(chan error, 1)
	go func() {
		if err := s.Serve(listener); err != nil {
			errCh <- err
		}
	}()

	// read os signals
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	wg := &sync.WaitGroup{}

	wg.Go(func() {
		// update on startup
		slog.Info("initial update started")
		if err := updater.Update(ctx); err != nil {
			if !errors.Is(err, core.ErrAlreadyExists) {
				slog.Error("initial update", "err", err)
			}
		}

		// periodic update
		ticker := time.NewTicker(cfg.XKCD.CheckPeriod)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				slog.Info("periodic update started")
				if err := updater.Update(ctx); err != nil {
					if errors.Is(err, core.ErrAlreadyExists) {
						slog.Debug("periodic update skipped: already running")
					} else {
						slog.Error("periodic update", "err", err)
					}
				}
			case <-ctx.Done():
				return
			}
		}
	})

	select {
	case <-ctx.Done():
		slog.Info("gracefully shutting down server")
		s.GracefulStop()

		// wait update before closing notifier
		wg.Wait()

		slog.Info("server exiting")

	case err := <-errCh:
		return fmt.Errorf("server error: %w", err)
	}

	return nil
}
