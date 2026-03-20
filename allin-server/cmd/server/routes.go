package main

import (
	"net/http"

	"github.com/allin/server/internal/room"
	"github.com/allin/server/internal/user"
	"github.com/allin/server/internal/ws"
)

// registerRoutes 将所有 HTTP 路由挂载到 mux。
//
// 路由一览：
//
//	GET  /health                — 健康检查，返回 {"status":"ok"}，用于容器探活
//
//	POST /api/auth/register     — 注册新用户（无需鉴权）
//	POST /api/auth/login        — 登录，返回 JWT token（无需鉴权）
//	GET  /api/me                — 获取当前登录用户信息（需 JWT）
//
//	POST /api/rooms             — 创建房间，返回房间码和配置（需 JWT）
//	GET  /api/rooms/{code}      — 查询房间基本信息（无需鉴权，供加入前预览）
//
//	GET  /api/ws?room=CODE      — WebSocket 升级入口，JWT 通过 ?token= 或 Authorization 头传入
func registerRoutes(
	mux *http.ServeMux,
	userHandler *user.Handler,
	roomHandler *room.Handler,
	wsHandler *ws.Handler,
	authMW func(http.Handler) http.Handler,
) {
	// ── 健康检查 ──────────────────────────────────────────────────────
	// 不依赖数据库，仅确认进程存活；可用于 k8s liveness probe。
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`)) //nolint:errcheck
	})

	// ── 认证 ──────────────────────────────────────────────────────────
	// 注册：校验用户名/密码格式，bcrypt 哈希存储，返回 JWT + 用户信息。
	mux.HandleFunc("POST /api/auth/register", userHandler.Register)
	// 登录：校验凭证，返回 JWT（有效期 7 天）+ 用户信息。
	mux.HandleFunc("POST /api/auth/login", userHandler.Login)
	// 获取当前用户：解析 JWT 后查库返回完整 User 对象（含筹码余额）。
	mux.Handle("GET /api/me", authMW(http.HandlerFunc(userHandler.Me)))

	// ── 房间 ──────────────────────────────────────────────────────────
	// 创建房间：按请求体中的 RoomConfig 生成 6 位房间码，持久化到内存。
	mux.Handle("POST /api/rooms", authMW(http.HandlerFunc(roomHandler.Create)))
	// 查询房间：通过房间码获取配置和状态，用于前端加入前的信息展示。
	mux.HandleFunc("GET /api/rooms/{code}", roomHandler.Get)

	// ── WebSocket ─────────────────────────────────────────────────────
	// 游戏实时通道：HTTP → WS 升级，鉴权后加入对应房间的 Hub。
	// Hub 不存在时自动创建并启动 Game Engine goroutine。
	// Token 可通过 ?token=xxx 或 Authorization: Bearer xxx 传入。
	mux.HandleFunc("GET /api/ws", wsHandler.ServeWS)
}
