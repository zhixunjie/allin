# 实现进度

> 最后更新：2026-03-18
> 当前阶段：Phase 2 — 游戏引擎（后端完成，待集成测试）

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

## Phase 4：React 面板 + 聊天

- [ ] `src/react/panels/ActionPanel.tsx`：操作按钮
- [ ] `src/react/panels/BetSlider.tsx`：下注滑块 + 快捷按钮
- [ ] `src/react/panels/ChatPanel.tsx`：聊天框
- [ ] `src/react/panels/HandHistory.tsx`：本手结果弹层
- [ ] `src/react/panels/RoomInfo.tsx`：房间码 + 邀请链接
- [ ] 后端：聊天消息中继（限速）
- [ ] 后端：补充筹码接口
- [ ] 后端：暂离逻辑

---

## Phase 5：上线准备

- [ ] 优雅关闭（排空活跃牌局）
- [ ] 空房间 GC（30 分钟后回收）
- [ ] Dockerfile 多阶段构建
- [ ] docker-compose.yml（app + mysql）
- [ ] Nginx 配置（WebSocket 反代）
- [ ] 前端断线自动重连（指数退避）
- [ ] 连接丢失 UI 提示
- [ ] 结构化日志（slog）
- [ ] 健康检查接口（`GET /health`）
