package model

import (
	"encoding/json"
	"time"
)

// HandHistoryRecord 是写入 hand_history 表的完整记录，由引擎在每手结算后异步保存。
type HandHistoryRecord struct {
	RoomID      int64           `db:"room_id"`      // 关联 room_history.id
	HandNum     int             `db:"hand_num"`     // 本手在房间内的编号（从 1 递增）
	PlayersJSON json.RawMessage `db:"players_json"` // 座位快照数组（开局时各玩家筹码/位置）
	ActionsJSON json.RawMessage `db:"actions_json"` // 全程行动日志数组（player/action/amount/street）
	ResultJSON  json.RawMessage `db:"result_json"`  // 结算结果（赢家/金额/牌型/all_players）
	PlayedAt    time.Time       `db:"played_at"`    // 手牌结束时间（UTC）
}

// HandHistoryEntry 是 GET /api/rooms/:code/hands 返回给前端的简化视图，省略 players_json 以减少流量。
type HandHistoryEntry struct {
	HandNum     int             `db:"hand_num"     json:"hand_num"`  // 手牌编号
	ResultJSON  json.RawMessage `db:"result_json"  json:"result"`    // 结算结果（赢家/金额/牌型）
	ActionsJSON json.RawMessage `db:"actions_json" json:"actions"`   // 全程行动日志
	PlayedAt    time.Time       `db:"played_at"    json:"played_at"` // 手牌结束时间
}
