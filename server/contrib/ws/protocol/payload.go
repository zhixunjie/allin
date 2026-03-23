package protocol

import (
	"encoding/json"
	"time"

	"github.com/allin/server/contrib/game/model"
)

// Envelope 是所有服务端 → 客户端消息的通用包装。
type Envelope struct {
	Type    MsgType         `json:"type"`    // 事件类型
	Seq     int64           `json:"seq"`     // 服务端递增序列号，客户端可用于检测消息丢失
	Ts      int64           `json:"ts"`      // 服务端发送时间戳（Unix 毫秒）
	Payload json.RawMessage `json:"payload"` // 具体事件载荷，按 Type 解析为对应 Payload 结构体
}

// NewEnvelope 构建一个带当前时间戳的服务端事件信封。
func NewEnvelope(msgType MsgType, payload any) (Envelope, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return Envelope{}, err
	}
	return Envelope{
		Type:    msgType,
		Ts:      time.Now().UnixMilli(),
		Payload: raw,
	}, nil
}

// MustNewEnvelope 类似 NewEnvelope，但在序列化错误时 panic（对已知类型安全）。
func MustNewEnvelope(msgType MsgType, payload any) Envelope {
	e, err := NewEnvelope(msgType, payload)
	if err != nil {
		panic(err)
	}
	return e
}

// MsgType 是服务端 → 客户端的事件类型。
type MsgType string

const (
	TypeConnected      MsgType = "connected"       // ConnectedPayload 连接成功，附带玩家 ID 和当前游戏快照；
	TypePlayerJoined   MsgType = "player_joined"   // PlayerJoinedPayload 玩家加入或重连；
	TypePlayerLeft     MsgType = "player_left"     // PlayerLeftPayload 玩家离桌；
	TypeGameStarted    MsgType = "game_started"    // GameStartedPayload 新一手牌开始，含庄家/盲注信息；
	TypeHoleCards      MsgType = "hole_cards"      // HoleCardsPayload 仅发给当事玩家的底牌；
	TypeCardsDealt     MsgType = "cards_dealt"     // CardsDealtPayload 通知所有人哪些座位已发牌（背面）；
	TypeStreetStarted  MsgType = "street_started"  // StreetStartedPayload 进入新街道（Flop/Turn/River），附带公共牌；
	TypeActionRequired MsgType = "action_required" // ActionRequiredPayload 轮到指定玩家行动，附带倒计时和可用操作参数；
	TypeActionTaken    MsgType = "action_taken"    // ActionTakenPayload 玩家已行动，广播行动结果和筹码变化；
	TypeActionTimeout  MsgType = "action_timeout"  // ActionTimeoutPayload 玩家超时，服务端自动 fold 或 check；
	TypeShowdown       MsgType = "showdown"        // ShowdownPayload 摊牌，公开所有未弃牌玩家的手牌；
	TypeHandResult     MsgType = "hand_result"     // HandResultPayload 本手结算，包含赢家和筹码分配；
	TypeChatMessage    MsgType = "chat_message"    // ChatPayload 聊天消息（双向）；
	TypeError          MsgType = "error"           // ErrorPayload 服务端业务错误；
	TypeSitOut         MsgType = "sit_out_status"  // SitOutPayload 玩家离座/归座状态变更；
	TypeReadyStatus    MsgType = "ready_status"    // ReadyStatusPayload 结算间隙已准备人数广播；
	TypeStackUpdated   MsgType = "stack_updated"   // StackUpdatedPayload 玩家补充筹码后广播；
)

// 定义所有服务端 → 客户端的事件载荷结构体。

// ConnectedPayload 在 WebSocket 连接成功后发送给客户端。
// GameSnapshot 使用 any 以避免循环导入：game → ws → game。
type ConnectedPayload struct {
	PlayerID     string `json:"player_id"`               // 当前连接玩家的用户 ID
	DisplayName  string `json:"display_name"`            // 玩家显示名称
	RoomCode     string `json:"room_code"`               // 所在房间码
	ChipBalance  int64  `json:"chip_balance"`            // 买入后账户余额
	GameSnapshot any    `json:"game_snapshot,omitempty"` // 重连时附带的完整游戏快照
}

// GameStartedPayload 在新一手牌开始时广播。
type GameStartedPayload struct {
	HandNum    int   `json:"hand_num"`    // 本手牌编号（从 1 开始递增）
	DealerSeat int   `json:"dealer_seat"` // 庄家座位号
	SBSeat     int   `json:"sb_seat"`     // 小盲注座位号
	BBSeat     int   `json:"bb_seat"`     // 大盲注座位号
	SmallBlind int64 `json:"small_blind"` // 小盲注金额
	BigBlind   int64 `json:"big_blind"`   // 大盲注金额
}

// HoleCardsPayload 私密发送给玩家，包含其手牌。
type HoleCardsPayload struct {
	PlayerID string   `json:"player_id"` // 持有该手牌的玩家 ID
	Hole     []string `json:"hole"`      // 底牌列表，e.g. ["Ac","Kd"]
}

// CardsDealtPayload 通知所有玩家哪些座位收到了手牌（背面展示）。
type CardsDealtPayload struct {
	Seats []int `json:"seats"` // 已发牌的座位号列表
}

// StreetStartedPayload 在新的下注回合开始时广播。
type StreetStartedPayload struct {
	Street    string   `json:"street"`    // 街道名称：flop/turn/river
	Community []string `json:"community"` // 当前公共牌列表
	Pot       int64    `json:"pot"`       // 进入本街道时的底池总额
}

// ActionRequiredPayload 广播以提示指定玩家行动。
type ActionRequiredPayload struct {
	PlayerID   string `json:"player_id"`   // 需要行动的玩家 ID
	SeatIndex  int    `json:"seat_index"`  // 需要行动的座位号
	DeadlineTs int64  `json:"deadline_ts"` // 行动截止时间（Unix 毫秒）
	CurrentBet int64  `json:"current_bet"` // 本街当前最大下注额
	CallAmount int64  `json:"call_amount"` // 跟注所需金额（= CurrentBet - 玩家已投入）
	MinRaise   int64  `json:"min_raise"`   // 最小加注增量
	Stack      int64  `json:"stack"`       // 玩家当前剩余筹码
	Pot        int64  `json:"pot"`         // 当前底池总额（含各边池）
}

// ActionTakenPayload 在玩家行动后广播。
type ActionTakenPayload struct {
	PlayerID string       `json:"player_id"` // 执行行动的玩家 ID
	Action   model.Action `json:"action"`    // 行动类型
	Amount   int64        `json:"amount"`    // 行动金额（check/fold 为 0）
	Stack    int64        `json:"stack"`     // 行动后玩家剩余筹码
	TotalPot int64        `json:"total_pot"` // 行动后底池总额
}

// ActionTimeoutPayload 在玩家计时器耗尽时广播。
type ActionTimeoutPayload struct {
	PlayerID string       `json:"player_id"` // 超时玩家的 ID
	Action   model.Action `json:"action"`    // 自动执行的行动
	Stack    int64        `json:"stack"`     // 超时处理后玩家剩余筹码
	TotalPot int64        `json:"total_pot"` // 处理后底池总额
}

// PlayerJoinedPayload 在玩家加入或重连时广播。
type PlayerJoinedPayload struct {
	PlayerID    string `json:"player_id"`        // 加入玩家的用户 ID
	DisplayName string `json:"display_name"`     // 加入玩家的显示名称
	SeatIndex   int    `json:"seat_index"`       // 分配到的座位号
	Stack       int64  `json:"stack"`            // 买入带入的桌面筹码
	IsReconnect bool   `json:"is_reconnect"`     // true 表示重连，false 表示首次入座
	IsBot       bool   `json:"is_bot,omitempty"` // true 表示该玩家是 AI bot
}

// PlayerLeftPayload 在玩家离桌时广播。
type PlayerLeftPayload struct {
	PlayerID  string `json:"player_id"`  // 离桌玩家的用户 ID
	SeatIndex int    `json:"seat_index"` // 离桌玩家的座位号
}

// ChatPayload 承载一条聊天消息。
type ChatPayload struct {
	SenderID    string `json:"sender_id"`    // 发送者用户 ID
	DisplayName string `json:"display_name"` // 发送者显示名称
	Text        string `json:"text"`         // 消息正文
	Ts          int64  `json:"ts"`           // 服务端接收时间戳（Unix 毫秒）
}

// SitOutPayload 在玩家离座/归座时广播。
type SitOutPayload struct {
	PlayerID  string `json:"player_id"`  // 变更状态的玩家 ID
	SeatIndex int    `json:"seat_index"` // 玩家座位号
	SitOut    bool   `json:"sit_out"`    // true = 离座，false = 归座
}

// StackUpdatedPayload 在玩家补充筹码后广播。
type StackUpdatedPayload struct {
	PlayerID string `json:"player_id"` // 筹码变动的玩家 ID
	Stack    int64  `json:"stack"`     // 变动后桌面筹码总额
	Delta    int64  `json:"delta"`     // 本次变动金额（正数）
}

// ReadyStatusPayload 在结算间隙广播已准备人数。
type ReadyStatusPayload struct {
	ReadyCount int `json:"ready_count"` // 已点击"准备下一局"的玩家数
	TotalCount int `json:"total_count"` // 需要准备的玩家总数
}

type (
	// ShowdownPayload 是 TypeShowdown 事件的载荷（所有未弃牌玩家的手牌列表）。
	ShowdownPayload []ShowdownReveal

	// ShowdownReveal 是 TypeShowdown 事件中单个玩家的摊牌信息。
	ShowdownReveal struct {
		PlayerID  string   `json:"player_id"`  // 玩家 ID
		SeatIndex int      `json:"seat_index"` // 座位号
		Hole      []string `json:"hole"`       // 底牌，e.g. ["Ac","Kd"]
		HandName  string   `json:"hand_name"`  // 成牌名称，e.g. "Full House"
	}
)

type (
	// HandResultPayload 是 TypeHandResult 事件的载荷。
	HandResultPayload struct {
		Winners          []HandResultWinner `json:"winners"`             // 赢家列表及各自赢得的金额
		Seats            []HandResultSeat   `json:"seats,omitempty"`     // 各座位筹码快照（摊牌时提供）
		BestHand         []string           `json:"best_hand,omitempty"` // 主赢家的最佳五张牌
		AllPlayers       []HandResultPlayer `json:"all_players"`         // 所有玩家摘要（含弃牌者）
		NextHandDelaySec int                `json:"next_hand_delay_sec"` // 下一手开始前的等待秒数
	}
	// HandResultWinner 是 HandResultPayload 中的赢家记录。
	HandResultWinner struct {
		PlayerID string `json:"player_id"`           // 赢家玩家 ID
		Amount   int64  `json:"amount"`              // 本手获得的筹码总额
		HandName string `json:"hand_name,omitempty"` // 成牌名称（摊牌时有值，未摊牌为空）
	}
	// HandResultSeat 是 HandResultPayload 中的座位筹码快照（手牌结束后）。
	HandResultSeat struct {
		PlayerID string `json:"player_id"` // 玩家 ID
		Stack    int64  `json:"stack"`     // 手牌结束后桌面筹码
	}
	// HandResultPlayer 是 HandResultPayload 中每位玩家的摘要（含弃牌者）。
	HandResultPlayer struct {
		PlayerID    string   `json:"player_id"`           // 玩家 ID
		DisplayName string   `json:"display_name"`        // 显示名称
		Hole        []string `json:"hole"`                // 底牌（弃牌者为空数组）
		HandName    string   `json:"hand_name,omitempty"` // 成牌名称（仅摊牌时有值）
		Folded      bool     `json:"folded"`              // 是否已弃牌
		IsWinner    bool     `json:"is_winner"`           // 是否为本手赢家
	}
)
