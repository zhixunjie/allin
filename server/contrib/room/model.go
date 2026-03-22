package room

// SetBotType 表示房间 bot 的整体风格主题。
type SetBotType string

const (
	SetBotTypeMixed      SetBotType = "mixed"      // 混合（TAG→LAG→Station→Rock 循环分配）
	SetBotTypeAggressive SetBotType = "aggressive" // 激进主题（TAG 与 LAG 交替）
	SetBotTypePassive    SetBotType = "passive"    // 被动主题（Rock 与 Station 交替）
	SetBotTypeRandom     SetBotType = "random"     // 随机（每个 bot 独立随机选取风格）
)

// RoomState 表示房间的生命周期状态。
type RoomState string

const (
	RoomStateLobby   RoomState = "lobby"
	RoomStatePlaying RoomState = "playing"
)
