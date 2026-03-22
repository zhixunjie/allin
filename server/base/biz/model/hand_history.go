package model

import (
	"encoding/json"
	"time"
)

const (
	TableNameHandHistory = "hand_history" // 手牌历史表
	TableNameRoomHistory = "room_history" // 房间历史表
)

// HandHistoryRecord 是写入 hand_history 表的完整记录，由引擎在每手结算后异步保存。
type HandHistoryRecord struct {
	RoomID      int64           `json:"room_id"      db:"room_id"`      // 关联 room_history.id
	HandNum     int             `json:"hand_num"     db:"hand_num"`     // 本手在房间内的编号（从 1 递增）
	PlayersJSON json.RawMessage `json:"players_json" db:"players_json"` // 座位快照数组（开局时各玩家筹码/位置）
	ActionsJSON json.RawMessage `json:"actions_json" db:"actions_json"` // 全程行动日志数组（player/action/amount/street）
	ResultJSON  json.RawMessage `json:"result_json"  db:"result_json"`  // 结算结果（赢家/金额/牌型/all_players）
	PlayedAt    time.Time       `json:"played_at"    db:"played_at"`    // 手牌结束时间（UTC）
}

// HandHistoryEntry 是 GET /api/rooms/:code/hands 返回给前端的简化视图，省略 players_json 以减少流量。
type HandHistoryEntry struct {
	HandNum     int             `json:"hand_num"  db:"hand_num"`     // 手牌编号
	ResultJSON  json.RawMessage `json:"result"    db:"result_json"`  // 结算结果（赢家/金额/牌型）
	ActionsJSON json.RawMessage `json:"actions"   db:"actions_json"` // 全程行动日志
	PlayedAt    time.Time       `json:"played_at" db:"played_at"`    // 手牌结束时间
}
