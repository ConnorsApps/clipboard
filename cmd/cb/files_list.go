package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/ConnorsApps/clipboard/pkg/cbclient"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v3"
)

func filesListCommand() *cli.Command {
	return &cli.Command{
		Name:    "list",
		Aliases: []string{"l", "ls"},
		Usage:   "List uploaded files",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "json",
				Usage: "output as JSON",
			},
		},
		Action: func(_ context.Context, cmd *cli.Command) error {
			cfg := mustLoadConfig()
			client := cbclient.NewClient(cfg.ServerURL, cfg.Token)
			files, err := client.ListFiles()
			if err != nil {
				return err
			}

			if cmd.Bool("json") {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(files)
			}

			if len(files) == 0 {
				log.Info().Msg("No files uploaded")
				return nil
			}
			for _, f := range files {
				fmt.Printf("%-24s  %-32s  %8s  %s\n", f.ID, f.Name, humanSize(f.Size), f.UploadedAt)
			}
			return nil
		},
	}
}
