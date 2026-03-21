package dao

import (
	"fmt"

	"github.com/allin/server/base/biz/model"
)

type userDao struct{}

// Create inserts a new user row and sets u.ID from the auto-increment value.
func (d *userDao) Create(u *model.User) error {
	result, err := DBM.Exec(
		`INSERT INTO users (username, password_hash, display_name, chip_balance, created_at)
		 VALUES (?, ?, ?, ?, ?)`,
		u.Username, u.PasswordHash, u.DisplayName, u.ChipBalance, u.CreatedAt,
	)
	if err != nil {
		if isDuplicateEntry(err) {
			return model.ErrUsernameTaken
		}
		return fmt.Errorf("userDao.Create: %w", err)
	}
	u.ID, _ = result.LastInsertId()
	return nil
}

// GetByUsername returns the user with the given username.
func (d *userDao) GetByUsername(username string) (*model.User, error) {
	u := &model.User{}
	err := DBM.Get(u,
		`SELECT id, username, password_hash, display_name, chip_balance, created_at
		 FROM users WHERE username = ?`, username,
	)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return nil, model.ErrUserNotFound
		}
		return nil, fmt.Errorf("userDao.GetByUsername: %w", err)
	}
	return u, nil
}

// GetByID returns the user with the given ID.
func (d *userDao) GetByID(id int64) (*model.User, error) {
	u := &model.User{}
	err := DBM.Get(u,
		`SELECT id, username, password_hash, display_name, chip_balance, created_at
		 FROM users WHERE id = ?`, id,
	)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return nil, model.ErrUserNotFound
		}
		return nil, fmt.Errorf("userDao.GetByID: %w", err)
	}
	return u, nil
}

// AdjustChips atomically adds delta to chip_balance and writes a ledger entry.
func (d *userDao) AdjustChips(userID int64, delta int64, reason, refID string) error {
	tx, err := DBM.Begin()
	if err != nil {
		return fmt.Errorf("userDao.AdjustChips: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if _, err := tx.Exec(
		`UPDATE users SET chip_balance = chip_balance + ? WHERE id = ?`, delta, userID,
	); err != nil {
		return fmt.Errorf("userDao.AdjustChips: update: %w", err)
	}
	if _, err := tx.Exec(
		`INSERT INTO chip_ledger (user_id, delta, reason, ref_id, created_at)
		 VALUES (?, ?, ?, ?, NOW())`, userID, delta, reason, refID,
	); err != nil {
		return fmt.Errorf("userDao.AdjustChips: ledger: %w", err)
	}
	return tx.Commit()
}

func isDuplicateEntry(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return containsStr(s, "Error 1062") || containsStr(s, "Duplicate entry")
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
