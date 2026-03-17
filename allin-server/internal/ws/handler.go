package ws

import (
	"log/slog"
	"net/http"
	"strings"
	"sync"

	"github.com/allin/server/internal/auth"
	"github.com/allin/server/internal/room"
)

// Handler handles WebSocket upgrade requests.
type Handler struct {
	roomManager *room.Manager
	jwtSecret   string

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

	// 3. Get or create hub for this room
	hub := h.getOrCreateHub(rm.Code)

	// 4. Upgrade to WebSocket
	client, err := NewClient(hub, w, r, claims.UserID, claims.DisplayName)
	if err != nil {
		slog.Error("ws: upgrade failed", "err", err)
		return
	}

	// 5. Send initial "connected" event to this client only
	env := MustEvent(TypeConnected, ConnectedPayload{
		PlayerID:    claims.UserID,
		DisplayName: claims.DisplayName,
		RoomCode:    rm.Code,
	})
	hub.SendTo(claims.UserID, env)

	// 6. Broadcast to others that this player connected
	hub.Broadcast(MustEvent(TypePlayerJoined, PlayerJoinedPayload{
		PlayerID:    claims.UserID,
		DisplayName: claims.DisplayName,
		SeatIndex:   -1, // not seated yet; game engine assigns seat in Phase 2
		IsReconnect: false,
	}))

	// 7. Start pumps — WritePump in goroutine, ReadPump blocks until close
	go client.WritePump()
	client.ReadPump()
}

// getOrCreateHub returns the hub for the given room code, creating it if needed.
func (h *Handler) getOrCreateHub(code string) *Hub {
	h.hubsMu.RLock()
	hub, ok := h.hubs[code]
	h.hubsMu.RUnlock()
	if ok {
		return hub
	}

	h.hubsMu.Lock()
	defer h.hubsMu.Unlock()
	// Double-check after acquiring write lock
	if hub, ok = h.hubs[code]; ok {
		return hub
	}
	hub = NewHub(code)
	go hub.Run()
	h.hubs[code] = hub
	slog.Info("ws: hub created", "room", code)
	return hub
}

// RemoveHub removes the hub for a room that has been closed.
func (h *Handler) RemoveHub(code string) {
	h.hubsMu.Lock()
	delete(h.hubs, code)
	h.hubsMu.Unlock()
}

// extractToken pulls JWT from Authorization header or ?token= query param.
func extractToken(r *http.Request) string {
	if hdr := r.Header.Get("Authorization"); strings.HasPrefix(hdr, "Bearer ") {
		return strings.TrimPrefix(hdr, "Bearer ")
	}
	return r.URL.Query().Get("token")
}
