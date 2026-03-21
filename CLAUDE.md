# AllIn — Claude 项目说明

## 项目结构

```
allin/
├── server/             # Go 后端（单 Go module）
│   ├── go.mod          # module github.com/allin/server
│   ├── base/           # 微服务主体
│   │   ├── main.go
│   │   ├── router.go
│   │   ├── config.yaml
│   │   └── biz/
│   │       ├── dao/      # DB 访问层（sqlx）
│   │       ├── handler/  # HTTP 处理器（Hertz）
│   │       ├── model/    # 数据模型
│   │       ├── mw/       # 中间件（JWT）
│   │       └── service/  # 业务逻辑
│   └── contrib/        # 共享基础包
│       ├── auth/         # JWT + bcrypt
│       ├── eval/         # 手牌评估器
│       ├── game/         # 游戏引擎（状态机）
│       ├── room/         # 房间模型 + 内存管理
│       └── ws/           # WebSocket Hub + Client
├── ui-web/             # 前端（Vite + React + PixiJS v8）
└── docs/               # 设计文档和进度追踪
```

## 本地开发启动

### 前置条件
- MySQL 8.0 运行在 `127.0.0.1:13306`，root 无密码
- 连接命令：`mysql -h 127.0.0.1 -P 13306 -u root`
- 数据库 `allin` 已存在，启动时 AutoMigrate 自动建表

### 后端（server/）
```bash
cd server
go run ./base/
```
- 配置文件：`server/base/config.yaml`（Viper 加载）
- 监听 `:8080`
- 健康检查：`curl http://localhost:8080/health`

### 前端（ui-web/）
```bash
cd ui-web
npm run dev
```
- 默认 `:5173`，端口占用时顺延（5174、5175…）
- `/api` 代理到 `http://localhost:8080`（含 WebSocket）

### 验证后端是否已在运行
```bash
curl http://localhost:8080/health
```

## 技术栈

| 层 | 技术 |
|----|------|
| 后端语言 | Go 1.25 |
| HTTP 框架 | CloudWeGo Hertz v0.10.4 |
| WebSocket | gorilla/websocket（通过 hertz common/adaptor 桥接） |
| 数据库驱动 | jmoiron/sqlx + go-sql-driver/mysql |
| 配置 | spf13/viper（config.yaml） |
| 鉴权 | JWT HS256，7 天有效期 |
| 前端构建 | Vite 5 + TypeScript |
| UI | React 18 + CSS Modules |
| 2D 渲染 | PixiJS v8（WebGL，程序化渲染，无外部图片） |
| 状态管理 | Zustand |
| 路由 | React Router v6 |

## 关键设计决策

- **分层架构**：`handler → service → dao`，与 shopman/server/base 完全对齐。
- **无循环依赖**：`contrib/game` 导入 `contrib/ws/room`；`contrib/ws` 不导入 `contrib/game`。通过 `ws.Handler.SetEngineStarter(func)` 回调在 `main.go` 中注入 Engine 工厂。
- **游戏引擎**：单 goroutine，`select` 监听 `hub.Inbound` + 可重置 timer channel（`nil` channel 不触发 select）。
- **座位旋转**：PixiJS TableScene 将本地玩家始终旋转到底部（display index 0）。
- **手牌评估**：纯 Go 枚举 C(7,5)=21 种五牌组合，`HandRanks.dat` 为可选加速路径。
- **边底池**：按 TotalBet 升序迭代，逐层计算各玩家贡献。
- **买入生命周期**：玩家加入房间时从 `chip_balance` 扣除买入额（`biz/dao.UserDao.AdjustChips`）；断线时将剩余 stack 归还账户。

## 测试账号

密码均为 `123456`，初始筹码各 10,000。

| 账号 | 显示名 |
|------|--------|
| test1 | 测试玩家1 |
| test2 | 测试玩家2 |
| test3 | 测试玩家3 |
| test4 | 测试玩家4 |

多窗口测试：用多个浏览器窗口或无痕模式分别登录不同账号，一人创建房间，其余用房间码加入。

## 进度
详见 `docs/PROGRESS.md`。当前阶段：Phase 10（服务端重构）已完成。

## Docs Conventions

- 文档存放目录：`docs`
- **SQL 文件**：所有 DDL 统一存放在 `docs/sql/`
- **SQL 备份文件**：如果我要求从本地数据使用 mysqldump 导出数据库备份文件，路径使用：`docs/sql/bak`，导出命令为：

  ```
  mysqldump --databases <数据库> --set-gtid-purged=OFF \
  --host=<主机> \
  --port=<端口> -u<用户名> -p<密码>
  ```

## Infrastructure

### MySQL

- Host: `127.0.0.1:13306` / User: `root` / Password: (empty)
- 数据库：`allin`

---

## Data Model Conventions

- **Struct Tags**：同时使用 `json` 和 `db` tag（sqlx 扫描需要 `db`）
- **ID 类型**：用户/房间 ID 均为 `string`（UUID v4）
- **错误处理**：`fmt.Errorf("xxx failed: %w", err)`
- **错误不忽略**：禁止用 `_` 丢弃 `error`；后台 goroutine 中用 `slog.Error` 记录

---

## Code Patterns

### 分层模板

```go
// biz/dao/xxx_dao.go
type xxxDao struct{}
func (d *xxxDao) GetByID(id string) (*model.XXX, error) {
    item := &model.XXX{}
    err := DBM.Get(item, `SELECT * FROM xxx WHERE id = ?`, id)
    return item, err
}

// biz/service/xxx_svc.go
type XxxSvc struct{}
func NewXxxSvc() *XxxSvc { return &XxxSvc{} }
func (svc *XxxSvc) GetByID(id string) (*model.XXX, error) {
    return dao.XxxDao.GetByID(id)
}

// biz/handler/xxx.go
var Xxx XxxHandler
type XxxHandler struct{}
func (*XxxHandler) Get(ctx context.Context, c *app.RequestContext) {
    id := c.Param("id")
    item, err := service.Xxx.GetByID(id)
    if err != nil {
        c.JSON(404, map[string]string{"error": "not found"})
        return
    }
    c.JSON(200, item)
}
```

### 路由注册（router.go）

```go
func register(h *server.Hertz, ...) {
    api := h.Group("/api")
    api.GET("/xxx/:id", handler.Xxx.Get)
    api.POST("/xxx", mw.JWTMiddleware(), handler.Xxx.Create)
}
```
