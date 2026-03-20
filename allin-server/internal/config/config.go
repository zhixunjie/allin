package config

import (
	"os"
	"strings"
)

// Config 保存从环境变量加载的所有应用配置。
type Config struct {
	// 服务器
	ServerAddr string // 例如 ":8080"

	// MySQL
	MySQLDSN string // 例如 user:pass@tcp(127.0.0.1:3306)/allin?parseTime=true&charset=utf8mb4

	// JWT
	JWTSecret string

	// CORS
	AllowOrigins []string
}

// Load 从环境变量读取配置，提供合理的默认值。
func Load() *Config {
	return &Config{
		ServerAddr:   getEnv("SERVER_ADDR", ":8080"),
		MySQLDSN:     getEnv("MYSQL_DSN", "root:@tcp(127.0.0.1:13306)/allin?parseTime=true&charset=utf8mb4"),
		JWTSecret:    getEnv("JWT_SECRET", "change-me-in-production"),
		AllowOrigins: strings.Split(getEnv("ALLOW_ORIGINS", "http://localhost:5173"), ","),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
