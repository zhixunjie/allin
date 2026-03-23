package room

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/allin/server/base/biz/dao"
)

// Manager 在内存中维护所有活跃房间。
type Manager struct {
	mu    sync.RWMutex
	rooms map[string]*Room // key = 房间码
}

// NewManager 创建一个新的 Manager。
func NewManager() *Manager {
	return &Manager{rooms: make(map[string]*Room)}
}

// Create 校验配置、生成唯一房间码、持久化到 DB 并注册到内存。
func (m *Manager) Create(hostUserID int64, cfg RoomConfig) (*Room, error) {
	if err := cfg.validate(); err != nil {
		return nil, err
	}

	code, err := m.generateUniqueCode()
	if err != nil {
		return nil, err
	}

	r := &Room{
		Code:       code,
		HostUserID: hostUserID,
		Config:     cfg,
		State:      RoomStateLobby,
		CreatedAt:  time.Now(),
	}

	cfgJSON, err := json.Marshal(r.Config)
	if err != nil {
		return nil, fmt.Errorf("room config marshalling error: %s", err)
	}
	r.ID, err = dao.RoomDao.Persist(r.Code, r.HostUserID, cfgJSON, r.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("room.Create: %w", err)
	}

	m.mu.Lock()
	m.rooms[code] = r
	m.mu.Unlock()

	return r, nil
}

// Get 根据房间码返回对应的房间，不存在则返回 ErrNotFound。
func (m *Manager) Get(code string) (*Room, error) {
	m.mu.RLock()
	r, ok := m.rooms[code]
	m.mu.RUnlock()
	if !ok {
		return nil, ErrNotFound
	}
	return r, nil
}

// Close 将房间从内存中移除，并在 DB 中标记为已结束。
func (m *Manager) Close(code string) {
	m.mu.Lock()
	delete(m.rooms, code)
	m.mu.Unlock()

	dao.RoomDao.MarkEnded(code)
}

// generateUniqueCode 生成一个未被占用的房间码，最多尝试 10 次。
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

// StartGC 启动后台 goroutine，定期清理空闲超过 idleTimeout 的房间。
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
