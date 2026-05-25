package main

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/ConnorsApps/clipboard/pkg/cbclient"
	"github.com/gorilla/websocket"
	"github.com/urfave/cli/v3"
)

type wsContentMsg struct{ content string }
type wsErrorMsg struct{ err error }
type wsConnectedMsg struct{}
type wsReconnectingMsg struct{}
type submitResultMsg struct{ err error }

type liveMode int

const (
	liveModeView liveMode = iota
	liveModeEdit
)

const (
	liveEditHeight   = 5
	liveStatusHeight = 1
	liveHintHeight   = 1
)

type liveModel struct {
	viewport   viewport.Model
	textarea   textarea.Model
	mode       liveMode
	content    string
	lastUpdate time.Time
	connStatus string
	statusMsg  string
	width      int
	height     int
	client     *cbclient.Client
}

func newLiveModel(client *cbclient.Client) liveModel {
	ta := textarea.New()
	ta.Placeholder = "Type new clipboard content..."
	ta.CharLimit = 0
	ta.ShowLineNumbers = false
	return liveModel{
		viewport:   viewport.New(),
		textarea:   ta,
		mode:       liveModeView,
		connStatus: "connecting",
		client:     client,
	}
}

func (m liveModel) Init() tea.Cmd {
	return nil
}

func (m liveModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m = m.recalcLayout()
		return m, nil

	case wsConnectedMsg:
		m.connStatus = "connected"
		m.statusMsg = ""
		return m, nil

	case wsReconnectingMsg:
		m.connStatus = "reconnecting"
		return m, nil

	case wsErrorMsg:
		m.connStatus = "disconnected"
		m.statusMsg = msg.err.Error()
		return m, nil

	case wsContentMsg:
		m.content = msg.content
		m.lastUpdate = time.Now()
		m.viewport.SetContent(msg.content)
		m.viewport.GotoBottom()
		return m, nil

	case submitResultMsg:
		if msg.err != nil {
			m.statusMsg = "error: " + msg.err.Error()
		} else {
			m.statusMsg = "sent!"
			m.mode = liveModeView
			m.textarea.Reset()
			m = m.recalcLayout()
		}
		return m, nil

	case tea.KeyPressMsg:
		return m.handleKey(msg)
	}

	var cmd tea.Cmd
	if m.mode == liveModeEdit {
		m.textarea, cmd = m.textarea.Update(msg)
	} else {
		m.viewport, cmd = m.viewport.Update(msg)
	}
	return m, cmd
}

func (m liveModel) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch m.mode {
	case liveModeView:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "e", "i":
			m.mode = liveModeEdit
			m.textarea.SetValue(m.content)
			m = m.recalcLayout()
			return m, m.textarea.Focus()
		case "ctrl+d":
			m.statusMsg = "clearing..."
			return m, m.submitCmd("")
		default:
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			return m, cmd
		}

	case liveModeEdit:
		switch msg.String() {
		case "esc":
			m.mode = liveModeView
			m.textarea.Blur()
			m.textarea.Reset()
			m.statusMsg = ""
			m = m.recalcLayout()
			return m, nil
		case "ctrl+s":
			content := m.textarea.Value()
			m.statusMsg = "sending..."
			return m, m.submitCmd(content)
		default:
			var cmd tea.Cmd
			m.textarea, cmd = m.textarea.Update(msg)
			return m, cmd
		}
	}
	return m, nil
}

func (m liveModel) submitCmd(content string) tea.Cmd {
	return func() tea.Msg {
		return submitResultMsg{err: m.client.SetClipboard(content)}
	}
}

func (m liveModel) recalcLayout() liveModel {
	editZone := 0
	if m.mode == liveModeEdit {
		editZone = liveEditHeight + liveHintHeight
	}
	vpHeight := m.height - liveStatusHeight - editZone
	if vpHeight < 1 {
		vpHeight = 1
	}
	m.viewport.SetWidth(m.width)
	m.viewport.SetHeight(vpHeight)
	m.textarea.SetWidth(m.width)
	m.textarea.SetHeight(liveEditHeight)
	return m
}

func (m liveModel) View() tea.View {
	var b strings.Builder
	b.WriteString(m.viewport.View())
	b.WriteString("\n")
	b.WriteString(m.liveStatusBar())
	if m.mode == liveModeEdit {
		b.WriteString("\n")
		b.WriteString(m.textarea.View())
		b.WriteString("\n")
		b.WriteString(dimStyle.Render("ctrl+s submit · esc cancel"))
	}
	v := tea.NewView(b.String())
	v.AltScreen = true
	return v
}

func (m liveModel) liveStatusBar() string {
	var connStr string
	switch m.connStatus {
	case "connected":
		connStr = successStyle.Render("● connected")
	case "connecting", "reconnecting":
		connStr = dimStyle.Render("◌ " + m.connStatus)
	default:
		connStr = errStyle.Render("✕ disconnected")
	}

	center := ""
	if m.statusMsg != "" {
		center = "  " + dimStyle.Render(m.statusMsg)
	}

	var right string
	if !m.lastUpdate.IsZero() {
		right = dimStyle.Render("updated " + m.lastUpdate.Format("15:04:05"))
	}
	if m.mode == liveModeView {
		if right != "" {
			right += "  "
		}
		right += dimStyle.Render("[e]dit  [ctrl+d]clear  [q]uit")
	}

	used := lipgloss.Width(connStr) + lipgloss.Width(center) + lipgloss.Width(right)
	pad := m.width - used
	if pad < 0 {
		pad = 0
	}
	return connStr + center + strings.Repeat(" ", pad) + right
}

func liveConnectAndListen(p *tea.Program, wsURL string) {
	const maxBackoff = 30 * time.Second
	backoff := time.Second

	for {
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		if err != nil {
			p.Send(wsReconnectingMsg{})
			time.Sleep(backoff)
			backoff = min(backoff*2, maxBackoff)
			continue
		}
		backoff = time.Second
		p.Send(wsConnectedMsg{})

		for {
			_, data, err := conn.ReadMessage()
			if err != nil {
				conn.Close()
				if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
					return
				}
				p.Send(wsReconnectingMsg{})
				time.Sleep(backoff)
				backoff = min(backoff*2, maxBackoff)
				break
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
			p.Send(wsContentMsg{content: msg.Content})
		}
	}
}

func liveCommand() *cli.Command {
	return &cli.Command{
		Name:  "live",
		Usage: "Live clipboard viewer and editor",
		Action: func(_ context.Context, _ *cli.Command) error {
			cfg := mustLoadConfig()
			client := cbclient.NewClient(cfg.ServerURL, cfg.Token)
			wsURL, err := client.WebSocketURL()
			if err != nil {
				return err
			}
			m := newLiveModel(client)
			p := tea.NewProgram(m)
			go liveConnectAndListen(p, wsURL)
			_, err = p.Run()
			return err
		},
	}
}
