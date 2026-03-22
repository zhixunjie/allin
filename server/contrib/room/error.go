package room

import "errors"

var (
	ErrNotFound     = errors.New("room not found")
	ErrRoomFull     = errors.New("room is full")
	ErrCodeConflict = errors.New("code generation conflict, retry")
)
