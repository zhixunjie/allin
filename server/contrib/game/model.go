package game

import (
	"fmt"

	"github.com/allin/server/contrib/eval"
	"github.com/allin/server/contrib/room"
)

// Street 表示当前手牌所处的下注回合阶段。
// 状态转移由 engine.go 中的 advanceOrEnd / nextStreet 驱动：
//
//	Idle → PreFlop（startHand 发底牌，postBlind 完成盲注）
//	PreFlop → Flop（下注完成，dealCommunity 发 3 张公共牌）
//	Flop → Turn（下注完成，dealCommunity 发第 4 张公共牌）
//	Turn → River（下注完成，dealCommunity 发第 5 张公共牌）
//	River → Showdown（下注完成，runShowdown 比牌并分配底池）
//	Showdown → Idle（结算完成，等待 handStartDelay 后开始下一手）
type Street uint8

const (
	// StreetIdle 是闲置状态：手牌尚未开始或上一手牌已结算完毕。
	// 在此状态下等待足够玩家（≥2 名，非离座、非断线、有筹码）后由定时器触发 startHand。
	StreetIdle Street = 0

	// StreetPreFlop 是翻牌前阶段：底牌已发，盲注已下，从 UTG 开始第一轮下注。
	StreetPreFlop Street = 1

	// StreetFlop 是翻牌圈：3 张公共牌已揭开，从小盲（或其后第一个活跃玩家）开始下注。
	StreetFlop Street = 2

	// StreetTurn 是转牌圈：第 4 张公共牌已揭开，再次从小盲方向开始下注。
	StreetTurn Street = 3

	// StreetRiver 是河牌圈：第 5 张（最后一张）公共牌已揭开，进行最后一轮下注。
	StreetRiver Street = 4

	// StreetShowdown 是摊牌阶段：所有下注结束，runShowdown 比较手牌并按底池分配筹码。
	// 结算完成后广播 HandResult，随后重置为 StreetIdle。
	StreetShowdown Street = 5
)

func (s Street) String() string {
	switch s {
	case StreetIdle:
		return "idle"
	case StreetPreFlop:
		return "preflop"
	case StreetFlop:
		return "flop"
	case StreetTurn:
		return "turn"
	case StreetRiver:
		return "river"
	case StreetShowdown:
		return "showdown"
	default:
		return "unknown"
	}
}

// Card 是一张扑克牌。
type Card struct {
	Rank byte // '2'…'A'
	Suit byte // 'c','d','h','s'
}

func (c Card) String() string { return string([]byte{c.Rank, c.Suit}) }

func (c Card) toEval() eval.Card { return eval.Card{Rank: c.Rank, Suit: c.Suit} }

// Player 表示已入座的玩家。
type Player struct {
	UserID      string // 玩家唯一 ID（bot 以 "bot_" 开头）
	DisplayName string // 显示名称
	SeatIndex   int    // 座位号 0–8
	Stack       int64  // 未参与下注的筹码余额
	Bet         int64  // 当前回合累计下注额
	TotalBet    int64  // 本手牌总下注额（用于边池计算）

	Hole     [2]Card  // 手牌（两张底牌）
	Folded   bool     // 是否已弃牌
	AllIn    bool     // 是否全押
	SitOut   bool     // 是否离座等待
	IsBot    bool     // 是否为 AI 机器人
	BotStyle BotStyle // bot 风格（tag/lag/station/rock；仅 IsBot=true 时有效）

	// ActedThisStreet 标记本街道内该玩家是否已行动；
	// 当有玩家加注超过其下注额时重置为 false，迫使其重新决策。
	ActedThisStreet bool

	Disconnected bool // 是否因断线而暂时离开（手牌进行中保留座位）
}

// Pot 表示主池或边池。
type Pot struct {
	Amount   int64    // 池中筹码总额
	Eligible []string // 有资格赢得该池的用户 ID 列表
}

// GameState 是一个房间的完整内存游戏状态。
type GameState struct {
	Street    Street     // 当前游戏阶段
	HandNum   int        // 本局手牌编号（从 1 开始递增）
	Community []Card     // 公共牌（0–5 张）
	Seats     [9]*Player // 座位数组，nil 表示空座

	DealerSeat int // 庄家按钮所在座位号
	SBSeat     int // 小盲所在座位号
	BBSeat     int // 大盲所在座位号
	ActionSeat int // 当前待行动的座位号；-1 表示无待行动

	CurrentBet int64 // 本回合当前最高下注额
	MinRaise   int64 // 最低加注增量（上次加注幅度）

	Config room.RoomConfig // 房间配置（盲注/买入等）
}

// SeatPlayer 将玩家安排到第一个空座位。
func (gs *GameState) SeatPlayer(p *Player) bool {
	for i := range gs.Seats {
		if gs.Seats[i] == nil {
			p.SeatIndex = i
			gs.Seats[i] = p
			return true
		}
	}
	return false
}

// UnseatPlayer 根据 userID 移除玩家。
func (gs *GameState) UnseatPlayer(userID string) {
	for i, p := range gs.Seats {
		if p != nil && p.UserID == userID {
			gs.Seats[i] = nil
			return
		}
	}
}

// FindPlayer 返回指定 userID 的玩家。
func (gs *GameState) FindPlayer(userID string) *Player {
	for _, p := range gs.Seats {
		if p != nil && p.UserID == userID {
			return p
		}
	}
	return nil
}

// ActivePlayers 返回未弃牌、未离座的玩家。
func (gs *GameState) ActivePlayers() []*Player {
	var out []*Player
	for _, p := range gs.Seats {
		if p != nil && !p.Folded && !p.SitOut {
			out = append(out, p)
		}
	}
	return out
}

// SeatedCount 返回已入座的玩家数量。
func (gs *GameState) SeatedCount() int {
	n := 0
	for _, p := range gs.Seats {
		if p != nil {
			n++
		}
	}
	return n
}

// EligibleToStart 返回可以参与新一手牌的玩家。
func (gs *GameState) EligibleToStart() []*Player {
	var out []*Player
	for _, p := range gs.Seats {
		if p != nil && !p.SitOut && !p.Disconnected && p.Stack > 0 {
			out = append(out, p)
		}
	}
	return out
}

// nextActiveSeat 返回 'from' 之后下一个未弃牌、未离座的座位（循环查找）。
// 如果没有找到则返回 -1。
func (gs *GameState) nextActiveSeat(from int) int {
	for i := 1; i <= 9; i++ {
		idx := (from + i) % 9
		p := gs.Seats[idx]
		if p != nil && !p.Folded && !p.SitOut {
			return idx
		}
	}
	return -1
}

// nextActableSeat 返回下一个仍可下注/加注/跟注的座位（非全押/弃牌/离座）。
func (gs *GameState) nextActableSeat(from int) int {
	for i := 1; i <= 9; i++ {
		idx := (from + i) % 9
		p := gs.Seats[idx]
		if p != nil && !p.Folded && !p.SitOut && !p.AllIn {
			return idx
		}
	}
	return -1
}

// BettingRoundOver 当没有活跃玩家需要继续行动时返回 true。
func (gs *GameState) BettingRoundOver() bool {
	for _, p := range gs.Seats {
		if p == nil || p.Folded || p.SitOut || p.AllIn {
			continue
		}
		if !p.ActedThisStreet {
			return false
		}
		if p.Bet < gs.CurrentBet {
			return false
		}
	}
	return true
}

// TotalPot 返回所有玩家下注的总和。
func (gs *GameState) TotalPot() int64 {
	var total int64
	for _, p := range gs.Seats {
		if p != nil {
			total += p.TotalBet
		}
	}
	return total
}

// ---- 快照类型（通过 WS 发送） ----

// GameSnapshot 是发送给客户端的完整状态。
type GameSnapshot struct {
	Street     string          `json:"street"`
	HandNum    int             `json:"hand_num"`
	Community  []string        `json:"community"`
	Seats      []SeatSnapshot  `json:"seats"`
	Pot        int64           `json:"pot"`
	DealerSeat int             `json:"dealer_seat"`
	ActionSeat int             `json:"action_seat"`
	CurrentBet int64           `json:"current_bet"`
	MinRaise   int64           `json:"min_raise"`
	Config     room.RoomConfig `json:"config"`
}

// SeatSnapshot 是 GameSnapshot 中单个座位的状态。
type SeatSnapshot struct {
	SeatIndex   int      `json:"seat_index"`
	UserID      string   `json:"user_id"`
	DisplayName string   `json:"display_name"`
	Stack       int64    `json:"stack"`
	Bet         int64    `json:"bet"`
	Folded      bool     `json:"folded"`
	AllIn       bool     `json:"all_in"`
	SitOut       bool     `json:"sit_out"`
	Disconnected bool     `json:"disconnected,omitempty"`
	IsBot        bool     `json:"is_bot,omitempty"`
	Hole         []string `json:"hole,omitempty"` // 仅对请求的玩家可见
}

// Snapshot 构建 GameSnapshot，仅为 viewerID 填充手牌。
func (gs *GameState) Snapshot(viewerID string) GameSnapshot {
	snap := GameSnapshot{
		Street:     gs.Street.String(),
		HandNum:    gs.HandNum,
		Community:  cardsToStrings(gs.Community),
		Pot:        gs.TotalPot(),
		DealerSeat: gs.DealerSeat,
		ActionSeat: gs.ActionSeat,
		CurrentBet: gs.CurrentBet,
		MinRaise:   gs.MinRaise,
		Config:     gs.Config,
	}
	for _, p := range gs.Seats {
		if p == nil {
			continue
		}
		ss := SeatSnapshot{
			SeatIndex:    p.SeatIndex,
			UserID:       p.UserID,
			DisplayName:  p.DisplayName,
			Stack:        p.Stack,
			Bet:          p.Bet,
			Folded:       p.Folded,
			AllIn:        p.AllIn,
			SitOut:       p.SitOut,
			Disconnected: p.Disconnected,
			IsBot:        p.IsBot,
		}
		if p.UserID == viewerID && gs.Street != StreetIdle {
			ss.Hole = []string{p.Hole[0].String(), p.Hole[1].String()}
		}
		snap.Seats = append(snap.Seats, ss)
	}
	return snap
}

func cardsToStrings(cards []Card) []string {
	if len(cards) == 0 {
		return []string{}
	}
	out := make([]string, len(cards))
	for i, c := range cards {
		out[i] = c.String()
	}
	return out
}

// EvaluateHand 返回玩家最佳 7 张牌手牌的评估等级。
func EvaluateHand(hole [2]Card, community []Card) (uint32, string) {
	if len(community) < 3 {
		return 0xFFFFFFFF, ""
	}
	cards := [7]eval.Card{
		hole[0].toEval(),
		hole[1].toEval(),
	}
	for i, c := range community {
		if i >= 5 {
			break
		}
		cards[2+i] = c.toEval()
	}
	rank := eval.Evaluate7(cards)
	return rank, fmt.Sprintf("%s", eval.Describe(rank))
}

// BestFiveStrings 返回玩家手牌+公共牌中最佳五张的字符串切片。
// community 不足 3 张时返回 nil（翻牌前不评估）。
func BestFiveStrings(hole [2]Card, community []Card) []string {
	if len(community) < 3 {
		return nil
	}
	cards := [7]eval.Card{
		hole[0].toEval(),
		hole[1].toEval(),
	}
	for i, c := range community {
		if i >= 5 {
			break
		}
		cards[2+i] = c.toEval()
	}
	best := eval.BestFive(cards)
	out := make([]string, 5)
	for i, c := range best {
		out[i] = c.String()
	}
	return out
}
