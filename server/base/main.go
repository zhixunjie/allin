package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/hertz-contrib/cors"
	"github.com/spf13/viper"

	"github.com/allin/server/base/biz/dao"
	"github.com/allin/server/base/biz/service"
	"github.com/allin/server/contrib/game"
	"github.com/allin/server/contrib/room"
	"github.com/allin/server/contrib/ws"
)

func init() {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./base") // when running `go run ./base/` from module root
	viper.AutomaticEnv()
	if err := viper.ReadInConfig(); err != nil {
		fmt.Fprintf(os.Stderr, "config: %v (using defaults)\n", err)
	}
}

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	// DAO layer (connects MySQL, runs AutoMigrate)
	dao.Init()

	// Room infrastructure
	roomManager := room.NewManager()

	// Service layer
	service.Init(roomManager)

	// WebSocket + game engine
	wsHandler := ws.NewHandler(roomManager, viper.GetString("jwt.secret"))
	registry := game.NewRegistry()

	wsHandler.SetEngineStarter(func(hub *ws.Hub, rm *room.Room) {
		eng := game.NewEngine(hub, rm, registry)
		eng.SetOnEmpty(func() {
			service.Room.Close(rm.Code)
			wsHandler.RemoveHub(rm.Code)
			slog.Info("room: closed after last player left", "code", rm.Code)
		})
		go eng.Run()
	})

	roomManager.StartGC(5*time.Minute, 30*time.Minute, wsHandler.ClientCount)

	// Hertz server
	h := server.New(
		server.WithHostPorts(viper.GetString("server.addr")),
		server.WithExitWaitTime(4*time.Second),
	)

	// CORS
	h.Use(cors.New(cors.Config{
		AllowOrigins:     viper.GetStringSlice("cors.allow_origins"),
		AllowMethods:     []string{"GET", "POST", "OPTIONS"},
		AllowHeaders:     []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
	}))

	register(h, wsHandler)

	// Stop game engines on graceful shutdown
	h.Engine.OnShutdown = append(h.Engine.OnShutdown, func(ctx context.Context) {
		registry.StopAll()
		slog.Info("game engines stopped")
	})

	slog.Info("server starting", "addr", viper.GetString("server.addr"))
	h.Spin()
}
