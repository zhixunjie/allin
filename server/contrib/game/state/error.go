package state

import "errors"

var (
	ErrNotYourTurn     = errors.New("not your turn")   // 不是该玩家的行动回合
	ErrInvalidAction   = errors.New("invalid action")  // 当前局面下该行动不合法
	ErrInvalidAmount   = errors.New("invalid amount")  // 下注/加注金额不合法
	ErrGameNotActive   = errors.New("game not active") // 游戏尚未开始或已结束
	ErrPlayerNotSeated = errors.New("player not seated")
)
