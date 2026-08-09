package websocket

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	gorillaws "github.com/gorilla/websocket"
)

func testNow() time.Time {
	return time.Date(2026, 8, 7, 12, 0, 0, 0, time.UTC)
}

func newTestConnPair(t *testing.T) (*gorillaws.Conn, *gorillaws.Conn) {
	t.Helper()
	upgrader := gorillaws.Upgrader{}
	var (
		mu         sync.Mutex
		serverConn *gorillaws.Conn
		ready      = make(chan struct{})
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		mu.Lock()
		serverConn = conn
		mu.Unlock()
		close(ready)
		<-r.Context().Done()
		_ = conn.Close()
	}))
	t.Cleanup(server.Close)

	clientConn, _, err := gorillaws.DefaultDialer.Dial("ws"+server.URL[4:], nil)
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}
	t.Cleanup(func() { _ = clientConn.Close() })

	select {
	case <-ready:
	case <-time.After(2 * time.Second):
		t.Fatal("server websocket not ready")
	}

	mu.Lock()
	defer mu.Unlock()
	if serverConn == nil {
		t.Fatal("server websocket connection missing")
	}
	t.Cleanup(func() { _ = serverConn.Close() })
	return clientConn, serverConn
}

func newTestClient(t *testing.T, hub *Hub, documentID, workspaceID, userID uuid.UUID) *Client {
	t.Helper()
	conn, _ := newTestConnPair(t)
	client := &Client{
		ConnectionID: uuid.New(),
		UserID:       userID,
		DocumentID:   documentID,
		WorkspaceID:  workspaceID,
		Hub:          hub,
		Conn:         conn,
		Send:         make(chan []byte, 16),
	}
	client.Ready.Store(true)
	return client
}
