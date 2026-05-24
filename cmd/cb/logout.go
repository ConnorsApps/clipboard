package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/urfave/cli/v3"
)

func logoutCommand() *cli.Command {
	return &cli.Command{
		Name:  "logout",
		Usage: "Remove saved credentials",
		Action: func(_ context.Context, _ *cli.Command) error {
			err := os.Remove(configPath())
			if err != nil && !errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("failed to remove config: %w", err)
			}
			fmt.Fprintln(os.Stderr, successStyle.Render("Logged out"))
			return nil
		},
	}
}
