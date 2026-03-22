package state

import "github.com/allin/server/contrib/game/model"

// SeatPlayer 将玩家安排到第一个空座位，成功返回 true，无座位返回 false。
func (gs *GameStateMachine) SeatPlayer(p *model.Player) bool {
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
func (gs *GameStateMachine) FindPlayer(userID string) *model.Player {
	for _, p := range gs.Seats {
		if p != nil && p.UserID == userID {
			return p
		}
	}
	return nil
}

// ActivePlayers 返回未弃牌、未离座的玩家。
func (gs *GameStateMachine) ActivePlayers() []*model.Player {
	var out []*model.Player
	for _, p := range gs.Seats {
		if p != nil && !p.Folded && !p.SitOut {
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

// EligibleToStart 返回可以参与新一手牌的玩家：未离座、未断线、有筹码。
func (gs *GameStateMachine) EligibleToStart() []*model.Player {
	var out []*model.Player
	for _, p := range gs.Seats {
		if p != nil && !p.SitOut && !p.Disconnected && p.Stack > 0 {
			out = append(out, p)
		}
	}
	return out
}

// NextActiveSeat 返回 from 之后下一个未弃牌、未离座的座位（循环查找）。
// 如果没有找到则返回 -1。
func (gs *GameStateMachine) NextActiveSeat(from int) int {
	for i := 1; i <= 9; i++ {
		idx := (from + i) % 9
		p := gs.Seats[idx]
		if p != nil && !p.Folded && !p.SitOut {
			return idx
		}
	}
	return -1
}

// NextActableSeat 返回下一个仍可下注/加注/跟注的座位（非全押/弃牌/离座）。
func (gs *GameStateMachine) NextActableSeat(from int) int {
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
func (gs *GameStateMachine) BettingRoundOver() bool {
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

// CanAct 返回所有仍可参与下注决策的玩家：未弃牌、未离座、未全押。
func (gs *GameStateMachine) CanAct() []*model.Player {
	var out []*model.Player
	for _, p := range gs.Seats {
		if p != nil && !p.Folded && !p.SitOut && !p.AllIn {
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
