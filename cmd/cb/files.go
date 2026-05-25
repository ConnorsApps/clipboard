package main

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/urfave/cli/v3"
)

const (
	downloadMaxRetries = 3
	downloadBaseDelay  = time.Second
	downloadMaxDelay   = 30 * time.Second
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
		Name:    "files",
		Aliases: []string{"f", "file"},
		Usage:   "Manage files on the clipboard server",
		Commands: []*cli.Command{
			filesListCommand(),
			filesDownloadCommand(),
			filesUploadCommand(),
			filesClearCommand(),
		},
	}
}
