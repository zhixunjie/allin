package dao

import (
	"fmt"
	"time"
)

type roomDao struct{}

// Persist writes a new room to room_history.
func (d *roomDao) Persist(id, code, hostUserID string, cfgJSON []byte, createdAt time.Time) error {
	_, err := DBM.Exec(
		`INSERT INTO room_history (id, room_code, host_user_id, config_json, started_at)
		 VALUES (?, ?, ?, ?, ?)`,
		id, code, hostUserID, cfgJSON, createdAt,
	)
	if err != nil {
		return fmt.Errorf("roomDao.Persist: %w", err)
	}
	return nil
}

// MarkEnded sets ended_at for the given room code.
func (d *roomDao) MarkEnded(code string) {
	DBM.Exec( //nolint:errcheck
		`UPDATE room_history SET ended_at = NOW() WHERE room_code = ? AND ended_at IS NULL`, code,
	)
}
