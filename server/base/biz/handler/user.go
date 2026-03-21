package handler

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"

	"github.com/allin/server/base/biz/model"
	"github.com/allin/server/base/biz/mw"
	"github.com/allin/server/base/biz/service"
)

var User UserHandler

type UserHandler struct{}

// Register handles POST /api/auth/register
func (*UserHandler) Register(ctx context.Context, c *app.RequestContext) {
	var req struct {
		Username    string `json:"username"`
		Password    string `json:"password"`
		DisplayName string `json:"display_name"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, map[string]string{"error": "invalid request body"})
		return
	}

	u, token, err := service.User.Register(req.Username, req.Password, req.DisplayName)
	if err != nil {
		if err == model.ErrUsernameTaken {
			c.JSON(409, map[string]string{"error": "username already taken"})
			return
		}
		c.JSON(400, map[string]string{"error": err.Error()})
		return
	}
	c.JSON(201, map[string]any{"token": token, "user": u})
}

// Login handles POST /api/auth/login
func (*UserHandler) Login(ctx context.Context, c *app.RequestContext) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, map[string]string{"error": "invalid request body"})
		return
	}

	u, token, err := service.User.Login(req.Username, req.Password)
	if err != nil {
		c.JSON(401, map[string]string{"error": err.Error()})
		return
	}
	c.JSON(200, map[string]any{"token": token, "user": u})
}

// Me handles GET /api/me
func (*UserHandler) Me(ctx context.Context, c *app.RequestContext) {
	userID := mw.GetUserID(c)
	u, err := service.User.GetByID(userID)
	if err != nil {
		c.JSON(404, map[string]string{"error": "user not found"})
		return
	}
	c.JSON(200, u)
}
