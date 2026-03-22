package dao

import (
	"fmt"
	"log/slog"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/spf13/viper"
)

// DBM 是全局 MySQL 主库连接，由 Init() 初始化后供各 dao 使用。
var DBM *sqlx.DB

var (
	UserDao        = &userDao{}        // 用户表 DAO
	RoomDao        = &roomDao{}        // 房间历史 DAO
	HandHistoryDao = &handHistoryDao{} // 手牌历史 DAO
)

// Init 使用 Viper 配置连接 MySQL。
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
