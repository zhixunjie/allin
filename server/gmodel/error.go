package gmodel

import "errors"

var (
	// room 相关错误
	ErrRoomNotFound     = errors.New("room not found")                  // 找不到对应房间码的房间
	ErrRoomFull         = errors.New("room is full")                    // 房间人数已满
	ErrRoomCodeConflict = errors.New("code generation conflict, retry") // 房间码生成冲突，需重试

	// game 相关错误
	ErrGameNotActive   = errors.New("game not active")   // 游戏尚未开始或已结束
	ErrNotYourTurn     = errors.New("not your turn")     // 不是该玩家的行动回合
	ErrInvalidAction   = errors.New("invalid action")    // 当前局面下该行动不合法
	ErrInvalidAmount   = errors.New("invalid amount")    // 下注/加注金额不合法
	ErrPlayerNotSeated = errors.New("player not seated") // 玩家不在座位上
)
