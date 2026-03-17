package room

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/allin/server/internal/store"
	"github.com/google/uuid"
)

var (
	ErrNotFound     = errors.New("room not found")
	ErrRoomFull     = errors.New("room is full")
	ErrCodeConflict = errors.New("code generation conflict, retry")
)

// Manager holds all active rooms in memory and handles persistence.
type Manager struct {
	mu    sync.RWMutex
	rooms map[string]*Room // key = room code
}

// NewManager creates a new RoomManager.
func NewManager() *Manager {
	return &Manager{rooms: make(map[string]*Room)}
}

// Create creates a new room, persists it, and registers it in memory.
func (m *Manager) Create(hostUserID string, cfg RoomConfig) (*Room, error) {
	if err := validateConfig(cfg); err != nil {
		return nil, err
	}

	code, err := m.generateUniqueCode()
	if err != nil {
		return nil, err
	}

	r := &Room{
		ID:         uuid.New().String(),
		Code:       code,
		HostUserID: hostUserID,
		Config:     cfg,
		State:      RoomStateLobby,
		CreatedAt:  time.Now(),
	}

	// Persist to room_history
	cfgJSON, _ := json.Marshal(cfg)
	if _, err := store.DB.Exec(
		`INSERT INTO room_history (id, room_code, host_user_id, config_json, started_at)
		 VALUES (?, ?, ?, ?, ?)`,
		r.ID, r.Code, r.HostUserID, cfgJSON, r.CreatedAt,
	); err != nil {
		return nil, fmt.Errorf("room.Create: persist: %w", err)
	}

	m.mu.Lock()
	m.rooms[code] = r
	m.mu.Unlock()

	return r, nil
}

// Get returns the room with the given code.
func (m *Manager) Get(code string) (*Room, error) {
	m.mu.RLock()
	r, ok := m.rooms[code]
	m.mu.RUnlock()
	if !ok {
		return nil, ErrNotFound
	}
	return r, nil
}

// Close marks a room as ended and removes it from memory.
func (m *Manager) Close(code string) {
	m.mu.Lock()
	delete(m.rooms, code)
	m.mu.Unlock()

	store.DB.Exec( //nolint:errcheck
		`UPDATE room_history SET ended_at = NOW() WHERE room_code = ? AND ended_at IS NULL`, code,
	)
}

// generateUniqueCode generates a code not already in use (up to 10 attempts).
func (m *Manager) generateUniqueCode() (string, error) {
	for i := 0; i < 10; i++ {
		code, err := GenerateCode()
		if err != nil {
			return "", err
		}
		m.mu.RLock()
		_, exists := m.rooms[code]
		m.mu.RUnlock()
		if !exists {
			return code, nil
		}
	}
	return "", ErrCodeConflict
}

func validateConfig(cfg RoomConfig) error {
	if cfg.SmallBlind <= 0 {
		return errors.New("small_blind must be positive")
	}
	if cfg.BigBlind != cfg.SmallBlind*2 {
		return errors.New("big_blind must be 2x small_blind")
	}
	if cfg.MinBuyIn < cfg.BigBlind*10 {
		return errors.New("min_buy_in must be at least 10x big_blind")
	}
	if cfg.MaxBuyIn < cfg.MinBuyIn {
		return errors.New("max_buy_in must be >= min_buy_in")
	}
	if cfg.MaxPlayers < 2 || cfg.MaxPlayers > 9 {
		return errors.New("max_players must be between 2 and 9")
	}
	if cfg.ActionTimeSec == 0 {
		cfg.ActionTimeSec = 30
	}
	if cfg.ActionTimeSec < 5 || cfg.ActionTimeSec > 120 {
		return errors.New("action_time_sec must be between 5 and 120")
	}
	return nil
}
