package room

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/allin/server/internal/auth"
)

// Handler bundles HTTP handlers for room endpoints.
type Handler struct {
	manager *Manager
}

func NewHandler(manager *Manager) *Handler {
	return &Handler{manager: manager}
}

// Create handles POST /api/rooms
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	hostUserID := auth.UserIDFromCtx(r.Context())

	var cfg RoomConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if cfg.ActionTimeSec == 0 {
		cfg.ActionTimeSec = 30
	}

	room, err := h.manager.Create(hostUserID, cfg)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, room)
}

// Get handles GET /api/rooms/{code}
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	code := strings.ToUpper(strings.TrimSpace(r.PathValue("code")))
	room, err := h.manager.Get(code)
	if err != nil {
		writeError(w, http.StatusNotFound, "room not found")
		return
	}
	writeJSON(w, http.StatusOK, room)
}

// --- helpers ---

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
