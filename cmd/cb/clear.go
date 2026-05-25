package main

import (
	"context"

	"github.com/ConnorsApps/clipboard/pkg/cbclient"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v3"
)

func clearCommand() *cli.Command {
	return &cli.Command{
		Name:  "clear",
		Usage: "Clear clipboard content",
		Action: func(_ context.Context, _ *cli.Command) error {
			cfg := mustLoadConfig()
			client := cbclient.NewClient(cfg.ServerURL, cfg.Token)
			if err := client.SetClipboard(""); err != nil {
				return err
			}
			log.Info().Msg("Clipboard cleared")
			return nil
		},
	}
}
