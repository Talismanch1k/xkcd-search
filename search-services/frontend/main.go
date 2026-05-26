package main

import (
	"context"
	"embed"
	"errors"
	"flag"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"yadro.com/course/frontend/adapters/api"
	"yadro.com/course/frontend/adapters/rest"
	"yadro.com/course/frontend/internal/config"
	"yadro.com/course/frontend/internal/session"
	"yadro.com/course/pkg/logger"
)

//go:embed templates static
var content embed.FS

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
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	apiClient := api.New(cfg.APIAddress)
	sessions := session.NewStore(ctx, cfg.SessionTTL)

	tmpl, err := template.ParseFS(content, "templates/*.html")
	if err != nil {
		return fmt.Errorf("parse templates: %w", err)
	}

	mux := http.NewServeMux()

	mux.Handle("GET /static/", http.FileServerFS(content))
	mux.Handle("GET /", rest.NewIndexHandler(tmpl))
	mux.Handle("GET /feed", rest.NewFeedHandler(tmpl, apiClient, cfg.SearchLimit))
	mux.Handle("GET /feed/discover", rest.NewDiscoverHandler(tmpl, apiClient, apiClient, cfg.SearchLimit))
	mux.Handle("GET /feed/next", rest.NewFeedNextHandler(apiClient, apiClient, cfg.SearchLimit))

	adminAuth := rest.RequireAuth(sessions)
	mux.Handle("GET /admin", rest.NewAdminHandler(tmpl, sessions, apiClient))
	mux.Handle("POST /admin/login", rest.NewAdminLoginHandler(sessions, apiClient))
	mux.Handle("POST /admin/logout", rest.NewAdminLogoutHandler(sessions))
	mux.Handle("POST /admin/update", adminAuth(rest.NewAdminUpdateHandler(sessions, apiClient)))
	mux.Handle("POST /admin/drop", adminAuth(rest.NewAdminDropHandler(sessions, apiClient)))

	srv := &http.Server{
		Addr:              cfg.Address,
		Handler:           mux,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
	}

	slog.Info("starting frontend", "addr", srv.Addr)
	errCh := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		slog.Info("gracefully shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("forced shutdown: %w", err)
		}
		slog.Info("server exiting")
	case err := <-errCh:
		return fmt.Errorf("server: %w", err)
	}
	return nil
}
