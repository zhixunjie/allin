package ws

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"

	"github.com/allin/server/contrib/auth"
	"github.com/allin/server/contrib/room"
)

// EngineStarter 是在为房间创建新 RoomConn 时调用的函数。
// 游戏引擎应在此回调中创建和启动。
type EngineStarter func(rc *RoomConn, rm *room.Room)

// Handler 负责处理 WebSocket 升级请求，并维护每个房间的 RoomConn 生命周期。
type Handler struct {
	roomManager   *room.Manager  // 用于按房间码查询房间元数据
	jwtSecret     string         // JWT 验证密钥，从 config.yaml 注入
	engineStarter EngineStarter  // 创建 RoomConn 时触发的回调，用于启动游戏引擎

	roomConnsMu sync.RWMutex          // 保护 roomConns 并发读写
	roomConns   map[string]*RoomConn  // 活跃 RoomConn 索引，key = 房间码
}

func NewHandler(roomManager *room.Manager, jwtSecret string) *Handler {
	return &Handler{
		roomManager: roomManager,
		jwtSecret:   jwtSecret,
		roomConns:   make(map[string]*RoomConn),
	}
}

// SetEngineStarter 注册创建 RoomConn 时启动游戏引擎的回调。
func (h *Handler) SetEngineStarter(fn EngineStarter) {
	h.engineStarter = fn
}

// ServeWS handles GET /api/ws?room=CODE
func (h *Handler) ServeWS(w http.ResponseWriter, r *http.Request) {
	// 1. Authenticate
	token := extractToken(r)
	if token == "" {
		http.Error(w, "missing token", http.StatusUnauthorized)
		return
	}
	claims, err := auth.ParseToken(h.jwtSecret, token)
	if err != nil {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}

	// 2. Look up room
	code := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("room")))
	rm, err := h.roomManager.Get(code)
	if err != nil {
		http.Error(w, "room not found", http.StatusNotFound)
		return
	}

	// 3. Get or create RoomConn (and engine) for this room
	rc := h.getOrCreateRoomConn(rm)

	// 4. Upgrade to WebSocket
	// UserID is int64 in JWT claims; WS/game layers use string representation.
	userIDStr := fmt.Sprintf("%d", claims.UserID)
	client, err := NewClient(rc, w, r, userIDStr, claims.DisplayName)
	if err != nil {
		slog.Error("ws: upgrade failed", "err", err)
		return
	}

	// 5. Start pumps — WritePump in goroutine, ReadPump blocks until close
	go client.WritePump()
	client.ReadPump()
}

// getOrCreateRoomConn 返回给定房间的 RoomConn，不存在则新建。
func (h *Handler) getOrCreateRoomConn(rm *room.Room) *RoomConn {
	h.roomConnsMu.RLock()
	rc, ok := h.roomConns[rm.Code]
	h.roomConnsMu.RUnlock()
	if ok {
		return rc
	}

	h.roomConnsMu.Lock()
	defer h.roomConnsMu.Unlock()
	if rc, ok = h.roomConns[rm.Code]; ok {
		return rc
	}

	rc = NewRoomConn(rm.Code)
	go rc.Run()
	h.roomConns[rm.Code] = rc
	slog.Info("ws: room conn created", "room", rm.Code)

	if h.engineStarter != nil {
		h.engineStarter(rc, rm)
	}
	return rc
}

// RemoveHub 移除已关闭房间的 RoomConn。
func (h *Handler) RemoveHub(code string) {
	h.roomConnsMu.Lock()
	delete(h.roomConns, code)
	h.roomConnsMu.Unlock()
}

// ClientCount 返回给定房间已连接的客户端数量。
func (h *Handler) ClientCount(code string) int {
	h.roomConnsMu.RLock()
	rc, ok := h.roomConns[code]
	h.roomConnsMu.RUnlock()
	if !ok {
		return 0
	}
	return rc.ClientCount()
}

// extractToken pulls JWT from Authorization header or ?token= query param.
func extractToken(r *http.Request) string {
	if hdr := r.Header.Get("Authorization"); strings.HasPrefix(hdr, "Bearer ") {
		return strings.TrimPrefix(hdr, "Bearer ")
	}
	return r.URL.Query().Get("token")
}
