package ws

import (
	"encoding/json"
	"time"
)

// ---- Message types ----
// Server → Client (events)
const (
	TypeConnected      = "connected"
	TypePlayerJoined   = "player_joined"
	TypePlayerLeft     = "player_left"
	TypeGameStarted    = "game_started"
	TypeHoleCards      = "hole_cards"
	TypeCardsDealt     = "cards_dealt"
	TypeStreetStarted  = "street_started"
	TypeActionRequired = "action_required"
	TypeActionTaken    = "action_taken"
	TypeActionTimeout  = "action_timeout"
	TypeShowdown       = "showdown"
	TypeHandResult     = "hand_result"
	TypeChatMessage    = "chat_message"
	TypeError          = "error"
	TypeStackUpdated   = "stack_updated"
)

// Client → Server (commands)
const (
	CmdJoinRoom = "join_room"
	CmdAction   = "action"
	CmdChat     = "chat"
	CmdAddChips = "add_chips"
	CmdSitOut   = "sit_out"
)

// Envelope is the common wrapper for all WebSocket messages.
type Envelope struct {
	Type    string          `json:"type"`
	Seq     int64           `json:"seq"`
	Ts      int64           `json:"ts"` // unix milliseconds
	Payload json.RawMessage `json:"payload"`
}

// NewEvent builds a server-sent event envelope with the current timestamp.
func NewEvent(msgType string, payload any) (Envelope, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return Envelope{}, err
	}
	return Envelope{
		Type:    msgType,
		Ts:      time.Now().UnixMilli(),
		Payload: raw,
	}, nil
}

// MustEvent is like NewEvent but panics on marshal error (safe for known types).
func MustEvent(msgType string, payload any) Envelope {
	e, err := NewEvent(msgType, payload)
	if err != nil {
		panic(err)
	}
	return e
}

// ---- Payload structs ----

// ConnectedPayload is sent to a client upon successful WebSocket connection.
type ConnectedPayload struct {
	PlayerID    string `json:"player_id"`
	DisplayName string `json:"display_name"`
	RoomCode    string `json:"room_code"`
	// GameSnapshot will be added in Phase 2
}

// PlayerJoinedPayload is broadcast when a player joins or reconnects.
type PlayerJoinedPayload struct {
	PlayerID    string `json:"player_id"`
	DisplayName string `json:"display_name"`
	SeatIndex   int    `json:"seat_index"`
	Stack       int64  `json:"stack"`
	IsReconnect bool   `json:"is_reconnect"`
}

// PlayerLeftPayload is broadcast when a player disconnects.
type PlayerLeftPayload struct {
	PlayerID  string `json:"player_id"`
	SeatIndex int    `json:"seat_index"`
}

// ChatPayload carries a chat message.
type ChatPayload struct {
	SenderID    string `json:"sender_id"`
	DisplayName string `json:"display_name"`
	Text        string `json:"text"`
	Ts          int64  `json:"ts"`
}

// ErrorPayload carries an error response to a client command.
type ErrorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	RefSeq  int64  `json:"ref_seq"`
}

// ---- Incoming command payloads ----

// JoinRoomCmd is the payload for CmdJoinRoom.
type JoinRoomCmd struct {
	RoomCode  string `json:"room_code"`
	SeatIndex *int   `json:"seat_index"` // optional preferred seat
}

// ActionCmd is the payload for CmdAction.
type ActionCmd struct {
	Action string `json:"action"` // fold/check/call/bet/raise/all_in
	Amount int64  `json:"amount"`
}

// ChatCmd is the payload for CmdChat.
type ChatCmd struct {
	Text string `json:"text"`
}

// AddChipsCmd is the payload for CmdAddChips.
type AddChipsCmd struct {
	Amount int64 `json:"amount"`
}

// SitOutCmd is the payload for CmdSitOut.
type SitOutCmd struct {
	SitOut bool `json:"sit_out"`
}
