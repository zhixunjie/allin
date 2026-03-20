package ws

import (
	"encoding/json"
	"log/slog"
	"sync"
)

// Hub 管理单个房间的所有 WebSocket 客户端。
// 它是客户端 send 通道的唯一写入者，防止数据竞争。
type Hub struct {
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
	Env         Envelope
}

// NewHub 为给定房间创建一个新的 Hub。
func NewHub(roomCode string) *Hub {
	return &Hub{
		roomCode:   roomCode,
		clients:    make(map[string]*Client),
		Inbound:    make(chan InboundMessage, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

// Run 启动 hub 事件循环。应在专用 goroutine 中调用。
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client.UserID] = client
			h.mu.Unlock()
			slog.Info("ws: client registered", "room", h.roomCode, "user", client.UserID)

		case client := <-h.unregister:
			h.mu.Lock()
			if existing, ok := h.clients[client.UserID]; ok && existing == client {
				delete(h.clients, client.UserID)
				close(client.send)
			}
			h.mu.Unlock()
			slog.Info("ws: client unregistered", "room", h.roomCode, "user", client.UserID)

			// 通知游戏引擎有玩家断开连接。
			h.Inbound <- InboundMessage{
				SenderID: client.UserID,
				Env:      Envelope{Type: "disconnect"},
			}
		}
	}
}

// Broadcast 向房间内所有已连接的客户端发送消息。
func (h *Hub) Broadcast(env Envelope) {
	data, err := json.Marshal(env)
	if err != nil {
		slog.Error("ws: broadcast marshal", "err", err)
		return
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, client := range h.clients {
		select {
		case client.send <- data:
		default:
			slog.Warn("ws: client send buffer full, dropping message", "user", client.UserID)
		}
	}
}

// SendTo 向一个特定客户端发送消息（例如手牌）。
func (h *Hub) SendTo(userID string, env Envelope) {
	data, err := json.Marshal(env)
	if err != nil {
		slog.Error("ws: send marshal", "err", err)
		return
	}
	h.mu.RLock()
	client, ok := h.clients[userID]
	h.mu.RUnlock()
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
func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// IsConnected 报告用户当前是否有活跃连接。
func (h *Hub) IsConnected(userID string) bool {
	h.mu.RLock()
	_, ok := h.clients[userID]
	h.mu.RUnlock()
	return ok
}

// DisplayName 返回已连接用户的显示名称，未连接则返回空字符串。
func (h *Hub) DisplayName(userID string) string {
	h.mu.RLock()
	c, ok := h.clients[userID]
	h.mu.RUnlock()
	if !ok {
		return ""
	}
	return c.DisplayName
}
