package auth

import (
	"context"
	"net/http"
	"strings"
)

type contextKey string

const (
	ContextUserID      contextKey = "user_id"
	ContextDisplayName contextKey = "display_name"
)

// Middleware 返回一个验证 Bearer JWT 的 HTTP 中间件。
// 验证成功后将 user_id 和 display_name 存入请求上下文。
func Middleware(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractToken(r)
			if token == "" {
				http.Error(w, `{"error":"missing token"}`, http.StatusUnauthorized)
				return
			}
			claims, err := ParseToken(secret, token)
			if err != nil {
				http.Error(w, `{"error":"invalid token"}`, http.StatusUnauthorized)
				return
			}
			ctx := context.WithValue(r.Context(), ContextUserID, claims.UserID)
			ctx = context.WithValue(ctx, ContextDisplayName, claims.DisplayName)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// extractToken 从 Authorization 请求头或 ?token= 查询参数中提取 JWT。
func extractToken(r *http.Request) string {
	if h := r.Header.Get("Authorization"); strings.HasPrefix(h, "Bearer ") {
		return strings.TrimPrefix(h, "Bearer ")
	}
	return r.URL.Query().Get("token")
}

// UserIDFromCtx 从请求上下文中获取已认证的用户 ID。
func UserIDFromCtx(ctx context.Context) string {
	v, _ := ctx.Value(ContextUserID).(string)
	return v
}

// DisplayNameFromCtx 从请求上下文中获取显示名称。
func DisplayNameFromCtx(ctx context.Context) string {
	v, _ := ctx.Value(ContextDisplayName).(string)
	return v
}
