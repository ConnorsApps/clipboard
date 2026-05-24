package main

import (
	"context"
	"fmt"
	"os"

	"charm.land/lipgloss/v2"
	"github.com/urfave/cli/v3"
)

var (
	errStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true)
	successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	dimStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
)

func fatal(msg string) {
	fmt.Fprintln(os.Stderr, errStyle.Render("Error: ")+msg)
	os.Exit(1)
}

func mustLoadConfig() *Config {
	cfg, err := loadConfig()
	if err != nil {
		fatal("not logged in — run 'cb login' first")
	}
	return cfg
}

func main() {
	cmd := &cli.Command{
		Name:                  "cb",
		Usage:                 "clipboard CLI",
		Version:               "0.1.0",
		EnableShellCompletion: true,
		Commands: []*cli.Command{
			loginCommand(),
			getCommand(),
			setCommand(),
			watchCommand(),
			logoutCommand(),
			filesCommand(),
			{
				Name:   "version",
				Hidden: true,
				Action: func(_ context.Context, cmd *cli.Command) error {
					fmt.Println(cmd.Root().Version)
					return nil
				},
			},
		},
		ExitErrHandler: func(_ context.Context, _ *cli.Command, err error) {
			if err != nil {
				fmt.Fprintln(os.Stderr, errStyle.Render("Error: ")+err.Error())
				os.Exit(1)
			}
		},
	}

	cmd.Run(context.Background(), os.Args)
}
