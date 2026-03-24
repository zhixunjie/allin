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
		if p.UserID == viewerID && gs.Street != gmodel.StreetIdle {
			ss.Hole = []string{p.Hole[0].String(), p.Hole[1].String()}
		}
		snap.Seats = append(snap.Seats, ss)
	}
	return snap
}
