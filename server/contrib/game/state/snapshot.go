package state

import "github.com/allin/server/gmodel"

// Snapshot 构建 GameSnapshot，仅为 viewerID 填充手牌。
func (gs *GameStateMachine) Snapshot(viewerID string) GameSnapshot {
	snap := GameSnapshot{
		Street:     gs.Street.String(),
		HandNum:    gs.HandNum,
		Community:  CardsToStrings(gs.Community),
		Pot:        gs.TotalPot(),
		DealerSeat: gs.DealerSeat,
		ActionSeat: gs.ActionSeat,
		CurrentBet: gs.CurrentBet,
		MinRaise:   gs.MinRaise,
		DeadlineTs: gs.ActionDeadlineMs,
		Config:     gs.Config,
	}
	for _, p := range gs.Seats {
		if p == nil {
			continue
		}
		ss := SeatSnapshot{
			SeatIndex:       p.SeatIndex,
			UserID:          p.UserID,
			DisplayName:     p.DisplayName,
			Stack:           p.Stack,
			Bet:             p.Bet,
			Folded:          p.Folded,
			AllIn:           p.AllIn,
			SitOut:          p.SitOut,
			Disconnected:    p.Disconnected,
			IsBot:           p.IsBot,
			WaitForNextHand: p.WaitForNextHand,
		}
		if gs.Street != gmodel.StreetIdle && !p.WaitForNextHand && !p.Folded {
			if p.UserID == viewerID {
				ss.Hole = []string{p.Hole[0].String(), p.Hole[1].String()}
			} else {
				ss.Hole = []string{"?", "?"}
			}
		}
		snap.Seats = append(snap.Seats, ss)
	}
	return snap
}
