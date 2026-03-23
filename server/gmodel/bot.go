package model

import "math/rand"

// RoomBotType 表示房间 bot 的整体风格主题。
type RoomBotType string

const (
	RoomBotTypeMixed      RoomBotType = "mixed"      // 混合（TAG→LAG→Station→Rock 循环分配）
	RoomBotTypeAggressive RoomBotType = "aggressive" // 激进主题（TAG 与 LAG 交替）
	RoomBotTypePassive    RoomBotType = "passive"    // 被动主题（Rock 与 Station 交替）
	RoomBotTypeRandom     RoomBotType = "random"     // 随机（每个 bot 独立随机选取风格）
)

var AllRoomBotType = []RoomBotType{RoomBotTypeMixed, RoomBotTypeAggressive, RoomBotTypePassive, RoomBotTypeRandom}

var AllBotStyle = []BotStyle{BotStyleTag, BotStyleLag, BotStyleStation, BotStyleRock}

// BotStyle 表示单个 bot 的风格流派（用于决策阈值映射）。
type BotStyle string

const (
	BotStyleTag     BotStyle = "tag"     // 紧凶（TAG）：选牌严格，有牌强打
	BotStyleLag     BotStyle = "lag"     // 松凶（LAG）：宽泛入局，频繁加注
	BotStyleStation BotStyle = "station" // 松被动（Calling Station）：喜欢跟注，很少主动
	BotStyleRock    BotStyle = "rock"    // 紧被动（Rock）：极度保守，只玩超强牌
)

// AssignStyle 根据房间风格主题（RoomBotType）和 bot 序号，返回单个 bot 的具体风格。
//
//	mixed（默认）: TAG→LAG→Station→Rock 循环
//	aggressive:    TAG→LAG 交替
//	passive:       Rock→Station 交替
//	random:        每个 bot 独立随机
func AssignStyle(roomStyle RoomBotType, index int) BotStyle {
	switch roomStyle {
	case RoomBotTypeAggressive:
		return []BotStyle{BotStyleTag, BotStyleLag}[index%2]
	case RoomBotTypePassive:
		return []BotStyle{BotStyleRock, BotStyleStation}[index%2]
	case RoomBotTypeRandom:
		return AllBotStyle[rand.Intn(len(AllBotStyle))]
	default: // mixed 或空字符串
		return AllBotStyle[index%len(AllBotStyle)]
	}
}
