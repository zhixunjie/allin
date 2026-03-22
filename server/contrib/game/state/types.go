package state

import (
	"github.com/allin/server/contrib/game/model"
	"github.com/allin/server/contrib/room"
)

// Street、Card、Player 的权威定义在 contrib/game/model 包中；
// 此处提供 type alias，保持现有调用方无需修改导入路径。
type (
	Street = model.Street // 当前手牌阶段
	Card   = model.Card   // 一张扑克牌
	Player = model.Player // 已入座的玩家
)

// Street 常量别名（alias 不重新导出常量，需在此列出）
const (
	StreetIdle     = model.StreetIdle
	StreetPreFlop  = model.StreetPreFlop
	StreetFlop     = model.StreetFlop
	StreetTurn     = model.StreetTurn
	StreetRiver    = model.StreetRiver
	StreetShowdown = model.StreetShowdown
)

// GameStateMachine 是一个房间的完整内存游戏状态。
type GameStateMachine struct {
	Street    Street     // 当前游戏阶段
	HandNum   int        // 本局手牌编号（从 1 开始递增）
	Community []Card     // 公共牌（0–5 张）
	Seats     [9]*Player // 座位数组，nil 表示空座

	DealerSeat int // 庄家按钮所在座位号
	SBSeat     int // 小盲所在座位号
	BBSeat     int // 大盲所在座位号
	ActionSeat int // 当前待行动的座位号；-1 表示无待行动

	CurrentBet int64 // 本回合当前最高下注额
	MinRaise   int64 // 最低加注增量（上次加注幅度）

	Config room.RoomConfig // 房间配置（盲注/买入等）
}
