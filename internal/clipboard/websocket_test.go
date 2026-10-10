package clipboard

import (
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func dial(t *testing.T, srv *httptest.Server) *websocket.Conn {
	t.Helper()
	url := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws?token=t"
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatal(err)
	}
	return conn
}

func TestHandleWebSocketSendsInitialState(t *testing.T) {
	m := NewManager()
	m.GetOrCreate("u").SetContent("hello")
	getUserID := func(string) (string, bool, error) { return "u", true, nil }
	filesForUser := func(string) interface{} { return []string{"a"} }
	srv := httptest.NewServer(m.HandleWebSocket(getUserID, filesForUser))
	defer srv.Close()

	conn := dial(t, srv)
	defer conn.Close()

	var first, second struct {
		Type    string   `json:"type"`
		Content string   `json:"content"`
		Files   []string `json:"files"`
	}
	if err := conn.ReadJSON(&first); err != nil {
		t.Fatal(err)
	}
	if err := conn.ReadJSON(&second); err != nil {
		t.Fatal(err)
	}
	if first.Type != "content" || first.Content != "hello" {
		t.Errorf("first message = %+v, want content \"hello\"", first)
	}
	if second.Type != "files_list" || len(second.Files) != 1 || second.Files[0] != "a" {
		t.Errorf("second message = %+v, want files_list [a]", second)
	}
}

// Broadcasts come from many goroutines (each client's reader plus tusd upload
// callbacks). gorilla/websocket panics on concurrent writes to one connection,
// so this would crash without the per-connection write lock.
func TestConcurrentBroadcasts(t *testing.T) {
	m := NewManager()
	getUserID := func(string) (string, bool, error) { return "u", true, nil }
	srv := httptest.NewServer(m.HandleWebSocket(getUserID, func(string) interface{} { return []string{} }))
	defer srv.Close()

	const broadcasters, perBroadcaster = 8, 50
	// Each client receives content + files_list on connect, then every broadcast
	const wantPerClient = 2 + broadcasters*perBroadcaster

	var readers sync.WaitGroup
	for range 2 {
		conn := dial(t, srv)
		defer conn.Close()
		readers.Add(1)
		go func() {
			defer readers.Done()
			for range wantPerClient {
				if _, _, err := conn.ReadMessage(); err != nil {
					t.Error(err)
					return
				}
			}
		}()
	}

	// Wait until both clients are registered before broadcasting
	s := m.GetOrCreate("u")
	deadline := time.Now().Add(5 * time.Second)
	for {
		s.clientsMu.RLock()
		n := len(s.clients)
		s.clientsMu.RUnlock()
		if n == 2 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("only %d clients registered", n)
		}
		time.Sleep(time.Millisecond)
	}

	var writers sync.WaitGroup
	for i := range broadcasters {
		writers.Add(1)
		go func() {
			defer writers.Done()
			for range perBroadcaster {
				if i%2 == 0 {
					s.UpdateContent("x")
				} else {
					s.BroadcastFilesList([]string{"f"})
				}
			}
		}()
	}
	writers.Wait()
	readers.Wait()
}

func TestPingPong(t *testing.T) {
	m := NewManager()
	getUserID := func(string) (string, bool, error) { return "u", true, nil }
	srv := httptest.NewServer(m.HandleWebSocket(getUserID, nil))
	defer srv.Close()

	conn := dial(t, srv)
	defer conn.Close()
	if err := conn.WriteJSON(WSMessage{Type: Ping}); err != nil {
		t.Fatal(err)
	}

	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	for {
		var msg WSMessage
		if err := conn.ReadJSON(&msg); err != nil {
			t.Fatalf("no pong: %v", err)
		}
		if msg.Type == Pong {
			return
		}
	}
}

// Every client, including the sender, gets each change tagged with an increasing version
func TestUpdateEchoesWithVersion(t *testing.T) {
	m := NewManager()
	getUserID := func(string) (string, bool, error) { return "u", true, nil }
	srv := httptest.NewServer(m.HandleWebSocket(getUserID, nil))
	defer srv.Close()

	sender, other := dial(t, srv), dial(t, srv)
	defer sender.Close()
	defer other.Close()

	readContent := func(conn *websocket.Conn) WSMessage {
		t.Helper()
		conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		for {
			var msg WSMessage
			if err := conn.ReadJSON(&msg); err != nil {
				t.Fatal(err)
			}
			if msg.Type == "content" {
				return msg
			}
		}
	}
	readContent(sender) // initial state
	readContent(other)

	for i, text := range []string{"one", "two"} {
		if err := sender.WriteJSON(WSMessage{Type: Update, Content: text}); err != nil {
			t.Fatal(err)
		}
		for name, conn := range map[string]*websocket.Conn{"sender": sender, "other": other} {
			msg := readContent(conn)
			if msg.Content != text || msg.Version != uint64(i+1) {
				t.Errorf("%s got %q v%d, want %q v%d", name, msg.Content, msg.Version, text, i+1)
			}
		}
	}
}
