package ws

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/allin/server/contrib/ws/protocol"
	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 4096
)

// Client 是与房间 RoomConn 关联的单个 WebSocket 连接。
type Client struct {
	rc          *RoomConn       // 所属房间的消息总线
	conn        *websocket.Conn // 底层 WebSocket 连接
	send        chan []byte     // 出站消息队列，由 WritePump 消费
	UserID      string          // 玩家用户 ID
	DisplayName string          // 玩家显示名称
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Origin 验证在 HTTP 服务器层通过 CORS 处理。
		return true
	},
}

// NewClient 将 HTTP 连接升级为 WebSocket 并注册到 RoomConn。
func NewClient(rc *RoomConn, w http.ResponseWriter, r *http.Request, userID, displayName string) (*Client, error) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return nil, err
	}
	c := &Client{
		rc:          rc,
		conn:        conn,
		send:        make(chan []byte, 256),
		UserID:      userID,
		DisplayName: displayName,
	}
	rc.register <- c
	return c, nil
}

// ReadPump 从 WebSocket 连接读取消息并转发到 RoomConn。
// 此方法应在独立的 goroutine 中调用。
func (c *Client) ReadPump() {
	defer func() {
		c.rc.unregister <- c
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

		var env protocol.CmdEnvelope
		if err := json.Unmarshal(data, &env); err != nil {
			slog.Warn("ws: invalid envelope", "user", c.UserID, "err", err)
			continue
		}

		select {
		case c.rc.Inbound <- protocol.InboundMessage{
			SenderID:    c.UserID,
			DisplayName: c.DisplayName,
			Env:         env,
		}:
		default:
			slog.Warn("ws: inbound channel full, dropping message", "user", c.UserID, "type", env.Type)
		}
	}
}

// WritePump 将出站消息推送到 WebSocket 连接。
// 此方法应在独立的 goroutine 中调用。
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
				// RoomConn 关闭了通道。
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
