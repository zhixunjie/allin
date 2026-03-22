package state

import (
	"github.com/allin/server/contrib/game/model"
	"github.com/allin/server/contrib/room"
)

// GameStateMachine 是一个房间的完整内存游戏状态。
type GameStateMachine struct {
	Street    model.Street     // 当前游戏阶段
	HandNum   int              // 本局手牌编号（从 1 开始递增）
	Community []model.Card     // 公共牌（0–5 张）
	Seats     [9]*model.Player // 座位数组，nil 表示空座

	DealerSeat int // 庄家按钮所在座位号
	SBSeat     int // 小盲所在座位号
	BBSeat     int // 大盲所在座位号
	ActionSeat int // 当前待行动的座位号；-1 表示无待行动

	CurrentBet int64 // 本回合当前最高下注额
	MinRaise   int64 // 最低加注增量（上次加注幅度）

	Config room.RoomConfig // 房间配置（盲注/买入等）
}
