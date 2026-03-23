package state

import "github.com/allin/server/gmodel"

// BotSituation 描述 bot 做决策时的完整局面快照。
// 在触发 bot 行动前从 GameStateMachine 中提取，传入 Personality.decide。
type BotSituation struct {
	Street     model.Street  // 当前街道（PreFlop/Flop/Turn/River）
	Hole       [2]model.Card // bot 的两张底牌
	Community  []model.Card  // 当前公共牌（0–5 张）
	CurrentBet int64         // 本轮最大下注额（需跟注到此金额）
	PlayerBet  int64         // bot 本街已下注金额
	Stack      int64         // bot 当前桌面筹码
	BigBlind   int64         // 大盲注额（用于计算底池赔率和加注倍数）
	Pot        int64         // 当前总底池（含本街所有下注）
}

// NewBotSituation 从 GameStateMachine 和指定玩家中提取局面快照。
func NewBotSituation(gs *GameStateMachine, p *model.Player) BotSituation {
	community := make([]model.Card, len(gs.Community))
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
