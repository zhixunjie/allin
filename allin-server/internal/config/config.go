package config

import (
	"os"
	"strings"
)

// Config holds all application configuration loaded from environment variables.
type Config struct {
	// Server
	ServerAddr string // e.g. ":8080"

	// MySQL
	MySQLDSN string // e.g. user:pass@tcp(127.0.0.1:3306)/allin?parseTime=true&charset=utf8mb4

	// JWT
	JWTSecret string

	// CORS
	AllowOrigins []string
}

// Load reads configuration from environment variables with sensible defaults.
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
