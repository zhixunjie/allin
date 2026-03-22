package handler

import (
	"context"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"

	"github.com/allin/server/base/biz/dao"
	"github.com/allin/server/base/biz/mw"
	"github.com/allin/server/base/biz/service"
	"github.com/allin/server/contrib/room"
)

var Room RoomHandler

type RoomHandler struct{}

// Create handles POST /api/rooms
func (*RoomHandler) Create(ctx context.Context, c *app.RequestContext) {
	hostUserID := mw.GetUserID(c)

	var cfg room.RoomConfig
	if err := c.BindJSON(&cfg); err != nil {
		c.JSON(400, map[string]string{"error": "invalid request body"})
		return
	}
	if cfg.ActionTimeSec == 0 {
		cfg.ActionTimeSec = 30
	}

	rm, err := service.Room.Create(hostUserID, cfg)
	if err != nil {
		c.JSON(400, map[string]string{"error": err.Error()})
		return
	}
	c.JSON(201, rm)
}

// Get handles GET /api/rooms/:code
func (*RoomHandler) Get(ctx context.Context, c *app.RequestContext) {
	code := strings.ToUpper(strings.TrimSpace(c.Param("code")))
	rm, err := service.Room.Get(code)
	if err != nil {
		c.JSON(404, map[string]string{"error": "room not found"})
		return
	}
	c.JSON(200, rm)
}

// GetHands handles GET /api/rooms/:code/hands
// 返回该房间最近 20 手的结果摘要。
func (*RoomHandler) GetHands(ctx context.Context, c *app.RequestContext) {
	code := strings.ToUpper(strings.TrimSpace(c.Param("code")))
	entries, err := dao.HandHistoryDao.GetByRoomCode(code, 20)
	if err != nil {
		c.JSON(500, map[string]string{"error": "server error"})
		return
	}
	if entries == nil {
		entries = []dao.HandHistoryEntry{}
	}
	c.JSON(200, entries)
}
