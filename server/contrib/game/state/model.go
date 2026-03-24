package state

import (
	"github.com/allin/server/contrib/room"
	"github.com/allin/server/gmodel"
)

// ── 底池 ──────────────────────────────────────────────────────────────────────

// Pot 表示一个底池（主池或边池）。
type Pot struct {
	Amount   int64    // 池中筹码总额
	Eligible []string // 有资格赢得该池的用户 ID 列表（全押玩家只能赢取对应层级）
}

// ── Bot 决策局面 ──────────────────────────────────────────────────────────────

// BotSituation 描述 bot 做决策时的完整局面快照。
// 在触发 bot 行动前从 GameStateMachine 中提取，传入 Personality.decide。
type BotSituation struct {
	Street     gmodel.Street  // 当前街道（PreFlop/Flop/Turn/River）
	Hole       [2]gmodel.Card // bot 的两张底牌
	Community  []gmodel.Card  // 当前公共牌（0–5 张）
	CurrentBet int64          // 本轮最大下注额（需跟注到此金额）
	PlayerBet  int64          // bot 本街已下注金额
	Stack      int64          // bot 当前桌面筹码
	BigBlind   int64          // 大盲注额（用于计算底池赔率和加注倍数）
	Pot        int64          // 当前总底池（含本街所有下注）
}

// ── WebSocket 快照 ────────────────────────────────────────────────────────────

// GameSnapshot 是发送给客户端的完整状态，在重连或首次入座时下发。
type GameSnapshot struct {
	Street     string          `json:"street"`      // 当前街道名称（idle/preflop/…）
	HandNum    int             `json:"hand_num"`    // 当前手牌编号（从 1 递增）
	Community  []string        `json:"community"`   // 公共牌字符串列表
	Seats      []SeatSnapshot  `json:"seats"`       // 所有已入座玩家的状态列表
	Pot        int64           `json:"pot"`         // 当前总底池
	DealerSeat int             `json:"dealer_seat"` // 庄家座位号
	ActionSeat int             `json:"action_seat"`          // 当前待行动座位号；-1 表示无
	CurrentBet int64           `json:"current_bet"`          // 本回合最高下注额
	MinRaise   int64           `json:"min_raise"`            // 最低加注增量
	DeadlineTs int64           `json:"deadline_ts,omitempty"` // 当前行动截止时间（Unix 毫秒）；0 或省略表示无待行动
	Config     room.RoomConfig `json:"config"`               // 房间配置（盲注/买入等）
}

// SeatSnapshot 是 GameSnapshot 中单个座位的实时状态。
type SeatSnapshot struct {
	SeatIndex    int      `json:"seat_index"`             // 座位号 0–8
	UserID       string   `json:"user_id"`                // 玩家唯一 ID
	DisplayName  string   `json:"display_name"`           // 显示名称
	Stack        int64    `json:"stack"`                  // 桌面筹码余额
	Bet          int64    `json:"bet"`                    // 本街已下注金额
	Folded       bool     `json:"folded"`                 // 是否已弃牌
	AllIn        bool     `json:"all_in"`                 // 是否全押
	SitOut       bool     `json:"sit_out"`                // 是否离座
	Disconnected bool     `json:"disconnected,omitempty"` // 是否断线（手牌中保留座位）
	IsBot        bool     `json:"is_bot,omitempty"`       // 是否为 AI bot
	Hole         []string `json:"hole,omitempty"`         // 底牌（仅对请求玩家自身可见）
}
