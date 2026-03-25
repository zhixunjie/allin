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
	"github.com/allin/server/contrib/game/engine"
	"github.com/allin/server/contrib/room"
	"github.com/allin/server/contrib/ws"
)

func init() {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./base") // go run ./base/ 时从模块根目录查找
	viper.AutomaticEnv()
	// 允许通过环境变量 JWT_SECRET 覆盖配置文件中的 jwt.secret
	viper.BindEnv("jwt.secret", "JWT_SECRET") //nolint:errcheck
	if err := viper.ReadInConfig(); err != nil {
		fmt.Fprintf(os.Stderr, "config: %v (using defaults)\n", err)
	}
}

func main() {
	initLogger()

	if viper.GetString("jwt.secret") == "change-me-in-production" {
		slog.Warn("SECURITY: jwt.secret is using the insecure default value; set JWT_SECRET env var or update config.yaml")
	}

	// 基础设施层
	dao.Init()
	roomManager := room.NewManager()
	service.Init(roomManager)

	// 游戏层
	wsHandler, registry := initGame(roomManager)

	// HTTP 服务器
	runServer(wsHandler, registry)
}

// initLogger 初始化结构化日志，输出到 stdout。
func initLogger() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))
}

// initGame 初始化 WebSocket Handler 和游戏引擎注册表，注入引擎启动回调。
func initGame(roomManager *room.Manager) (*ws.Handler, *engine.Registry) {
	wsHandler := ws.NewHandler(roomManager, viper.GetString("jwt.secret"), viper.GetStringSlice("cors.allow_origins"))
	registry := engine.NewRegistry()

	wsHandler.SetEngineStarter(func(rc *ws.RoomConn, rm *room.Room) {
		eng := engine.NewEngine(rc, rm, registry)
		eng.SetOnEmpty(func() {
			service.Room.Close(rm.Code)
			wsHandler.RemoveRoomConn(rm.Code)
			slog.Info("room: closed after last player left", "code", rm.Code)
		})
		go eng.Run()
	})

	roomManager.StartGC(5*time.Minute, 30*time.Minute, wsHandler.ClientCount)

	return wsHandler, registry
}

// runServer 创建 Hertz 实例，配置 CORS、路由和优雅关闭，然后阻塞运行直到进程退出。
func runServer(wsHandler *ws.Handler, registry *engine.Registry) {
	h := server.New(
		server.WithHostPorts(viper.GetString("server.addr")),
		server.WithExitWaitTime(4*time.Second),
	)

	h.Use(cors.New(cors.Config{
		AllowOrigins:     viper.GetStringSlice("cors.allow_origins"),
		AllowMethods:     []string{"GET", "POST", "OPTIONS"},
		AllowHeaders:     []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
	}))

	register(h, wsHandler)

	h.Engine.OnShutdown = append(h.Engine.OnShutdown, func(ctx context.Context) {
		registry.StopAll()
		slog.Info("game engines stopped")
	})

	slog.Info("server starting", "addr", viper.GetString("server.addr"))
	h.Spin()
}
