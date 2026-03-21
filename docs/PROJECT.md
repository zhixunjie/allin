# AllIn — 熟人组局德州扑克

## 产品定位

面向熟人圈子的 No-Limit Texas Hold'em Cash Game 平台。
玩家通过 **房间码** 或 **邀请链接** 加入私人牌局，虚拟筹码娱乐，先做 PC Web，后续扩展 H5 / iOS / Android。

---

## 技术栈

| 层 | 选型 |
|----|------|
| 后端语言 | Go 1.25 |
| HTTP 框架 | Hertz v0.10.4（CloudWeGo） |
| WebSocket | gorilla/websocket（via contrib/ws） |
| 配置管理 | Viper（`server/base/config.yaml`） |
| 数据库访问 | sqlx + go-sql-driver/mysql |
| 持久化 | MySQL 8.0（AutoMigrate 自动建表） |
| 鉴权 | JWT HS256，7 天有效期 |
| 游戏状态 | 纯内存（Go struct + channel 单写者模式） |
| 手牌评估 | 纯 Go 枚举 C(7,5)=21 种五牌组合（HandRanks.dat 为可选加速） |
| 前端构建 | Vite 5 + TypeScript |
| UI | React 18 + CSS Modules |
| 2D 渲染 | PixiJS v8（WebGL，程序化渲染，无外部图片） |
| 状态管理 | Zustand |
| 路由 | React Router v6 |

---

## 目录结构

```
allin/
├── server/                     # Go 后端（单 module: github.com/allin/server）
│   ├── base/                   # 微服务主体
│   │   ├── main.go             # 启动入口（Hertz + Viper）
│   │   ├── router.go           # 路由注册
│   │   ├── config.yaml         # 本地配置
│   │   └── biz/
│   │       ├── handler/        # HTTP Handler（User / Room）
│   │       ├── service/        # 业务逻辑（UserSvc / RoomSvc）
│   │       ├── dao/            # 数据访问（userDao / roomDao + AutoMigrate）
│   │       ├── mw/             # 中间件（JWT）
│   │       └── model/          # 数据模型（User struct + 错误定义）
│   ├── contrib/                # 可复用组件
│   │   ├── ws/                 # WebSocket Hub + Client + Handler
│   │   ├── room/               # RoomManager（内存 + GC）
│   │   ├── game/               # 游戏引擎（状态机 + AI bot）
│   │   ├── eval/               # 手牌评估器
│   │   └── auth/               # JWT 签发/验证 + bcrypt
│   └── go.mod
│
├── ui-web/                     # React + PixiJS 前端
│   ├── src/
│   │   ├── api/                # HTTP + WebSocket 客户端
│   │   ├── store/              # Zustand stores（auth / room / game / chat）
│   │   ├── pixi/               # PixiJS 场景 + 组件
│   │   ├── react/              # React 页面 + 面板
│   │   └── hooks/
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

- Go 1.25+
- Node.js 20+
- MySQL 8.0（本地 `127.0.0.1:13306`，root 无密码，`allin` 库已存在）

### 启动后端

```bash
cd server
go run ./base
# 首次启动自动建表，输出：dao: auto-migrated 4 tables
# 健康检查：curl http://localhost:8080/health
```

### 启动前端

```bash
cd ui-web
npm install
npm run dev
# 默认 http://localhost:5173，端口占用时顺延
```

### 验证后端是否已在运行

```bash
curl http://localhost:8080/health
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
| POST | `/api/rooms` | 创建房间：`{ small_blind, big_blind, min_buy_in, max_buy_in, max_players, bot_count, bot_style }` |
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

## AI Bot 风格系统

### 风格人设参数表

| 参数 | 含义 | TAG（紧凶） | LAG（松凶） | Station（松被动） | Rock（紧被动） |
|------|------|:-----------:|:-----------:|:-----------------:|:--------------:|
| PreflopEnterThreshold | 主动入局所需最低 preflop 强度 | 0.65 | 0.35 | 0.30 | 0.78 |
| PreflopRaiseThreshold | preflop 选择加注而非跟注的门槛 | 0.80 | 0.50 | 0.85 | 0.92 |
| PostflopBetThreshold  | postflop 主动下注所需最低强度  | 0.55 | 0.35 | 0.70 | 0.72 |
| PostflopFoldThreshold | postflop 面对下注时弃牌的强度上限 | 0.30 | 0.15 | 0.05 | 0.48 |
| BluffRate             | 手牌偏弱时仍激进行动的概率 | 0.08 | 0.22 | 0.02 | 0.02 |

### 风格主题（RoomConfig.BotStyle）

| 值 | 中文 | 分配规则 |
|----|------|---------|
| `mixed`（默认/空） | 混合 | 按序号循环：TAG→LAG→Station→Rock |
| `aggressive` | 激进 | 按序号交替：TAG→LAG→TAG→LAG… |
| `passive` | 被动 | 按序号交替：Rock→Station→Rock… |
| `random` | 随机 | 每个 bot 独立随机选一种 |

### 手牌强度映射

**Preflop（无公共牌）**：对子 `0.5 + (rank-2)/24×0.5`，非对子按点数和归一化，同花 +0.04，连牌 +0.02。

**Postflop 成牌类别**：

| 牌型 | 强度 | 牌型 | 强度 |
|------|:----:|------|:----:|
| 同花顺 | 1.00 | 三条 | 0.60 |
| 四条   | 0.95 | 两对 | 0.45 |
| 葫芦   | 0.88 | 一对 | 0.30 |
| 同花   | 0.78 | 高牌 | 0.15 |
| 顺子   | 0.70 | | |

---

## 邀请链接格式

```
https://allin.example.com/join/XXXXXX
```

前端读取 URL 参数中的 6 位房间码，请求 `/api/rooms/:code` 显示房间信息后引导用户加入。
