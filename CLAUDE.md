# AllIn — Claude 项目说明

## 项目结构

```
allin/
├── allin-server/   # Go 后端（net/http, gorilla/websocket, MySQL）
├── allin-web/      # 前端（Vite + React + TypeScript + PixiJS v8）
└── docs/           # 设计文档和进度追踪
```

## 本地开发启动

### 前置条件
- MySQL 8.0 运行在 `127.0.0.1:13306`，root 无密码
- 连接命令：`mysql -h 127.0.0.1 -P 13306 -u root`
- 数据库 `allin` 已存在，启动时 AutoMigrate 自动建表

### 后端（allin-server/）
```bash
go run ./cmd/server
```
- 监听 `:8080`
- 健康检查：`curl http://localhost:8080/health`

### 前端（allin-web/）
```bash
npm run dev
```
- 默认 `:5173`，端口占用时顺延
- `/api` 代理到 `http://localhost:8080`（含 WebSocket）

### 验证后端是否已在运行
```bash
curl http://localhost:8080/health
```

## 技术栈

| 层 | 技术 |
|----|------|
| 后端语言 | Go 1.25 |
| HTTP/WS | `net/http` + `gorilla/websocket` |
| 数据库 | MySQL 8.0，`go-sql-driver/mysql` |
| 鉴权 | JWT HS256，7 天有效期 |
| 前端构建 | Vite 5 + TypeScript |
| UI | React 18 + CSS Modules |
| 2D 渲染 | PixiJS v8（WebGL，程序化渲染，无外部图片） |
| 状态管理 | Zustand |
| 路由 | React Router v6 |

## 关键设计决策

- **无循环依赖**：`game` 包导入 `ws`；`ws` 不导入 `game`。通过 `ws.Handler.SetEngineStarter(func)` 回调在 `main.go` 中注入 Engine 工厂。
- **游戏引擎**：单 goroutine，`select` 监听 `hub.Inbound` + 可重置 timer channel（`nil` channel 不触发 select）。
- **座位旋转**：PixiJS TableScene 将本地玩家始终旋转到底部（display index 0）。
- **手牌评估**：纯 Go 枚举 C(7,5)=21 种五牌组合，`HandRanks.dat` 为可选加速路径。
- **边底池**：按 TotalBet 升序迭代，逐层计算各玩家贡献。

## 前端渲染分工规则

| 层 | 负责内容 |
|----|---------|
| **PixiJS** | 所有游戏画面：牌桌、手牌、公共牌、座位、筹码、动画、玩家信息 |
| **React** | 纯 UI 控件：聊天、连接状态、大厅/登录页 |

> 不得在 React JSX 中重复渲染游戏画面元素，防止双重渲染 bug。

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
详见 `docs/PROGRESS.md`。当前阶段：Phase 8（UI 视觉重绘）进行中。
