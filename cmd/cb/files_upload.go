package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ConnorsApps/clipboard/pkg/cbclient"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v3"
)

func filesUploadCommand() *cli.Command {
	return &cli.Command{
		Name:      "upload",
		Aliases:   []string{"u", "ul"},
		Usage:     "Upload a file",
		ArgsUsage: "<path>",
		ShellComplete: func(_ context.Context, cmd *cli.Command) {
			prefix := cmd.Args().First()
			dir, file := filepath.Split(prefix)
			readDir := dir
			if readDir == "" {
				readDir = "."
			}
			entries, err := os.ReadDir(readDir)
			if err != nil {
				return
			}
			for _, e := range entries {
				if !strings.HasPrefix(e.Name(), file) {
					continue
				}
				completion := dir + e.Name()
				if e.IsDir() {
					completion += "/"
				}
				fmt.Println(completion)
			}
		},
		Action: func(_ context.Context, cmd *cli.Command) error {
			if cmd.Args().Len() == 0 {
				return fmt.Errorf("path argument required")
			}
			path := cmd.Args().First()

			f, err := os.Open(path)
			if err != nil {
				return fmt.Errorf("failed to open file: %w", err)
			}
			defer f.Close()

			fi, err := f.Stat()
			if err != nil {
				return fmt.Errorf("failed to stat file: %w", err)
			}

			cfg := mustLoadConfig()
			client := cbclient.NewClient(cfg.ServerURL, cfg.Token)

			pr := &progressReader{r: f, total: fi.Size()}
			if err := client.UploadFileFromReader(pr, fi.Size(), path); err != nil {
				return err
			}
			fmt.Fprintln(os.Stderr)
			log.Info().Str("path", path).Msg("Uploaded")
			return nil
		},
	}
}
