package model

// Action 表示玩家行动类型（值与 ws/protocol 的 JSON 字段匹配）。
type Action string

const (
	ActionFold  Action = "fold"   // 弃牌：放弃本手
	ActionCheck Action = "check"  // 让牌：无需跟注时过牌
	ActionCall  Action = "call"   // 跟注：跟上当前最大下注
	ActionBet   Action = "bet"    // 下注：本街首次主动投入筹码
	ActionRaise Action = "raise"  // 加注：在已有下注基础上继续提高
	ActionAllIn Action = "all_in" // 全押：将所有筹码投入底池
)
