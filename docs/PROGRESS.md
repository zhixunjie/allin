# 实现进度

> 最后更新：2026-03-22
> 当前阶段：Phase 12 完成

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

## Phase 12：服务端功能完善 ✅

### 断线重连（保留座位）

- [x] `Player.Disconnected bool`：手牌进行中断线时保留座位，标记为 Disconnected
- [x] `handleDisconnect`：手牌间隙断线立即离座 cashOut；手牌中断线仅自动弃牌（当前轮）或标记弃牌，保留座位等待重连
- [x] `handleJoinRoom` 重连路径：检测到 `Disconnected==true` 时直接恢复（清除标记、发送快照、广播 `player_joined` IsReconnect）
- [x] `cleanupDisconnected()`：手牌结束后统一清理仍处于断线状态的玩家并 cashOut，广播 `player_left`
- [x] `SeatSnapshot.Disconnected` 字段下发前端，前端 `SeatSnapshot` 类型同步更新
- [x] `EligibleToStart()` 排除 Disconnected 玩家（不参与下一手计数）

### `add_chips` DB 校验

- [x] `handleAddChips`：从 DB 查询 `chip_balance`，验证余额充足后调用 `AdjustChips(-added, "add_chips", roomCode)` 扣除，防止无限补充
- [x] 余额不足时返回 `insufficient_chips` 错误给客户端

### `sit_out` 事件类型修正

- [x] 新增 `TypeSitOut = "sit_out_status"` 及 `SitOutPayload{PlayerID, SeatIndex, SitOut}`
- [x] `handleSitOut` 改为广播 `TypeSitOut`（原错误地广播 `player_joined`）
- [x] 前端新增 `WSEventType.SitOutStatus`、`SitOutPayload` 类型、`applySitOut` store 方法
- [x] `useWebSocket` 订阅 `sit_out_status` 事件，正确更新座位 `sit_out` 字段

### 主动离桌（leave_table）

- [x] 新增 `CmdLeaveTable = "leave_table"` 命令
- [x] `handleLeaveTable`：仅允许手牌间隙，离座 cashOut，广播 `player_left`
- [x] 提取公共逻辑 `maybeStartEmptyTimer()`，统一处理「人类全离场 → 清 bot → 宽限期」
- [x] `kickBrokePlayers()` 复用同一逻辑，消除重复代码
- [x] 前端「离开牌桌」按钮先发送 `leave_table` 命令，再跳转 `/lobby`
- [x] 前端 `WSEventType.LeaveTable` 枚举值补充

### 手牌历史持久化

- [x] 新建 `server/base/biz/dao/hand_history_dao.go`：`HandHistoryRecord` 结构体 + `Save()` 方法
- [x] `dao/init.go` 注册 `HandHistoryDao`
- [x] `saveHandHistory(resultJSON)`：异步 goroutine 写库，不阻塞引擎主循环
- [x] `runShowdown` 和 `awardUncontested` 均在广播 `hand_result` 后调用 `saveHandHistory`
- [x] 保存字段：`room_id`、`hand_num`、`players_json`（座位快照）、`result_json`（赢家/金额）、`played_at`
- [x] 数据库四张表均已建立：`users`、`room_history`、`hand_history`、`chip_ledger`

---

## Phase 13：缺口修复 ✅

- [x] **`handleSitOut` 归座触发开局**：玩家从 sit_out 归座时，若 `Street == Idle` 且 `EligibleToStart() >= 2`，自动调用 `resetTimer(handStartDelay)` 开始倒计时
- [x] **前端订阅 `stack_updated`**：新增 `StackUpdatedPayload` 类型、`applyStackUpdated` store 方法，`useWebSocket` 订阅 `stack_updated` 事件；`add_chips` 后座位筹码实时刷新
- [x] **`hand_history.actions_json` 完整落库**：Engine 新增 `handActions []actionLogEntry`，`startHand` 时清空，`handleAction` 和 `handleTimeout` 追加记录（含 PlayerID / Action / Amount / Street）；`saveHandHistory` 将序列化后的完整行动日志写入 DB
- [x] **多赢家（平分底池）正确展示**：`RoundResultModal` 改为遍历 `winners` 数组，每位赢家单独一行；最佳五张牌仅首位赢家显示

---

---

## Phase 14：功能补全 ✅

### 账户余额自动刷新
- [x] `useWebSocket`：`player_joined`（自身首次入桌）时 fetch `/api/me` 更新 `chip_balance`
- [x] `useWebSocket`：`stack_updated`（自身补充筹码）时 fetch `/api/me` 更新 `chip_balance`
- [x] `LobbyPage`：组件挂载时 fetch `/api/me`，确保离桌 cashOut 后余额显示最新值

### 补充筹码按钮（RoomPage）
- [x] 手牌间隙（`street === Idle`）且桌面筹码低于 `max_buy_in` 时显示「补充筹码」按钮
- [x] 弹窗支持自定义金额输入（25% / 50% / 全补 快捷预设，上限 `min(maxAdd, chip_balance)`）
- [x] 确认后发送 `add_chips` WS 命令，`stack_updated` 事件自动刷新桌面筹码和账户余额

### 筹码归零后领取（`POST /api/chips/claim`）
- [x] 后端：`UserSvc.ClaimFreeChips` — 余额 < 1000 时补发 10000 筹码，写入 `chip_ledger`
- [x] 后端：`UserHandler.ClaimChips` handler + 路由注册
- [x] 前端：`LobbyPage` 余额 < 1000 时显示「领取筹码」按钮，成功后更新 auth store

### 手牌历史查询（`GET /api/rooms/:code/hands`）
- [x] 后端：`HandHistoryDao.GetByRoomCode` — JOIN `room_history` 按房间码查最近 20 手，返回 `hand_num / result / played_at`
- [x] 后端：`RoomHandler.GetHands` handler + 路由注册（JWT 保护）
- [x] 前端：`HandHistory` 面板重写为按需加载的历史列表（每行展示手牌编号、赢家、金额、牌型、时间）
- [x] `RoomPage` 顶部栏添加「历史」切换按钮，点击显示/隐藏 `HandHistory` 面板

### 行动计时器 Banner 移除
- [x] 删除 `hooks/useActionTimer.ts`
- [x] `RoomPage` 移除所有计时器 Banner 相关注释代码（PixiJS `TimerArc` 弧形倒计时保留）

## Phase 15：缺口修复与完善 ✅

### Bug 修复：HandHistory 数据解析错误
- [x] 修正 `HandEntry.result` 类型（从错误的 `Array` 改为 `{ winners, all_players, best_hand, seats }` 对象）
- [x] `HandHistory.tsx` 改为读 `hand.result.winners`，并通过 `all_players` 建立 `player_id → display_name` 映射，显示可读玩家名

### 买入金额自选（加入房间）
- [x] `roomStore` 新增 `selectedBuyIn` 字段（0 = 使用服务端默认 max_buy_in）
- [x] `LobbyPage` 加入确认弹窗：当 `min_buy_in < max_buy_in` 时显示滑块，范围 `[min, min(max, balance)]`
- [x] `useWebSocket` 在发送 `join_room` 时携带 `selectedBuyIn`（0 时省略，服务端自动用 max）

### 邀请链接复制按钮
- [x] `RoomPage` 顶部栏「邀请」按钮，点击复制 `/join/:code` 到剪贴板，2 秒后文字恢复

### ChatPanel 恢复挂载
- [x] `RoomPage` 重新 import 并渲染 `ChatPanel`（折叠状态常驻右下角）

### PROJECT.md 同步更新
- [x] Section 2 加入 `POST /api/chips/claim`
- [x] Section 3 加入 `GET /api/rooms/:code/hands`
- [x] Section 9 筹码流动表加入 `claim_free`
- [x] Section 10 修正 `players_json` / `result_json` 字段描述
- [x] Section 11 补充 `selectedBuyIn` 说明
- [x] Section 13 新增 RoomPage/LobbyPage 功能说明，修正 ChatPanel/HandHistory 描述
