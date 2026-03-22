package model

import (
	"encoding/json"
	"time"
)

// RoomHistory 对应 room_history 表，记录一个房间的完整生命周期。
type RoomHistory struct {
	ID         int64           `json:"id"          db:"id"`           // 房间记录自增ID
	RoomCode   string          `json:"room_code"   db:"room_code"`    // 6位房间码
	HostUserID int64           `json:"host_user_id" db:"host_user_id"` // 房主用户ID
	ConfigJSON json.RawMessage `json:"config_json" db:"config_json"`  // 房间配置快照（RoomConfig JSON）
	StartedAt  time.Time       `json:"started_at"  db:"started_at"`   // 房间创建时间（UTC）
	EndedAt    *time.Time      `json:"ended_at"    db:"ended_at"`     // 房间关闭时间；NULL 表示仍在进行
}
