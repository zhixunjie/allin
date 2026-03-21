package dao

import (
	"encoding/json"
	"fmt"
	"time"
)

type handHistoryDao struct{}

// HandHistoryRecord 是保存到 hand_history 表的单条记录。
type HandHistoryRecord struct {
	RoomID      int64           `db:"room_id"`
	HandNum     int             `db:"hand_num"`
	PlayersJSON json.RawMessage `db:"players_json"`
	ActionsJSON json.RawMessage `db:"actions_json"`
	ResultJSON  json.RawMessage `db:"result_json"`
	PlayedAt    time.Time       `db:"played_at"`
}

// Save 插入一条手牌历史记录。
func (d *handHistoryDao) Save(r HandHistoryRecord) error {
	_, err := DBM.Exec(
		`INSERT INTO hand_history (room_id, hand_num, players_json, actions_json, result_json, played_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		r.RoomID, r.HandNum, r.PlayersJSON, r.ActionsJSON, r.ResultJSON, r.PlayedAt,
	)
	if err != nil {
		return fmt.Errorf("handHistoryDao.Save: %w", err)
	}
	return nil
}
