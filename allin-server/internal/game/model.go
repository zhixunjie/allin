package game

import (
	"fmt"

	"github.com/allin/server/internal/eval"
	"github.com/allin/server/internal/room"
)

// Street represents a betting round phase.
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

// Card is a playing card.
type Card struct {
	Rank byte // '2'…'A'
	Suit byte // 'c','d','h','s'
}

func (c Card) String() string { return string([]byte{c.Rank, c.Suit}) }

func (c Card) toEval() eval.Card { return eval.Card{Rank: c.Rank, Suit: c.Suit} }

// Player represents a seated player.
type Player struct {
	UserID      string
	DisplayName string
	SeatIndex   int
	Stack       int64 // chips not in play
	Bet         int64 // current street bet
	TotalBet    int64 // total bet this hand (for side-pot calc)

	Hole   [2]Card
	Folded bool
	AllIn  bool
	SitOut bool

	// ActedThisStreet is reset to false when the current bet is raised above the player's bet.
	ActedThisStreet bool
}

// Pot represents a main or side pot.
type Pot struct {
	Amount   int64
	Eligible []string // UserIDs
}

// GameState is the complete in-memory game state for one room.
type GameState struct {
	Street     Street
	HandNum    int
	Community  []Card
	Seats      [9]*Player // nil = empty seat

	DealerSeat int // button
	SBSeat     int
	BBSeat     int
	ActionSeat int // -1 when no action pending

	CurrentBet int64 // highest bet this street
	MinRaise   int64 // minimum raise increment

	Config room.RoomConfig
}

// SeatPlayer places a player in the first available seat.
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

// UnseatPlayer removes a player by userID.
func (gs *GameState) UnseatPlayer(userID string) {
	for i, p := range gs.Seats {
		if p != nil && p.UserID == userID {
			gs.Seats[i] = nil
			return
		}
	}
}

// FindPlayer returns the player with the given userID.
func (gs *GameState) FindPlayer(userID string) *Player {
	for _, p := range gs.Seats {
		if p != nil && p.UserID == userID {
			return p
		}
	}
	return nil
}

// ActivePlayers returns non-folded, non-sitting-out players.
func (gs *GameState) ActivePlayers() []*Player {
	var out []*Player
	for _, p := range gs.Seats {
		if p != nil && !p.Folded && !p.SitOut {
			out = append(out, p)
		}
	}
	return out
}

// SeatedCount returns the number of occupied seats.
func (gs *GameState) SeatedCount() int {
	n := 0
	for _, p := range gs.Seats {
		if p != nil {
			n++
		}
	}
	return n
}

// EligibleToStart returns players who can participate in a new hand.
func (gs *GameState) EligibleToStart() []*Player {
	var out []*Player
	for _, p := range gs.Seats {
		if p != nil && !p.SitOut && p.Stack > 0 {
			out = append(out, p)
		}
	}
	return out
}

// nextActiveSeat returns the next non-folded, non-sitout seat after 'from' (wraps around).
// Returns -1 if none found.
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

// nextActableSeat returns the next seat that can still bet/raise/call (not all-in/folded/sitout).
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

// BettingRoundOver returns true when no active player needs to act further.
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

// TotalPot returns the sum of all bets across all players.
func (gs *GameState) TotalPot() int64 {
	var total int64
	for _, p := range gs.Seats {
		if p != nil {
			total += p.TotalBet
		}
	}
	return total
}

// ---- Snapshot types (sent over WS) ----

// GameSnapshot is the full state sent to a client.
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

// SeatSnapshot is one seat's state in a GameSnapshot.
type SeatSnapshot struct {
	SeatIndex   int      `json:"seat_index"`
	UserID      string   `json:"user_id"`
	DisplayName string   `json:"display_name"`
	Stack       int64    `json:"stack"`
	Bet         int64    `json:"bet"`
	Folded      bool     `json:"folded"`
	AllIn       bool     `json:"all_in"`
	SitOut      bool     `json:"sit_out"`
	Hole        []string `json:"hole,omitempty"` // only for the requesting player
}

// Snapshot builds a GameSnapshot, filling hole cards only for viewerID.
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

// EvaluateHand returns the eval rank for a player's best 7-card hand.
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
