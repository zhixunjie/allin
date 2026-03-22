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

// HandHistoryEntry 是查询接口返回给前端的简化视图。
type HandHistoryEntry struct {
	HandNum     int             `db:"hand_num"     json:"hand_num"`
	ResultJSON  json.RawMessage `db:"result_json"  json:"result"`
	ActionsJSON json.RawMessage `db:"actions_json" json:"actions"`
	PlayedAt    time.Time       `db:"played_at"    json:"played_at"`
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

// GetByRoomCode 返回指定房间最近 limit 手的结果摘要，按手牌编号倒序。
func (d *handHistoryDao) GetByRoomCode(code string, limit int) ([]HandHistoryEntry, error) {
	var rows []HandHistoryEntry
	err := DBM.Select(&rows,
		`SELECT hh.hand_num, hh.result_json, hh.actions_json, hh.played_at
		 FROM hand_history hh
		 JOIN room_history rh ON hh.room_id = rh.id
		 WHERE rh.room_code = ?
		 ORDER BY hh.hand_num DESC
		 LIMIT ?`,
		code, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("handHistoryDao.GetByRoomCode: %w", err)
	}
	return rows, nil
}
