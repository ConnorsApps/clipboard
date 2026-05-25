package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/ConnorsApps/clipboard/pkg/cbclient"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v3"
	"golang.org/x/term"
)

func humanSize(b int64) string {
	switch {
	case b >= 1<<30:
		return fmt.Sprintf("%.1f GB", float64(b)/(1<<30))
	case b >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(b)/(1<<20))
	case b >= 1<<10:
		return fmt.Sprintf("%.1f KB", float64(b)/(1<<10))
	default:
		return fmt.Sprintf("%d B", b)
	}
}

// progressReader wraps an io.Reader and prints transfer progress to stderr.
type progressReader struct {
	r     io.Reader
	total int64
	read  int64
}

func (p *progressReader) Read(buf []byte) (int, error) {
	n, err := p.r.Read(buf)
	p.read += int64(n)
	if p.total > 0 {
		fmt.Fprintf(os.Stderr, "\r%s / %s", humanSize(p.read), humanSize(p.total))
	} else {
		fmt.Fprintf(os.Stderr, "\r%s", humanSize(p.read))
	}
	return n, err
}

func filesCommand() *cli.Command {
	return &cli.Command{
		Name:  "files",
		Usage: "Manage files on the clipboard server",
		Commands: []*cli.Command{
			filesListCommand(),
			filesDownloadCommand(),
			filesUploadCommand(),
			filesDeleteCommand(),
		},
	}
}

func filesListCommand() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "List uploaded files",
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

func filesDownloadCommand() *cli.Command {
	return &cli.Command{
		Name:      "download",
		Usage:     "Download a file",
		ArgsUsage: "[id]",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Usage:   "save to this path instead of the remote filename",
			},
		},
		Action: func(_ context.Context, cmd *cli.Command) error {
			cfg := mustLoadConfig()
			client := cbclient.NewClient(cfg.ServerURL, cfg.Token)

			id := cmd.Args().First()
			var remoteName string

			if id == "" {
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
				fmt.Fprint(os.Stderr, "Pick a file [1]: ")
				line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
				line = strings.TrimSpace(line)
				if line == "" {
					line = "1"
				}
				n, err := strconv.Atoi(line)
				if err != nil || n < 1 || n > len(files) {
					return fmt.Errorf("invalid selection")
				}
				id = files[n-1].ID
				remoteName = files[n-1].Name
			}

			body, size, err := client.DownloadFile(id)
			if err != nil {
				return err
			}
			defer body.Close()

			// Pipe mode: stream to stdout silently.
			if !term.IsTerminal(int(os.Stdout.Fd())) {
				_, err = io.Copy(os.Stdout, body)
				return err
			}

			// Terminal mode: save to file with progress.
			if remoteName == "" {
				// Look up the filename from the file list when the id was provided directly.
				files, err := client.ListFiles()
				if err == nil {
					for _, f := range files {
						if f.ID == id {
							remoteName = f.Name
							break
						}
					}
				}
				if remoteName == "" {
					remoteName = id
				}
			}

			outPath := cmd.String("output")
			if outPath == "" {
				outPath = remoteName
			}

			f, err := os.Create(outPath)
			if err != nil {
				return fmt.Errorf("failed to create file: %w", err)
			}
			defer f.Close()

			pr := &progressReader{r: body, total: size}
			if _, err := io.Copy(f, pr); err != nil {
				return err
			}
			fmt.Fprintln(os.Stderr) // newline after progress
			log.Info().Str("path", outPath).Msg("Saved")
			return nil
		},
	}
}

func filesUploadCommand() *cli.Command {
	return &cli.Command{
		Name:      "upload",
		Usage:     "Upload a file",
		ArgsUsage: "<path>",
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

func filesDeleteCommand() *cli.Command {
	return &cli.Command{
		Name:      "delete",
		Usage:     "Delete a file by ID",
		ArgsUsage: "<id>",
		Action: func(_ context.Context, cmd *cli.Command) error {
			if cmd.Args().Len() == 0 {
				return fmt.Errorf("id argument required")
			}
			id := cmd.Args().First()
			cfg := mustLoadConfig()
			client := cbclient.NewClient(cfg.ServerURL, cfg.Token)
			if err := client.DeleteFile(id); err != nil {
				return err
			}
			log.Info().Str("id", id).Msg("Deleted")
			return nil
		},
	}
}
