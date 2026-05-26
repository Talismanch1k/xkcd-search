package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"yadro.com/course/api/adapters/aaa"
	"yadro.com/course/api/adapters/rest"
	"yadro.com/course/api/adapters/rest/middleware"
	"yadro.com/course/api/adapters/search"
	"yadro.com/course/api/adapters/update"
	"yadro.com/course/api/adapters/words"
	"yadro.com/course/api/core"
	"yadro.com/course/api/internal/config"
	"yadro.com/course/pkg/closer"
	"yadro.com/course/pkg/logger"
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
	auth, err := aaa.New(cfg.TokenTTL)
	if err != nil {
		slog.Error("init auth adapter", "err", err)
		return err
	}

	wordsClient, err := words.NewClient(cfg.WordsAddress)
	if err != nil {
		slog.Error("init words adapter", "err", err)
		return err
	}
	defer closer.CloseOrLog(wordsClient)

	updateClient, err := update.NewClient(cfg.UpdateAddress)
	if err != nil {
		slog.Error("init update adapter", "err", err)
		return err
	}
	defer closer.CloseOrLog(updateClient)

	searchClient, err := search.NewClient(cfg.SearchAddress)
	if err != nil {
		slog.Error("init search adapter", "err", err)
		return err
	}
	defer closer.CloseOrLog(searchClient)

	pingers := map[string]core.Pinger{
		"words":  wordsClient,
		"update": updateClient,
		"search": searchClient,
	}

	mux := http.NewServeMux()

	// Handlers
	mux.Handle("GET /api/ping", rest.NewPingHandler(pingers, cfg.HTTPServer.Timeout))
	mux.Handle("POST /api/login", rest.NewLoginHandler(auth))
	mux.Handle("GET /metrics", rest.NewMetricsHandler())

	// Norm (words) handlers
	normHandler := rest.NewNormHandler(wordsClient)
	mux.Handle("GET /api/words",
		http.TimeoutHandler(
			normHandler,
			cfg.HTTPServer.Timeout,
			"connection timed out",
		),
	)

	// update handlers
	mux.Handle("GET /api/db/stats", rest.NewStatsHandler(updateClient))
	mux.Handle("GET /api/db/status", rest.NewStatusHandler(updateClient))
	mux.Handle("POST /api/db/update", middleware.Auth(rest.NewUpdateHandler(updateClient), auth))

	// drops in cascade: database, search index
	mux.Handle("DELETE /api/db", middleware.Auth(rest.NewDropHandler(updateClient), auth))

	// search handlers
	mux.Handle("GET /api/comics/{id}", rest.NewComicHandler(searchClient))
	mux.Handle("GET /api/search", middleware.Concurrency(rest.NewSearchHandler(searchClient), cfg.SearchConcurrency))
	mux.Handle("GET /api/isearch", middleware.Rate(rest.NewISearchHandler(searchClient), cfg.SearchRate))

	srv := &http.Server{
		Addr:              cfg.HTTPServer.Address,
		Handler:           middleware.WithMetrics(mux),
		ReadTimeout:       cfg.HTTPServer.ReadTimeout,
		WriteTimeout:      cfg.HTTPServer.WriteTimeout,
		IdleTimeout:       cfg.HTTPServer.IdleTimeout,
		ReadHeaderTimeout: 5 * time.Second,
	}

	slog.Info("starting server", "addr", srv.Addr)
	errCh := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	// read os signals
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	select {
	// graceful shutdown
	case <-ctx.Done():
		slog.Info("gracefully shutting down server")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("service forced to shutdown: %w", err)
		}
		slog.Info("server exiting")

	case err := <-errCh:
		return fmt.Errorf("server running: %w", err)
	}

	return nil
}
