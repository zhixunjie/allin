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

	// ActedThisStreet 标记本街道内该玩家是否已行动；
	// 当有玩家加注超过其下注额时重置为 false，迫使其重新决策。
	ActedThisStreet bool

	Disconnected bool // 是否因断线而暂时离开（手牌进行中保留座位）
}
