# contrib/game — 游戏引擎模块

德州扑克游戏引擎，按职责拆分为四个子包。

## 目录结构

```
contrib/game/
├── model/               # 纯数据类型（无游戏逻辑）
│   ├── street.go        # Street 类型及阶段常量（Idle/PreFlop/Flop/Turn/River/Showdown）
│   ├── card.go          # Card 结构体（Rank + Suit）
│   └── player.go        # Player 结构体（座位、筹码、底牌、状态标志）
│
├── state/               # 游戏状态机（数据 + 规则逻辑）
│   ├── types.go         # GameStateMachine、Pot；model 类型的 alias 导出
│   ├── state_machine.go # GameStateMachine 方法：入座/离座/查找/轮次推进
│   ├── action.go        # Action 类型；ValidateAction / ApplyAction 方法
│   ├── snapshot.go      # GameSnapshot / SeatSnapshot；Snapshot()、EvaluateHand()
│   ├── situation.go     # BotSituation（bot 决策所需局面快照）
│   ├── pot.go           # BuildPots — 边池计算
│   ├── deck.go          # NewShuffledDeck — 生成并洗牌
│   └── error.go         # 错误变量（ErrNotYourTurn 等）
│
├── bot/                 # AI 机器人
│   ├── bot.go           # Bot 结构体；ScheduleAction — 延迟注入行动
│   ├── bot_personality.go # BotStyle 类型；Personality 结构体；decide 决策逻辑
│   └── util.go          # GenUserID / GenUserName / IsBotID / AssignStyle
│
└── engine/              # 引擎事件循环（单 goroutine 状态机驱动）
    ├── engine.go        # Engine 结构体；NewEngine / Run / Stop
    ├── registry.go      # Registry — 多引擎生命周期管理（优雅关闭）
    ├── hand.go          # 手牌流程：startHand / nextStreet / runShowdown / 准备系统
    ├── seat.go          # 座位管理：加入 / 断线 / 离桌 / bot 入座 / 清场
    └── util.go          # sendSnapshot / sendError / saveHandHistory / nextEligibleSeatAfter
```

## 包依赖关系

```
model  ←  state  ←  bot  ←  engine
                              ↑
                         (ws / room / dao)
```

- **model**：无内部依赖，仅定义纯数据结构。
- **state**：导入 `model`（type alias）和 `room`（配置），包含全部规则逻辑。
- **bot**：导入 `state`（局面类型、行动常量）和 `ws`（注入行动消息），实现 AI 决策。
- **engine**：导入 `state`、`bot`、`ws`、`room`、`dao`，驱动完整的事件循环。

## 关键设计

| 方面 | 说明 |
|------|------|
| 并发安全 | Engine.Run() 运行在单一 goroutine，所有状态修改无需加锁 |
| Bot 行动 | goroutine 延迟 1–3 秒后向 `rc.Inbound` 写入行动消息，由 Run() 统一处理 |
| 边池计算 | `state.BuildPots` 按 TotalBet 升序迭代，自动拆分 all-in 边池 |
| 手牌评估 | `state.EvaluateHand` 枚举 C(7,5)=21 种五牌组合，返回可比较的 rank 值 |
| 买入/结算 | 加入时扣 `chip_balance`，离桌/断线时归还剩余 stack |
