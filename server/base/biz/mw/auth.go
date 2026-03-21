package mw

import (
	"context"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/spf13/viper"

	"github.com/allin/server/contrib/auth"
)

const (
	ctxUserID      = "user_id"
	ctxDisplayName = "display_name"
)

// JWTMiddleware returns a Hertz handler that validates Bearer JWT and injects claims.
func JWTMiddleware() app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		token := extractToken(c)
		if token == "" {
			c.JSON(401, map[string]string{"error": "missing token"})
			c.Abort()
			return
		}
		claims, err := auth.ParseToken(viper.GetString("jwt.secret"), token)
		if err != nil {
			c.JSON(401, map[string]string{"error": "invalid token"})
			c.Abort()
			return
		}
		c.Set(ctxUserID, claims.UserID)
		c.Set(ctxDisplayName, claims.DisplayName)
		c.Next(ctx)
	}
}

func extractToken(c *app.RequestContext) string {
	if hdr := string(c.GetHeader("Authorization")); strings.HasPrefix(hdr, "Bearer ") {
		return strings.TrimPrefix(hdr, "Bearer ")
	}
	return c.Query("token")
}

// GetUserID returns the authenticated user ID set by JWTMiddleware.
func GetUserID(c *app.RequestContext) int64 {
	v, _ := c.Get(ctxUserID)
	id, _ := v.(int64)
	return id
}

// GetDisplayName returns the display name set by JWTMiddleware.
func GetDisplayName(c *app.RequestContext) string {
	v, _ := c.Get(ctxDisplayName)
	name, _ := v.(string)
	return name
}
