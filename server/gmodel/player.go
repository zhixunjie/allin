package gmodel

// Player 表示已入座的玩家。
type Player struct {
	UserID      string   // 玩家唯一 ID（bot 以 "bot_" 开头）
	DisplayName string   // 显示名称
	SeatIndex   int      // 座位号 0–8
	Stack       int64    // 未参与下注的筹码余额
	Bet         int64    // 当前回合累计下注额
	TotalBet    int64    // 本手牌总下注额（用于边池计算）
	Hole        [2]Card  // 手牌（两张底牌）
	Folded      bool     // 是否已弃牌
	AllIn       bool     // 是否全押
	SitOut      bool     // 是否离座等待
	IsBot       bool     // 是否为 AI 机器人
	BotStyle    BotStyle // bot 风格标识（仅 IsBot=true 时有效）

	// 标记本街道内该玩家是否已行动；
	// 当有玩家加注超过其下注额时重置为 false，迫使其重新决策。
	ActedThisStreet bool

	// 是否因断线而暂时离开（手牌进行中保留座位）
	Disconnected bool

	// 是否在手牌进行中入座，需等待下一手牌才能参与。
	// 由引擎在中途入座时自动置 true，下一手牌开始时自动清除，无需玩家手动操作。
	WaitForNextHand bool
}

// ── 玩家状态判断方法 ─────────────────────────────────────────────────────────

// DealsIn 本手牌是否应发底牌：未离座且未等待下一手。
// 用于：发牌循环、dealtSeats 列表、底牌私发过滤。
func (p *Player) DealsIn() bool {
	return !p.SitOut && !p.WaitForNextHand
}

// Active 是否为本手牌的活跃参与者：已发牌、未弃牌。
// 用于：ActivePlayers、NextActiveSeat、摊牌评估。
func (p *Player) Active() bool {
	return !p.Folded && !p.SitOut && !p.WaitForNextHand
}

// CanBet 是否仍可参与本轮下注决策：活跃且未全押。
// 用于：BettingRoundOver、NextActableSeat、CanAct。
func (p *Player) CanBet() bool {
	return !p.Folded && !p.SitOut && !p.AllIn && !p.WaitForNextHand
}

// ReadyToStart 是否满足参与新手牌的条件：有筹码、未离座、未断线、未等待。
// 用于：EligibleToStart、scheduleNextHand bot 列表。
func (p *Player) ReadyToStart() bool {
	return !p.SitOut && !p.Disconnected && !p.WaitForNextHand && p.Stack > 0
}
