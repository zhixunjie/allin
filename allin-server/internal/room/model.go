package room

import (
	"sync"
	"time"
)

// RoomConfig is the configuration set by the host when creating a room.
type RoomConfig struct {
	SmallBlind    int64 `json:"small_blind"`
	BigBlind      int64 `json:"big_blind"`
	MinBuyIn      int64 `json:"min_buy_in"`
	MaxBuyIn      int64 `json:"max_buy_in"`
	MaxPlayers    int   `json:"max_players"`     // 2-9
	ActionTimeSec int   `json:"action_time_sec"` // default 30
}

// RoomState represents the lifecycle state of a room.
type RoomState string

const (
	RoomStateLobby   RoomState = "lobby"
	RoomStatePlaying RoomState = "playing"
)

// Room is a single poker room. The in-memory fields (hub, engine, players) are
// managed by the RoomManager and never persisted.
type Room struct {
	ID          string     `json:"id"`
	Code        string     `json:"code"`
	HostUserID  string     `json:"host_user_id"`
	Config      RoomConfig `json:"config"`
	State       RoomState  `json:"state"`
	CreatedAt   time.Time  `json:"created_at"`
	PlayerCount int        `json:"player_count"` // live count for lobby display

	mu           sync.Mutex
	lastActivity time.Time // updated on player join/leave
}

// Touch records the current time as the latest activity on the room.
func (r *Room) Touch() {
	r.mu.Lock()
	r.lastActivity = time.Now()
	r.mu.Unlock()
}

// IdleDuration returns how long the room has been inactive.
// Falls back to time since creation if Touch has never been called.
func (r *Room) IdleDuration() time.Duration {
	r.mu.Lock()
	t := r.lastActivity
	r.mu.Unlock()
	if t.IsZero() {
		return time.Since(r.CreatedAt)
	}
	return time.Since(t)
}
