package main

import (
	"context"
	"fmt"
	"os"

	"github.com/ConnorsApps/clipboard/pkg/cbclient"
	"github.com/urfave/cli/v3"
	"golang.org/x/term"
)

func getCommand() *cli.Command {
	return &cli.Command{
		Name:    "get",
		Aliases: []string{"g", "paste"},
		Usage:   "Print clipboard content",
		Action: func(_ context.Context, _ *cli.Command) error {
			cfg := mustLoadConfig()
			client := cbclient.NewClient(cfg.ServerURL, cfg.Token)
			content, err := client.GetClipboard()
			if err != nil {
				return err
			}
			if term.IsTerminal(int(os.Stdout.Fd())) {
				fmt.Println(content)
			} else {
				fmt.Print(content)
			}
			return nil
		},
	}
}
