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

	// 使用带下界守卫的原子更新，防止并发买入导致余额透支。
	// delta 为负数时，AND chip_balance + ? >= 0 确保余额不会低于零。
	result, err := tx.Exec(
		fmt.Sprintf(`UPDATE %s SET chip_balance = chip_balance + ? WHERE id = ? AND chip_balance + ? >= 0`, model.TableNameUser),
		delta, userID, delta,
	)
	if err != nil {
		return fmt.Errorf("userDao.AdjustChips: update: %w", err)
	}
	if delta < 0 {
		if rows, _ := result.RowsAffected(); rows == 0 {
			return fmt.Errorf("userDao.AdjustChips: %w", model.ErrInsufficientChips)
		}
	}
	if _, err := tx.Exec(
		fmt.Sprintf(`INSERT INTO %s (user_id, delta, reason, ref_id, created_at) VALUES (?, ?, ?, ?, NOW())`, model.TableNameChipLedger),
		userID, delta, reason, refID,
	); err != nil {
		return fmt.Errorf("userDao.AdjustChips: ledger: %w", err)
	}
	return tx.Commit()
}

// ClaimChipsIfBelow 原子地在 chip_balance < threshold 时将 delta 加到余额，并写账本。
// 返回 (true, nil) 表示成功补发；(false, nil) 表示余额已满足条件，未执行更新。
func (d *userDao) ClaimChipsIfBelow(userID int64, threshold, delta int64, reason string) (bool, error) {
	tx, err := DBM.Begin()
	if err != nil {
		return false, fmt.Errorf("userDao.ClaimChipsIfBelow: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	result, err := tx.Exec(
		fmt.Sprintf(`UPDATE %s SET chip_balance = chip_balance + ? WHERE id = ? AND chip_balance < ?`, model.TableNameUser),
		delta, userID, threshold,
	)
	if err != nil {
		return false, fmt.Errorf("userDao.ClaimChipsIfBelow: update: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return false, nil // 余额已满足条件，无需补发
	}
	if _, err := tx.Exec(
		fmt.Sprintf(`INSERT INTO %s (user_id, delta, reason, ref_id, created_at) VALUES (?, ?, ?, ?, NOW())`, model.TableNameChipLedger),
		userID, delta, reason, "",
	); err != nil {
		return false, fmt.Errorf("userDao.ClaimChipsIfBelow: ledger: %w", err)
	}
	return true, tx.Commit()
}

func isDuplicateEntry(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return strings.Contains(s, "Error 1062") || strings.Contains(s, "Duplicate entry")
}
