package main

import (
	"context"
	"fmt"
	"os"

	"github.com/ConnorsApps/clipboard/pkg/cbclient"
	"github.com/atotto/clipboard"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v3"
	"golang.org/x/term"
)

func getCommand() *cli.Command {
	return &cli.Command{
		Name:    "get",
		Aliases: []string{"g", "paste"},
		Usage:   "Print clipboard content and copy it to the local system clipboard",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "copy",
				Aliases: []string{"c"},
				Usage:   "copy to the local system clipboard (default: on when printing to a terminal, off when piped; use --copy=false to disable)",
			},
		},
		Action: func(_ context.Context, cmd *cli.Command) error {
			cfg := mustLoadConfig()
			client := cbclient.NewClient(cfg.ServerURL, cfg.Token)
			content, err := client.GetClipboard()
			if err != nil {
				return err
			}

			isTTY := term.IsTerminal(int(os.Stdout.Fd()))
			if isTTY {
				fmt.Println(content)
			} else {
				fmt.Print(content)
			}

			// Piped output stays side-effect free unless --copy is passed.
			shouldCopy := isTTY
			if cmd.IsSet("copy") {
				shouldCopy = cmd.Bool("copy")
			}
			if shouldCopy {
				if err := clipboard.WriteAll(content); err != nil {
					log.Warn().Err(err).Msg("Could not copy to the system clipboard (on Linux, install xclip, xsel or wl-clipboard)")
				} else if isTTY {
					log.Info().Msg("Copied to clipboard")
				}
			}
			return nil
		},
	}
}
