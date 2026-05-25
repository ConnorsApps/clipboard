package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v3"
)

func logoutCommand() *cli.Command {
	return &cli.Command{
		Name:    "logout",
		Aliases: []string{"lo"},
		Usage:   "Remove saved credentials",
		Action: func(_ context.Context, _ *cli.Command) error {
			err := os.Remove(configPath())
			if err != nil && !errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("failed to remove config: %w", err)
			}
			log.Info().Msg("Logged out")
			return nil
		},
	}
}
