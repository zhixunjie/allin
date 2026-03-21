package model

import (
	"errors"
	"time"
)

var (
	ErrUserNotFound  = errors.New("user not found")
	ErrUsernameTaken = errors.New("username already taken")
)

// User represents a registered player.
type User struct {
	ID           int64     `json:"id"           db:"id"`
	Username     string    `json:"username"     db:"username"`
	PasswordHash string    `json:"-"            db:"password_hash"`
	DisplayName  string    `json:"display_name" db:"display_name"`
	ChipBalance  int64     `json:"chip_balance" db:"chip_balance"`
	CreatedAt    time.Time `json:"created_at"   db:"created_at"`
}
