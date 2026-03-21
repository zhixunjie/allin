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
	UserDao        = &userDao{}
	RoomDao        = &roomDao{}
	HandHistoryDao = &handHistoryDao{}
)

// Init connects to MySQL using config from Viper.
// 表结构由 docs/sql/allin.sql 维护，启动前手动执行一次即可。
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
}
