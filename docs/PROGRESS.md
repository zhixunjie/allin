# 实现进度

> 最后更新：2026-03-25
> 当前阶段：Phase 10 完成，所有功能已上线

---

## Phase 1：基础设施 & 用户系统 ✅

**目标**：Go 项目骨架、数据库连接、用户注册/登录鉴权。

- [x] Go module 初始化（`server/go.mod`，module `github.com/allin/server`）
- [x] 目录结构：`server/base/`（3 层架构）、`server/contrib/`（共享组件）
- [x] Viper 配置加载（`server/base/config.yaml`）
- [x] sqlx + MySQL 连接池，表结构通过 `docs/sql/allin.sql` 手动维护（`users`、`room_history`、`hand_history`、`chip_ledger`）
- [x] `contrib/auth`：bcrypt 密码哈希 + JWT HS256 签发/验证（7 天有效期）
- [x] `biz/dao/user_dao.go`：用户 CRUD + `AdjustChips`
- [x] `biz/service/user_svc.go`：注册、登录、查询
- [x] `biz/handler/user.go`：`POST /api/auth/register`、`POST /api/auth/login`、`GET /api/me`
- [x] `biz/mw/jwt.go`：Hertz JWT 中间件（Viper 读密钥）
- [x] `POST /api/chips/claim`：余额 < 1000 时补发 10000 筹码（写 `chip_ledger`）

---

## Phase 2：WebSocket 通信层 ✅

**目标**：实时双向消息传递基础设施，支撑游戏所有事件推送。

- [x] `contrib/ws/client.go`：WebSocket Client（gorilla/websocket，hertz adaptor 桥接）
- [x] `contrib/ws/room_conn.go`：`RoomConn`——单房间连接管理器（`register` / `unregister` / `broadcast` / `Inbound` channel），替代传统 Hub 设计，天然隔离房间间消息
- [x] `contrib/ws/protocol/command.go`：上行消息协议（`CmdEnvelope`，`CmdType` 枚举：`join_room`、`action`、`ready`、`add_chips`、`leave_table`、`sit_out`、`chat`、`disconnect`，及各命令载荷结构体）
- [x] `contrib/ws/protocol/payload.go`：下行消息协议（`Envelope`，`MsgType` 枚举，及所有事件载荷结构体）
- [x] `contrib/ws/handler.go`：`ws.Handler`——WS 升级入口，维护 `roomConns` 索引，内置 JWT 鉴权；`ServeWS` 通过 `adaptor.HertzHandler` 桥接到 Hertz 路由
- [x] `main.go`：`wsHandler.SetEngineStarter` 回调注入，解耦 `contrib/ws` 与 `contrib/game/engine`

---

## Phase 3：游戏引擎核心 ✅

**目标**：完整的德州扑克规则实现，单 goroutine 状态机。

- [x] `server/gmodel/`：全局共享数据模型（`Card`、`Player`、`Action`、`Street`、`HandCategory`、`SeatIndex` 常量、`BotStyle` 枚举）
- [x] `contrib/eval`：HandRanks.dat 加速路径 + 纯 Go `Evaluate7`（枚举 C(7,5)=21 种组合）+ 牌型描述
- [x] `contrib/game/state/`：纯状态逻辑（无 I/O）
  - `state_machine.go`：`GameStateMachine`（街道、座位数组、盲注位、当前下注、配置）
  - `deck.go`：洗牌（Fisher-Yates）+ 发牌
  - `pot.go`：`BuildPots` 边底池算法（按 TotalBet 升序迭代，逐层计算贡献）
  - `action.go`：`ValidateAction` + `ApplyAction`
  - `snapshot.go`：将 `GameStateMachine` 序列化为前端快照
  - `eval.go`：集成 `contrib/eval`，供结算使用
  - `situation.go`：局面评估辅助（用于 Bot 决策）
- [x] `contrib/game/engine/`：I/O + 驱动层
  - `engine.go`：`Engine` 主循环（`select` 监听 `rc.Inbound` + 可重置 timer channel）；状态流转 `Idle → PreFlop → Flop → Turn → River → Showdown`；盲注发布、发洞牌、公共牌翻开、行动超时自动 fold；`runShowdown`（`BuildPots` 结算 + 多赢家平分）；`awardUncontested`（所有人弃牌直接派奖）
  - `hand.go`：单手牌生命周期（startHand、dealHoleCards、dealCommunity 等）
  - `seat.go`：座位管理（加入、离开、断线、补位）
  - `registry.go`：`Registry` 全局引擎表（`SIGTERM → StopAll`）
- [x] 单元测试：手牌评估 15 用例 ✅，边底池计算 20 场景 ✅

---

## Phase 4：房间管理 & 引擎集成 ✅

**目标**：房间生命周期管理，WS 消息路由到游戏引擎。

- [x] `contrib/room/manager.go`：`RoomManager`（内存 map + 互斥锁），房间码生成（6位大写字母）
- [x] `contrib/room/room.go`：`Room` 模型（含 `BotCount`、`MinBuyIn`、`MaxBuyIn` 配置）
- [x] `biz/dao/room_dao.go`：房间持久化（`room_history` 表）
- [x] `biz/handler/room.go`：`POST /api/rooms`（创建）、`GET /api/rooms/:code`（查询）
- [x] Engine `handleJoinRoom`：带入金额校验（`min_buy_in ≤ buy_in ≤ max_buy_in`，0 值默认 max）、从 `chip_balance` 扣除买入额
- [x] Engine `handleLeaveTable`（`leave_table` 命令）：手牌间隙离座 cashOut，广播 `player_left`
- [x] 断线重连：手牌进行中断线保留座位（`gmodel.Player.Disconnected`）；手牌结束后 `cleanupDisconnected` 统一 cashOut；重连路径恢复座位
- [x] 优雅关闭：`engine.Registry` + SIGTERM → `StopAll()`
- [x] 空房间 GC：`StartGC` 每 5 分钟扫描，30 分钟无活动的房间回收

---

## Phase 5：前端基础 & PixiJS 牌桌 ✅

**目标**：Vite + React + PixiJS 项目骨架，核心牌桌渲染。

- [x] Vite 5 + TypeScript + React 18 项目初始化（`ui-web/`）
- [x] 依赖：PixiJS v8、Zustand、React Router v6
- [x] `/api` + WebSocket 代理到 `http://localhost:8080`
- [x] Zustand stores：`authStore`、`roomStore`、`gameStore`
- [x] `hooks/useWebSocket.ts`：连接管理 + 消息分发 + 断线指数退避重连
- [x] `LoginPage`、`LobbyPage`（创建/加入房间）、`RoomPage`（主游戏页）
- [x] PixiJS `TableScene`：座位自动旋转（本地玩家始终在底部，display index 0）
- [x] `CardSprite`：正牌 / 背牌程序化绘制（无外部图片）
- [x] `SeatSprite`：头像框、昵称、筹码、下注徽章、位置标签（BTN/SB/BB）、光晕区分本地/远端
- [x] `PotDisplay`：底池金额展示
- [x] `TimerArc`：PixiJS 弧形行动倒计时
- [x] `DealAnimation`：发牌动画
- [x] `ChipAnimation`：筹码收集动画
- [x] Canvas 宽高比锁定（`aspectRatio: 1200/700`，防止 CSS 垂直拉伸）
- [x] 3 人桌位置标签修正：BTN 改为 `BTN/UTG`

---

## Phase 6：前端交互面板 ✅

**目标**：玩家操作 UI、聊天、信息展示。

- [x] `ActionPanel`：fold / check / call / raise 按钮，显示合法行动范围
- [x] `BetSlider`：下注金额滑块（BB 倍数快捷按钮 + 自定义输入）
- [x] `ChatPanel`：聊天消息（最多 200 条），折叠状态常驻右下角
- [x] `RoomInfo`：房间码、盲注结构、在线人数
- [x] `HandHistory` 面板：按需加载最近 20 手，展示赢家、金额、牌型、时间（`GET /api/rooms/:code/hands`）
- [x] `RoundResultModal`：结算弹窗（多赢家遍历展示、平分合并、最佳五张牌）；破产时切换为「再次买入」面板（滑块选额、校验余额/座位、入座成功自动关闭）；「开始下一局」按钮发送 `ready` 命令
- [x] `ActionLogPanel`：实时行动日志面板，显示当局每步操作（阶段标签 + 玩家名 + 行动类型 + 金额）
- [x] 邀请链接复制按钮（2 秒后文字恢复）
- [x] `RoomPage` 顶部栏：房间码、邀请、历史切换

---

## Phase 7：AI Bot 系统 ✅

**目标**：可配置风格的 AI 陪玩，自动补位，游戏始终流畅。

- [x] 创建房间时指定 `bot_count`，Bot 自动入座（`"bot_"` 前缀 ID）
- [x] `contrib/game/bot/bot.go`：`Bot` 结构体，持有 `RoomConn` 引用，1–3 秒随机延迟后通过 `rc.Inbound` 投递行动（与真人走完全相同的引擎路径）
- [x] `contrib/game/bot/personality.go`：四种风格（`TAG` 紧凶 / `LAG` 松凶 / `Station` 松被动 / `Rock` 紧被动）+ 四种主题（混合 / 激进 / 被动 / 随机）
- [x] `contrib/game/bot/util.go` + `contrib/game/state/situation.go`：Preflop / Postflop 手牌强度评估 + 风格感知决策 + `BluffRate` 虚张声势
- [x] Bot 破产后 8 秒自动补位（`engine.botReplaceDelay`），补位后满足条件自动开局
- [x] 真人全部离开时清场 Bot，30 秒宽限期等待真人重连
- [x] 前端 Bot 标识（蓝色边框区分）

---

## Phase 8：数据持久化 & 账户功能 ✅

**目标**：手牌历史落库，账户筹码与桌面筹码双向同步。

- [x] `biz/dao/hand_history_dao.go`：`HandHistoryRecord`（`room_id`、`hand_num`、`players_json`、`result_json`、`actions_json`、`played_at`）
- [x] Engine 结算后异步写库（不阻塞主循环）；`handActions []actionLogEntry` 记录完整行动日志
- [x] `GET /api/rooms/:code/hands`：JOIN `room_history` 查最近 20 手（JWT 保护）
- [x] 账户余额自动刷新：`player_joined`（首次入桌）时 fetch `/api/me`；`LobbyPage` 挂载时拉取最新余额
- [x] `LobbyPage` 加入确认弹窗：`min_buy_in < max_buy_in` 时显示买入滑块（上限 `min(max, balance)`）
- [x] `LobbyPage`：余额 < 1000 时显示「领取筹码」按钮

---

## Phase 9：游戏体验完善 ✅

**目标**：补齐规则细节，修复 UI 缺陷，完善准备流程。

- [x] **准备系统**：手牌结算后等待 10 秒，Bot 自动准备；玩家点「开始下一局」发送 `ready` 命令；全员准备后 500ms 立即开局（`ready_status` 事件广播 `ready_count/total_count`）
- [x] **盲注初始展示**：`game_started` 事件携带 `small_blind`/`big_blind`，前端按 `sb_seat`/`bb_seat` 初始化下注显示
- [x] **多赢家合并**：同一玩家赢主池+边池时筹码累加，`applyHandResult` 按 `player_id` 做 reduce
- [x] **筹码归零踢出**：`kickBrokePlayers` 在 `runShowdown`/`awardUncontested` 后调用，自动移除 Stack==0 玩家
- [x] **sit_out 归座触发开局**：从 sit_out 归座时，若 `Street == Idle` 且满足开局条件，自动启动倒计时
- [x] **弹窗层级**：`RoundResultModal` z-index=70 > 破产弹窗 z-index=50，结算界面始终优先

---

## Phase 10：部署 & 生产就绪 ✅

**目标**：容器化，前端自动重连，创建房间默认值优化。

- [x] Dockerfile 多阶段构建（Go + Node）+ docker-compose + nginx 反代
- [x] 前端断线自动重连（指数退避）+ 连接丢失 UI 提示
- [x] `LobbyPage` 监听 `bigBlind` 变化，自动将 `maxBuyIn` 设为 `100×BB`、`minBuyIn` 设为 `20×BB`

---

## 待办 / 可选优化

- [ ] 观战模式（只读 WS 连接，不占座位）
- [ ] 锦标赛模式（固定买入，递增盲注，淘汰制）
- [ ] 手牌历史详情页（逐行动回放）
- [ ] 移动端响应式适配