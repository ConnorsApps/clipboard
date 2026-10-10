package clipboard

import (
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

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
	for {
		s.clientsMu.RLock()
		n := len(s.clients)
		s.clientsMu.RUnlock()
		if n == 2 {
			break
		}
	}

	var writers sync.WaitGroup
	for i := range broadcasters {
		writers.Add(1)
		go func() {
			defer writers.Done()
			for range perBroadcaster {
				if i%2 == 0 {
					s.Broadcast(WSMessage{Type: "content", Content: "x"}, nil)
				} else {
					s.BroadcastFilesList([]string{"f"})
				}
			}
		}()
	}
	writers.Wait()
	readers.Wait()
}
