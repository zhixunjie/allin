package user

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"
	"unicode"

	"github.com/allin/server/internal/auth"
	"github.com/google/uuid"
)

// Handler bundles the HTTP handlers for user endpoints.
type Handler struct {
	jwtSecret string
}

func NewHandler(jwtSecret string) *Handler {
	return &Handler{jwtSecret: jwtSecret}
}

// Register handles POST /api/auth/register
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username    string `json:"username"`
		Password    string `json:"password"`
		DisplayName string `json:"display_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.Username = strings.TrimSpace(req.Username)
	req.DisplayName = strings.TrimSpace(req.DisplayName)

	if err := validateRegister(req.Username, req.Password, req.DisplayName); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server error")
		return
	}

	u := &User{
		ID:           uuid.New().String(),
		Username:     req.Username,
		PasswordHash: hash,
		DisplayName:  req.DisplayName,
		ChipBalance:  10000,
		CreatedAt:    time.Now(),
	}
	if err := Create(u); err != nil {
		if err == ErrUsernameTaken {
			writeError(w, http.StatusConflict, "username already taken")
			return
		}
		writeError(w, http.StatusInternalServerError, "server error")
		return
	}

	token, err := auth.IssueToken(h.jwtSecret, u.ID, u.DisplayName)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server error")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"token": token, "user": u})
}

// Login handles POST /api/auth/login
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	u, err := GetByUsername(req.Username)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid username or password")
		return
	}
	if !auth.CheckPassword(req.Password, u.PasswordHash) {
		writeError(w, http.StatusUnauthorized, "invalid username or password")
		return
	}

	token, err := auth.IssueToken(h.jwtSecret, u.ID, u.DisplayName)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"token": token, "user": u})
}

// Me handles GET /api/me
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromCtx(r.Context())
	u, err := GetByID(userID)
	if err != nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}
	writeJSON(w, http.StatusOK, u)
}

// --- helpers ---

func validateRegister(username, password, displayName string) error {
	if len(username) < 3 || len(username) > 32 {
		return errorf("username must be 3-32 characters")
	}
	for _, r := range username {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' {
			return errorf("username may only contain letters, digits and underscores")
		}
	}
	if len(password) < 6 {
		return errorf("password must be at least 6 characters")
	}
	if len(displayName) < 1 || len(displayName) > 32 {
		return errorf("display_name must be 1-32 characters")
	}
	return nil
}

type appError struct{ msg string }

func (e *appError) Error() string { return e.msg }
func errorf(msg string) error     { return &appError{msg: msg} }

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
