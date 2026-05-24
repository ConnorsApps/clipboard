package main

import (
	"context"
	"fmt"

	"github.com/ConnorsApps/clipboard/pkg/cbclient"
	"github.com/urfave/cli/v3"
)

func getCommand() *cli.Command {
	return &cli.Command{
		Name:  "get",
		Usage: "Print clipboard content",
		Action: func(_ context.Context, _ *cli.Command) error {
			cfg := mustLoadConfig()
			client := cbclient.NewClient(cfg.ServerURL, cfg.Token)
			content, err := client.GetClipboard()
			if err != nil {
				return err
			}
			fmt.Print(content)
			return nil
		},
	}
}
