package main

import (
	"context"
	"log/slog"
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
	// Structured logging
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	// Load config
	cfg := config.Load()

	// Connect to MySQL
	if err := store.Connect(cfg.MySQLDSN); err != nil {
		slog.Error("failed to connect to mysql", "err", err)
		os.Exit(1)
	}
	defer store.DB.Close()

	// Auto-migrate tables
	if err := store.AutoMigrate(); err != nil {
		slog.Error("failed to auto-migrate", "err", err)
		os.Exit(1)
	}

	// Initialise handlers
	roomManager := room.NewManager()
	userHandler := user.NewHandler(cfg.JWTSecret)
	roomHandler := room.NewHandler(roomManager)
	wsHandler := ws.NewHandler(roomManager, cfg.JWTSecret)
	authMW := auth.Middleware(cfg.JWTSecret)

	// Engine registry for graceful shutdown.
	registry := game.NewRegistry()

	// Wire game engine: start an Engine goroutine whenever a new hub is created.
	wsHandler.SetEngineStarter(func(hub *ws.Hub, rm *room.Room) {
		eng := game.NewEngine(hub, rm, registry)
		eng.SetOnEmpty(func() {
			roomManager.Close(rm.Code)
			wsHandler.RemoveHub(rm.Code)
			slog.Info("room: closed after last player left", "code", rm.Code)
		})
		go eng.Run()
	})

	// Room GC: remove rooms idle for >30 min every 5 minutes.
	roomManager.StartGC(5*time.Minute, 30*time.Minute, wsHandler.ClientCount)

	// Routes
	mux := http.NewServeMux()

	// Health
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	// Auth
	mux.HandleFunc("POST /api/auth/register", userHandler.Register)
	mux.HandleFunc("POST /api/auth/login", userHandler.Login)
	mux.Handle("GET /api/me", authMW(http.HandlerFunc(userHandler.Me)))

	// Rooms
	mux.Handle("POST /api/rooms", authMW(http.HandlerFunc(roomHandler.Create)))
	mux.HandleFunc("GET /api/rooms/{code}", roomHandler.Get)

	// WebSocket
	mux.HandleFunc("GET /api/ws", wsHandler.ServeWS)

	// CORS + logging middleware
	handler := corsMiddleware(cfg.AllowOrigins, loggingMiddleware(mux))

	// HTTP server
	srv := &http.Server{
		Addr:         cfg.ServerAddr,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 0, // disabled for WebSocket connections
		IdleTimeout:  60 * time.Second,
	}

	// Start server
	go func() {
		slog.Info("server starting", "addr", cfg.ServerAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	// Graceful shutdown on SIGINT/SIGTERM
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down...")

	// Stop all game engines first (finish or abort active hands).
	registry.StopAll()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("shutdown error", "err", err)
	}
	slog.Info("server stopped")
}

// corsMiddleware adds CORS headers for the configured origins.
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

// loggingMiddleware logs each request method, path, and duration.
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
