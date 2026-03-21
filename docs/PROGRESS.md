# 实现进度

> 最后更新：2026-03-22
> 当前阶段：Phase 10 完成

---

## Phase 1：基础设施 ✅

### 后端

- [x] Go module 初始化（`server/go.mod`，module `github.com/allin/server`）
- [x] 目录结构创建（`server/base/`、`server/contrib/`）
- [x] `internal/config`：环境变量加载
- [x] `internal/store/mysql.go`：MySQL 连接池 + 自动建表
- [x] `internal/auth/password.go`：bcrypt 哈希
- [x] `internal/auth/jwt.go`：JWT 签发/验证（HS256，7天有效期）
- [x] `internal/auth/middleware.go`：HTTP 鉴权中间件
- [x] `internal/user/model.go`、`repository.go`、`handler.go`
- [x] `internal/room/`：房间模型、房间码生成、RoomManager、Handler
- [x] `internal/ws/`：WebSocket Hub + Client + Handler
- [x] `cmd/server/main.go`：主入口，路由注册，CORS + 日志中间件

### 前端

- [x] Vite + React + TypeScript 项目初始化（`ui-web/`）
- [x] 依赖安装（PixiJS v8、Zustand、React Router v6）
- [x] HTTP / WebSocket API 封装、Auth/Room Zustand store
- [x] LoginPage、LobbyPage、基础路由、Vite 代理配置

---

## Phase 2：游戏引擎 ✅ 编译通过，单测全绿

- [x] `internal/eval/`：HandRanks.dat 加载器 + 纯 Go Evaluate7（枚举 21 种组合）+ 牌型描述
- [x] `internal/game/model.go`：GameState、Player、Card、Pot、GameSnapshot
- [x] `internal/game/deck.go`：洗牌 + 发牌
- [x] `internal/game/pot.go`：BuildPots 边底池算法
- [x] `internal/game/action.go`：ValidateAction + ApplyAction
- [x] `internal/game/engine.go`：Engine 主循环（单 goroutine，preflop→showdown 状态机，行动超时）
- [x] 单元测试：手牌评估 15 用例 ✅，边底池计算 20 场景 ✅

---

## Phase 3：PixiJS 牌桌 + WS 集成 ✅

- [x] PixiJS v8 Application、CardSprite、SeatSprite、PotDisplay、TimerArc
- [x] TableScene（Zustand subscribe 驱动，座位自动旋转）
- [x] DealAnimation、ChipAnimation
- [x] GameSnapshot Zustand store、useWebSocket、useGameState、useActionTimer
- [x] RoomPage（PixiJS canvas + React 覆盖层）

---

## Phase 4：React 面板 + 聊天 ✅

- [x] ActionPanel、BetSlider、ChatPanel、HandHistory、RoomInfo
- [x] 聊天消息 store（最多 200 条）
- [x] 后端：聊天中继（1 秒限速）、补充筹码接口、暂离逻辑

---

## Phase 5：上线准备 ✅

- [x] 优雅关闭（game.Registry + SIGTERM → StopAll()）
- [x] 空房间 GC（StartGC 5min/30min）
- [x] Dockerfile 多阶段构建 + docker-compose + nginx
- [x] 前端断线自动重连（指数退避）+ 连接丢失 UI 提示

---

## Phase 6：AI 玩家 ✅

- [x] 创建房间时指定 AI 玩家数（`bot_count`）
- [x] AI 自动入座、自动行动（1–3 秒随机延迟，走与真人相同路径）
- [x] 简单决策策略（check/bet/call/raise/fold 概率权重）
- [x] 真人全部离开时清场 + 30s 宽限期重连 bots 重新入座
- [x] 前端 bot 标识（🤖 前缀 + 蓝色边框）

---

## Phase 7：AI 玩家风格升级 ✅

- [x] 四种 bot 风格：TAG（紧凶）、LAG（松凶）、Station（松被动）、Rock（紧被动）
- [x] 四种风格主题：混合 / 激进 / 被动 / 随机
- [x] Preflop / Postflop 强度评估 + 风格感知决策 + BluffRate 虚张声势

---

## Phase 8：渲染修复 ✅

- [x] Canvas 宽高比修复（`aspectRatio: 1200/700`，防止 CSS 垂直拉伸）
- [x] 手牌可见性修复（删除多余 `else if` 分支）
- [x] 手牌渲染统一到 PixiJS（移除 React overlay 的 HTML 手牌）

---

## Phase 9：UI 视觉精调 ✅

- [x] 本地玩家/远端玩家头像框光晕扩展（多层扩散，结构对称，色调区分）
- [x] 移除 4 方位角装饰弧，统一头像半径（AVATAR_R_REMOTE = 48）
- [x] 字体调整：昵称 12→15，筹码 11→14，位置标签 10→12，下注徽章 11→13
- [x] 所有玩家手牌统一向右偏移 18px，避免与光晕重叠
- [x] 代码整理：ChipAnimation 移除冗余引用，DealAnimation 简化 tick 逻辑

---

## Phase 10：后端架构重构 ✅

将 `allin-server/` 重构为对齐微服务规范的 `server/` 目录结构。

### 目录变更

- `allin-server/` → `server/`（单 Go module：`github.com/allin/server`）
- `server/base/`：微服务主体（Hertz HTTP server + 3 层架构）
- `server/contrib/`：可复用组件（`ws/`、`room/`、`game/`、`eval/`、`auth/`）
- `allin-web/` → `ui-web/`

### 后端技术栈升级

- HTTP 框架：`net/http` → **Hertz v0.10.4**（CloudWeGo）
- 配置管理：环境变量（`os.Getenv`） → **Viper**（`server/base/config.yaml`）
- 数据库访问：`database/sql` + `go-sql-driver` → **sqlx**（`db:` tag 扫描）
- 中间件：`net/http.Handler` 链 → Hertz `app.HandlerFunc` 链

### 3 层架构落地

- `server/base/biz/handler/`：`UserHandler`（Register / Login / Me）、`RoomHandler`（Create / Get）
- `server/base/biz/service/`：`UserSvc`（注册、登录、查询）、`RoomSvc`（创建、查询、关闭）
- `server/base/biz/dao/`：`userDao`（CRUD + AdjustChips）、`roomDao`（Persist / MarkEnded）
- `server/base/biz/mw/`：`JWTMiddleware()`（Hertz app.HandlerFunc，Viper 读密钥）
- `server/base/biz/model/`：`User` struct（`db:` tag）、`ErrUserNotFound`、`ErrUsernameTaken`

### 路由 & 配置

- `server/base/router.go`：Hertz 路由注册（`/health`、`/api/auth/*`、`/api/me`、`/api/rooms*`、`/api/ws`）
- WebSocket 通过内置 adaptor `github.com/cloudwego/hertz/pkg/common/adaptor.HertzHandler` 桥接
- CORS 通过 `github.com/hertz-contrib/cors` 配置，允许源从 `config.yaml` 读取
- `server/base/config.yaml`：server.addr / mysql.dsn / jwt.secret / cors.allow_origins

### contrib 包适配

- `contrib/room/manager.go`：`Create/Close` 直接调用 `base/biz/dao.RoomDao`（移除旧 store 依赖）
- `contrib/game/engine.go`：用户查询 / 筹码调整改为调用 `base/biz/dao.UserDao`

### 已删除

- `cmd/`、`internal/config/`、`internal/store/`、`internal/user/`、`internal/room/dao.go`、`internal/room/service.go`、`internal/room/handler.go`、`internal/resp/`、`internal/auth/middleware.go`
- `server/base/.env.example`

---

## Phase 11：游戏引擎缺口修复 ✅

- [x] **筹码归零踢出**：`kickBrokePlayers()` 在 `runShowdown` / `awardUncontested` 后调用，自动移除 Stack==0 的玩家并广播 `player_left`；人类全离开时清场 bot 并启动宽限期
- [x] **带入金额校验**：`JoinRoomCmd` 新增 `buy_in` 字段；`handleJoinRoom` 验证 `min_buy_in ≤ buy_in ≤ max_buy_in`，0 值默认使用 MaxBuyIn
- [x] **All-in 超额退还**：已由 `BuildPots` 正确处理——超额下注作为独立边池归属于该玩家，无需额外处理

---

## TODO：已知缺口

### 游戏引擎

- [ ] 手牌历史持久化：无 `hand_history` 表，牌局结果只广播不落库
