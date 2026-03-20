package main

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/allin/server/internal/auth"
	"github.com/allin/server/internal/config"
	"github.com/allin/server/internal/game"
	"github.com/allin/server/internal/room"
	"github.com/allin/server/internal/store"
	"github.com/allin/server/internal/user"
	"github.com/allin/server/internal/ws"
)

func main() {
	// 结构化日志
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	// 加载配置
	cfg := config.Load()

	// 连接 MySQL
	if err := store.Connect(cfg.MySQLDSN); err != nil {
		slog.Error("failed to connect to mysql", "err", err)
		os.Exit(1)
	}
	defer store.DB.Close()

	// 自动建表
	if err := store.AutoMigrate(); err != nil {
		slog.Error("failed to auto-migrate", "err", err)
		os.Exit(1)
	}

	// 初始化处理器
	roomManager := room.NewManager()
	userHandler := user.NewHandler(cfg.JWTSecret)
	roomHandler := room.NewHandler(roomManager)
	wsHandler := ws.NewHandler(roomManager, cfg.JWTSecret)
	authMW := auth.Middleware(cfg.JWTSecret)

	// 引擎注册表，用于优雅关闭。
	registry := game.NewRegistry()

	// 连接游戏引擎：每当创建新 hub 时启动一个 Engine goroutine。
	wsHandler.SetEngineStarter(func(hub *ws.Hub, rm *room.Room) {
		eng := game.NewEngine(hub, rm, registry)
		eng.SetOnEmpty(func() {
			roomManager.Close(rm.Code)
			wsHandler.RemoveHub(rm.Code)
			slog.Info("room: closed after last player left", "code", rm.Code)
		})
		go eng.Run()
	})

	// 房间 GC：每 5 分钟清理空闲超过 30 分钟的房间。
	roomManager.StartGC(5*time.Minute, 30*time.Minute, wsHandler.ClientCount)

	// 路由（详见 routes.go）
	mux := http.NewServeMux()
	registerRoutes(mux, userHandler, roomHandler, wsHandler, authMW)

	// CORS + 日志中间件
	handler := corsMiddleware(cfg.AllowOrigins, loggingMiddleware(mux))

	// HTTP 服务器
	srv := &http.Server{
		Addr:         cfg.ServerAddr,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 0, // 为 WebSocket 连接禁用写超时
		IdleTimeout:  60 * time.Second,
	}

	// 启动服务器
	go func() {
		slog.Info("server starting", "addr", cfg.ServerAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	// 收到 SIGINT/SIGTERM 时优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down...")

	// 先停止所有游戏引擎（完成或中止进行中的牌局）。
	registry.StopAll()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("shutdown error", "err", err)
	}
	slog.Info("server stopped")
}

// corsMiddleware 为已配置的来源添加 CORS 响应头。
func corsMiddleware(allowOrigins []string, next http.Handler) http.Handler {
	originSet := make(map[string]struct{}, len(allowOrigins))
	for _, o := range allowOrigins {
		originSet[o] = struct{}{}
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if _, ok := originSet[origin]; ok {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// loggingMiddleware 记录每个请求的方法、路径和耗时。
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w, status: 200}
		next.ServeHTTP(rw, r)
		slog.Info("http",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rw.status,
			"dur", time.Since(start).String(),
		)
	})
}

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

// Hijack 委托给底层 ResponseWriter，以支持 WebSocket 升级。
func (rw *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h, ok := rw.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("underlying ResponseWriter does not implement http.Hijacker")
	}
	return h.Hijack()
}
