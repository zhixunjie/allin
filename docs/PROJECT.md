# AllIn — 功能点与实现逻辑全览

> 最后更新：2026-03-25

---

## 目录

1. [系统架构](#1-系统架构)
2. [用户系统](#2-用户系统)
3. [房间系统](#3-房间系统)
4. [WebSocket 通信层](#4-websocket-通信层)
5. [游戏引擎（状态机）](#5-游戏引擎状态机)
6. [手牌评估器](#6-手牌评估器)
7. [边底池算法](#7-边底池算法)
8. [AI Bot 系统](#8-ai-bot-系统)
9. [筹码账务系统](#9-筹码账务系统)
10. [手牌历史持久化](#10-手牌历史持久化)
11. [前端状态管理](#11-前端状态管理)
12. [前端 PixiJS 渲染](#12-前端-pixijs-渲染)
13. [前端 React 面板](#13-前端-react-面板)
14. [数据库表结构](#14-数据库表结构)

---

## 1. 系统架构

### 分层结构

```
客户端 (React + PixiJS)
    ↕ HTTP REST (Hertz)
    ↕ WebSocket (gorilla/websocket via adaptor)
server/base/biz/handler  →  service  →  dao
                                          ↕ sqlx / MySQL
server/contrib/ws/Hub   ←→  game/Engine
                              ↕
                        contrib/room/Manager
                        contrib/eval (手牌评估)
```

### 无循环依赖原则

- `contrib/game` 导入 `contrib/ws`、`contrib/room`、`contrib/eval`、`base/biz/dao`
- `contrib/ws` 不导入 `contrib/game`（通过 `ws.Handler.SetEngineStarter` 回调注入 Engine 工厂，在 `main.go` 完成装配）
- `contrib/room/manager` 直接调用 `base/biz/dao.RoomDao`

### 关键常量

| 常量 | 值 | 说明 |
|------|-----|------|
| `handStartDelay` | 10s | 两手牌之间的等待时间（全员已准备时缩短为 500ms） |
| `chatRateLimit` | 1s | 聊天消息限速 |
| `emptyGracePeriod` | 30s | 人类全离场后 bot 清场宽限期 |
| 默认 `ActionTimeSec` | 30s | 玩家行动超时时间 |
| `botReplaceDelay` | 8s | bot 破产后自动补位延迟 |

---

## 2. 用户系统

### HTTP 接口

| 方法 | 路径 | 鉴权 | 说明 |
|------|------|------|------|
| POST | `/api/auth/register` | 无 | 注册，初始筹码 10,000 |
| POST | `/api/auth/login` | 无 | 登录，返回 JWT |
| GET | `/api/me` | JWT | 查询当前用户信息 |
| POST | `/api/chips/claim` | JWT | 余额 < 1000 时领取 10,000 免费筹码 |
| GET | `/health` | 无 | 健康检查 |

### 注册逻辑 (`UserSvc.Register`)

1. 校验用户名/密码/显示名非空
2. `bcrypt` 哈希密码（`auth/password.go`）
3. `userDao.Create` 写库，主键冲突返回 `ErrUsernameTaken`（HTTP 409）
4. `auth/jwt.go` 签发 HS256 JWT（7 天有效期，密钥从 `config.yaml` 读取）

### 登录逻辑 (`UserSvc.Login`)

1. `userDao.GetByUsername` 查询，未找到返回 `ErrUserNotFound`（HTTP 401）
2. `bcrypt.CompareHashAndPassword` 验证密码
3. 签发 JWT，返回 token + 用户信息

### JWT 中间件 (`mw.JWTMiddleware`)

- 从 `Authorization: Bearer <token>` 提取 token
- `auth/jwt.go` 解析并校验，将 `userID` 注入 Hertz context（`mw.GetUserID(c)` 取出）

---

## 3. 房间系统

### HTTP 接口

| 方法 | 路径 | 鉴权 | 说明 |
|------|------|------|------|
| POST | `/api/rooms` | JWT | 创建房间 |
| GET | `/api/rooms/:code` | 无 | 查询房间信息 |
| GET | `/api/rooms/:code/hands` | JWT | 查询房间最近 20 手记录 |

### 创建房间逻辑 (`room.Manager.Create`)

1. `validateConfig` 校验房间参数：
   - `big_blind == small_blind * 2`
   - `min_buy_in >= big_blind * 10`
   - `max_buy_in >= min_buy_in`
   - `max_players` 在 2–9 之间
   - `bot_count` 在 0 到 `max_players-1` 之间
   - `bot_style` 只能是 `mixed/aggressive/passive/random`
   - `action_time_sec` 在 5–120 之间（0 时默认为 30）
2. 生成唯一 6 位房间码（最多重试 10 次）
3. `dao.RoomDao.Persist` 写库（`room_history` 表）
4. 注册到内存 `rooms map[string]*Room`
5. 在 `ws.Handler.ServeWS` 中为该房间创建 `Hub` + `Engine`，分别在独立 goroutine 中运行

### 房间 GC (`Manager.StartGC`)

- 每 5 分钟扫描一次
- 条件：无连接客户端 && 空闲时长 > 30 分钟
- 满足条件则 `Manager.Close`：从内存移除 + 数据库标记 `ended_at`

### 优雅关闭

- 监听 `SIGTERM`，调用 `game.Registry.StopAll()`，关闭所有引擎 goroutine

---

## 4. WebSocket 通信层

### 连接建立 (`ws.Handler.ServeWS`)

1. Upgrade HTTP → WebSocket（`gorilla/websocket`，通过 Hertz `adaptor.HertzHandler` 桥接）
2. 从 JWT（query param `token` 或 `Authorization` header）解析 `userID` 和 `displayName`
3. 创建或复用该房间的 `Hub`，按需启动 `Hub.Run()` 和 `Engine.Run()`
4. 创建 `Client`，注册到 Hub（`hub.register <- client`）
5. 启动 `client.writePump`（goroutine）和 `client.readPump`（当前 goroutine）

### Hub (`contrib/ws/hub.go`)

- `clients map[string]*Client`（以 userID 为键）
- `Inbound chan InboundMessage`（容量 256，游戏引擎消费）
- `register / unregister chan *Client` 序列化注册操作，避免并发写 map
- 客户端断开时自动向 `Inbound` 注入 `CmdDisconnect` 消息

### 消息信封格式

```json
{ "type": "action_required", "seq": 42, "ts": 1710000000000, "payload": { ... } }
```

### 服务端 → 客户端（事件类型）

| 类型 | 说明 |
|------|------|
| `connected` | WS 握手成功，附带游戏快照（重连场景） |
| `player_joined` | 玩家入座或重连（`is_reconnect` 区分） |
| `player_left` | 玩家离场 |
| `game_started` | 新一手牌开始（含庄/盲座位信息） |
| `hole_cards` | 私密发送本人手牌（仅当事人） |
| `cards_dealt` | 广播哪些座位收到了手牌（仅座位列表，不含牌面） |
| `street_started` | 新街道开始（含公共牌/底池） |
| `action_required` | 轮到某玩家行动（含截止时间戳、跟注额、最小加注额） |
| `action_taken` | 玩家已行动（含行动类型/金额/剩余筹码/总底池） |
| `action_timeout` | 玩家行动超时，自动弃牌或过牌 |
| `showdown` | 摊牌，公开各玩家手牌及牌型 |
| `hand_result` | 本手结算（含 winners/seats/best_hand/all_players） |
| `chat_message` | 聊天消息中继 |
| `sit_out_status` | 玩家离座/归座状态变更 |
| `stack_updated` | 玩家筹码变化（补充筹码后广播） |
| `ready_status` | 当前准备人数广播（含 `ready_count`、`total_count`） |
| `error` | 错误响应（含错误码 code、RefSeq） |

### 客户端 → 服务端（命令类型）

| 类型 | 说明 |
|------|------|
| `join_room` | 加入房间（含 `room_code`、可选 `buy_in`） |
| `action` | 玩家行动（`fold/check/call/bet/raise/all_in` + `amount`） |
| `chat` | 发送聊天消息（限 1–200 字符，1 秒限速） |
| `add_chips` | 补充筹码（手牌间隙，从账户余额扣除） |
| `sit_out` | 离座/归座切换 |
| `leave_table` | 主动离桌（仅限手牌间隙） |
| `ready` | 玩家准备好开始下一手（结算后发送，全员准备则立即开局） |

---

## 5. 游戏引擎（状态机）

### 架构

- 每个房间一个 `Engine` 实例，在独立 goroutine 中运行 `Engine.Run()`
- **单 goroutine** 处理所有状态变更，无需加锁
- `select` 监听四个 channel：`hub.Inbound`（玩家消息）、`timerC`（可重置计时器）、`botReplaceC`（bot 补位定时器）、`quit`（退出）
- `timerC = nil` 时该分支永不触发（惰性定时器实现可重置效果）

### 街道状态机

```
Idle → PreFlop → Flop → Turn → River → Showdown → Idle
              ↘ (只剩 1 名活跃玩家)
                 awardUncontested → Idle
```

### 手牌开始流程 (`startHand`)

1. 清空 `handActions` 行动日志（`handActions = handActions[:0]`）
2. `HandNum++`，`Street = PreFlop`，清空公共牌
3. `nextEligibleSeatAfter` 移动庄家按钮
4. 单挑（2 人）：庄家 = 小盲，对方 = 大盲；多人：庄家右 = 小盲，再右 = 大盲
5. 重置所有玩家状态（`Bet/TotalBet/Folded/AllIn/ActedThisStreet/Hole`）
6. `postBlind`：扣除盲注（筹码不足则全押）
7. 洗牌（Fisher-Yates），轮转发牌（座位顺序先各发 1 张，共发 2 轮）
8. 广播 `game_started`
9. `SendTo` 私密发送各玩家手牌（`hole_cards`），广播 `cards_dealt`（仅座位列表）
10. 单挑时庄家翻牌前先行动；多人时从大盲左边第一个合格玩家开始
11. 广播 `action_required`，启动行动计时器

### 行动处理 (`handleAction`)

1. 解析 `ActionCmd` payload
2. `ValidateAction` 校验合法性：
   - 游戏必须处于活跃街道（非 Idle/Showdown）
   - 必须是当前行动座位
   - 玩家未弃牌/未全押
   - `check`：当前无欠注
   - `call`：当前有欠注可跟
   - `bet`：当前无下注且金额 >= 大盲
   - `raise`：当前有下注且目标总额 >= `CurrentBet + MinRaise`，且筹码足以加注
   - `all_in`：筹码 > 0
3. `ApplyAction` 修改内存状态（激进行为重置其他玩家 `ActedThisStreet = false`）
4. 追加到 `handActions` 日志（PlayerID/Action/Amount/Street）
5. 广播 `action_taken`
6. `advanceOrEnd`

### `ApplyAction` 细节

| 行动 | 逻辑 |
|------|------|
| `fold` | `Folded = true`，`ActedThisStreet = true` |
| `check` | `ActedThisStreet = true` |
| `call` | 扣除 `min(toCall, stack)`，筹码耗尽则 `AllIn = true` |
| `bet` | 设置 `CurrentBet`，更新 `MinRaise = amount`；激进行为重置其他人 |
| `raise` | 目标总额赋值给 `Bet`，更新 `MinRaise = raiseBy`；激进行为重置其他人 |
| `all_in` | 全部筹码入池，若超过 `CurrentBet` 则更新 `CurrentBet` 和 `MinRaise` |

### 推进逻辑 (`advanceOrEnd`)

1. `ActivePlayers()`（未弃/未离座）只剩 1 人 → `awardUncontested`
2. `BettingRoundOver()`：所有未弃/未离/未全押玩家都已行动且 `Bet >= CurrentBet` → `nextStreet`
3. `nextActableSeat`（未弃/未离/未全押）返回下一个行动座位 → 广播 `action_required`
4. 若无可行动座位（全员全押）→ `nextStreet`

### 街道推进 (`nextStreet`)

1. 重置 `Bet=0`、`ActedThisStreet=false`、`CurrentBet=0`、`MinRaise=BB`
2. PreFlop → Flop（发 3 张），Flop → Turn（发 1 张），Turn → River（发 1 张），River → `runShowdown`
3. 广播 `street_started`
4. 若 `CanAct()` 为空（全员全押）：`ActionSeat=-1`，设 2 秒定时器自动推进
5. 否则翻牌后从庄家左边第一个活跃玩家开始

### 行动超时 (`handleTimeout`)

- `Street == Idle`：手牌开始计时器触发，`EligibleToStart() >= 2` 则 `startHand`
- `ActionSeat == -1`：全员全押自动推进，调用 `nextStreet`
- 否则：`Bet >= CurrentBet` 则自动 `check`，否则自动 `fold`，广播 `action_timeout`，追加行动日志

### 摊牌 (`runShowdown`)

1. 公开所有未弃牌玩家手牌，调用 `EvaluateHand` 计算牌型名，广播 `showdown`
2. `BuildPots` 计算主池和边池
3. 逐池：`EvaluateHand` 找最强玩家（rank 最小），平分底池，余数归第一赢家
4. 取首位赢家最佳五张 `BestFiveStrings`
5. 构建 `all_players`（含弃牌者信息），广播 `hand_result`
6. 异步 `saveHandHistory`，调用 `kickBrokePlayers`、`cleanupDisconnected`
7. `Street = Idle`，调用 `scheduleNextHand`

### 未摊牌颁奖 (`awardUncontested`)

- 活跃玩家仅剩 1 人时，将所有玩家 `TotalBet` 之和全部颁给该玩家
- 广播 `hand_result`（赢家手牌不公开，`all_players` 中弃牌者手牌为空）
- 同样异步保存历史，执行清理逻辑，调用 `scheduleNextHand`

### 断线处理 (`handleDisconnect`)

- Bot ID 注入的假消息直接忽略（`IsBotID` 检测）
- **手牌间隙断线**：立即离座，`cashOut` 返还筹码，广播 `player_left`，调用 `maybeStartEmptyTimer`
- **手牌中断线**：
  - `Disconnected = true`，保留座位
  - 若当前轮到该玩家：执行 `ApplyAction(fold)`，`advanceOrEnd`
  - 否则：直接 `Folded = true`，`checkHandOver`

### 断线重连 (`handleJoinRoom` 重连路径)

- `FindPlayer` 找到同 userID 且 `Disconnected == true`
- 清除标记，`sendSnapshot` 发送当前快照
- 广播 `player_joined`（`is_reconnect: true`）

### 手牌结束后清理 (`cleanupDisconnected`)

- 遍历所有 `Disconnected == true` 的玩家，离座并 `cashOut`，广播 `player_left`

### 筹码归零踢出 (`kickBrokePlayers`)

- 遍历所有 `Stack == 0` 的玩家，离座，广播 `player_left`
- 调用 `maybeStartEmptyTimer`

### 空场处理 (`maybeStartEmptyTimer`)

- 统计人类玩家数，若为 0：移除所有 bot 座位，`botsSeated = false`
- 启动 30 秒宽限期（`time.AfterFunc`），触发 `onEmpty` 回调（`Manager.Close`）

### 补充筹码 (`handleAddChips`)

- 仅允许手牌间隙（`Street == Idle`）
- 新筹码量 = `min(stack + amount, MaxBuyIn)`，计算实际增量
- 查 DB 校验余额充足，`AdjustChips(-added, "add_chips", roomCode)` 扣除
- 广播 `stack_updated`

### 离座/归座 (`handleSitOut`)

- 广播 `sit_out_status`
- 离座且当前轮到该玩家：自动弃牌，`advanceOrEnd`
- 归座且 `Street == Idle` 且 `EligibleToStart() >= 2`：启动 `handStartDelay` 计时器

### 主动离桌 (`handleLeaveTable`)

- 仅允许手牌间隙，否则返回 `hand_in_progress` 错误
- 离座 `cashOut`，广播 `player_left`，调用 `maybeStartEmptyTimer`

### 准备系统 (`scheduleNextHand` / `handleReady`)

- `scheduleNextHand`：每手结束后调用，替代原内联 `resetTimer`
  1. 清空 `readyPlayers` 集合；Bot 自动标记为已准备
  2. 广播初始 `ready_status`（`ready_count / total_count`）
  3. 若全员已准备（`allEligibleReady`）→ 500ms 后直接开局；否则等待 `handStartDelay`（10s）
- `handleReady`：收到客户端 `ready` 命令后标记该玩家已准备，广播最新 `ready_status`；若此时全员已准备则立即缩短计时器至 500ms 开局
- `broadcastReadyStatus`：广播 `{ ready_count, total_count }` 给房间内所有客户端
- `allEligibleReady`：判断所有 `EligibleToStart()` 玩家均已在 `readyPlayers` 中

### Bot 破产自动补位 (`botReplaceC`)

- `kickBrokePlayers` 检测被踢出的破产 bot；若场内仍有人类玩家，启动 `botReplaceDelay`（8s）计时器
- 计时结束后调用 `seatBots()` 补充 bot 席位；若 `Street == Idle` 且满足开局条件则触发下一手倒计时
- `hasHumanPlayers()`：辅助函数，判断场内是否存在非 bot 玩家

### `EligibleToStart` 判定条件

玩家同时满足：`!SitOut && !Disconnected && Stack > 0`

---

## 6. 手牌评估器

**文件：** `server/contrib/eval/eval.go`、`describe.go`、`table.go`

### 双路径评估

- **快速路径**：`HandRanks.dat`（Two Plus Two 查找表）加载成功后使用，O(7) 查表
- **纯 Go 路径**：枚举 C(7,5)=21 种五牌组合，每组调用 `evaluateFive`，取最低等级

### `evaluateFive` 编码规则

`uint32 = cat<<20 | (15-v1)<<16 | (15-v2)<<12 | ...`（值越小越好）

| cat | 牌型 | cat | 牌型 |
|-----|------|-----|------|
| 0 | 同花顺 | 4 | 顺子 |
| 1 | 四条 | 5 | 三条 |
| 2 | 葫芦 | 6 | 两对 |
| 3 | 同花 | 7 | 一对 |
| — | — | 8 | 高牌 |

- 顺子特殊：A-2-3-4-5（轮子）最高牌按 5 计算
- `BestFive`：遍历 21 种组合返回构成最佳手牌的 5 张牌（用于摊牌展示）

### 调用接口

- `game.EvaluateHand(hole, community)` → `(rank uint32, handName string)`
- `game.BestFiveStrings(hole, community)` → `[]string`（公共牌 < 3 张时返回 nil）

---

## 7. 边底池算法

**文件：** `server/contrib/game/pot.go`

### 算法步骤

1. 收集所有 `TotalBet > 0` 的玩家，按 `TotalBet` 升序排列
2. 逐层迭代（每个新的 cap 对应一个底池）：
   - `amount`：所有玩家在 `(prevCap, cap]` 区间贡献之和
   - `eligible`：未弃牌且 `TotalBet >= cap` 的玩家列表
3. 返回 `[]Pot{Amount, Eligible}`

### 特性

- 弃牌玩家的超额下注贡献到底池但无资格赢取
- 全押玩家只能赢取其自身覆盖范围内的底池
- 平分时余数在 `runShowdown` 中归第一赢家

---

## 8. AI Bot 系统

**文件：** `server/contrib/game/bot.go`

### Bot 标识

- UserID 格式：`bot_<roomCode>_<index>`
- `IsBotID` 检测 `"bot_"` 前缀，所有跳过 DB 操作的逻辑都依赖此判断
- 显示名从预设名单循环：Alice / Bob / Charlie / Diana / Eve / Frank / Grace / Henry / Ivy / Jack

### 入座时机

- 首次有人类玩家加入时（`!e.botsSeated`），`seatBots()` 一次性入座全部 bot
- 人类全离场后清除所有 bot，`botsSeated = false`；若有人类重新加入则再次入座

### 四种风格参数

| 风格 | 类型 | Enter | Raise | Bet | Fold | Bluff |
|------|------|:-----:|:-----:|:---:|:----:|:-----:|
| TAG | 紧凶 | 0.65 | 0.80 | 0.55 | 0.30 | 0.08 |
| LAG | 松凶 | 0.35 | 0.50 | 0.35 | 0.15 | 0.22 |
| Station | 松被动 | 0.30 | 0.85 | 0.70 | 0.05 | 0.02 |
| Rock | 紧被动 | 0.78 | 0.92 | 0.72 | 0.48 | 0.02 |

### 风格主题分配

| 房间 `bot_style` | 分配规则 |
|------|------|
| `mixed`（默认/空） | TAG→LAG→Station→Rock 循环 |
| `aggressive` | TAG→LAG 交替 |
| `passive` | Rock→Station 交替 |
| `random` | 每个 bot 独立随机 |

### 手牌强度评估

**Preflop**：
- 口袋对子：`0.5 + (rank-2)/24.0 * 0.5`（22≈0.54，AA=1.0）
- 非对子：`(hi+lo-4)/(14+13-4)`，同花 +0.04，连牌 +0.02

**Postflop（公共牌 < 5 张）**：退回 Preflop 估算（避免零值填充导致评估越界）

**Postflop（公共牌 = 5 张）**：`EvaluateHand` 取 `rank>>20` 分类映射到强度表：
`[同花顺=1.0, 四条=0.95, 葫芦=0.88, 同花=0.78, 顺子=0.70, 三条=0.60, 两对=0.45, 一对=0.30, 高牌=0.15]`

### 决策逻辑 (`decideBotAction`)

**Preflop 无需跟注**（BB 免费过牌）：
- 强度 >= `RaiseThreshold` → raise(3×BB)；否则 → check

**Preflop 需跟注**：
- 强度 < `EnterThreshold` 且随机 > `BluffRate` → fold
- 强度 >= `RaiseThreshold` → raise(3×BB)
- 否则 → call（筹码不足则 all_in）

**Postflop 无人下注**：
- 强度 >= `BetThreshold` 或随机 < `BluffRate` → bet(0.6×pot，不低于 BB，不超过 stack 则 all_in)
- 否则 → check

**Postflop 面对下注**：
- 强度 < `FoldThreshold` 且随机 > `BluffRate` → fold
- 强度 >= `BetThreshold * 1.3` → raise(2.5×currentBet)
- 否则 → call（筹码不足则 all_in）

### 行动调度 (`scheduleAIAction`)

- 快照当前局面数据（只读），在新 goroutine 中延迟 1–3 秒后通过 `hub.Inbound` 注入行动
- 与真人走完全相同的 `ValidateAction → ApplyAction` 路径，过期行动被校验拒绝后静默丢弃

---

## 9. 筹码账务系统

**文件：** `server/base/biz/dao/user_dao.go`

### 筹码流动时机

| 事件 | 方向 | reason | ref_id |
|------|------|--------|--------|
| 加入房间（买入） | `-buyIn` | `buy_in` | roomCode |
| 离开/断线/踢出（现金兑出） | `+stack` | `cash_out` | roomCode |
| 补充筹码 | `-added` | `add_chips` | roomCode |
| 领取免费筹码 | `+10,000` | `claim_free` | — |

### `AdjustChips` 实现

事务操作：`UPDATE users SET chip_balance = chip_balance + ?` + `INSERT INTO chip_ledger`（完整审计日志）

### 余额校验

- **买入时**：查 `chip_balance >= buyIn`，不足返回 `insufficient_chips` 错误
- **补充时**：同样查库校验，新 stack 上限为 `MaxBuyIn`（超出部分截断后计算实际增量）

### Bot 跳过 DB

`IsBotID` 检测，bot 的所有筹码操作直接跳过数据库，stack 仅存在内存中

---

## 10. 手牌历史持久化

**文件：** `server/base/biz/dao/hand_history_dao.go`、`engine.go`

### 记录字段（`hand_history` 表）

| 字段 | 内容 |
|------|------|
| `room_id` | 房间数据库 ID |
| `hand_num` | 手牌编号（从 1 递增） |
| `players_json` | 座位快照（player_id/display_name/seat_index/stack） |
| `actions_json` | 完整行动序列（PlayerID/Action/Amount/Street） |
| `result_json` | 完整结算对象 `{ winners, seats, best_hand, all_players }`，同时作为 WS `hand_result` payload 广播 |
| `played_at` | 手牌完成时间戳 |

### 行动日志收集流程

1. `startHand` 时 `handActions = handActions[:0]` 清空
2. `handleAction`（真人和 bot 行动）追加 `actionLogEntry`
3. `handleTimeout`（超时自动行动）也追加记录
4. `runShowdown` / `awardUncontested` 结束后，异步 goroutine 调用 `HandHistoryDao.Save()`，不阻塞引擎主循环

---

## 11. 前端状态管理

### Store 架构

| Store | 文件 | 职责 |
|-------|------|------|
| `useGameStore` | `store/game.ts` | 游戏状态（GameSnapshot + 本地扩展字段） |
| `useAuthStore` | `store/auth.ts` | 用户认证（token/user），`updateChipBalance` 同步余额 |
| `useRoomStore` | `store/room.ts` | 房间信息 + `selectedBuyIn`（加入时用户选择的买入额，0=服务端默认） |
| `useChatStore` | `store/chat.ts` | 聊天消息列表（最多 200 条） |
| `useConnectionStore` | `store/connection.ts` | WebSocket 连接状态 |

### `useGameStore` 关键字段（超出 GameSnapshot 的扩展）

| 字段 | 说明 |
|------|------|
| `myUserId` | 本客户端用户 ID |
| `myHole` | 本人手牌（`hole_cards` 单独下发，不经过 seats） |
| `deadlineTs` | 当前行动截止时间（Unix 毫秒），驱动 `TimerArc` 倒计时 |
| `callAmount` | 跟注所需金额（`action_required` 更新） |
| `minRaiseAmount` | 最小加注额（`action_required` 更新） |
| `lastHandResult` | 上一手结算结果，驱动 `RoundResultModal` 显示，idle 时为 null |
| `readyCount` | 当前已准备的玩家数（`ready_status` 更新） |
| `readyTotal` | 需要准备的总玩家数（`ready_status` 更新） |

### WS 事件 → Store 方法映射

| WS 事件 | Store 方法 | 核心逻辑 |
|---------|-----------|---------|
| `connected` | `applyConnected` | 记录 myUserId；若附带快照则同步整局状态 |
| `game_started` | `applyGameStarted` | 重置座位下注/弃牌/全押/手牌状态，清空公共牌 |
| `hole_cards` | `applyHoleCards` | 仅更新 `myHole`（服务端只发给当事人） |
| `cards_dealt` | `applyCardsDealt` | 为已发牌座位设置背面占位 `['?','?']` |
| `street_started` | `applyStreetStarted` | 同步公共牌/底池/街道，重置本街下注 |
| `action_required` | `applyActionRequired` | 更新行动座位、倒计时、跟注/加注参数 |
| `action_taken` | `applyActionTaken` | 更新底池；更新该玩家筹码/下注/弃牌/全押状态 |
| `action_timeout` | `applyActionTaken` | 与 action_taken 相同处理（视为自动行动） |
| `showdown` | `applyShowdown` | 公开各玩家手牌，`street = Showdown` |
| `hand_result` | `applyHandResult` | 更新各玩家筹码，记录 `lastHandResult`，`street = Idle` |
| `player_joined` | `applyPlayerJoined` | 追加新座位（重连时若已存在则跳过） |
| `player_left` | `applyPlayerLeft` | 从座位列表移除 |
| `sit_out_status` | `applySitOut` | 更新座位 `sit_out` 字段 |
| `stack_updated` | `applyStackUpdated` | 更新指定座位 `stack` |
| `ready_status` | `applyReadyStatus` | 更新 `readyCount` / `readyTotal`；`game_started` / `reset` 时清零 |

### `useWebSocket` hook 生命周期

- `roomCode / token` 变化时重新建立连接
- WS `Open` 事件后自动发送 `join_room` 命令；若 WS 已 open 则直接发送（快速重渲染场景）
- 断线自动重连（指数退避，由 `wsClient` 实现）
- 组件卸载时：取消所有事件监听 → `wsClient.disconnect()` → 重置 gameStore 和 chatStore

### `useGameState` 派生状态

- `isMyTurn`：`action_seat === myServerSeat`
- `canCheck`：`callAmount === 0`（无欠注）
- `canCall`：`callAmount > 0`
- `canBet`：`current_bet === 0`（无人下注）
- `canRaise`：`current_bet > 0`

---

## 12. 前端 PixiJS 渲染

### 技术要点

- PixiJS v8，WebGL 渲染，程序化绘制（无外部图片资源）
- 画布固定比例 1200×700（`aspectRatio: 1200/700`，防 CSS 垂直拉伸）

### 场景层级（`TableScene`）

```
root (Container)
 ├─ 背景层    深空底色 + 星空颗粒 + 环境光晕
 ├─ 牌桌层    椭圆木框 + 毛毡绿 + 装饰线 + 位置标签
 ├─ 座位层    9 个 SeatSprite（0-8 号）
 │   ├─ 头像圆框（多层扩散光晕，本地/远端色调区分）
 │   ├─ 显示名 15px / 筹码 14px / 位置标签 12px
 │   ├─ 下注徽章 13px
 │   └─ 手牌（CardSprite × 2，右偏移 18px 避免与光晕重叠）
 ├─ 公共牌区  CardSprite × 5
 ├─ TimerArc  行动倒计时弧（绑定 deadlineTs）
 ├─ PotDisplay 底池金额
 ├─ 街道标签  当前街道名称
 └─ 庄家按钮  Dealer Button 图形
```

### 座位旋转

本地玩家始终旋转到底部（`displayIdx = 0`）：
- `server → display`：`(serverIdx - myServerSeat + 9) % 9`
- `display → server`：`(displayIdx + myServerSeat) % 9`
- `SEAT_POSITIONS[displayIdx]` 映射到画布坐标

### 数据驱动

- `useGameStore.subscribe` 驱动，状态变化时调用对应 `update*()` 方法
- `DealAnimation`：发牌动画（牌从庄家位置飞向各玩家）
- `ChipAnimation`：筹码动画（筹码从玩家位置飞向底池）

### 核心组件

| 组件 | 说明 |
|------|------|
| `CardSprite` | 单张牌，程序化绘制正面/背面，支持翻转动画 |
| `SeatSprite` | 座位，含头像/名字/筹码/手牌/下注标签/光晕/bot 标识（🤖 前缀 + 蓝色边框） |
| `PotDisplay` | 底池金额显示 |
| `TimerArc` | 圆弧倒计时，绑定 `deadlineTs` |
| `CasinoChip` | 筹码图形（用于 ChipAnimation） |

---

## 13. 前端 React 面板

### 页面路由

| 路径 | 页面 | 说明 |
|------|------|------|
| `/login` | `LoginPage` | 登录/注册表单 |
| `/lobby` | `LobbyPage` | 大厅，创建/加入房间 |
| `/room/:code` | `RoomPage` | 游戏房间（PixiJS canvas + React 覆盖层） |
| `/lab/*` | `LabPage` | 组件调试实验室（开发用） |

### 面板组件

| 组件 | 文件 | 说明 |
|------|------|------|
| `ActionPanel` | `panels/ActionPanel.tsx` | 行动按钮，仅 `isMyTurn` 时渲染 |
| `BetSlider` | `panels/BetSlider.tsx` | 下注滑块，含 1/4、1/2、3/4、pot 倍快捷预设 |
| `ChatPanel` | `panels/ChatPanel.tsx` | 聊天窗口（折叠/展开切换，最多 200 条历史，1s 限速） |
| `HandHistory` | `panels/HandHistory.tsx` | 历史手牌记录面板（按需展示，从 `/api/rooms/:code/hands` 拉取，赢家显示 display_name） |
| `RoomInfo` | `panels/RoomInfo.tsx` | 房间信息（盲注/买入范围/在线人数） |
| `RoundResultModal` | `panels/RoundResultModal.tsx` | 结算弹窗（多赢家逐行显示，首位赢家展示最佳五张；「开始下一局」发送 `ready` 命令并显示准备人数；满足条件时底部内嵌「补充筹码」滑块；z-index: z-[70]） |
| `ConnectionBanner` | `components/ConnectionBanner.tsx` | 断线提示横幅 |

### ActionPanel 行动逻辑

- `canCheck`：无欠注时显示过牌按钮
- `canCall`：有欠注时显示跟注按钮（附带金额）
- `canBet/canRaise`：决定显示「下注」还是「加注」
- 下注金额默认为 `minRaiseAmount`，范围 `[minBet, maxBet]`，支持底池比例快捷设置
- 全押始终显示

### RoomPage 功能

- **「邀请」按钮**：复制 `/join/:code` 链接到剪贴板，2 秒后恢复文字
- **「历史」按钮**：切换 `HandHistory` 面板显示/隐藏
- **聊天面板**：始终挂载，折叠状态下右下角显示消息徽标
- **破产弹窗**：检测 `gs.mySeat` 从非空变为空（非首次进入）时触发（z-index: z-[50]），提供「返回大厅」和「再次买入」两个选项；「再次买入」子界面含金额滑块，入座成功后自动关闭
- **「补充筹码」**：已合并至 `RoundResultModal`，不再在 `RoomPage` 独立弹窗

### 离桌逻辑（`RoomPage`）

1. 点击「离开牌桌」先发送 WS 命令 `leave_table`
2. 再跳转 `/lobby`（前端不等待服务端响应）
3. 手牌进行中服务端拒绝 `leave_table`（`hand_in_progress` 错误），不影响前端跳转

### LobbyPage 功能

- **进入时刷新余额**：`useEffect` 调用 `GET /api/me` 同步离桌后 cashOut 金额
- **领取免费筹码**：余额 < 1000 时头部显示「领取筹码」按钮，调用 `POST /api/chips/claim`
- **加入房间买入自选**：确认弹窗内含滑块，范围 `[min_buy_in, min(max_buy_in, balance)]`；选定值存入 `roomStore.selectedBuyIn`，WS 连接后发送 `join_room` 时附带
- **邀请链接处理**：路由 `/join/:code` 自动填充加入表单的房间码
- **BB 联动计算**：创建房间时监听 `bigBlind` 变化，自动将 `maxBuyIn` 设为 `100 × BB`、`minBuyIn` 设为 `20 × BB`

---

## 14. 数据库表结构

### `users`

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | BIGINT PK AUTO_INCREMENT | 用户 ID |
| `username` | VARCHAR UNIQUE | 登录名 |
| `password_hash` | VARCHAR | bcrypt 哈希 |
| `display_name` | VARCHAR | 显示名 |
| `chip_balance` | BIGINT | 账户筹码余额 |
| `created_at` | DATETIME | 注册时间 |

### `room_history`

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | BIGINT PK | 房间 ID |
| `code` | VARCHAR UNIQUE | 6 位房间码 |
| `host_user_id` | BIGINT | 创建者 ID |
| `config_json` | JSON | 房间配置快照 |
| `created_at` | DATETIME | 创建时间 |
| `ended_at` | DATETIME NULL | 关闭时间（GC 或 SIGTERM 后写入） |

### `hand_history`

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | BIGINT PK | 记录 ID |
| `room_id` | BIGINT | 房间 ID |
| `hand_num` | INT | 手牌编号 |
| `players_json` | JSON | 座位信息快照 |
| `actions_json` | JSON | 完整行动日志 |
| `result_json` | JSON | 赢家/金额信息 |
| `played_at` | DATETIME | 手牌完成时间 |

### `chip_ledger`

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | BIGINT PK | 记录 ID |
| `user_id` | BIGINT | 用户 ID |
| `delta` | BIGINT | 变动金额（正数=增加，负数=减少） |
| `reason` | VARCHAR | 原因（`buy_in` / `cash_out` / `add_chips` / `claim_free`） |
| `ref_id` | VARCHAR | 关联 ID（房间码） |
| `created_at` | DATETIME | 变动时间 |