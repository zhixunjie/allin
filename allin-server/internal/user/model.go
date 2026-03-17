package user

import "time"

// User represents a registered player.
type User struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	DisplayName  string    `json:"display_name"`
	ChipBalance  int64     `json:"chip_balance"`
	CreatedAt    time.Time `json:"created_at"`
}
