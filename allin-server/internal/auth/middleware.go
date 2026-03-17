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

// Middleware returns an HTTP middleware that validates a Bearer JWT.
// On success it stores user_id and display_name into the request context.
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

// extractToken pulls the JWT from Authorization header or ?token= query param.
func extractToken(r *http.Request) string {
	if h := r.Header.Get("Authorization"); strings.HasPrefix(h, "Bearer ") {
		return strings.TrimPrefix(h, "Bearer ")
	}
	return r.URL.Query().Get("token")
}

// UserIDFromCtx retrieves the authenticated user ID from a request context.
func UserIDFromCtx(ctx context.Context) string {
	v, _ := ctx.Value(ContextUserID).(string)
	return v
}

// DisplayNameFromCtx retrieves the display name from a request context.
func DisplayNameFromCtx(ctx context.Context) string {
	v, _ := ctx.Value(ContextDisplayName).(string)
	return v
}
