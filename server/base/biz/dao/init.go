package dao

import (
	"fmt"
	"log/slog"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/spf13/viper"
)

// DBM is the global MySQL connection (master).
var DBM *sqlx.DB

var (
	UserDao = &userDao{}
	RoomDao = &roomDao{}
)

// Init connects to MySQL using config from Viper and runs AutoMigrate.
func Init() {
	dsn := viper.GetString("mysql.dsn")
	db, err := sqlx.Connect("mysql", dsn)
	if err != nil {
		panic(fmt.Errorf("dao.Init: connect mysql: %w", err))
	}
	db.SetMaxOpenConns(50)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)
	DBM = db
	slog.Info("dao: connected to mysql")

	if err := autoMigrate(); err != nil {
		panic(fmt.Errorf("dao.Init: auto-migrate: %w", err))
	}
}

func autoMigrate() error {
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
		if _, err := DBM.Exec(stmt); err != nil {
			return err
		}
	}
	slog.Info("dao: auto-migrated 4 tables")
	return nil
}
