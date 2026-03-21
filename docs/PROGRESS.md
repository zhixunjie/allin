# 实现进度

> 最后更新：2026-03-21
> 当前阶段：Phase 9 完成

---

## Phase 1：基础设施

### 后端 ✅ 代码已完成，待 MySQL 建库验证

- [x] Go module 初始化（`allin-server/go.mod`）
- [x] 目录结构创建
- [x] `internal/config`：环境变量加载（默认 127.0.0.1:13306）
- [x] `internal/store/mysql.go`：MySQL 连接池 + 自动建表（CREATE TABLE IF NOT EXISTS）
- [x] `internal/auth/password.go`：bcrypt 哈希
- [x] `internal/auth/jwt.go`：JWT 签发/验证（HS256，7天有效期）
- [x] `internal/auth/middleware.go`：HTTP 鉴权中间件
- [x] `internal/user/model.go`：User struct
- [x] `internal/user/repository.go`：用户 CRUD + AdjustChips
- [x] `internal/user/handler.go`：注册/登录/获取用户接口
- [x] `internal/room/model.go`：Room、RoomConfig struct
- [x] `internal/room/code.go`：6 位房间码生成（无歧义字符集）
- [x] `internal/room/manager.go`：RoomManager（内存 + 持久化）
- [x] `internal/room/handler.go`：创建/查询房间接口
- [x] `internal/ws/message.go`：Envelope + 所有消息类型常量
- [x] `internal/ws/hub.go`：WebSocket Hub（broadcast/register/unregister/SendTo）
- [x] `internal/ws/client.go`：ReadPump + WritePump（ping/pong 保活）
- [x] `internal/ws/handler.go`：HTTP → WebSocket Upgrade（含 hub 注册表）
- [x] `cmd/server/main.go`：主入口，路由注册，CORS + 日志中间件
- [x] `go build ./...` 编译通过，`go vet ./...` 无报错
- [ ] **待办**：手动建库 `allin`，跑 `go run ./cmd/server` 验证自动建表

### 前端 ✅ 基础完成，`npm run build` 通过

- [x] Vite + React + TypeScript 项目初始化（`allin-web/`）
- [x] 依赖安装（PixiJS v8、Zustand、React Router v6）
- [x] `src/api/http.ts`：fetch 封装 + 类型化 API（authAPI / roomAPI）
- [x] `src/api/ws.ts`：WebSocket 单例 + 事件总线（on/off/send）
- [x] `src/store/auth.ts`：JWT + User 鉴权状态（localStorage 持久化）
- [x] `src/store/room.ts`：房间状态
- [x] `src/react/pages/LoginPage.tsx`：登录/注册页（含 CSS Module 样式）
- [x] `src/react/pages/LobbyPage.tsx`：大厅页（创建/加入房间，含邀请码自动填入）
- [x] 基础路由配置（BrowserRouter + RequireAuth 守卫）
- [x] `vite.config.ts`：开发代理 `/api` → `localhost:8080`（含 ws）

---

## Phase 2：游戏引擎 ✅ 编译通过，单测全绿

- [x] `internal/eval/table.go`：HandRanks.dat 加载器（可选，不存在时自动 fallback）
- [x] `internal/eval/eval.go`：纯 Go Evaluate7（枚举 21 种 5 选组合）+ T+2 fast path
- [x] `internal/eval/describe.go`：rank → 牌型名称（含 Royal Flush 判断）
- [x] `internal/game/model.go`：GameState、Player、Card、Pot、GameSnapshot 全套 struct
- [x] `internal/game/deck.go`：洗牌 + 发牌（math/rand/v2）
- [x] `internal/game/pot.go`：BuildPots 边底池算法（支持多人全押 + 弃牌贡献）
- [x] `internal/game/action.go`：ValidateAction + ApplyAction（fold/check/call/bet/raise/all_in）
- [x] `internal/game/engine.go`：Engine 主循环（单 goroutine + timer channel 模式）
  - join_room 入座 / disconnect 自动弃牌
  - preflop→flop→turn→river→showdown 状态机
  - 行动超时自动 fold/check
  - 建底池 + 摊牌评估 + 芯片分配（含边底池平分）
  - 非争议锅直接归属（免摊牌）
- [x] `internal/ws/handler.go`：EngineStarter callback（新 hub → 自动启动 Engine）
- [x] `internal/ws/message.go`：新增所有游戏事件 payload 结构
- [x] `internal/ws/hub.go`：新增 DisplayName() 方法
- [x] `cmd/server/main.go`：注册 engine factory
- [x] 单元测试：手牌评估 15 个用例（全类型 + 排序正确性）✅
- [x] 单元测试：边底池计算 20 个场景 ✅
- [ ] 单元测试：完整手牌流程（preflop → showdown）— Phase 3 集成测试覆盖

---

## Phase 3：PixiJS 牌桌 + WS 集成 ✅ 构建通过

- [x] `src/pixi/app.ts`：PixiJS v8 Application 异步初始化，自适应分辨率
- [x] `src/pixi/assets.ts`：常量定义（纯程序化渲染，无需外部图片）
- [x] `src/pixi/components/CardSprite.ts`：单张牌（正面/背面）
- [x] `src/pixi/components/SeatSprite.ts`：玩家座位（头像/名称/筹码/状态）
- [x] `src/pixi/components/PotDisplay.ts`：底池显示
- [x] `src/pixi/components/TimerArc.ts`：圆弧倒计时（绿→黄→红）
- [x] `src/pixi/scenes/TableScene.ts`：主场景（Zustand subscribe 驱动，座位自动旋转）
- [x] `src/pixi/scenes/DealAnimation.ts`：发牌飞行动画
- [x] `src/pixi/scenes/ChipAnimation.ts`：筹码入池动画
- [x] `src/store/game.ts`：完整 GameSnapshot Zustand store（处理所有 WS 事件）
- [x] `src/hooks/useWebSocket.ts`：WS 事件订阅 → store dispatch
- [x] `src/hooks/useGameState.ts`：派生 UI 状态（isMyTurn / canCheck/Call/Bet/Raise）
- [x] `src/hooks/useActionTimer.ts`：本地倒计时（250ms 精度）
- [x] `src/react/pages/RoomPage.tsx`：PixiJS canvas + React 覆盖层（操作面板/下注滑块/手牌）
- [x] `src/App.tsx`：新增 `/room/:code` 路由
- [x] WS → Zustand → PixiJS 桥接完成，`npm run build` 通过

---

## Phase 4：React 面板 + 聊天 ✅ 构建通过

- [x] `src/react/panels/ActionPanel.tsx`：操作按钮（fold/check/call/bet/raise/all-in）
- [x] `src/react/panels/BetSlider.tsx`：下注滑块 + 快捷按钮（1/3、1/2、3/4、Pot、All-In）
- [x] `src/react/panels/ChatPanel.tsx`：可折叠聊天框，Enter 发送，限 200 字
- [x] `src/react/panels/HandHistory.tsx`：本手结果弹层（5 秒自动消失）
- [x] `src/react/panels/RoomInfo.tsx`：房间码 + 一键复制邀请链接
- [x] `src/store/chat.ts`：Zustand 聊天消息 store（最多 200 条）
- [x] `src/store/game.ts`：新增 `lastHandResult` 字段（HandHistory 驱动）
- [x] `src/hooks/useWebSocket.ts`：新增 `chat_message` → `useChatStore.addMessage()`
- [x] `src/react/pages/RoomPage.tsx`：重构，使用所有面板组件
- [x] 后端：聊天消息中继（1 秒限速）
- [x] 后端：补充筹码接口（`add_chips`，遵守 MaxBuyIn 上限）
- [x] 后端：暂离逻辑（`sit_out`，自动弃牌）

---

## Phase 5：上线准备 ✅ 构建通过

- [x] 优雅关闭：`game.Registry` 跟踪所有 Engine，SIGTERM 时 `StopAll()` 等待全部退出
- [x] 空房间 GC：`room.Manager.StartGC(5min, 30min, clientCountFn)`，最后玩家离开时立即关闭
- [x] Dockerfile 多阶段构建（go-builder + node-builder + distroless 最终镜像）
- [x] `docker-compose.yml`（server + mysql + nginx）
- [x] `nginx/nginx.conf`（静态文件 + `/api/` 反代 + WebSocket Upgrade）
- [x] 前端断线自动重连（指数退避 1s→2s→4s…最大 30s，最多 10 次）
- [x] 连接丢失 UI 提示（顶部红色横幅，显示重连进度）
- [x] 结构化日志（slog）✅ Phase 2 已完成
- [x] 健康检查接口（`GET /health`）✅ Phase 2 已完成

---

## Phase 6：AI 玩家 ✅ 构建通过

### 功能点

- [x] **创建房间时可指定 AI 玩家数**：大厅页新增"AI 玩家数"输入框（min=0, max=人数上限-1），创建请求携带 `bot_count`；后端 `validateConfig` 校验范围
- [x] **AI 自动入座**：第一个真人通过 WS 加入房间时，Engine 在同一 goroutine 内调用 `seatBots()`，将配置数量的 bot 依次入座并广播 `player_joined`
- [x] **AI 自动行动**：轮到 AI 时，`broadcastActionRequired` 启动独立 goroutine，随机延迟 1–3 秒后向 `hub.Inbound` 注入 `CmdAction`，走与真人相同的 `ValidateAction → ApplyAction` 路径
- [x] **AI 决策策略**：无需跟注时 80% check / 20% bet(2×BB)；面对下注时 75% call / 20% raise(2× current) / 5% fold
- [x] **线程安全**：goroutine 只读快照变量，写操作经 buffered channel (256)，引擎主循环单写者；`ValidateAction` 拒绝过期的 bot 行动
- [x] **真人全部离开时清场**：统计非 bot 的人类玩家数，人类=0 时移除全部 bot 座位并重置 `botsSeated`，启动 30s 宽限期计时器；宽限期内有真人重连则 bots 重新入座
- [x] **前端 AI 标识**：bot 座位名称前加 🤖 前缀，边框与头像固定为蓝色（`0x4060a0`），与真人（哈希色头像 + 金色激活边框）视觉区分

### 改动文件

- [x] `internal/room/model.go`：`RoomConfig` 新增 `BotCount int`
- [x] `internal/room/manager.go`：`validateConfig` 校验 `bot_count >= 0 && bot_count < max_players`
- [x] `internal/game/model.go`：`Player` 新增 `IsBot bool`；`SeatSnapshot` 新增 `is_bot` JSON 字段；`Snapshot()` 填充 `IsBot`
- [x] `internal/ws/message.go`：`PlayerJoinedPayload` 新增 `IsBot bool`
- [x] `internal/game/bot.go`（新建）：`IsBotID()`、`botUserID()`、`botDisplayName()`、`scheduleAIAction()`、`decideBotAction()`
- [x] `internal/game/engine.go`：`Engine.botsSeated`；第一个真人加入时 `seatBots()`；`broadcastActionRequired` 触发 `scheduleAIAction`；`handleDisconnect` 守卫 + 全人类离开时清除 bot 座位并重置 `botsSeated`
- [x] `src/api/http.ts`：`RoomConfig` 新增 `bot_count?: number`
- [x] `src/store/game.ts`：`SeatSnapshot` 新增 `is_bot?: boolean`；`applyPlayerJoined` 填充 `is_bot`
- [x] `src/react/pages/LobbyPage.tsx`：新增 AI 玩家数输入框
- [x] `src/pixi/components/SeatSprite.ts`：bot 显示名前缀 🤖，蓝色边框 + 蓝色头像

---

## Phase 7：AI 玩家风格升级 ✅ 构建通过

### 功能点

- [x] **四种 bot 风格**：TAG（紧凶）、LAG（松凶）、Station（松被动）、Rock（紧被动），每种风格有独立的入局/加注/下注/弃牌阈值和虚张声势率
- [x] **风格主题**：创建房间时可选混合 / 激进 / 被动 / 随机四种主题，决定同桌 bot 的风格分配规则
- [x] **Preflop 强度评估**：对子按等级线性映射，非对子按点数和 + 同花/连牌加成
- [x] **Postflop 强度评估**：调用现有 `EvaluateHand()`，按成牌类别（SF→HC）映射 0–1 强度
- [x] **风格感知决策**：preflop 按 EnterThreshold/RaiseThreshold 决定入局/加注/弃牌；postflop 按 BetThreshold/FoldThreshold 决定下注/弃牌/加注；BluffRate 引入随机虚张声势

### 改动文件

- [x] `internal/room/model.go`：`RoomConfig` 新增 `BotStyle string`
- [x] `internal/room/manager.go`：`validateConfig` 校验 `bot_style` 合法值
- [x] `internal/game/model.go`：`Player` 新增 `BotStyle string`
- [x] `internal/game/bot.go`：全面重写，新增 `BotPersonality`、`assignBotStyle()`、`preflopStrength()`、`postflopStrength()`、`handStrength()`、`decideBotAction()`
- [x] `internal/game/engine.go`：`seatBots()` 调用 `assignBotStyle()` 写入 `p.BotStyle`；`scheduleAIAction()` 传入手牌快照
- [x] `src/api/http.ts`：`RoomConfig` 新增 `bot_style?: string`
- [x] `src/react/pages/LobbyPage.tsx`：AI 风格下拉选择框（混合 / 激进 / 被动 / 随机）

---

## Phase 8：渲染修复 ✅ 构建通过

- [x] **Canvas 宽高比修复**（`src/pixi/app.ts`）：移除 `height: '100%'`，改用 `aspectRatio: 1200/700`，防止 CSS 垂直拉伸导致整体模糊
- [x] **手牌可见性修复**（`src/pixi/components/SeatSprite.ts`）：删除 `else if (isActive)` 背面牌分支（showdown 前 `seat.hole` 本就是 undefined，无需此逻辑）；手牌逻辑统一为 `isLocal ? myHole : seat.hole`
- [x] **手牌渲染统一到 PixiJS**（`src/react/pages/RoomPage.tsx`）：移除 React overlay 的 `myCards` HTML 版手牌（含 `isRedSuit`/`suitSymbol` 辅助函数和对应 CSS），`SeatSprite` 统一渲染本地玩家和摊牌手牌

---

---

## Phase 9：UI 视觉精调 ✅

### 头像框

- [x] **本地玩家（YOU）头像框光晕扩展**：最外层从 R+24 扩到 R+40，主环线宽加粗至 5px，active / normal 两态分别调整 alpha 梯度，整体气势更足
- [x] **远端玩家 normal 态升级**：原银蓝双环 + 瞄准框改为多层银白向外扩散光晕（R+1 ~ R+32），与本地玩家金色光晕结构对称但色调区分
- [x] **远端玩家 active 态扩展**：最外层从 R+18 扩到 R+32，主环 4.5px，更大气
- [x] **移除 4 方位角装饰弧**：简化视觉，去掉瞄准框感装饰
- [x] **统一头像半径**：`AVATAR_R_REMOTE` 由 40 调整为 48，所有玩家头像大小一致

### 字体

- [x] 远端玩家昵称 12 → 15
- [x] 远端玩家筹码金额 11 → 14
- [x] 位置标签（BTN/SB/BB）10 → 12
- [x] 下注徽章金额 11 → 13，金币图标 12 → 14
- [x] 筹码余额副标签 8 → 10
- [x] 本地玩家 YOU 徽章 13 → 10（偏小更协调）
- [x] 本地玩家筹码金额 22 → 16

### 手牌位置

- [x] 所有玩家手牌统一向右偏移 18px（card0: R+12→R+30，card1: R+48→R+66），避免与光晕重叠

### 代码整理

- [x] **`ChipAnimation.ts`**：移除 `FlyChip.label`（是 `g` 子节点，无需独立引用）
- [x] **`DealAnimation.ts`**：移除未使用的 `vx/vy/progress` 字段，改为直接存 `speed`，tick 逻辑同步简化

---

## TODO：已知缺口

### 游戏引擎

- [ ] 筹码归零踢出：栈为 0 的玩家未自动离桌，需在 `hand_result` 分配后检查并 UnseatPlayer
- [ ] 带入金额校验：`handleJoinRoom` 未按 `min_buy_in` / `max_buy_in` 校验初始筹码，玩家永远以 MaxBuyIn 入座
- [ ] All-in 超额退还：`runShowdown` 中赢家若 all-in 金额超过其他人 TotalBet，多余部分未退还（`awardUncontested` 有处理，showdown 路径缺失）
- [ ] 手牌历史持久化：无 `hand_history` 表，牌局结果只广播不落库

### 本地测试 Bug（已修复，记录留档）

- [x] `websocket: response does not implement http.Hijacker`：`loggingMiddleware` 的 `responseWriter` 未代理 `Hijack()`，导致 WS 升级失败 → 已修复
- [x] React StrictMode 双重挂载导致 WS 立即断开 → 已移除 StrictMode
- [x] `onEmpty` 立即销毁房间导致重连 404 → 已改为 30s 宽限期
