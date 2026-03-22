package model

// Street 表示当前手牌所处的下注回合阶段。
// 状态转移由 engine 中的 advanceOrEnd / nextStreet 驱动：
//
//	Idle → PreFlop → Flop → Turn → River → Showdown → Idle
type Street uint8

const (
	StreetIdle     Street = 0 // 闲置：等待足够玩家（≥2）后启动下一手
	StreetPreFlop  Street = 1 // 翻牌前：底牌已发，盲注已下
	StreetFlop     Street = 2 // 翻牌：3 张公共牌已揭开
	StreetTurn     Street = 3 // 转牌：第 4 张公共牌已揭开
	StreetRiver    Street = 4 // 河牌：第 5 张公共牌已揭开
	StreetShowdown Street = 5 // 摊牌：结算并分配底池
)

// String 返回街道的可读名称，用于日志和 JSON 序列化。
func (s Street) String() string {
	switch s {
	case StreetIdle:
		return "idle"
	case StreetPreFlop:
		return "preflop"
	case StreetFlop:
		return "flop"
	case StreetTurn:
		return "turn"
	case StreetRiver:
		return "river"
	case StreetShowdown:
		return "showdown"
	default:
		return "unknown"
	}
}
