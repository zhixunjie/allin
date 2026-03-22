package model

import (
	"errors"
	"time"
)

const (
	TableNameUser       = "users"       // 用户账户表
	TableNameChipLedger = "chip_ledger" // 筹码变动账本表
)

var (
	ErrUserNotFound  = errors.New("user not found")
	ErrUsernameTaken = errors.New("username already taken")
)

// User 表示一个已注册的玩家账户。
type User struct {
	ID           int64     `json:"id"           db:"id"`            // 自增主键
	Username     string    `json:"username"     db:"username"`      // 登录用户名（唯一）
	PasswordHash string    `json:"-"            db:"password_hash"` // bcrypt 哈希，不对外暴露
	DisplayName  string    `json:"display_name" db:"display_name"`  // 前端展示名
	ChipBalance  int64     `json:"chip_balance" db:"chip_balance"`  // 账户筹码余额
	CreatedAt    time.Time `json:"created_at"   db:"created_at"`    // 注册时间（UTC）
}
