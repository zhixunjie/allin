package room

import (
	"sync"
	"time"
)

// Room 是一个扑克房间。内存中的字段（hub、engine、players）
// 由 RoomManager 管理，不会持久化。
type Room struct {
	ID          int64      `json:"id"`
	Code        string     `json:"code"`
	HostUserID  int64      `json:"host_user_id"`
	Config      RoomConfig `json:"config"`
	State       RoomState  `json:"state"`
	CreatedAt   time.Time  `json:"created_at"`
	PlayerCount int        `json:"player_count"` // 大厅显示的实时人数

	mu           sync.Mutex
	lastActivity time.Time // 在玩家加入/离开时更新
}

// Touch 记录当前时间为房间的最近活动时间。
func (r *Room) Touch() {
	r.mu.Lock()
	r.lastActivity = time.Now()
	r.mu.Unlock()
}

// IdleDuration 返回房间空闲了多长时间。
// 如果从未调用过 Touch，则回退为自创建以来的时间。
func (r *Room) IdleDuration() time.Duration {
	r.mu.Lock()
	t := r.lastActivity
	r.mu.Unlock()
	if t.IsZero() {
		return time.Since(r.CreatedAt)
	}
	return time.Since(t)
}
