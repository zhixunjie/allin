package ws

import (
	"log/slog"
	"net/http"
	"strings"
	"sync"

	"github.com/allin/server/internal/auth"
	"github.com/allin/server/internal/room"
)

// EngineStarter is a function called when a new hub is created for a room.
// The game engine should be created and started inside this callback.
type EngineStarter func(hub *Hub, rm *room.Room)

// Handler handles WebSocket upgrade requests.
type Handler struct {
	roomManager   *room.Manager
	jwtSecret     string
	engineStarter EngineStarter

	hubsMu sync.RWMutex
	hubs   map[string]*Hub // key = room code
}

func NewHandler(roomManager *room.Manager, jwtSecret string) *Handler {
	return &Handler{
		roomManager: roomManager,
		jwtSecret:   jwtSecret,
		hubs:        make(map[string]*Hub),
	}
}

// SetEngineStarter registers the callback used to start a game engine when a hub is created.
func (h *Handler) SetEngineStarter(fn EngineStarter) {
	h.engineStarter = fn
}

// ServeWS handles GET /api/ws?room=CODE
func (h *Handler) ServeWS(w http.ResponseWriter, r *http.Request) {
	// 1. Authenticate
	token := extractToken(r)
	if token == "" {
		http.Error(w, "missing token", http.StatusUnauthorized)
		return
	}
	claims, err := auth.ParseToken(h.jwtSecret, token)
	if err != nil {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}

	// 2. Look up room
	code := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("room")))
	rm, err := h.roomManager.Get(code)
	if err != nil {
		http.Error(w, "room not found", http.StatusNotFound)
		return
	}

	// 3. Get or create hub (and engine) for this room
	hub := h.getOrCreateHub(rm)

	// 4. Upgrade to WebSocket
	client, err := NewClient(hub, w, r, claims.UserID, claims.DisplayName)
	if err != nil {
		slog.Error("ws: upgrade failed", "err", err)
		return
	}

	// 5. Start pumps — WritePump in goroutine, ReadPump blocks until close
	go client.WritePump()
	client.ReadPump()
}

// getOrCreateHub returns the hub for the given room, creating it if needed.
func (h *Handler) getOrCreateHub(rm *room.Room) *Hub {
	h.hubsMu.RLock()
	hub, ok := h.hubs[rm.Code]
	h.hubsMu.RUnlock()
	if ok {
		return hub
	}

	h.hubsMu.Lock()
	defer h.hubsMu.Unlock()
	if hub, ok = h.hubs[rm.Code]; ok {
		return hub
	}

	hub = NewHub(rm.Code)
	go hub.Run()
	h.hubs[rm.Code] = hub
	slog.Info("ws: hub created", "room", rm.Code)

	if h.engineStarter != nil {
		h.engineStarter(hub, rm)
	}
	return hub
}

// RemoveHub removes the hub for a room that has been closed.
func (h *Handler) RemoveHub(code string) {
	h.hubsMu.Lock()
	delete(h.hubs, code)
	h.hubsMu.Unlock()
}

// ClientCount returns the number of connected clients for the given room.
func (h *Handler) ClientCount(code string) int {
	h.hubsMu.RLock()
	hub, ok := h.hubs[code]
	h.hubsMu.RUnlock()
	if !ok {
		return 0
	}
	return hub.ClientCount()
}

// extractToken pulls JWT from Authorization header or ?token= query param.
func extractToken(r *http.Request) string {
	if hdr := r.Header.Get("Authorization"); strings.HasPrefix(hdr, "Bearer ") {
		return strings.TrimPrefix(hdr, "Bearer ")
	}
	return r.URL.Query().Get("token")
}
