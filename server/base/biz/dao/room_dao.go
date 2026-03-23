package dao

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/allin/server/base/biz/model"
)

type roomDao struct{}

// Persist 将新房间写入 room_history 表，返回自增 ID。
func (d *roomDao) Persist(code string, hostUserID int64, cfgJSON []byte, createdAt time.Time) (int64, error) {
	query := fmt.Sprintf(`INSERT INTO %s (room_code, host_user_id, config_json, started_at) VALUES (?, ?, ?, ?)`, model.TableNameRoomHistory)
	result, err := DBM.Exec(query, code, hostUserID, cfgJSON, createdAt)
	if err != nil {
		return 0, fmt.Errorf("roomDao.Persist: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("roomDao.Persist: last insert id: %w", err)
	}
	return id, nil
}

// MarkEnded 将指定房间码的 ended_at 设为当前时间。
func (d *roomDao) MarkEnded(code string) {
	query := fmt.Sprintf(`UPDATE %s SET ended_at = NOW() WHERE room_code = ? AND ended_at IS NULL`, model.TableNameRoomHistory)
	if _, err := DBM.Exec(query, code); err != nil {
		slog.Error("roomDao.MarkEnded: failed", "code", code, "err", err)
	}
}
