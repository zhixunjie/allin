package user

import "time"

// User 表示一个已注册的玩家。
type User struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	DisplayName  string    `json:"display_name"`
	ChipBalance  int64     `json:"chip_balance"`
	CreatedAt    time.Time `json:"created_at"`
}
