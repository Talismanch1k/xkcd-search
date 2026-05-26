package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"
	"yadro.com/course/pkg/closer"
	"yadro.com/course/pkg/interceptors"
	"yadro.com/course/pkg/logger"
	"yadro.com/course/search/core"
	"yadro.com/course/search/internal/config"

	searchpb "yadro.com/course/proto/search"
	"yadro.com/course/search/adapters/db"
	searchgrpc "yadro.com/course/search/adapters/grpc"
	"yadro.com/course/search/adapters/score"
	"yadro.com/course/search/adapters/subscriber"
	"yadro.com/course/search/adapters/timer"
	"yadro.com/course/search/adapters/words"
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

	// database adapter (READONLY for search)
	storage, err := db.New(cfg.DBAddress)
	if err != nil {
		return fmt.Errorf("connect to db: %w", err)
	}
	defer closer.CloseOrLog(storage)

	// words adapter
	words, err := words.NewClient(cfg.WordsAddress)
	if err != nil {
		return fmt.Errorf("create words client: %w", err)
	}
	defer closer.CloseOrLog(words)

	// search service
	search := core.NewService(storage, words, score.DefaultScorer{})

	// read os signals
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// events subscriber
	sub, err := subscriber.New(cfg.BrokerAddress, search)
	if err != nil {
		return fmt.Errorf("create broker subscriber: %w", err)
	}
	defer closer.CloseOrLog(sub)

	if err := sub.Start(ctx); err != nil {
		return fmt.Errorf("start broker subscriber: %w", err)
	}

	// build index before starting
	timer.New(search, cfg.ISearchInterval).Start(ctx)

	// grpc server
	listener, err := net.Listen("tcp", cfg.Address)
	if err != nil {
		return fmt.Errorf("grpc listen: %w", err)
	}

	s := grpc.NewServer(grpc.ConnectionTimeout(5*time.Second),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle:     30 * time.Second,
			MaxConnectionAge:      1 * time.Hour,
			MaxConnectionAgeGrace: 5 * time.Minute,
			Time:                  30 * time.Second,
			Timeout:               5 * time.Second,
		}),
		grpc.UnaryInterceptor(interceptors.Logging()),
	)
	searchpb.RegisterSearchServiceServer(s, searchgrpc.NewServer(search))
	reflection.Register(s)

	slog.Info("server listening", "address", cfg.Address)
	errCh := make(chan error, 1)
	go func() {
		if err := s.Serve(listener); err != nil {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		slog.Info("gracefully shutting down server")
		s.GracefulStop()
		slog.Info("server exiting")

	case err := <-errCh:
		return fmt.Errorf("working server: %w", err)
	}

	return nil
}
