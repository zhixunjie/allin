package model

// SetBotType 表示房间 bot 的整体风格主题。
type SetBotType string

const (
	RoomBotTypeMixed      SetBotType = "mixed"      // 混合（TAG→LAG→Station→Rock 循环分配）
	RoomBotTypeAggressive SetBotType = "aggressive" // 激进主题（TAG 与 LAG 交替）
	RoomBotTypePassive    SetBotType = "passive"    // 被动主题（Rock 与 Station 交替）
	RoomBotTypeRandom     SetBotType = "random"     // 随机（每个 bot 独立随机选取风格）
)

var AllRoomBotType = []SetBotType{
	RoomBotTypeMixed,
	RoomBotTypeAggressive,
	RoomBotTypePassive,
	RoomBotTypeRandom,
}
