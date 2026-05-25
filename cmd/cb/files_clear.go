package main

import (
	"context"
	"fmt"

	"github.com/ConnorsApps/clipboard/pkg/cbclient"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v3"
)

func filesClearCommand() *cli.Command {
	return &cli.Command{
		Name:    "clear",
		Aliases: []string{"cl", "wipe"},
		Usage:   "Delete all uploaded files",
		Action: func(_ context.Context, _ *cli.Command) error {
			cfg := mustLoadConfig()
			client := cbclient.NewClient(cfg.ServerURL, cfg.Token)
			files, err := client.ListFiles()
			if err != nil {
				return err
			}
			if len(files) == 0 {
				log.Info().Msg("No files to clear")
				return nil
			}
			for _, f := range files {
				if err := client.DeleteFile(f.ID); err != nil {
					return fmt.Errorf("deleting %s: %w", f.Name, err)
				}
			}
			log.Info().Int("count", len(files)).Msg("Files cleared")
			return nil
		},
	}
}
