package state

import (
	"fmt"

	"github.com/allin/server/gmodel"
)

// ValidateAction 检查给定的行动对该玩家是否合法。
func (gs *GameStateMachine) ValidateAction(userID string, action gmodel.Action, amount int64) error {
	if gs.Street == gmodel.StreetIdle || gs.Street == gmodel.StreetShowdown {
		return gmodel.ErrGameNotActive
	}
	p := gs.FindPlayer(userID)
	if p == nil {
		return gmodel.ErrPlayerNotSeated
	}
	if gs.ActionSeat != p.SeatIndex {
		return gmodel.ErrNotYourTurn
	}
	if p.Folded || p.AllIn {
		return gmodel.ErrInvalidAction
	}

	switch action {
	case gmodel.ActionFold:
		return nil

	case gmodel.ActionCheck:
		if p.Bet < gs.CurrentBet {
			return fmt.Errorf("%w: must call %d or raise", gmodel.ErrInvalidAction, gs.CurrentBet-p.Bet)
		}
		return nil

	case gmodel.ActionCall:
		if p.Bet >= gs.CurrentBet {
			return fmt.Errorf("%w: no bet to call, use check", gmodel.ErrInvalidAction)
		}
		return nil

	case gmodel.ActionBet:
		if gs.CurrentBet > 0 {
			return fmt.Errorf("%w: there is already a bet, use raise", gmodel.ErrInvalidAction)
		}
		if amount < gs.Config.BigBlind {
			return fmt.Errorf("%w: bet must be at least %d (one big blind)", gmodel.ErrInvalidAmount, gs.Config.BigBlind)
		}
		if amount > p.Stack {
			return fmt.Errorf("%w: not enough chips", gmodel.ErrInvalidAmount)
		}
		return nil

	case gmodel.ActionRaise:
		if gs.CurrentBet == 0 {
			return fmt.Errorf("%w: no bet to raise, use bet", gmodel.ErrInvalidAction)
		}
		toCall := gs.CurrentBet - p.Bet
		minRaiseTotal := gs.CurrentBet + gs.MinRaise
		if amount < minRaiseTotal && amount != p.Stack+p.Bet {
			return fmt.Errorf("%w: raise must be to at least %d", gmodel.ErrInvalidAmount, minRaiseTotal)
		}
		if amount > p.Stack+p.Bet {
			return fmt.Errorf("%w: raise amount exceeds stack", gmodel.ErrInvalidAmount)
		}
		if toCall >= p.Stack {
			return fmt.Errorf("%w: not enough chips to raise, use all_in", gmodel.ErrInvalidAmount)
		}
		return nil

	case gmodel.ActionAllIn:
		if p.Stack == 0 {
			return fmt.Errorf("%w: no chips to go all-in", gmodel.ErrInvalidAmount)
		}
		return nil

	default:
		return fmt.Errorf("%w: unknown action %q", gmodel.ErrInvalidAction, action)
	}
}

// ApplyAction 修改 gs 以应用已验证的行动。
// 如果行动是激进行为（下注/加注）需要其他人重新行动，则返回 true。
func (gs *GameStateMachine) ApplyAction(userID string, action gmodel.Action, amount int64) bool {
	p := gs.FindPlayer(userID)
	if p == nil {
		return false
	}
	aggression := false

	switch action {
	case gmodel.ActionFold:
		p.Folded = true
		p.ActedThisStreet = true

	case gmodel.ActionCheck:
		p.ActedThisStreet = true

	case gmodel.ActionCall:
		toCall := gs.CurrentBet - p.Bet
		if toCall > p.Stack {
			toCall = p.Stack
			p.AllIn = true
		}
		p.Bet += toCall
		p.TotalBet += toCall
		p.Stack -= toCall
		p.ActedThisStreet = true

	case gmodel.ActionBet:
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

	case gmodel.ActionRaise:
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

	case gmodel.ActionAllIn:
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

	// 激进行为（bet/raise）：抬高了 CurrentBet，其他仍可行动的玩家须重新决策。
	if aggression {
		for _, op := range gs.Seats {
			if op != nil && op.UserID != userID && op.CanBet() {
				op.ActedThisStreet = false
			}
		}
	}

	return aggression
}
