package engine

import "github.com/allin/server/gmodel"

// ── 行动日志 ──────────────────────────────────────────────────────────────────

// actionLogEntry 记录单次玩家行动，手牌结束后序列化写入 hand_history.actions_json。
type actionLogEntry struct {
	PlayerID string       `json:"player_id"` // 执行行动的玩家 ID
	Action   gmodel.Action `json:"action"`   // 行动类型（fold/check/call/bet/raise/all_in）
	Amount   int64        `json:"amount"`    // 行动金额（check/fold 为 0）
	Street   string       `json:"street"`    // 行动所在街道（preflop/flop/turn/river）
}

// ── 手牌历史存档 ──────────────────────────────────────────────────────────────

// playerSnap 是写入 hand_history.players_json 的玩家座位快照，记录开局时状态。
type playerSnap struct {
	PlayerID    string `json:"player_id"`    // 玩家唯一 ID
	DisplayName string `json:"display_name"` // 显示名称
	SeatIndex   int    `json:"seat_index"`   // 座位号 0–8
	Stack       int64  `json:"stack"`        // 开局时桌面筹码
}

// ── TypeShowdown 消息体 ───────────────────────────────────────────────────────

// revealEntry 是 TypeShowdown 消息中单个玩家的摊牌数据。
type revealEntry struct {
	PlayerID  string   `json:"player_id"`  // 摊牌玩家 ID
	SeatIndex int      `json:"seat_index"` // 座位号
	Hole      []string `json:"hole"`       // 两张底牌字符串，如 ["As","Kd"]
	HandName  string   `json:"hand_name"`  // 牌型名称，如 "Full House"
}

// ── TypeHandResult 消息体 ────────────────────────────────────────────────────

// handResultWinner 是 TypeHandResult 消息中单个赢家的结算条目。
// showdown 路径会填写 HandName；uncontested（其余人弃牌）路径留空。
type handResultWinner struct {
	PlayerID string `json:"player_id"`          // 赢家玩家 ID
	Amount   int64  `json:"amount"`              // 赢得筹码数
	HandName string `json:"hand_name,omitempty"` // 牌型名称（uncontested 时为空）
}

// handResultSeat 是 TypeHandResult 消息中每个座位的筹码快照（结算后余额）。
type handResultSeat struct {
	PlayerID string `json:"player_id"` // 玩家 ID
	Stack    int64  `json:"stack"`     // 结算后桌面筹码
}

// handResultPlayer 是 TypeHandResult 消息中每位玩家的完整详情，用于前端展示结算弹窗。
// uncontested 路径 Hole 为空切片，HandName 为空字符串。
type handResultPlayer struct {
	PlayerID    string   `json:"player_id"`          // 玩家 ID
	DisplayName string   `json:"display_name"`        // 显示名称
	Hole        []string `json:"hole"`                // 底牌（弃牌者为 []，未公开者为 ["?","?"]）
	HandName    string   `json:"hand_name,omitempty"` // 牌型名称（弃牌或 uncontested 时为空）
	Folded      bool     `json:"folded"`              // 是否已弃牌
	IsWinner    bool     `json:"is_winner"`           // 是否为本手赢家
}

// handResultPayload 是 TypeHandResult 的完整消息体。
type handResultPayload struct {
	Winners          []handResultWinner `json:"winners"`                     // 赢家列表（主池+边池各一条，同一玩家可能多条）
	Seats            []handResultSeat   `json:"seats,omitempty"`             // 结算后各座位筹码（uncontested 时省略）
	BestHand         []string           `json:"best_hand,omitempty"`         // 赢家最佳五张牌（uncontested 时省略）
	AllPlayers       []handResultPlayer `json:"all_players"`                 // 所有入座玩家详情（用于结算弹窗）
	NextHandDelaySec int                `json:"next_hand_delay_sec"`         // 下一手开始前的等待秒数
}
