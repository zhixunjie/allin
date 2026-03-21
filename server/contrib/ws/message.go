package ws

import (
	"encoding/json"
	"time"
)

// ---- 消息类型 ----
// 服务端 → 客户端（事件）
const (
	TypeConnected      = "connected"
	TypePlayerJoined   = "player_joined"
	TypePlayerLeft     = "player_left"
	TypeGameStarted    = "game_started"
	TypeHoleCards      = "hole_cards"
	TypeCardsDealt     = "cards_dealt"
	TypeStreetStarted  = "street_started"
	TypeActionRequired = "action_required"
	TypeActionTaken    = "action_taken"
	TypeActionTimeout  = "action_timeout"
	TypeShowdown       = "showdown"
	TypeHandResult     = "hand_result"
	TypeChatMessage    = "chat_message"
	TypeError          = "error"
	TypeStackUpdated   = "stack_updated"
)

// 客户端 → 服务端（命令）
const (
	CmdJoinRoom = "join_room"
	CmdAction   = "action"
	CmdChat     = "chat"
	CmdAddChips = "add_chips"
	CmdSitOut   = "sit_out"
)

// Envelope 是所有 WebSocket 消息的通用包装。
type Envelope struct {
	Type    string          `json:"type"`
	Seq     int64           `json:"seq"`
	Ts      int64           `json:"ts"` // Unix 毫秒时间戳
	Payload json.RawMessage `json:"payload"`
}

// NewEvent 构建一个带当前时间戳的服务端事件信封。
func NewEvent(msgType string, payload any) (Envelope, error) {
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

// MustEvent 类似 NewEvent，但在序列化错误时 panic（对已知类型安全）。
func MustEvent(msgType string, payload any) Envelope {
	e, err := NewEvent(msgType, payload)
	if err != nil {
		panic(err)
	}
	return e
}

// ---- 载荷结构体 ----

// ConnectedPayload 在 WebSocket 连接成功后发送给客户端。
// GameSnapshot 使用指针以避免循环导入：game → ws → game。
// ws 包接受任意值；game 包负责填充。
type ConnectedPayload struct {
	PlayerID     string      `json:"player_id"`
	DisplayName  string      `json:"display_name"`
	RoomCode     string      `json:"room_code"`
	GameSnapshot interface{} `json:"game_snapshot,omitempty"`
}

// ---- 游戏事件载荷（服务端 → 客户端） ----

// GameStartedPayload 在新一手牌开始时广播。
type GameStartedPayload struct {
	HandNum    int   `json:"hand_num"`
	DealerSeat int   `json:"dealer_seat"`
	SBSeat     int   `json:"sb_seat"`
	BBSeat     int   `json:"bb_seat"`
	SmallBlind int64 `json:"small_blind"`
	BigBlind   int64 `json:"big_blind"`
}

// HoleCardsPayload 私密发送给玩家，包含其手牌。
type HoleCardsPayload struct {
	PlayerID string   `json:"player_id"`
	Hole     []string `json:"hole"` // e.g. ["Ac","Kd"]
}

// CardsDealtPayload 通知所有玩家哪些座位收到了手牌。
type CardsDealtPayload struct {
	Seats []int `json:"seats"`
}

// StreetStartedPayload 在新的下注回合开始时广播。
type StreetStartedPayload struct {
	Street    string   `json:"street"`
	Community []string `json:"community"`
	Pot       int64    `json:"pot"`
}

// ActionRequiredPayload 广播以提示玩家行动。
type ActionRequiredPayload struct {
	PlayerID   string `json:"player_id"`
	SeatIndex  int    `json:"seat_index"`
	DeadlineTs int64  `json:"deadline_ts"` // Unix 毫秒时间戳
	CurrentBet int64  `json:"current_bet"`
	CallAmount int64  `json:"call_amount"`
	MinRaise   int64  `json:"min_raise"`
	Stack      int64  `json:"stack"`
	Pot        int64  `json:"pot"`
}

// ActionTakenPayload 在玩家行动后广播。
type ActionTakenPayload struct {
	PlayerID string `json:"player_id"`
	Action   string `json:"action"`
	Amount   int64  `json:"amount"`
	Stack    int64  `json:"stack"`
	TotalPot int64  `json:"total_pot"`
}

// ActionTimeoutPayload 在玩家计时器耗尽时广播。
type ActionTimeoutPayload struct {
	PlayerID string `json:"player_id"`
	Action   string `json:"action"` // "fold" or "check"
}

// PlayerJoinedPayload 在玩家加入或重连时广播。
type PlayerJoinedPayload struct {
	PlayerID    string `json:"player_id"`
	DisplayName string `json:"display_name"`
	SeatIndex   int    `json:"seat_index"`
	Stack       int64  `json:"stack"`
	IsReconnect bool   `json:"is_reconnect"`
	IsBot       bool   `json:"is_bot,omitempty"`
}

// PlayerLeftPayload 在玩家断开连接时广播。
type PlayerLeftPayload struct {
	PlayerID  string `json:"player_id"`
	SeatIndex int    `json:"seat_index"`
}

// ChatPayload 承载一条聊天消息。
type ChatPayload struct {
	SenderID    string `json:"sender_id"`
	DisplayName string `json:"display_name"`
	Text        string `json:"text"`
	Ts          int64  `json:"ts"`
}

// ErrorPayload 承载对客户端命令的错误响应。
type ErrorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	RefSeq  int64  `json:"ref_seq"`
}

// StackUpdatedPayload 在手牌外玩家筹码变化时广播。
type StackUpdatedPayload struct {
	PlayerID string `json:"player_id"`
	Stack    int64  `json:"stack"`
	Delta    int64  `json:"delta"`
}

// ---- 传入命令载荷 ----

// JoinRoomCmd 是 CmdJoinRoom 的载荷。
type JoinRoomCmd struct {
	RoomCode  string `json:"room_code"`
	SeatIndex *int   `json:"seat_index"` // 可选的首选座位
	BuyIn     int64  `json:"buy_in"`     // 买入金额；0 表示使用 MaxBuyIn
}

// ActionCmd 是 CmdAction 的载荷。
type ActionCmd struct {
	Action string `json:"action"` // fold/check/call/bet/raise/all_in
	Amount int64  `json:"amount"`
}

// ChatCmd 是 CmdChat 的载荷。
type ChatCmd struct {
	Text string `json:"text"`
}

// AddChipsCmd 是 CmdAddChips 的载荷。
type AddChipsCmd struct {
	Amount int64 `json:"amount"`
}

// SitOutCmd 是 CmdSitOut 的载荷。
type SitOutCmd struct {
	SitOut bool `json:"sit_out"`
}
