package dao

import (
	"fmt"
	"time"
)

type roomDao struct{}

// Persist writes a new room to room_history and returns the auto-increment ID.
func (d *roomDao) Persist(code string, hostUserID int64, cfgJSON []byte, createdAt time.Time) (int64, error) {
	result, err := DBM.Exec(
		`INSERT INTO room_history (room_code, host_user_id, config_json, started_at)
		 VALUES (?, ?, ?, ?)`,
		code, hostUserID, cfgJSON, createdAt,
	)
	if err != nil {
		return 0, fmt.Errorf("roomDao.Persist: %w", err)
	}
	id, _ := result.LastInsertId()
	return id, nil
}

// MarkEnded sets ended_at for the given room code.
func (d *roomDao) MarkEnded(code string) {
	DBM.Exec( //nolint:errcheck
		`UPDATE room_history SET ended_at = NOW() WHERE room_code = ? AND ended_at IS NULL`, code,
	)
}
