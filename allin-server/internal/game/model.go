package game

import (
	"fmt"

	"github.com/allin/server/internal/eval"
	"github.com/allin/server/internal/room"
)

// Street 表示下注回合阶段。
type Street uint8

const (
	StreetIdle     Street = 0
	StreetPreFlop  Street = 1
	StreetFlop     Street = 2
	StreetTurn     Street = 3
	StreetRiver    Street = 4
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
	UserID      string
	DisplayName string
	SeatIndex   int
	Stack       int64 // 未参与下注的筹码
	Bet         int64 // 当前回合下注额
	TotalBet    int64 // 本手牌总下注额（用于边池计算）

	Hole   [2]Card
	Folded bool
	AllIn  bool
	SitOut bool
	IsBot    bool
	BotStyle string // tag|lag|station|rock（仅在 IsBot=true 时设置）

	// ActedThisStreet 在当前下注被加注超过玩家下注额时重置为 false。
	ActedThisStreet bool
}

// Pot 表示主池或边池。
type Pot struct {
	Amount   int64
	Eligible []string // 有资格的用户 ID
}

// GameState 是一个房间的完整内存游戏状态。
type GameState struct {
	Street     Street
	HandNum    int
	Community  []Card
	Seats      [9]*Player // nil = 空座位

	DealerSeat int // 庄家按钮
	SBSeat     int
	BBSeat     int
	ActionSeat int // -1 表示无待行动

	CurrentBet int64 // 本回合最高下注额
	MinRaise   int64 // 最低加注增量

	Config room.RoomConfig
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
		if p != nil && !p.SitOut && p.Stack > 0 {
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
	Street     string         `json:"street"`
	HandNum    int            `json:"hand_num"`
	Community  []string       `json:"community"`
	Seats      []SeatSnapshot `json:"seats"`
	Pot        int64          `json:"pot"`
	DealerSeat int            `json:"dealer_seat"`
	ActionSeat int            `json:"action_seat"`
	CurrentBet int64          `json:"current_bet"`
	MinRaise   int64          `json:"min_raise"`
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
	SitOut      bool     `json:"sit_out"`
	IsBot       bool     `json:"is_bot,omitempty"`
	Hole        []string `json:"hole,omitempty"` // 仅对请求的玩家可见
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
			SeatIndex:   p.SeatIndex,
			UserID:      p.UserID,
			DisplayName: p.DisplayName,
			Stack:       p.Stack,
			Bet:         p.Bet,
			Folded:      p.Folded,
			AllIn:       p.AllIn,
			SitOut:      p.SitOut,
			IsBot:       p.IsBot,
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
