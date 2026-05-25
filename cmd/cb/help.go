package main

import (
	"fmt"
	"os"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/urfave/cli/v3"
	"golang.org/x/term"
)

var headingStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Bold(true)

func init() {
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		return
	}
	cli.RootCommandHelpTemplate = colorizeHeadings(cli.RootCommandHelpTemplate)
	cli.CommandHelpTemplate = colorizeHeadings(cli.CommandHelpTemplate)
	cli.SubcommandHelpTemplate = colorizeHeadings(cli.SubcommandHelpTemplate)
}

// colorizeHeadings wraps urfave/cli's section headings in ANSI escapes.
// Two passes via sentinels keep prefix-overlapping labels (OPTIONS / GLOBAL OPTIONS)
// from being double-wrapped.
func colorizeHeadings(tmpl string) string {
	headings := []string{
		"GLOBAL OPTIONS:",
		"DESCRIPTION:",
		"COPYRIGHT:",
		"CATEGORY:",
		"COMMANDS:",
		"VERSION:",
		"OPTIONS:",
		"USAGE:",
		"NAME:",
		"AUTHOR",
	}
	type pair struct{ sentinel, styled string }
	pairs := make([]pair, 0, len(headings))
	for i, h := range headings {
		s := fmt.Sprintf("\x00H%d\x00", i)
		tmpl = strings.ReplaceAll(tmpl, h, s)
		pairs = append(pairs, pair{s, headingStyle.Render(h)})
	}
	for _, p := range pairs {
		tmpl = strings.ReplaceAll(tmpl, p.sentinel, p.styled)
	}
	return tmpl
}
