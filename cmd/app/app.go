package app

import (
	"log/slog"
	"os"

	"github.com/alexbro4u/gotemplate/internal/config"
	"github.com/alexbro4u/gotemplate/internal/core"

	"github.com/spf13/cobra"
)

func New(cfg *config.Config) *cobra.Command {
	return &cobra.Command{
		Use:   "gotemplate",
		Short: "Start gotemplate server",
		Run: func(_ *cobra.Command, _ []string) {
			if err := core.Run(cfg); err != nil {
				slog.Default().Error("failed to run app", slog.Any("error", err))
				os.Exit(1)
			}
		},
	}
}
