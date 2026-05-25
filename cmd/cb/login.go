package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v3"
	"golang.org/x/term"
)

func loginCommand() *cli.Command {
	return &cli.Command{
		Name:    "login",
		Aliases: []string{"li"},
		Usage:   "Authenticate with a clipboard server",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "server",
				Usage: "Server URL (e.g. http://localhost:8080)",
			},
		},
		Action: func(_ context.Context, cmd *cli.Command) error {
			serverURL := cmd.String("server")
			reader := bufio.NewReader(os.Stdin)

			if serverURL == "" {
				fmt.Fprint(os.Stderr, "Server URL: ")
				url, _ := reader.ReadString('\n')
				serverURL = strings.TrimSpace(url)
			}
			serverURL = strings.TrimRight(serverURL, "/")

			fmt.Fprint(os.Stderr, "Password: ")
			passwordBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
			fmt.Fprintln(os.Stderr)
			if err != nil {
				return fmt.Errorf("failed to read password: %w", err)
			}

			body, _ := json.Marshal(map[string]string{"password": string(passwordBytes)})
			resp, err := http.Post(serverURL+"/api/login", "application/json", bytes.NewReader(body))
			if err != nil {
				return fmt.Errorf("login request failed: %w", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode == 401 {
				return fmt.Errorf("incorrect password")
			}
			if resp.StatusCode != 200 {
				return fmt.Errorf("login failed (HTTP %d)", resp.StatusCode)
			}

			var result struct {
				Token string `json:"token"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				return fmt.Errorf("invalid response from server: %w", err)
			}

			if err := saveConfig(&Config{ServerURL: serverURL, Token: result.Token}); err != nil {
				return fmt.Errorf("failed to save config: %w", err)
			}

			log.Info().Msg("Logged in successfully")
			return nil
		},
	}
}
