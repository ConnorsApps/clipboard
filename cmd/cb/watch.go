package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
	"github.com/urfave/cli/v3"
)

func watchCommand() *cli.Command {
	return &cli.Command{
		Name:  "watch",
		Usage: "Stream clipboard updates in real time",
		Action: func(_ context.Context, _ *cli.Command) error {
			cfg := mustLoadConfig()

			serverURL, err := url.Parse(cfg.ServerURL)
			if err != nil {
				return fmt.Errorf("invalid server URL: %w", err)
			}
			switch serverURL.Scheme {
			case "http":
				serverURL.Scheme = "ws"
			case "https":
				serverURL.Scheme = "wss"
			}
			serverURL.Path = "/ws"
			serverURL.RawQuery = "token=" + url.QueryEscape(cfg.Token)

			conn, _, err := websocket.DefaultDialer.Dial(serverURL.String(), nil)
			if err != nil {
				return fmt.Errorf("failed to connect: %w", err)
			}
			defer conn.Close()

			fmt.Fprintln(os.Stderr, dimStyle.Render("Watching clipboard (Ctrl-C to stop)..."))

			sig := make(chan os.Signal, 1)
			signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
			go func() {
				<-sig
				fmt.Fprintln(os.Stderr)
				conn.WriteMessage(websocket.CloseMessage,
					websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
				os.Exit(0)
			}()

			for {
				_, data, err := conn.ReadMessage()
				if err != nil {
					if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
						break
					}
					return err
				}

				var msg struct {
					Type    string `json:"type"`
					Content string `json:"content"`
				}
				if err := json.Unmarshal(data, &msg); err != nil {
					continue
				}
				if msg.Type != "content" {
					continue
				}

				ts := dimStyle.Render("── " + time.Now().Format("15:04:05") + " ──")
				fmt.Println(ts)
				fmt.Println(msg.Content)
				if !strings.HasSuffix(msg.Content, "\n") {
					fmt.Println()
				}
			}
			return nil
		},
	}
}
