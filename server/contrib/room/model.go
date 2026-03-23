package room

// RoomState 表示房间的生命周期状态。
type RoomState string

const (
	RoomStateLobby   RoomState = "lobby"
	RoomStatePlaying RoomState = "playing"
)
