package mw

import (
	"context"
	"net"
	"sync"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
)

// fixedWindowLimiter 是一个基于固定时间窗口的 IP 速率限制器。
type fixedWindowLimiter struct {
	mu       sync.Mutex
	windows  map[string]*windowEntry // key = IP 地址
	limit    int                     // 每个窗口内允许的最大请求数
	duration time.Duration           // 窗口时长
}

type windowEntry struct {
	count   int       // 当前窗口已请求次数
	resetAt time.Time // 窗口重置时间
}

func newFixedWindowLimiter(limit int, duration time.Duration) *fixedWindowLimiter {
	return &fixedWindowLimiter{
		windows:  make(map[string]*windowEntry),
		limit:    limit,
		duration: duration,
	}
}

func (l *fixedWindowLimiter) allow(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	e, ok := l.windows[ip]
	if !ok || now.After(e.resetAt) {
		l.windows[ip] = &windowEntry{count: 1, resetAt: now.Add(l.duration)}
		return true
	}
	if e.count >= l.limit {
		return false
	}
	e.count++
	return true
}

// authLimiter 限制认证接口：每个 IP 每分钟最多 10 次请求。
var authLimiter = newFixedWindowLimiter(10, time.Minute)

// RateLimitAuth 对登录/注册接口按 IP 进行速率限制（10 次/分钟）。
func RateLimitAuth() app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		ip, _, err := net.SplitHostPort(c.RemoteAddr().String())
		if err != nil {
			ip = c.RemoteAddr().String()
		}
		if !authLimiter.allow(ip) {
			c.JSON(429, map[string]string{"error": "too many requests, please try again later"})
			c.Abort()
			return
		}
		c.Next(ctx)
	}
}
