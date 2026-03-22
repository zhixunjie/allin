package model

import "time"

// ChipLedger 对应 chip_ledger 表，记录每一笔筹码变动流水。
type ChipLedger struct {
	ID        int64     `json:"id"         db:"id"`         // 流水自增ID
	UserID    int64     `json:"user_id"    db:"user_id"`    // 用户ID
	Delta     int64     `json:"delta"      db:"delta"`      // 筹码变动量（正=收入，负=支出）
	Reason    string    `json:"reason"     db:"reason"`     // 变动原因（buy_in/cash_out/add_chips）
	RefID     string    `json:"ref_id"     db:"ref_id"`     // 关联业务ID（如房间码）；可为空
	CreatedAt time.Time `json:"created_at" db:"created_at"` // 记录时间（UTC）
}
