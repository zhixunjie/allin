package room

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/allin/server/base/biz/dao"
	"github.com/google/uuid"
)

var (
	ErrNotFound     = errors.New("room not found")
	ErrRoomFull     = errors.New("room is full")
	ErrCodeConflict = errors.New("code generation conflict, retry")
)

// Manager holds all active rooms in memory.
type Manager struct {
	mu    sync.RWMutex
	rooms map[string]*Room // key = room code
}

// NewManager creates a new Manager.
func NewManager() *Manager {
	return &Manager{rooms: make(map[string]*Room)}
}

// Create validates config, generates a unique code, persists to DB, and registers in memory.
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

	cfgJSON, _ := json.Marshal(r.Config)
	if err := dao.RoomDao.Persist(r.ID, r.Code, r.HostUserID, cfgJSON, r.CreatedAt); err != nil {
		return nil, fmt.Errorf("room.Create: %w", err)
	}

	m.mu.Lock()
	m.rooms[code] = r
	m.mu.Unlock()

	return r, nil
}

// Get returns the room for the given code.
func (m *Manager) Get(code string) (*Room, error) {
	m.mu.RLock()
	r, ok := m.rooms[code]
	m.mu.RUnlock()
	if !ok {
		return nil, ErrNotFound
	}
	return r, nil
}

// Close removes the room from memory and marks it ended in DB.
func (m *Manager) Close(code string) {
	m.mu.Lock()
	delete(m.rooms, code)
	m.mu.Unlock()

	dao.RoomDao.MarkEnded(code)
}

// generateUniqueCode generates an unused room code (up to 10 attempts).
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

// StartGC starts a background goroutine that removes rooms idle longer than idleTimeout.
func (m *Manager) StartGC(interval, idleTimeout time.Duration, clientCount func(string) int) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			m.gc(idleTimeout, clientCount)
		}
	}()
}

func (m *Manager) gc(idleTimeout time.Duration, clientCount func(string) int) {
	m.mu.RLock()
	var toClose []string
	for code, rm := range m.rooms {
		if clientCount(code) == 0 && rm.IdleDuration() > idleTimeout {
			toClose = append(toClose, code)
		}
	}
	m.mu.RUnlock()

	for _, code := range toClose {
		m.Close(code)
		slog.Info("room: GC removed idle room", "code", code)
	}
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
	if cfg.BotCount < 0 || cfg.BotCount >= cfg.MaxPlayers {
		return errors.New("bot_count must be >= 0 and < max_players")
	}
	switch cfg.BotStyle {
	case "", "mixed", "aggressive", "passive", "random":
		// valid
	default:
		return errors.New("bot_style must be mixed, aggressive, passive, or random")
	}
	if cfg.ActionTimeSec == 0 {
		cfg.ActionTimeSec = 30
	}
	if cfg.ActionTimeSec < 5 || cfg.ActionTimeSec > 120 {
		return errors.New("action_time_sec must be between 5 and 120")
	}
	return nil
}
