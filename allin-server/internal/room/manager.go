package room

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
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

// Manager 在内存中保存所有活跃房间并处理持久化。
type Manager struct {
	mu    sync.RWMutex
	rooms map[string]*Room // 键 = 房间码
}

// NewManager 创建一个新的 RoomManager。
func NewManager() *Manager {
	return &Manager{rooms: make(map[string]*Room)}
}

// Create 创建新房间，持久化并注册到内存。
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

	// 持久化到 room_history
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

// Get 返回给定房间码的房间。
func (m *Manager) Get(code string) (*Room, error) {
	m.mu.RLock()
	r, ok := m.rooms[code]
	m.mu.RUnlock()
	if !ok {
		return nil, ErrNotFound
	}
	return r, nil
}

// Close 将房间标记为已结束并从内存中移除。
func (m *Manager) Close(code string) {
	m.mu.Lock()
	delete(m.rooms, code)
	m.mu.Unlock()

	store.DB.Exec( //nolint:errcheck
		`UPDATE room_history SET ended_at = NOW() WHERE room_code = ? AND ended_at IS NULL`, code,
	)
}

// generateUniqueCode 生成一个未被使用的房间码（最多尝试 10 次）。
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

// StartGC 启动后台 goroutine，移除超过 idleTimeout 时间没有连接客户端的房间。
// clientCount(code) 必须返回该房间的实时 WS 客户端数量。
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
