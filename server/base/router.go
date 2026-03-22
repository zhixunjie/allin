package main

import (
	"context"
	"net/http"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/adaptor"

	"github.com/allin/server/base/biz/handler"
	"github.com/allin/server/base/biz/mw"
	"github.com/allin/server/contrib/ws"
)

func register(h *server.Hertz, wsHandler *ws.Handler) {
	h.GET("/health", func(ctx context.Context, c *app.RequestContext) {
		c.JSON(200, map[string]string{"status": "ok"})
	})

	api := h.Group("/api")
	{
		api.POST("/auth/register", handler.User.Register)
		api.POST("/auth/login", handler.User.Login)
		api.GET("/me", mw.JWTMiddleware(), handler.User.Me)

		api.POST("/chips/claim", mw.JWTMiddleware(), handler.User.ClaimChips)

		api.POST("/rooms", mw.JWTMiddleware(), handler.Room.Create)
		api.GET("/rooms/:code", handler.Room.Get)
		api.GET("/rooms/:code/hands", mw.JWTMiddleware(), handler.Room.GetHands)

		// WebSocket: ServeWS stays as net/http, bridged via built-in adaptor
		api.GET("/ws", adaptor.HertzHandler(http.HandlerFunc(wsHandler.ServeWS)))
	}
}
