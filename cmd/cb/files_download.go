package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ConnorsApps/clipboard/pkg/cbclient"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v3"
	"golang.org/x/term"
)

func filesDownloadCommand() *cli.Command {
	return &cli.Command{
		Name:      "download",
		Aliases:   []string{"d", "dl"},
		Usage:     "Download a file",
		ArgsUsage: "[id]",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Usage:   "save to this path instead of the remote filename",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			cfg := mustLoadConfig()
			client := cbclient.NewClient(cfg.ServerURL, cfg.Token)
			outPath := cmd.String("output")

			id := cmd.Args().First()
			var toDownload []cbclient.FileInfo

			if id != "" {
				toDownload = []cbclient.FileInfo{{ID: id}}
			} else {
				files, err := client.ListFiles()
				if err != nil {
					return fmt.Errorf("failed to list files: %w", err)
				}
				if len(files) == 0 {
					return fmt.Errorf("no files uploaded")
				}
				for i, f := range files {
					fmt.Fprintf(os.Stderr, "  %d) %-32s  %s\n", i+1, f.Name, humanSize(f.Size))
				}
				fmt.Fprint(os.Stderr, "Pick a file (enter for all): ")
				line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
				line = strings.TrimSpace(line)
				if line == "" {
					toDownload = files
				} else {
					n, err := strconv.Atoi(line)
					if err != nil || n < 1 || n > len(files) {
						return fmt.Errorf("invalid selection")
					}
					toDownload = files[n-1 : n]
				}
			}

			if outPath != "" && len(toDownload) > 1 {
				return fmt.Errorf("--output cannot be used when downloading all files")
			}

			// Pipe mode: stream a single file to stdout silently.
			if len(toDownload) == 1 && !term.IsTerminal(int(os.Stdout.Fd())) {
				body, _, err := client.DownloadFile(toDownload[0].ID)
				if err != nil {
					return err
				}
				defer body.Close()
				_, err = io.Copy(os.Stdout, body)
				return err
			}

			for _, fileInfo := range toDownload {
				remoteName := fileInfo.Name
				if remoteName == "" {
					// Look up the filename when the ID was provided directly.
					allFiles, err := client.ListFiles()
					if err == nil {
						for _, f := range allFiles {
							if f.ID == fileInfo.ID {
								remoteName = f.Name
								break
							}
						}
					}
					if remoteName == "" {
						remoteName = fileInfo.ID
					}
				}

				dest := outPath
				if dest == "" {
					dest = remoteName
				}

				body, size, err := client.DownloadFileAt(fileInfo.ID, 0)
				if err != nil {
					return err
				}

				f, err := os.OpenFile(dest, os.O_RDWR|os.O_CREATE, 0644)
				if err != nil {
					body.Close()
					return fmt.Errorf("failed to create file: %w", err)
				}

				var written int64
				var lastErr error
				backoff := downloadBaseDelay
				for attempt := range downloadMaxRetries + 1 {
					if attempt > 0 {
						fmt.Fprintf(os.Stderr, "\nRetrying (%d/%d)...", attempt, downloadMaxRetries)
						select {
						case <-ctx.Done():
							f.Close()
							return ctx.Err()
						case <-time.After(backoff):
						}
						backoff = min(backoff*2, downloadMaxDelay)
						var remaining int64
						body, remaining, err = client.DownloadFileAt(fileInfo.ID, written)
						if err != nil {
							lastErr = err
							continue
						}
						if size > 0 && remaining > 0 && written+remaining != size {
							body.Close()
							lastErr = fmt.Errorf("file size changed during download")
							break
						}
					}
					f.Seek(written, io.SeekStart)
					pr := &progressReader{r: body, total: size, read: written}
					n, copyErr := io.Copy(f, pr)
					written += n
					body.Close()
					if copyErr == nil {
						lastErr = nil
						break
					}
					lastErr = copyErr
				}
				fmt.Fprintln(os.Stderr)
				f.Close()
				if lastErr != nil {
					return lastErr
				}
				log.Info().Str("path", dest).Msg("Saved")
			}

			return nil
		},
	}
}
