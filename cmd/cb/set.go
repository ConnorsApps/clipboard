package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/ConnorsApps/clipboard/pkg/cbclient"
	"github.com/urfave/cli/v3"
)

func setCommand() *cli.Command {
	return &cli.Command{
		Name:      "set",
		Usage:     "Set clipboard content (reads stdin if no arg)",
		ArgsUsage: "[text]",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "trim",
				Aliases: []string{"t"},
				Value:   true,
				Usage:   "trim trailing whitespace and newlines",
			},
		},
		Action: func(_ context.Context, cmd *cli.Command) error {
			cfg := mustLoadConfig()
			client := cbclient.NewClient(cfg.ServerURL, cfg.Token)

			var content string
			if cmd.Args().Len() > 0 {
				content = strings.Join(cmd.Args().Slice(), " ")
			} else {
				data, err := io.ReadAll(os.Stdin)
				if err != nil {
					return fmt.Errorf("failed to read stdin: %w", err)
				}
				content = string(data)
			}

			if cmd.Bool("trim") {
				content = strings.TrimRight(content, " \t\r\n")
			}

			return client.SetClipboard(content)
		},
	}
}
