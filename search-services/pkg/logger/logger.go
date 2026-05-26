package logger

import (
	"log/slog"
	"os"
	"strings"
)

var logLevel = new(slog.LevelVar) // info by default

func Setup() {
	opts := slog.HandlerOptions{
		Level: logLevel,
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &opts)))
}

func ChangeLevel(level string) {
	switch strings.ToUpper(level) {
	case "DEBUG":
		logLevel.Set(slog.LevelDebug)
	case "WARN":
		logLevel.Set(slog.LevelWarn)
	case "ERROR":
		logLevel.Set(slog.LevelError)
	default: // info and others
		logLevel.Set(slog.LevelInfo)
	}
}
