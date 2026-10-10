package clipboard

import (
	"fmt"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func serve(t *testing.T, m *Manager, files func(string) interface{}) *httptest.Server {
	getUserID := func(string) (string, bool, error) { return "u", true, nil }
	srv := httptest.NewServer(m.HandleWebSocket(getUserID, files))
	t.Cleanup(srv.Close)
	return srv
}

func dial(t *testing.T, srv *httptest.Server) *websocket.Conn {
	t.Helper()
	conn, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(srv.URL, "http"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	return conn
}

// next returns the next message of the given type
func next(t *testing.T, conn *websocket.Conn, typ WSMessageType) WSMessage {
	t.Helper()
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	for {
		var msg WSMessage
		if err := conn.ReadJSON(&msg); err != nil {
			t.Fatal(err)
		}
		if msg.Type == typ {
			return msg
		}
	}
}

func TestSync(t *testing.T) {
	m := NewManager()
	m.GetOrCreate("u").SetContent("hello")
	srv := serve(t, m, func(string) interface{} { return []string{"a"} })
	sender, other := dial(t, srv), dial(t, srv)
	conns := []*websocket.Conn{sender, other}

	for _, c := range conns {
		if msg := next(t, c, "content"); msg.Content != "hello" || msg.Version != 1 {
			t.Errorf("initial content %q v%d", msg.Content, msg.Version)
		}
		if files := fmt.Sprint(next(t, c, FilesList).Files); files != "[a]" {
			t.Errorf("initial files %s", files)
		}
	}

	// Changes reach every client, sender included, with increasing versions
	for i, text := range []string{"one", "two"} {
		sender.WriteJSON(WSMessage{Type: Update, Content: text})
		for _, c := range conns {
			if msg := next(t, c, "content"); msg.Content != text || msg.Version != uint64(i+2) {
				t.Errorf("got %q v%d, want %q v%d", msg.Content, msg.Version, text, i+2)
			}
		}
	}
}

// gorilla panics on concurrent writes to one connection
func TestConcurrentBroadcasts(t *testing.T) {
	m := NewManager()
	srv := serve(t, m, nil)
	s := m.GetOrCreate("u")
	const total = 400

	var readers sync.WaitGroup
	for range 2 {
		conn := dial(t, srv)
		next(t, conn, "content") // registered
		readers.Add(1)
		go func() {
			defer readers.Done()
			for range total {
				if _, _, err := conn.ReadMessage(); err != nil {
					t.Error(err)
					return
				}
			}
		}()
	}

	var writers sync.WaitGroup
	for i := range 8 {
		writers.Add(1)
		go func() {
			defer writers.Done()
			for range total / 8 {
				if i%2 == 0 {
					s.UpdateContent("x")
				} else {
					s.BroadcastFilesList(nil)
				}
			}
		}()
	}
	writers.Wait()
	readers.Wait()
}
