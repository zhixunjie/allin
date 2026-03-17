# AllIn — 熟人组局德州扑克

## 产品定位

面向熟人圈子的 No-Limit Texas Hold'em Cash Game 平台。
玩家通过 **房间码** 或 **邀请链接** 加入私人牌局，虚拟筹码娱乐，先做 PC Web，后续扩展 H5 / iOS / Android。

---

## 技术栈

| 层 | 选型 |
|----|------|
| 后端 | Go 1.22+（net/http + gorilla/websocket） |
| 前端游戏桌面 | PixiJS v8（WebGL Canvas 渲染） |
| 前端 UI 面板 | React 18 + TypeScript（DOM 覆盖层） |
| 状态管理 | Zustand |
| 构建工具 | Vite |
| 实时通信 | WebSocket + JSON |
| 鉴权 | JWT HS256，用户名/密码 |
| 持久化 | MySQL 8.0+（go-sql-driver/mysql） |
| 游戏状态 | 纯内存（Go struct + channel 单写者模式） |
| 手牌评估 | Two Plus Two 查表算法（HandRanks.dat） |

---

## 目录结构

```
allin/
├── allin-server/               # Go 后端
│   ├── cmd/server/main.go
│   ├── internal/
│   │   ├── auth/               # JWT + bcrypt
│   │   ├── user/               # 用户注册/登录
│   │   ├── room/               # 房间管理
│   │   ├── game/               # 游戏引擎（状态机 + 动作 + 牌型评估）
│   │   ├── eval/               # Two Plus Two 手牌评估器
│   │   ├── ws/                 # WebSocket Hub + Client
│   │   ├── store/              # MySQL 连接 + 自动建表
│   │   └── config/             # 环境变量配置
│   ├── assets/eval/HandRanks.dat
│   ├── go.mod
│   └── Makefile
│
├── allin-web/                  # React + PixiJS 前端
│   ├── src/
│   │   ├── api/                # HTTP + WebSocket 客户端
│   │   ├── store/              # Zustand stores
│   │   ├── pixi/               # PixiJS 场景 + 组件
│   │   ├── react/              # React 页面 + 面板
│   │   └── hooks/
│   ├── public/assets/          # 牌面图、筹码图、桌面图
│   ├── vite.config.ts
│   └── package.json
│
└── docs/
    ├── PROJECT.md              # 本文档
    └── PROGRESS.md             # 实现进度
```

---

## 本地开发环境

### 依赖
- Go 1.22+
- Node.js 20+
- MySQL 8.0+（本地服务，端口 3306）

### 1. 创建数据库

```sql
CREATE DATABASE allin CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
```

### 2. 配置后端环境变量

复制并编辑：
```bash
cp allin-server/.env.example allin-server/.env
```

`.env` 内容：
```
MYSQL_DSN=root:yourpassword@tcp(127.0.0.1:3306)/allin?parseTime=true&charset=utf8mb4
JWT_SECRET=your-secret-key-here
SERVER_ADDR=:8080
```

### 3. 启动后端

```bash
cd allin-server
go run ./cmd/server
# 首次启动自动建表，输出：[store] auto-migrated 4 tables
```

### 4. 启动前端

```bash
cd allin-web
npm install
npm run dev
# 访问 http://localhost:5173
```

---

## HTTP API

### 鉴权

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/auth/register` | 注册：`{ username, password, display_name }` |
| POST | `/api/auth/login` | 登录：`{ username, password }` → `{ token, user }` |
| GET  | `/api/me` | 当前用户信息（需 JWT） |

### 房间

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/rooms` | 创建房间：`{ small_blind, big_blind, min_buy_in, max_buy_in, max_players }` |
| GET  | `/api/rooms/:code` | 获取房间快照（邀请链接预览） |

### WebSocket

```
GET /api/ws?room=XXXXXX
Authorization: Bearer <token>
```

或：`GET /api/ws?room=XXXXXX&token=<jwt>`

---

## WebSocket 消息协议

### 公共信封

```json
{
  "type": "action_required",
  "seq": 42,
  "ts": 1710000000000,
  "payload": { ... }
}
```

### 服务端事件（Server → Client）

| type | 触发时机 |
|------|---------|
| `connected` | 连接建立，下发完整 GameSnapshot |
| `player_joined` | 玩家入座 |
| `player_left` | 玩家断线 |
| `game_started` | 新手牌开始 |
| `hole_cards` | 仅发给持牌玩家（2 张底牌） |
| `cards_dealt` | 通知他人某座位已发牌 |
| `street_started` | 翻/转/河牌街开始，含公共牌 |
| `action_required` | 轮到某玩家行动，含 `deadline_ts` |
| `action_taken` | 某玩家完成行动 |
| `action_timeout` | 超时自动弃牌/看牌 |
| `showdown` | 摊牌，含所有玩家底牌 |
| `hand_result` | 赢家 + 金额 + 各玩家剩余筹码 |
| `chat_message` | 聊天消息 |
| `error` | 命令错误响应 |

### 客户端命令（Client → Server）

| type | 说明 |
|------|------|
| `join_room` | 入座，含 `room_code` |
| `action` | `fold/check/call/bet/raise/all_in` + `amount` |
| `chat` | 聊天文本 |
| `add_chips` | 补充筹码 |
| `sit_out` | 暂离 |

---

## 游戏规则

- **模式**：No-Limit Texas Hold'em，现金局（Cash Game）
- **最大玩家数**：2-9 人
- **筹码**：虚拟筹码，可在手牌间补充
- **行动计时**：默认 30 秒，超时自动最优行动（可看牌则看牌，否则弃牌）
- **边底池**：支持多人全押产生的边底池，自动正确计算归属

---

## 邀请链接格式

```
https://allin.example.com/join/XXXXXX
```

前端读取 URL 参数中的 6 位房间码，请求 `/api/rooms/:code` 显示房间信息后引导用户加入。
