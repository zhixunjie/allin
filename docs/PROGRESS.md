# 实现进度

> 最后更新：2026-03-17
> 当前阶段：Phase 1 — 基础设施（后端完成，待验证 + 前端待开始）

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

### 前端

- [ ] Vite + React + TypeScript 项目初始化
- [ ] 依赖安装（PixiJS、Zustand、React Router）
- [ ] `src/api/http.ts`：fetch 封装
- [ ] `src/api/ws.ts`：WebSocket 单例 + 事件总线
- [ ] `src/store/auth.ts`：鉴权状态
- [ ] `src/store/room.ts`：房间状态
- [ ] `src/react/pages/LoginPage.tsx`：登录/注册页
- [ ] `src/react/pages/LobbyPage.tsx`：大厅页（创建/加入房间）
- [ ] 基础路由配置

---

## Phase 2：游戏引擎

- [ ] `internal/eval/table.go`：加载 HandRanks.dat
- [ ] `internal/eval/eval.go`：Evaluate7 函数
- [ ] `internal/eval/describe.go`：rank → 牌型名称
- [ ] `internal/game/model.go`：GameState、Player、Card、Pot struct
- [ ] `internal/game/deck.go`：洗牌 + 发牌
- [ ] `internal/game/state_machine.go`：FSM 状态与转换
- [ ] `internal/game/action.go`：动作校验与应用
- [ ] `internal/game/pot.go`：BuildPots 边底池算法
- [ ] `internal/game/timer.go`：行动倒计时 + 自动弃牌
- [ ] `internal/game/engine.go`：游戏主循环
- [ ] 单元测试：手牌评估正确性
- [ ] 单元测试：边底池计算（20+ 场景）
- [ ] 单元测试：完整手牌流程（preflop → showdown）

---

## Phase 3：PixiJS 牌桌 + WS 集成

- [ ] `src/pixi/app.ts`：PixiJS Application 初始化
- [ ] `src/pixi/assets.ts`：资源预加载（牌面、筹码、桌面）
- [ ] `src/pixi/components/CardSprite.ts`：单张牌（翻牌动画）
- [ ] `src/pixi/components/SeatSprite.ts`：玩家座位
- [ ] `src/pixi/components/PotDisplay.ts`：底池显示
- [ ] `src/pixi/components/TimerArc.ts`：圆弧倒计时
- [ ] `src/pixi/scenes/TableScene.ts`：主场景
- [ ] `src/pixi/scenes/DealAnimation.ts`：发牌飞行动画
- [ ] `src/pixi/scenes/ChipAnimation.ts`：筹码入池动画
- [ ] `src/store/game.ts`：GameSnapshot Zustand store
- [ ] `src/hooks/useWebSocket.ts`：WS 事件订阅
- [ ] `src/hooks/useGameState.ts`：派生 UI 状态
- [ ] `src/hooks/useActionTimer.ts`：本地倒计时
- [ ] `src/react/pages/RoomPage.tsx`：PixiJS canvas + React 覆盖层
- [ ] WS → Zustand → PixiJS 桥接完成，浏览器内可完整打一手牌

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
