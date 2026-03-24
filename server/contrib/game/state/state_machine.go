package state

import (
	"github.com/allin/server/contrib/room"
	"github.com/allin/server/gmodel"
)

// GameStateMachine 是一个房间的完整内存游戏状态。
type GameStateMachine struct {
	Street    gmodel.Street     // 当前游戏阶段
	HandNum   int              // 本局手牌编号（从 1 开始递增）
	Community []gmodel.Card     // 公共牌（0–5 张）
	Seats     [9]*gmodel.Player // 座位数组，nil 表示空座

	DealerSeat int // 庄家按钮所在座位号
	SBSeat     int // 小盲所在座位号
	BBSeat     int // 大盲所在座位号
	ActionSeat int // 当前待行动的座位号；-1 表示无待行动

	CurrentBet int64 // 本回合当前最高下注额
	MinRaise   int64 // 最低加注增量（上次加注幅度）

	ActionDeadlineMs int64 // 当前行动截止时间（Unix 毫秒）；0 表示无待行动

	Config room.RoomConfig // 房间配置（盲注/买入等）
}

// SeatPlayer 将玩家安排到第一个空座位，成功返回 true，无座位返回 false。
func (gs *GameStateMachine) SeatPlayer(p *gmodel.Player) bool {
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
func (gs *GameStateMachine) UnseatPlayer(userID string) {
	for i, p := range gs.Seats {
		if p != nil && p.UserID == userID {
			gs.Seats[i] = nil
			return
		}
	}
}

// FindPlayer 返回指定 userID 的玩家，不存在则返回 nil。
func (gs *GameStateMachine) FindPlayer(userID string) *gmodel.Player {
	for _, p := range gs.Seats {
		if p != nil && p.UserID == userID {
			return p
		}
	}
	return nil
}

// ActivePlayers 返回本手牌的活跃参与者（已发牌、未弃牌）。
func (gs *GameStateMachine) ActivePlayers() []*gmodel.Player {
	var out []*gmodel.Player
	for _, p := range gs.Seats {
		if p != nil && p.Active() {
			out = append(out, p)
		}
	}
	return out
}

// SeatedCount 返回已入座的玩家数量。
func (gs *GameStateMachine) SeatedCount() int {
	n := 0
	for _, p := range gs.Seats {
		if p != nil {
			n++
		}
	}
	return n
}

// EligibleToStart 返回满足参与新手牌条件的玩家。
func (gs *GameStateMachine) EligibleToStart() []*gmodel.Player {
	var out []*gmodel.Player
	for _, p := range gs.Seats {
		if p != nil && p.ReadyToStart() {
			out = append(out, p)
		}
	}
	return out
}

// NextActiveSeat 返回 from 之后下一个活跃参与者的座位（循环查找），未找到返回 NoSeat。
func (gs *GameStateMachine) NextActiveSeat(from int) int {
	for i := 1; i <= 9; i++ {
		idx := (from + i) % 9
		p := gs.Seats[idx]
		if p != nil && p.Active() {
			return idx
		}
	}
	return gmodel.NoSeat
}

// NextActableSeat 返回下一个仍可参与下注决策的座位（循环查找），未找到返回 NoSeat。
func (gs *GameStateMachine) NextActableSeat(from int) int {
	for i := 1; i <= 9; i++ {
		idx := (from + i) % 9
		p := gs.Seats[idx]
		if p != nil && p.CanBet() {
			return idx
		}
	}
	return gmodel.NoSeat
}

// BettingRoundOver 当所有可下注玩家均已行动且下注齐平时返回 true。
func (gs *GameStateMachine) BettingRoundOver() bool {
	for _, p := range gs.Seats {
		if p == nil || !p.CanBet() {
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

// CanAct 返回所有仍可参与下注决策的玩家。
func (gs *GameStateMachine) CanAct() []*gmodel.Player {
	var out []*gmodel.Player
	for _, p := range gs.Seats {
		if p != nil && p.CanBet() {
			out = append(out, p)
		}
	}
	return out
}

// TotalPot 返回所有玩家下注的总和。
func (gs *GameStateMachine) TotalPot() int64 {
	var total int64
	for _, p := range gs.Seats {
		if p != nil {
			total += p.TotalBet
		}
	}
	return total
}
