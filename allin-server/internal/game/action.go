package game

import (
	"errors"
	"fmt"
)

// Action names (must match ws.ActionCmd.Action values).
const (
	ActionFold  = "fold"
	ActionCheck = "check"
	ActionCall  = "call"
	ActionBet   = "bet"
	ActionRaise = "raise"
	ActionAllIn = "all_in"
)

var (
	ErrNotYourTurn    = errors.New("not your turn")
	ErrInvalidAction  = errors.New("invalid action")
	ErrInvalidAmount  = errors.New("invalid amount")
	ErrGameNotActive  = errors.New("game not active")
)

// ValidateAction checks whether the given action is legal for the player.
func ValidateAction(gs *GameState, userID, action string, amount int64) error {
	if gs.Street == StreetIdle || gs.Street == StreetShowdown {
		return ErrGameNotActive
	}
	p := gs.FindPlayer(userID)
	if p == nil {
		return errors.New("player not seated")
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
		// Calling is always legal when there's a bet to call.
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
		// Total amount player will have in (their current bet + raise amount from stack)
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

// ApplyAction mutates gs to apply the validated action.
// Returns true if the action was an aggression (bet/raise) requiring others to re-act.
func ApplyAction(gs *GameState, userID, action string, amount int64) bool {
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
			// All-in call
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
		newTotal := amount // 'amount' is the total the player wants to have bet after this action
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

	// If aggression: reset ActedThisStreet for all other active non-all-in players.
	if aggression {
		for _, op := range gs.Seats {
			if op != nil && op.UserID != userID && !op.Folded && !op.SitOut && !op.AllIn {
				op.ActedThisStreet = false
			}
		}
	}

	return aggression
}
