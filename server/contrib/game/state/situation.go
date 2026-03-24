package state

import "github.com/allin/server/gmodel"

// NewBotSituation 从 GameStateMachine 和指定玩家中提取局面快照。
func NewBotSituation(gs *GameStateMachine, p *gmodel.Player) BotSituation {
	community := make([]gmodel.Card, len(gs.Community))
	copy(community, gs.Community)
	return BotSituation{
		Street:     gs.Street,
		Hole:       p.Hole,
		Community:  community,
		CurrentBet: gs.CurrentBet,
		PlayerBet:  p.Bet,
		Stack:      p.Stack,
		BigBlind:   gs.Config.BigBlind,
		Pot:        gs.TotalPot(),
	}
}
