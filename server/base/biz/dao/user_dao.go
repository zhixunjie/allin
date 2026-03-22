package dao

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/allin/server/base/biz/model"
)

type userDao struct{}

// Create 插入新用户行，并将自增 ID 写回 u.ID。
func (d *userDao) Create(u *model.User) error {
	query := fmt.Sprintf(`INSERT INTO %s (username, password_hash, display_name, chip_balance, created_at) VALUES (?, ?, ?, ?, ?)`, model.TableNameUser)
	result, err := DBM.Exec(query, u.Username, u.PasswordHash, u.DisplayName, u.ChipBalance, u.CreatedAt)
	if err != nil {
		if isDuplicateEntry(err) {
			return model.ErrUsernameTaken
		}
		return fmt.Errorf("userDao.Create: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("userDao.Create: last insert id: %w", err)
	}
	u.ID = id
	return nil
}

// GetByUsername 返回指定用户名的用户，不存在时返回 ErrUserNotFound。
func (d *userDao) GetByUsername(username string) (*model.User, error) {
	u := &model.User{}
	query := fmt.Sprintf(`SELECT id, username, password_hash, display_name, chip_balance, created_at FROM %s WHERE username = ?`, model.TableNameUser)
	err := DBM.Get(u, query, username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, model.ErrUserNotFound
		}
		return nil, fmt.Errorf("userDao.GetByUsername: %w", err)
	}
	return u, nil
}

// GetByID 返回指定 ID 的用户，不存在时返回 ErrUserNotFound。
func (d *userDao) GetByID(id int64) (*model.User, error) {
	u := &model.User{}
	query := fmt.Sprintf(`SELECT id, username, password_hash, display_name, chip_balance, created_at FROM %s WHERE id = ?`, model.TableNameUser)
	err := DBM.Get(u, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, model.ErrUserNotFound
		}
		return nil, fmt.Errorf("userDao.GetByID: %w", err)
	}
	return u, nil
}

// AdjustChips 原子地将 delta 加到 chip_balance，同时写入账本记录。
func (d *userDao) AdjustChips(userID int64, delta int64, reason, refID string) error {
	tx, err := DBM.Begin()
	if err != nil {
		return fmt.Errorf("userDao.AdjustChips: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if _, err := tx.Exec(
		fmt.Sprintf(`UPDATE %s SET chip_balance = chip_balance + ? WHERE id = ?`, model.TableNameUser),
		delta, userID,
	); err != nil {
		return fmt.Errorf("userDao.AdjustChips: update: %w", err)
	}
	if _, err := tx.Exec(
		fmt.Sprintf(`INSERT INTO %s (user_id, delta, reason, ref_id, created_at) VALUES (?, ?, ?, ?, NOW())`, model.TableNameChipLedger),
		userID, delta, reason, refID,
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
	return strings.Contains(s, "Error 1062") || strings.Contains(s, "Duplicate entry")
}
