package dao

import (
	"fmt"

	"github.com/allin/server/base/biz/model"
)

type handHistoryDao struct{}

// Save 插入一条手牌历史记录。
func (d *handHistoryDao) Save(r model.HandHistoryRecord) error {
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
func (d *handHistoryDao) GetByRoomCode(code string, limit int) ([]model.HandHistoryEntry, error) {
	var rows []model.HandHistoryEntry
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
