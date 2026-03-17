package ws

import (
	"encoding/json"
	"log/slog"
	"sync"
)

// Hub manages all WebSocket clients for a single room.
// It is the sole writer to client send channels, preventing data races.
type Hub struct {
	roomCode string

	// Registered clients keyed by userID.
	clients map[string]*Client

	// Inbound messages from clients (processed by the game engine).
	Inbound chan InboundMessage

	// Channels for registration management.
	register   chan *Client
	unregister chan *Client

	mu sync.RWMutex
}

// InboundMessage wraps a client command with sender metadata.
type InboundMessage struct {
	SenderID    string
	DisplayName string
	Env         Envelope
}

// NewHub creates a new Hub for the given room.
func NewHub(roomCode string) *Hub {
	return &Hub{
		roomCode:   roomCode,
		clients:    make(map[string]*Client),
		Inbound:    make(chan InboundMessage, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

// Run starts the hub event loop. Call in a dedicated goroutine.
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

			// Notify game engine that a player disconnected.
			h.Inbound <- InboundMessage{
				SenderID: client.UserID,
				Env:      Envelope{Type: "disconnect"},
			}
		}
	}
}

// Broadcast sends a message to all connected clients in the room.
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

// SendTo sends a message to one specific client (e.g., hole cards).
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

// ClientCount returns the number of connected clients.
func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// IsConnected reports whether a user currently has an active connection.
func (h *Hub) IsConnected(userID string) bool {
	h.mu.RLock()
	_, ok := h.clients[userID]
	h.mu.RUnlock()
	return ok
}
