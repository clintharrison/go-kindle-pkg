package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"time"

	"github.com/lmittmann/tint"
)

func initLogger() {
	w := os.Stderr
	defaultLevel := slog.LevelInfo
	if os.Getenv("DEBUG") == "1" {
		defaultLevel = slog.LevelDebug
	}
	slog.SetDefault(slog.New(
		tint.NewHandler(w, &tint.Options{
			Level:      defaultLevel,
			TimeFormat: time.TimeOnly,
		}),
	))
}

func main() {
	ctx := context.Background()
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, os.Kill)
	defer cancel()

	initLogger()

	rootCmd := NewRootCmd()
	if err := rootCmd.ExecuteContext(ctx); err != nil {
		os.Exit(1)
	}

	os.Exit(0)
}
