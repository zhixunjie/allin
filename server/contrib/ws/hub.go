package ws

import (
	"encoding/json"
	"log/slog"
	"sync"
)

// RoomConn 管理单个房间的所有 WebSocket 客户端。
// 它是客户端 send 通道的唯一写入者，防止数据竞争。
type RoomConn struct {
	roomCode string

	// 以 userID 为键的已注册客户端。
	clients map[string]*Client

	// 来自客户端的入站消息（由游戏引擎处理）。
	Inbound chan InboundMessage

	// 注册管理通道。
	register   chan *Client
	unregister chan *Client

	mu sync.RWMutex
}

// InboundMessage 将客户端命令与发送者元数据包装在一起。
type InboundMessage struct {
	SenderID    string
	DisplayName string
	Env         CmdEnvelope
}

// NewRoomConn 为给定房间创建一个新的 RoomConn。
func NewRoomConn(roomCode string) *RoomConn {
	return &RoomConn{
		roomCode:   roomCode,
		clients:    make(map[string]*Client),
		Inbound:    make(chan InboundMessage, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

// Run 启动 RoomConn 事件循环。应在专用 goroutine 中调用。
func (rc *RoomConn) Run() {
	for {
		select {
		case client := <-rc.register:
			rc.mu.Lock()
			rc.clients[client.UserID] = client
			rc.mu.Unlock()
			slog.Info("ws: client registered", "room", rc.roomCode, "user", client.UserID)

		case client := <-rc.unregister:
			rc.mu.Lock()
			if existing, ok := rc.clients[client.UserID]; ok && existing == client {
				delete(rc.clients, client.UserID)
				close(client.send)
			}
			rc.mu.Unlock()
			slog.Info("ws: client unregistered", "room", rc.roomCode, "user", client.UserID)

			// 通知游戏引擎有玩家断开连接。
			rc.Inbound <- InboundMessage{
				SenderID: client.UserID,
				Env:      CmdEnvelope{Type: CmdDisconnect},
			}
		}
	}
}

// Broadcast 向房间内所有已连接的客户端发送消息。
func (rc *RoomConn) Broadcast(env Envelope) {
	data, err := json.Marshal(env)
	if err != nil {
		slog.Error("ws: broadcast marshal", "err", err)
		return
	}
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	for _, client := range rc.clients {
		select {
		case client.send <- data:
		default:
			slog.Warn("ws: client send buffer full, dropping message", "user", client.UserID)
		}
	}
}

// SendTo 向一个特定客户端发送消息（例如手牌）。
func (rc *RoomConn) SendTo(userID string, env Envelope) {
	data, err := json.Marshal(env)
	if err != nil {
		slog.Error("ws: send marshal", "err", err)
		return
	}
	rc.mu.RLock()
	client, ok := rc.clients[userID]
	rc.mu.RUnlock()
	if !ok {
		return
	}
	select {
	case client.send <- data:
	default:
		slog.Warn("ws: client send buffer full", "user", userID)
	}
}

// ClientCount 返回已连接的客户端数量。
func (rc *RoomConn) ClientCount() int {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	return len(rc.clients)
}

// IsConnected 报告用户当前是否有活跃连接。
func (rc *RoomConn) IsConnected(userID string) bool {
	rc.mu.RLock()
	_, ok := rc.clients[userID]
	rc.mu.RUnlock()
	return ok
}

// DisplayName 返回已连接用户的显示名称，未连接则返回空字符串。
func (rc *RoomConn) DisplayName(userID string) string {
	rc.mu.RLock()
	c, ok := rc.clients[userID]
	rc.mu.RUnlock()
	if !ok {
		return ""
	}
	return c.DisplayName
}
