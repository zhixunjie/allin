package state

import (
	"fmt"

	"github.com/allin/server/contrib/game/model"
)

// Action 表示玩家行动类型（值必须与 ws.ActionCmd.Action 的 JSON 字段匹配）。
type Action string

const (
	ActionFold  Action = "fold"   // 弃牌：放弃本手
	ActionCheck Action = "check"  // 让牌：无需跟注时过牌
	ActionCall  Action = "call"   // 跟注：跟上当前最大下注
	ActionBet   Action = "bet"    // 下注：本街首次主动投入筹码
	ActionRaise Action = "raise"  // 加注：在已有下注基础上继续提高
	ActionAllIn Action = "all_in" // 全押：将所有筹码投入底池
)

// ValidateAction 检查给定的行动对该玩家是否合法。
func (gs *GameStateMachine) ValidateAction(userID string, action Action, amount int64) error {
	if gs.Street == model.StreetIdle || gs.Street == model.StreetShowdown {
		return ErrGameNotActive
	}
	p := gs.FindPlayer(userID)
	if p == nil {
		return ErrPlayerNotSeated
	}
	if gs.ActionSeat != p.SeatIndex {
		return ErrNotYourTurn
	}
	if p.Folded || p.AllIn {
		return ErrInvalidAction
	}

	switch action {
	case ActionFold:
		return nil

	case ActionCheck:
		if p.Bet < gs.CurrentBet {
			return fmt.Errorf("%w: must call %d or raise", ErrInvalidAction, gs.CurrentBet-p.Bet)
		}
		return nil

	case ActionCall:
		if p.Bet >= gs.CurrentBet {
			return fmt.Errorf("%w: no bet to call, use check", ErrInvalidAction)
		}
		return nil

	case ActionBet:
		if gs.CurrentBet > 0 {
			return fmt.Errorf("%w: there is already a bet, use raise", ErrInvalidAction)
		}
		if amount < gs.Config.BigBlind {
			return fmt.Errorf("%w: bet must be at least %d (one big blind)", ErrInvalidAmount, gs.Config.BigBlind)
		}
		if amount > p.Stack {
			return fmt.Errorf("%w: not enough chips", ErrInvalidAmount)
		}
		return nil

	case ActionRaise:
		if gs.CurrentBet == 0 {
			return fmt.Errorf("%w: no bet to raise, use bet", ErrInvalidAction)
		}
		toCall := gs.CurrentBet - p.Bet
		minRaiseTotal := gs.CurrentBet + gs.MinRaise
		if amount < minRaiseTotal && amount != p.Stack+p.Bet {
			return fmt.Errorf("%w: raise must be to at least %d", ErrInvalidAmount, minRaiseTotal)
		}
		if toCall >= p.Stack {
			return fmt.Errorf("%w: not enough chips to raise, use all_in", ErrInvalidAmount)
		}
		return nil

	case ActionAllIn:
		if p.Stack == 0 {
			return fmt.Errorf("%w: no chips to go all-in", ErrInvalidAmount)
		}
		return nil

	default:
		return fmt.Errorf("%w: unknown action %q", ErrInvalidAction, action)
	}
}

// ApplyAction 修改 gs 以应用已验证的行动。
// 如果行动是激进行为（下注/加注）需要其他人重新行动，则返回 true。
func (gs *GameStateMachine) ApplyAction(userID string, action Action, amount int64) bool {
	p := gs.FindPlayer(userID)
	if p == nil {
		return false
	}
	aggression := false

	switch action {
	case ActionFold:
		p.Folded = true
		p.ActedThisStreet = true

	case ActionCheck:
		p.ActedThisStreet = true

	case ActionCall:
		toCall := gs.CurrentBet - p.Bet
		if toCall > p.Stack {
			toCall = p.Stack
			p.AllIn = true
		}
		p.Bet += toCall
		p.TotalBet += toCall
		p.Stack -= toCall
		p.ActedThisStreet = true

	case ActionBet:
		if amount > p.Stack {
			amount = p.Stack
			p.AllIn = true
		}
		p.Bet += amount
		p.TotalBet += amount
		p.Stack -= amount
		gs.MinRaise = amount
		gs.CurrentBet = p.Bet
		p.ActedThisStreet = true
		aggression = true

	case ActionRaise:
		newTotal := amount
		if newTotal > p.Bet+p.Stack {
			newTotal = p.Bet + p.Stack
			p.AllIn = true
		}
		raiseBy := newTotal - gs.CurrentBet
		added := newTotal - p.Bet
		p.Stack -= added
		p.TotalBet += added
		p.Bet = newTotal
		gs.MinRaise = raiseBy
		gs.CurrentBet = newTotal
		p.ActedThisStreet = true
		aggression = true

	case ActionAllIn:
		added := p.Stack
		p.Bet += added
		p.TotalBet += added
		p.Stack = 0
		p.AllIn = true
		if p.Bet > gs.CurrentBet {
			gs.MinRaise = p.Bet - gs.CurrentBet
			gs.CurrentBet = p.Bet
			p.ActedThisStreet = true
			aggression = true
		} else {
			p.ActedThisStreet = true
		}
	}

	// 激进行为：重置所有其他活跃非全押玩家的 ActedThisStreet。
	if aggression {
		for _, op := range gs.Seats {
			if op != nil && op.UserID != userID && !op.Folded && !op.SitOut && !op.AllIn {
				op.ActedThisStreet = false
			}
		}
	}

	return aggression
}
