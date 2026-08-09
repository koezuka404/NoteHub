package websocket

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	gorillaws "github.com/gorilla/websocket"
)

func TestClient_Close(t *testing.T) {
	client := newTestClient(t, NewHub(), uuid.New(), uuid.New(), uuid.New())
	client.Close()
	client.Close()
}

func TestClient_TrySend(t *testing.T) {
	client := newTestClient(t, NewHub(), uuid.New(), uuid.New(), uuid.New())
	client.TrySend([]byte("hello"))
	select {
	case got := <-client.Send:
		if string(got) != "hello" {
			t.Fatalf("payload = %q", got)
		}
	default:
		t.Fatal("expected message on send channel")
	}
}

func TestClient_TrySendBufferFullCloses(t *testing.T) {
	client := newTestClient(t, NewHub(), uuid.New(), uuid.New(), uuid.New())
	client.Send = make(chan []byte, 1)
	client.Ready.Store(true)
	client.TrySend([]byte("first"))
	client.TrySend([]byte("overflow"))
	if err := client.Conn.WriteMessage(gorillaws.TextMessage, []byte("probe")); err == nil {
		t.Fatal("expected closed connection after full buffer")
	}
}

func TestClient_WritePump(t *testing.T) {
	hub := NewHub()
	clientConn, serverConn := newTestConnPair(t)
	client := &Client{
		ConnectionID: uuid.New(),
		Hub:          hub,
		Conn:         clientConn,
		Send:         make(chan []byte, 4),
	}
	client.Ready.Store(true)

	go client.WritePump()
	client.TrySend([]byte("write-pump"))

	_ = serverConn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, msg, err := serverConn.ReadMessage()
	if err != nil {
		t.Fatalf("ReadMessage: %v", err)
	}
	if string(msg) != "write-pump" {
		t.Fatalf("msg = %q", msg)
	}
	close(client.Send)
	time.Sleep(50 * time.Millisecond)
}

func TestClient_WritePumpPing(t *testing.T) {
	orig := pingPeriodDuration
	pingPeriodDuration = 5 * time.Millisecond
	t.Cleanup(func() { pingPeriodDuration = orig })

	clientConn, _ := newTestConnPair(t)
	client := &Client{
		ConnectionID: uuid.New(),
		Hub:          NewHub(),
		Conn:         clientConn,
		Send:         make(chan []byte, 1),
	}
	go client.WritePump()
	time.Sleep(20 * time.Millisecond)
	close(client.Send)
	time.Sleep(20 * time.Millisecond)
}

func TestClient_ReadPump(t *testing.T) {
	hub := NewHub()
	clientConn, serverConn := newTestConnPair(t)
	client := &Client{
		ConnectionID: uuid.New(),
		UserID:       uuid.New(),
		DocumentID:   uuid.New(),
		WorkspaceID:  uuid.New(),
		Hub:          hub,
		Conn:         clientConn,
		Send:         make(chan []byte, 4),
	}
	client.Ready.Store(true)

	var (
		mu       sync.Mutex
		messages int
		done     = make(chan struct{}, 1)
	)
	onMessage := func(_ *Client, _ []byte) {
		mu.Lock()
		messages++
		mu.Unlock()
	}
	onClose := func(_ *Client) {
		done <- struct{}{}
	}

	go client.ReadPump(onMessage, onClose)
	if err := serverConn.WriteMessage(gorillaws.TextMessage, []byte(`{"type":"ping"}`)); err != nil {
		t.Fatalf("WriteMessage: %v", err)
	}
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	if messages != 1 {
		mu.Unlock()
		t.Fatalf("messages = %d", messages)
	}
	mu.Unlock()

	_ = clientConn.Close()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("expected onClose called")
	}
}

func TestClient_ReadPumpClosedChannel(t *testing.T) {
	hub := NewHub()
	clientConn, _ := newTestConnPair(t)
	client := &Client{
		ConnectionID: uuid.New(),
		DocumentID:   uuid.New(),
		WorkspaceID:  uuid.New(),
		Hub:          hub,
		Conn:         clientConn,
		Send:         make(chan []byte, 1),
	}
	client.Ready.Store(true)

	called := false
	go client.ReadPump(nil, func(_ *Client) { called = true })
	client.Close()
	time.Sleep(50 * time.Millisecond)
	if !called {
		t.Fatal("expected onClose on read error")
	}
}

func TestClient_ReadPumpSetReadDeadlineError(t *testing.T) {
	clientConn, _ := newTestConnPair(t)
	_ = clientConn.Close()
	client := &Client{
		ConnectionID: uuid.New(),
		DocumentID:   uuid.New(),
		Hub:          NewHub(),
		Conn:         clientConn,
		Send:         make(chan []byte, 1),
	}
	client.ReadPump(nil, nil)
}

func TestClient_ReadPumpPongHandler(t *testing.T) {
	clientConn, serverConn := newTestConnPair(t)
	client := &Client{
		ConnectionID: uuid.New(),
		DocumentID:   uuid.New(),
		Hub:          NewHub(),
		Conn:         clientConn,
		Send:         make(chan []byte, 1),
	}

	serverDone := make(chan struct{})
	go func() {
		defer close(serverDone)
		for {
			if _, _, err := serverConn.ReadMessage(); err != nil {
				return
			}
		}
	}()

	go client.ReadPump(nil, nil)
	if err := clientConn.WriteMessage(gorillaws.PingMessage, nil); err != nil {
		t.Fatalf("WriteMessage ping: %v", err)
	}
	time.Sleep(50 * time.Millisecond)
	_ = clientConn.Close()
	_ = serverConn.Close()
	select {
	case <-serverDone:
	case <-time.After(time.Second):
		t.Fatal("server read loop did not exit")
	}
}

var errTestConn = errors.New("connection closed")

func TestClient_WritePumpSetWriteDeadlineError(t *testing.T) {
	orig := connSetWriteDeadline
	connSetWriteDeadline = func(*gorillaws.Conn, time.Time) error { return errTestConn }
	t.Cleanup(func() { connSetWriteDeadline = orig })

	clientConn, _ := newTestConnPair(t)
	client := &Client{
		Hub:  NewHub(),
		Conn: clientConn,
		Send: make(chan []byte, 1),
	}
	go client.WritePump()
	client.Send <- []byte("msg")
	time.Sleep(20 * time.Millisecond)
}

func TestClient_WritePumpWriteMessageError(t *testing.T) {
	orig := connWriteMessage
	connWriteMessage = func(*gorillaws.Conn, int, []byte) error { return errTestConn }
	t.Cleanup(func() { connWriteMessage = orig })

	clientConn, _ := newTestConnPair(t)
	client := &Client{
		Hub:  NewHub(),
		Conn: clientConn,
		Send: make(chan []byte, 1),
	}
	go client.WritePump()
	client.Send <- []byte("msg")
	time.Sleep(20 * time.Millisecond)
}

func TestClient_WritePumpPingErrors(t *testing.T) {
	origPeriod := pingPeriodDuration
	origWrite := connWriteMessage
	pingPeriodDuration = 5 * time.Millisecond
	connWriteMessage = func(c *gorillaws.Conn, messageType int, data []byte) error {
		if messageType == gorillaws.PingMessage {
			return errTestConn
		}
		return origWrite(c, messageType, data)
	}
	t.Cleanup(func() {
		pingPeriodDuration = origPeriod
		connWriteMessage = origWrite
	})

	clientConn, _ := newTestConnPair(t)
	client := &Client{
		Hub:  NewHub(),
		Conn: clientConn,
		Send: make(chan []byte, 1),
	}
	go client.WritePump()
	time.Sleep(20 * time.Millisecond)
	close(client.Send)
}

func TestClient_WritePumpPingSetWriteDeadlineError(t *testing.T) {
	origPeriod := pingPeriodDuration
	origDeadline := connSetWriteDeadline
	pingPeriodDuration = 5 * time.Millisecond
	calls := 0
	connSetWriteDeadline = func(c *gorillaws.Conn, t time.Time) error {
		calls++
		if calls > 1 {
			return errTestConn
		}
		return origDeadline(c, t)
	}
	t.Cleanup(func() {
		pingPeriodDuration = origPeriod
		connSetWriteDeadline = origDeadline
	})

	clientConn, _ := newTestConnPair(t)
	client := &Client{
		Hub:  NewHub(),
		Conn: clientConn,
		Send: make(chan []byte, 1),
	}
	go client.WritePump()
	time.Sleep(20 * time.Millisecond)
	close(client.Send)
}

func TestClient_ReadPumpPongHandlerSetReadDeadlineError(t *testing.T) {
	orig := connSetReadDeadline
	calls := 0
	connSetReadDeadline = func(c *gorillaws.Conn, t time.Time) error {
		calls++
		if calls == 1 {
			return orig(c, t)
		}
		return errTestConn
	}
	t.Cleanup(func() { connSetReadDeadline = orig })

	clientConn, serverConn := newTestConnPair(t)
	client := &Client{
		DocumentID: uuid.New(),
		Hub:        NewHub(),
		Conn:       clientConn,
		Send:       make(chan []byte, 1),
	}

	serverDone := make(chan struct{})
	go func() {
		defer close(serverDone)
		for {
			if _, _, err := serverConn.ReadMessage(); err != nil {
				return
			}
		}
	}()

	go client.ReadPump(nil, nil)
	if err := clientConn.WriteMessage(gorillaws.PingMessage, nil); err != nil {
		t.Fatalf("WriteMessage ping: %v", err)
	}
	time.Sleep(50 * time.Millisecond)
	if calls < 2 {
		t.Fatalf("connSetReadDeadline calls = %d, want at least 2", calls)
	}
	_ = clientConn.Close()
	_ = serverConn.Close()
	<-serverDone
}
