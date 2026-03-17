package user

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/allin/server/internal/store"
)

var ErrNotFound = errors.New("user not found")
var ErrUsernameTaken = errors.New("username already taken")

// Create inserts a new user.
func Create(u *User) error {
	_, err := store.DB.Exec(
		`INSERT INTO users (id, username, password_hash, display_name, chip_balance, created_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		u.ID, u.Username, u.PasswordHash, u.DisplayName, u.ChipBalance, u.CreatedAt,
	)
	if err != nil {
		// MySQL duplicate entry error code 1062
		if isDuplicateEntry(err) {
			return ErrUsernameTaken
		}
		return fmt.Errorf("user.Create: %w", err)
	}
	return nil
}

// GetByUsername returns the user with the given username.
func GetByUsername(username string) (*User, error) {
	u := &User{}
	err := store.DB.QueryRow(
		`SELECT id, username, password_hash, display_name, chip_balance, created_at
		 FROM users WHERE username = ?`, username,
	).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.DisplayName, &u.ChipBalance, &u.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("user.GetByUsername: %w", err)
	}
	return u, nil
}

// GetByID returns the user with the given ID.
func GetByID(id string) (*User, error) {
	u := &User{}
	err := store.DB.QueryRow(
		`SELECT id, username, password_hash, display_name, chip_balance, created_at
		 FROM users WHERE id = ?`, id,
	).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.DisplayName, &u.ChipBalance, &u.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("user.GetByID: %w", err)
	}
	return u, nil
}

// AdjustChips atomically adds delta to a user's chip_balance.
// Use negative delta to deduct chips.
func AdjustChips(userID string, delta int64, reason, refID string) error {
	tx, err := store.DB.Begin()
	if err != nil {
		return fmt.Errorf("user.AdjustChips: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if _, err := tx.Exec(
		`UPDATE users SET chip_balance = chip_balance + ? WHERE id = ?`, delta, userID,
	); err != nil {
		return fmt.Errorf("user.AdjustChips: update: %w", err)
	}
	if _, err := tx.Exec(
		`INSERT INTO chip_ledger (user_id, delta, reason, ref_id, created_at)
		 VALUES (?, ?, ?, ?, NOW())`, userID, delta, reason, refID,
	); err != nil {
		return fmt.Errorf("user.AdjustChips: ledger: %w", err)
	}
	return tx.Commit()
}

// isDuplicateEntry detects MySQL error 1062 (duplicate key).
func isDuplicateEntry(err error) bool {
	if err == nil {
		return false
	}
	return contains(err.Error(), "Error 1062") || contains(err.Error(), "Duplicate entry")
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsStr(s, sub))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
