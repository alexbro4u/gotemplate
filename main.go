package main

import (
	"log/slog"
	"os"

	"github.com/alexbro4u/gotemplate/cmd/app"
	"github.com/alexbro4u/gotemplate/internal/config"
)

func main() {
	cfg, err := config.New()
	if err != nil {
		slog.Default().Error("failed to load config", slog.Any("error", err))
		os.Exit(1)
	}

	rootCmd := app.New(cfg)

	if execErr := rootCmd.Execute(); execErr != nil {
		slog.Default().Error("failed to execute command", slog.Any("error", execErr))
		os.Exit(1)
	}
}
