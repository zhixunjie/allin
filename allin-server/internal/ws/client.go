package ws

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 4096
)

// Client is a single WebSocket connection associated with a room hub.
type Client struct {
	hub         *Hub
	conn        *websocket.Conn
	send        chan []byte
	UserID      string
	DisplayName string
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Origin validation is handled at the HTTP server level via CORS.
		return true
	},
}

// NewClient upgrades an HTTP connection to WebSocket and registers it with the hub.
func NewClient(hub *Hub, w http.ResponseWriter, r *http.Request, userID, displayName string) (*Client, error) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return nil, err
	}
	c := &Client{
		hub:         hub,
		conn:        conn,
		send:        make(chan []byte, 256),
		UserID:      userID,
		DisplayName: displayName,
	}
	hub.register <- c
	return c, nil
}

// ReadPump reads messages from the WebSocket connection and forwards them to the hub.
// This method should be called in its own goroutine.
func (c *Client) ReadPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				slog.Warn("ws: read error", "user", c.UserID, "err", err)
			}
			break
		}

		var env Envelope
		if err := jsonUnmarshal(data, &env); err != nil {
			slog.Warn("ws: invalid envelope", "user", c.UserID, "err", err)
			continue
		}

		c.hub.Inbound <- InboundMessage{
			SenderID:    c.UserID,
			DisplayName: c.DisplayName,
			Env:         env,
		}
	}
}

// WritePump pumps outbound messages to the WebSocket connection.
// This method should be called in its own goroutine.
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Hub closed the channel.
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// jsonUnmarshal wraps json.Unmarshal for readability.
func jsonUnmarshal(data []byte, v any) error {
	return json.Unmarshal(data, v)
}
