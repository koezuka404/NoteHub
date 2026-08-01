package websocket

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	gorillaws "github.com/gorilla/websocket"
)

const (
	writeWait  = 10 * time.Second
	pongWait   = 60 * time.Second
	pingPeriod = (pongWait * 9) / 10
	maxMessage = 512 * 1024
)

type Client struct {
	ConnectionID uuid.UUID
	UserID       uuid.UUID
	DocumentID   uuid.UUID
	WorkspaceID  uuid.UUID
	Hub          *Hub
	Conn         *gorillaws.Conn
	Send         chan []byte
	Ready        atomic.Bool

	closeOnce sync.Once
}

func (c *Client) Close() {
	c.closeOnce.Do(func() {
		_ = c.Conn.Close()
	})
}

func (c *Client) TrySend(payload []byte) {
	select {
	case c.Send <- payload:
	default:
		c.Close()
	}
}

func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			if err := c.Conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				return
			}
			if !ok {
				_ = c.Conn.WriteMessage(gorillaws.CloseMessage, []byte{})
				return
			}
			if err := c.Conn.WriteMessage(gorillaws.TextMessage, message); err != nil {
				return
			}
		case <-ticker.C:
			if err := c.Conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				return
			}
			if err := c.Conn.WriteMessage(gorillaws.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *Client) ReadPump(onMessage func(*Client, []byte), onClose func(*Client)) {
	defer func() {
		c.Hub.Unregister(c)
		if onClose != nil {
			onClose(c)
		}
		c.Close()
	}()

	if err := c.Conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		return
	}
	c.Conn.SetPongHandler(func(string) error {
		return c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	})
	c.Conn.SetReadLimit(maxMessage)

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			return
		}
		onMessage(c, message)
	}
}
