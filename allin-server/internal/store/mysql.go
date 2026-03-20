package store

import (
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// DB 是全局数据库连接池。
var DB *sql.DB

// Connect 初始化 MySQL 连接池并验证连通性。
func Connect(dsn string) error {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("store: open mysql: %w", err)
	}
	db.SetMaxOpenConns(50)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		return fmt.Errorf("store: ping mysql: %w", err)
	}

	DB = db
	slog.Info("store: connected to mysql")
	return nil
}

// AutoMigrate 创建所有尚不存在的表。
func AutoMigrate() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id            CHAR(36)     NOT NULL PRIMARY KEY,
			username      VARCHAR(32)  NOT NULL,
			password_hash VARCHAR(60)  NOT NULL,
			display_name  VARCHAR(32)  NOT NULL,
			chip_balance  BIGINT       NOT NULL DEFAULT 10000,
			created_at    DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE KEY uk_username (username)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,

		`CREATE TABLE IF NOT EXISTS room_history (
			id           CHAR(36)    NOT NULL PRIMARY KEY,
			room_code    CHAR(6)     NOT NULL,
			host_user_id CHAR(36)    NOT NULL,
			config_json  JSON        NOT NULL,
			started_at   DATETIME    NOT NULL,
			ended_at     DATETIME,
			INDEX idx_room_code (room_code)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,

		`CREATE TABLE IF NOT EXISTS hand_history (
			id            CHAR(36)  NOT NULL PRIMARY KEY,
			room_id       CHAR(36)  NOT NULL,
			hand_num      INT       NOT NULL,
			players_json  JSON      NOT NULL,
			actions_json  JSON      NOT NULL,
			result_json   JSON      NOT NULL,
			played_at     DATETIME  NOT NULL,
			INDEX idx_hand_room (room_id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,

		`CREATE TABLE IF NOT EXISTS chip_ledger (
			id          BIGINT       NOT NULL AUTO_INCREMENT PRIMARY KEY,
			user_id     CHAR(36)     NOT NULL,
			delta       BIGINT       NOT NULL,
			reason      VARCHAR(32)  NOT NULL,
			ref_id      CHAR(36),
			created_at  DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
			INDEX idx_ledger_user (user_id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
	}

	for _, stmt := range stmts {
		if _, err := DB.Exec(stmt); err != nil {
			return fmt.Errorf("store: auto-migrate: %w", err)
		}
	}
	slog.Info("store: auto-migrated 4 tables")
	return nil
}
