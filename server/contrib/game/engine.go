package game

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	bizdao "github.com/allin/server/base/biz/dao"
	bizmodel "github.com/allin/server/base/biz/model"
	"github.com/allin/server/contrib/room"
	"github.com/allin/server/contrib/ws"
)

const (
	handStartDelay   = 10 * time.Second // 两手牌之间的等待时长，给玩家留出准备时间
	chatRateLimit    = time.Second      // 聊天消息发送频率上限：每人每秒最多 1 条
	emptyGracePeriod = 30 * time.Second // 所有人类玩家离开后，房间被回收前的宽限期
	botReplaceDelay  = 8 * time.Second  // bot 破产离桌后，等待真实玩家加入的宽限期；超时则补充新 bot
)

// Engine 驱动一个房间的游戏状态机。
// 所有对 GameState 的修改都严格发生在单一的 Run() goroutine 中，
// 无需任何锁即可保证并发安全。
type Engine struct {
	hub             *ws.RoomConn         // 该房间的 WebSocket 消息总线
	room            *room.Room           // 房间元数据与配置
	gs              *GameState           // 游戏状态（街道、座位、下注等）
	deck            []Card               // 当前手牌使用的洗牌牌组（每手重置）
	quit            chan struct{}        // 关闭此 channel 通知 Run() 退出
	chatLimiter     map[string]time.Time // 各玩家最近一次聊天时间，用于频率限制
	registry        *Registry            // 全局引擎注册表，为 nil 时不注册
	onEmpty         func()               // 所有人类玩家离开后触发的回调（用于回收房间）
	emptyTimer      *time.Timer          // 宽限期计时器，到期后执行 onEmpty
	botsSeated      bool                 // 标记 bot 是否已入座（首位人类玩家加入时触发一次）
	handActions     []actionLogEntry     // 当前手牌的行动序列，手牌结束后写入历史表
	botReplaceTimer *time.Timer          // bot 破产后等待补充的计时器
	botReplaceC     <-chan time.Time     // botReplaceTimer 对应的 channel，nil 时不触发
	readyPlayers    map[string]bool      // 在结算画面点击"开始下一局"的玩家集合
}

// actionLogEntry 记录单次行动，序列化后写入 hand_history.actions_json。
type actionLogEntry struct {
	PlayerID string `json:"player_id"` // 执行行动的玩家 ID
	Action   string `json:"action"`    // 行动类型（fold/check/call/bet/raise/all_in）
	Amount   int64  `json:"amount"`    // 行动金额（check/fold 为 0）
	Street   string `json:"street"`    // 行动所在街道（preflop/flop/turn/river）
}

// NewEngine 为给定的 RoomConn 和房间创建引擎。
func NewEngine(hub *ws.RoomConn, rm *room.Room, registry *Registry) *Engine {
	cfg := rm.Config
	if cfg.ActionTimeSec == 0 {
		cfg.ActionTimeSec = 30
	}
	e := &Engine{
		hub:         hub,
		room:        rm,
		chatLimiter: make(map[string]time.Time),
		gs: &GameState{
			Street:     StreetIdle,
			ActionSeat: -1,
			DealerSeat: -1,
			Config:     cfg,
		},
		quit:     make(chan struct{}),
		registry: registry,
	}
	if registry != nil {
		registry.track(e)
	}
	return e
}

// Stop 通知引擎 goroutine 退出。
func (e *Engine) Stop() { close(e.quit) }

// SetOnEmpty 注册最后一个玩家离开时调用的回调。
func (e *Engine) SetOnEmpty(fn func()) { e.onEmpty = fn }

// Run 是引擎的事件循环。应在专用 goroutine 中调用。
func (e *Engine) Run() {
	if e.registry != nil {
		defer e.registry.done(e)
	}
	var timerC <-chan time.Time
	var actionTimer *time.Timer

	resetTimer := func(d time.Duration) {
		if actionTimer != nil {
			actionTimer.Stop()
		}
		actionTimer = time.NewTimer(d)
		timerC = actionTimer.C
	}
	stopTimer := func() {
		if actionTimer != nil {
			actionTimer.Stop()
		}
		timerC = nil
	}

	for {
		select {
		case <-e.quit:
			stopTimer()
			return

		case msg := <-e.hub.Inbound:
			e.handleMessage(msg, resetTimer, stopTimer)

		case <-timerC:
			timerC = nil
			e.handleTimeout(resetTimer)

		case <-e.botReplaceC:
			// bot 破产宽限期结束：若仍有人类玩家且 bot 数量不足，补充新 bot
			e.botReplaceC = nil
			if e.hasHumanPlayers() {
				e.seatBots()
				if e.gs.Street == StreetIdle && len(e.gs.EligibleToStart()) >= 2 {
					resetTimer(handStartDelay)
				}
			}
		}
	}
}

// handleMessage 将入站消息按命令类型路由到对应处理函数。
func (e *Engine) handleMessage(
	msg ws.InboundMessage,
	resetTimer func(time.Duration),
	stopTimer func(),
) {
	switch msg.Env.Type {
	case ws.CmdJoinRoom: // 玩家加入或重连：分配座位、扣除买入、广播状态
		e.handleJoinRoom(msg, resetTimer)
	case ws.CmdAction: // 玩家行动（fold/check/call/bet/raise/all_in）：校验合法性、推进状态机
		e.handleAction(msg, resetTimer, stopTimer)
	case ws.CmdChat: // 聊天消息：限速后广播给房间内所有人
		e.handleChat(msg)
	case ws.CmdSitOut: // 离座/归座切换：更新 SitOut 标志，影响下一手参与资格
		e.handleSitOut(msg, resetTimer, stopTimer)
	case ws.CmdLeaveTable: // 主动离桌：仅限手牌间隙，归还筹码并移除座位
		e.handleLeaveTable(msg)
	case ws.CmdDisconnect: // 客户端优雅断开：手牌中保留座位并标记 Disconnected，间隙则直接离桌
		e.handleDisconnect(msg, resetTimer, stopTimer)
	case ws.CmdReady: // 结算画面点击"准备"：全员准备后提前 500ms 开始下一手
		e.handleReady(msg, resetTimer)
	}
}

// ---- 加入 ----

// handleJoinRoom 处理玩家加入房间请求。
// 若玩家已在座位（断线重连）则直接恢复状态；否则走完整买入入座流程。
func (e *Engine) handleJoinRoom(msg ws.InboundMessage, resetTimer func(time.Duration)) {
	// 断线重连：若玩家仍在座位（Disconnected=true），直接恢复。
	if existing := e.gs.FindPlayer(msg.SenderID); existing != nil {
		if existing.Disconnected {
			existing.Disconnected = false
			e.sendSnapshot(msg.SenderID)
			e.hub.Broadcast(ws.MustEvent(ws.TypePlayerJoined, ws.PlayerJoinedPayload{
				PlayerID:    existing.UserID,
				DisplayName: existing.DisplayName,
				SeatIndex:   existing.SeatIndex,
				Stack:       existing.Stack,
				IsReconnect: true,
			}))
		}
		return
	}
	e.room.Touch()
	// 当有玩家加入时取消待执行的房间关闭计时器。
	if e.emptyTimer != nil {
		e.emptyTimer.Stop()
		e.emptyTimer = nil
	}

	if e.gs.SeatedCount() >= e.room.Config.MaxPlayers {
		e.sendError(msg.SenderID, ws.ErrRoomFull, msg.Env.Seq)
		return
	}

	// 解析带入金额（0 表示默认使用 MaxBuyIn）。
	var cmd ws.JoinRoomCmd
	_ = json.Unmarshal(msg.Env.Payload, &cmd)
	buyIn := cmd.BuyIn
	if buyIn == 0 {
		buyIn = e.room.Config.MaxBuyIn
	}
	minBuyIn := e.room.Config.MinBuyIn
	maxBuyIn := e.room.Config.MaxBuyIn
	if buyIn < minBuyIn || buyIn > maxBuyIn {
		e.sendError(msg.SenderID, ws.ErrInvalidBuyIn, msg.Env.Seq,
			fmt.Sprintf("buy_in must be between %d and %d", minBuyIn, maxBuyIn))
		return
	}

	// 从用户账户扣除买入金额（bot 跳过 DB 操作）
	senderIntID, parseErr := strconv.ParseInt(msg.SenderID, 10, 64)
	if parseErr != nil {
		e.sendError(msg.SenderID, ws.ErrUserNotFound, msg.Env.Seq)
		return
	}
	u, err := bizdao.UserDao.GetByID(senderIntID)
	if err != nil {
		e.sendError(msg.SenderID, ws.ErrUserNotFound, msg.Env.Seq)
		return
	}
	if u.ChipBalance < buyIn {
		e.sendError(msg.SenderID, ws.ErrInsufficientChips, msg.Env.Seq,
			fmt.Sprintf("insufficient chips: need $%d, have $%d", buyIn, u.ChipBalance))
		return
	}
	if err := bizdao.UserDao.AdjustChips(senderIntID, -buyIn, "buy_in", e.room.Code); err != nil {
		slog.Error("game: failed to deduct buy-in", "user", msg.SenderID, "err", err)
		e.sendError(msg.SenderID, ws.ErrServerError, msg.Env.Seq, "failed to process buy-in")
		return
	}

	p := &Player{
		UserID:      msg.SenderID,
		DisplayName: msg.DisplayName,
		Stack:       buyIn,
	}
	e.gs.SeatPlayer(p)

	e.hub.Broadcast(ws.MustEvent(ws.TypePlayerJoined, ws.PlayerJoinedPayload{
		PlayerID:    p.UserID,
		DisplayName: p.DisplayName,
		SeatIndex:   p.SeatIndex,
		Stack:       p.Stack,
		IsReconnect: false,
	}))

	// 首次有人类玩家加入时安排 bot 入座。
	if !e.botsSeated {
		e.botsSeated = true
		e.seatBots()
	}

	// 向加入的玩家发送当前快照。
	e.sendSnapshot(msg.SenderID)

	// 如果有 ≥2 个合格玩家且没有正在进行的手牌，则自动开始。
	if e.gs.Street == StreetIdle && len(e.gs.EligibleToStart()) >= 2 {
		resetTimer(handStartDelay)
	}
}

// seatBots 将 AI 玩家安排到可用座位。
func (e *Engine) seatBots() {
	for i := 0; i < e.room.Config.BotCount; i++ {
		uid := botUserID(e.room.Code, i)
		if e.gs.FindPlayer(uid) != nil {
			continue // 已入座
		}
		p := &Player{
			UserID:      uid,
			DisplayName: botDisplayName(i),
			Stack:       e.room.Config.MaxBuyIn,
			IsBot:       true,
			BotStyle:    assignBotStyle(e.room.Config.BotStyle, i),
		}
		if !e.gs.SeatPlayer(p) {
			break // 没有更多座位
		}
		e.hub.Broadcast(ws.MustEvent(ws.TypePlayerJoined, ws.PlayerJoinedPayload{
			PlayerID:    p.UserID,
			DisplayName: p.DisplayName,
			SeatIndex:   p.SeatIndex,
			Stack:       p.Stack,
			IsBot:       true,
		}))
	}
}

// ---- 断开连接 ----

// handleDisconnect 处理玩家 WebSocket 断开事件。
// Idle 状态下立即离座并返还筹码；手牌进行中则保留座位标记断线，
// 仅在轮到该玩家行动时立即自动弃牌，其余情况等超时逻辑处理，保留重连机会。
func (e *Engine) handleDisconnect(msg ws.InboundMessage, resetTimer func(time.Duration), stopTimer func()) {
	// Bot ID 从没有真实的 WS 连接；忽略虚假的断开连接消息。
	if IsBotID(msg.SenderID) {
		return
	}

	p := e.gs.FindPlayer(msg.SenderID)
	if p == nil {
		return
	}

	if e.gs.Street == StreetIdle {
		// 手牌间隙断线：立即离座并返还筹码。
		e.hub.Broadcast(ws.MustEvent(ws.TypePlayerLeft, ws.PlayerLeftPayload{
			PlayerID:  p.UserID,
			SeatIndex: p.SeatIndex,
		}))
		stack := p.Stack
		e.gs.UnseatPlayer(msg.SenderID)
		e.cashOut(msg.SenderID, stack)
		e.room.Touch()
		e.maybeStartEmptyTimer()
		return
	}

	// 活跃手牌中断线：保留座位，标记断线。
	// 若正轮到该玩家行动则立即自动弃牌；否则仅标记断线，
	// 等轮到他时由超时逻辑处理，给断线重连留出时间。
	p.Disconnected = true
	if e.gs.ActionSeat == p.SeatIndex {
		ApplyAction(e.gs, p.UserID, ActionFold, 0)
		stopTimer()
		e.advanceOrEnd(resetTimer, stopTimer)
	}
}

// cashOut 将玩家剩余筹码返还到账户余额（bot 跳过）。
func (e *Engine) cashOut(userID string, stack int64) {
	if IsBotID(userID) || stack == 0 {
		return
	}
	uid, err := strconv.ParseInt(userID, 10, 64)
	if err != nil {
		slog.Error("game: cashOut invalid userID", "user", userID)
		return
	}
	if err := bizdao.UserDao.AdjustChips(uid, stack, "cash_out", e.room.Code); err != nil {
		slog.Error("game: failed to cash out", "user", userID, "stack", stack, "err", err)
	}
}

// ---- 行动 ----

// handleAction 处理玩家行动（fold/check/call/bet/raise/all_in）。
// 先校验合法性，通过后应用行动、记录日志、广播结果，再推进牌局。
func (e *Engine) handleAction(
	msg ws.InboundMessage,
	resetTimer func(time.Duration),
	stopTimer func(),
) {
	var cmd ws.ActionCmd
	if err := json.Unmarshal(msg.Env.Payload, &cmd); err != nil {
		e.sendError(msg.SenderID, ws.ErrBadPayload, msg.Env.Seq)
		return
	}

	if err := ValidateAction(e.gs, msg.SenderID, Action(cmd.Action), cmd.Amount); err != nil {
		e.sendError(msg.SenderID, ws.ErrInvalidAction, msg.Env.Seq, err.Error())
		return
	}

	ApplyAction(e.gs, msg.SenderID, Action(cmd.Action), cmd.Amount)

	p := e.gs.FindPlayer(msg.SenderID)
	var displayAmount int64
	if p != nil {
		displayAmount = p.Bet
	}

	// 记录到行动日志。
	e.handActions = append(e.handActions, actionLogEntry{
		PlayerID: msg.SenderID,
		Action:   cmd.Action,
		Amount:   cmd.Amount,
		Street:   e.gs.Street.String(),
	})

	e.hub.Broadcast(ws.MustEvent(ws.TypeActionTaken, ws.ActionTakenPayload{
		PlayerID: msg.SenderID,
		Action:   cmd.Action,
		Amount:   displayAmount,
		Stack:    p.Stack,
		TotalPot: e.gs.TotalPot(),
	}))

	stopTimer()
	e.advanceOrEnd(resetTimer, stopTimer)
}

// ---- 聊天 ----

// handleChat 处理聊天消息，并执行频率限制（每人每秒最多 1 条）。
func (e *Engine) handleChat(msg ws.InboundMessage) {
	// 频率限制：每个玩家每秒 1 条消息
	if last, ok := e.chatLimiter[msg.SenderID]; ok && time.Since(last) < chatRateLimit {
		return
	}
	e.chatLimiter[msg.SenderID] = time.Now()

	var cmd ws.ChatCmd
	if err := json.Unmarshal(msg.Env.Payload, &cmd); err != nil {
		return
	}
	if len(cmd.Text) == 0 || len(cmd.Text) > 200 {
		return
	}
	e.hub.Broadcast(ws.MustEvent(ws.TypeChatMessage, ws.ChatPayload{
		SenderID:    msg.SenderID,
		DisplayName: msg.DisplayName,
		Text:        cmd.Text,
		Ts:          time.Now().UnixMilli(),
	}))
}

// ---- 离座 ----

// handleSitOut 处理玩家离座/归座请求。
// 离座时若正轮到该玩家行动，自动弃牌；归座时若满足开局条件，启动倒计时。
func (e *Engine) handleSitOut(msg ws.InboundMessage, resetTimer func(time.Duration), stopTimer func()) {
	var cmd ws.SitOutCmd
	if err := json.Unmarshal(msg.Env.Payload, &cmd); err != nil {
		return
	}
	p := e.gs.FindPlayer(msg.SenderID)
	if p == nil {
		return
	}
	p.SitOut = cmd.SitOut

	e.hub.Broadcast(ws.MustEvent(ws.TypeSitOut, ws.SitOutPayload{
		PlayerID:  p.UserID,
		SeatIndex: p.SeatIndex,
		SitOut:    p.SitOut,
	}))

	// 离座：若在活跃手牌中且轮到该玩家，自动弃牌。
	if cmd.SitOut && e.gs.Street != StreetIdle && e.gs.ActionSeat == p.SeatIndex {
		ApplyAction(e.gs, p.UserID, ActionFold, 0)
		stopTimer()
		e.advanceOrEnd(resetTimer, stopTimer)
		return
	}
	// 归座：若处于空闲且满足开局条件，启动计时器。
	if !cmd.SitOut && e.gs.Street == StreetIdle && len(e.gs.EligibleToStart()) >= 2 {
		resetTimer(handStartDelay)
	}
}

// ---- 主动离桌 ----

// handleLeaveTable 处理玩家主动离桌请求。
// 只允许在 Idle 状态执行；移除玩家、返还筹码、广播离开事件。
func (e *Engine) handleLeaveTable(msg ws.InboundMessage) {
	if e.gs.Street != StreetIdle {
		e.sendError(msg.SenderID, ws.ErrHandInProgress, msg.Env.Seq)
		return
	}
	p := e.gs.FindPlayer(msg.SenderID)
	if p == nil {
		return
	}
	stack := p.Stack
	seatIdx := p.SeatIndex
	e.gs.UnseatPlayer(msg.SenderID)
	e.cashOut(msg.SenderID, stack)
	e.room.Touch()
	e.hub.Broadcast(ws.MustEvent(ws.TypePlayerLeft, ws.PlayerLeftPayload{
		PlayerID:  msg.SenderID,
		SeatIndex: seatIdx,
	}))
	e.maybeStartEmptyTimer()
}

// maybeStartEmptyTimer 在所有人类玩家离开时清场 bot 并启动宽限期。
func (e *Engine) maybeStartEmptyTimer() {
	humanCount := 0
	for _, sp := range e.gs.Seats {
		if sp != nil && !IsBotID(sp.UserID) {
			humanCount++
		}
	}
	if humanCount == 0 && e.onEmpty != nil {
		for _, sp := range e.gs.Seats {
			if sp != nil && IsBotID(sp.UserID) {
				e.gs.UnseatPlayer(sp.UserID)
			}
		}
		e.botsSeated = false
		if e.emptyTimer == nil {
			e.emptyTimer = time.AfterFunc(emptyGracePeriod, e.onEmpty)
		}
	}
}

// cleanupDisconnected 在手牌结束后移除所有仍处于断线状态的玩家并返还筹码。
func (e *Engine) cleanupDisconnected() {
	for _, p := range e.gs.Seats {
		if p == nil || !p.Disconnected {
			continue
		}
		uid := p.UserID
		seatIdx := p.SeatIndex
		stack := p.Stack
		e.gs.UnseatPlayer(uid)
		e.cashOut(uid, stack)
		e.hub.Broadcast(ws.MustEvent(ws.TypePlayerLeft, ws.PlayerLeftPayload{
			PlayerID:  uid,
			SeatIndex: seatIdx,
		}))
		slog.Info("game: disconnected player cleaned up", "room", e.room.Code, "player", uid)
	}
}

// ---- 超时 ----

// handleTimeout 处理计时器到期事件，分三种情况：
//  1. Idle 状态：手牌间隔计时结束，若仍满足条件则开始新手牌。
//  2. ActionSeat == -1：全员全押，无人可行动，自动推进到下一街道。
//  3. 正常行动超时：对当前行动玩家执行自动弃牌（有下注则 fold，否则 check）。
func (e *Engine) handleTimeout(resetTimer func(time.Duration)) {
	if e.gs.Street == StreetIdle {
		// 开始手牌计时器触发。
		if len(e.gs.EligibleToStart()) >= 2 {
			e.startHand(resetTimer)
		}
		return
	}

	// 全员全押自动推进：ActionSeat==-1 表示本街无人可行动，继续发牌。
	if e.gs.ActionSeat == -1 {
		e.nextStreet(resetTimer)
		return
	}

	// 行动超时：自动弃牌或过牌。
	p := e.gs.Seats[e.gs.ActionSeat]
	if p == nil {
		return
	}

	action := ActionFold
	if p.Bet >= e.gs.CurrentBet {
		action = ActionCheck
	}

	ApplyAction(e.gs, p.UserID, action, 0)
	e.hub.Broadcast(ws.MustEvent(ws.TypeActionTimeout, ws.ActionTimeoutPayload{
		PlayerID: p.UserID,
		Action:   string(action),
		Stack:    p.Stack,
		TotalPot: e.gs.TotalPot(),
	}))
	e.handActions = append(e.handActions, actionLogEntry{
		PlayerID: p.UserID,
		Action:   string(action),
		Amount:   0,
		Street:   e.gs.Street.String(),
	})
	e.advanceOrEnd(resetTimer, func() {})
}

// ---- 手牌流程 ----

// startHand 开始新一手牌：递增手牌编号、移动庄家按钮、分配盲注、洗牌发牌，
// 并广播 game_started / hole_cards / cards_dealt 事件，最后提示第一个行动玩家。
func (e *Engine) startHand(resetTimer func(time.Duration)) {
	e.handActions = e.handActions[:0] // 清空行动日志
	e.readyPlayers = nil              // 清空上一局的准备状态
	e.gs.HandNum++
	e.gs.Street = StreetPreFlop
	e.gs.Community = nil

	eligible := e.gs.EligibleToStart()

	// 移动庄家按钮。
	e.gs.DealerSeat = e.nextEligibleSeatAfter(e.gs.DealerSeat, eligible)

	// 分配小盲和大盲。
	if len(eligible) == 2 {
		// 单挑：庄家 = 小盲，对方 = 大盲。
		e.gs.SBSeat = e.gs.DealerSeat
		e.gs.BBSeat = e.nextEligibleSeatAfter(e.gs.DealerSeat, eligible)
	} else {
		e.gs.SBSeat = e.nextEligibleSeatAfter(e.gs.DealerSeat, eligible)
		e.gs.BBSeat = e.nextEligibleSeatAfter(e.gs.SBSeat, eligible)
	}

	// 重置玩家状态。
	for _, p := range e.gs.Seats {
		if p != nil {
			p.Bet = 0
			p.TotalBet = 0
			p.Folded = false
			p.AllIn = false
			p.ActedThisStreet = false
			p.Hole = [2]Card{}
		}
	}

	// 下盲注。
	sb := e.gs.Seats[e.gs.SBSeat]
	bb := e.gs.Seats[e.gs.BBSeat]

	postBlind(sb, e.gs.Config.SmallBlind)
	postBlind(bb, e.gs.Config.BigBlind)

	e.gs.CurrentBet = e.gs.Config.BigBlind
	e.gs.MinRaise = e.gs.Config.BigBlind

	// 洗牌并发牌。
	e.deck = newDeck()
	e.dealHoleCards()

	// 广播 game_started 事件。
	e.hub.Broadcast(ws.MustEvent(ws.TypeGameStarted, ws.GameStartedPayload{
		HandNum:    e.gs.HandNum,
		DealerSeat: e.gs.DealerSeat,
		SBSeat:     e.gs.SBSeat,
		BBSeat:     e.gs.BBSeat,
		SmallBlind: e.gs.Config.SmallBlind,
		BigBlind:   e.gs.Config.BigBlind,
	}))

	// 发送私密手牌。
	for _, p := range e.gs.Seats {
		if p == nil || p.SitOut {
			continue
		}
		e.hub.SendTo(p.UserID, ws.MustEvent(ws.TypeHoleCards, ws.HoleCardsPayload{
			PlayerID: p.UserID,
			Hole:     []string{p.Hole[0].String(), p.Hole[1].String()},
		}))
	}

	// 广播 cards_dealt（不透明 — 仅座位索引）。
	var dealtSeats []int
	for _, p := range e.gs.Seats {
		if p != nil && !p.SitOut {
			dealtSeats = append(dealtSeats, p.SeatIndex)
		}
	}
	e.hub.Broadcast(ws.MustEvent(ws.TypeCardsDealt, ws.CardsDealtPayload{Seats: dealtSeats}))

	// 翻牌前：行动从大盲左边开始。
	var firstActor int
	if len(eligible) == 2 {
		firstActor = e.gs.DealerSeat // 单挑：庄家（小盲）翻牌前先行动
	} else {
		firstActor = e.nextEligibleSeatAfter(e.gs.BBSeat, eligible)
	}
	e.gs.ActionSeat = firstActor

	e.broadcastActionRequired(resetTimer)
}

// postBlind 强制玩家下盲注：若筹码不足则全押，更新 Bet / TotalBet / Stack。
func postBlind(p *Player, amount int64) {
	if p == nil {
		return
	}
	if amount > p.Stack {
		amount = p.Stack
		p.AllIn = true
	}
	p.Bet += amount
	p.TotalBet += amount
	p.Stack -= amount
}

// dealHoleCards 按座位顺序循环两轮发牌，每位玩家发 2 张底牌。
// 已离座（SitOut）的玩家跳过，发完后收缩 deck 指针。
func (e *Engine) dealHoleCards() {
	idx := 0
	for round := 0; round < 2; round++ {
		for seat := 0; seat < 9; seat++ {
			p := e.gs.Seats[seat]
			if p == nil || p.SitOut {
				continue
			}
			p.Hole[round] = e.deck[idx]
			idx++
		}
	}
	e.deck = e.deck[idx:]
}

// dealCommunity 从牌组顶部取 n 张牌追加到公共牌区域。
// 翻牌发 3 张，转牌/河牌各发 1 张。
func (e *Engine) dealCommunity(n int) {
	for i := 0; i < n; i++ {
		e.gs.Community = append(e.gs.Community, e.deck[0])
		e.deck = e.deck[1:]
	}
}

// advanceOrEnd 在玩家行动或超时弃牌后决定下一步：
//   - 活跃玩家仅剩 1 人 → awardUncontested（无需摊牌）；
//   - 本回合下注已结束 → nextStreet 推进到下一街道；
//   - 所有剩余玩家全押 → nextStreet 继续发牌；
//   - 否则找到下一个可行动座位，广播 action_required。
func (e *Engine) advanceOrEnd(resetTimer func(time.Duration), stopTimer func()) {
	active := e.gs.ActivePlayers()

	// 只剩 1 个玩家 → 立即颁发底池。
	if len(active) == 1 {
		e.awardUncontested(active[0], resetTimer)
		return
	}

	// 下注回合结束？
	if e.gs.BettingRoundOver() {
		e.nextStreet(resetTimer)
		return
	}

	// 查找下一个可以行动的玩家。
	next := e.gs.nextActableSeat(e.gs.ActionSeat)
	if next == -1 {
		// 所有剩余玩家都已全押 → 发完公共牌。
		e.nextStreet(resetTimer)
		return
	}
	e.gs.ActionSeat = next
	e.broadcastActionRequired(resetTimer)
}

// nextStreet 将游戏推进到下一个下注街道：
// 重置玩家本回合下注，发公共牌（翻牌 3 张、转牌/河牌各 1 张），
// 广播 street_started 事件；若所有人全押则设置短延迟自动推进，
// 否则从庄家左边开始新一轮行动。河牌结束后调用 runShowdown。
func (e *Engine) nextStreet(resetTimer func(time.Duration)) {
	// 为新回合重置下注。
	for _, p := range e.gs.Seats {
		if p != nil {
			p.Bet = 0
			p.ActedThisStreet = false
		}
	}
	e.gs.CurrentBet = 0
	e.gs.MinRaise = e.gs.Config.BigBlind

	switch e.gs.Street {
	case StreetPreFlop:
		e.gs.Street = StreetFlop
		e.dealCommunity(3)
	case StreetFlop:
		e.gs.Street = StreetTurn
		e.dealCommunity(1)
	case StreetTurn:
		e.gs.Street = StreetRiver
		e.dealCommunity(1)
	case StreetRiver:
		e.runShowdown(resetTimer)
		return
	}

	e.hub.Broadcast(ws.MustEvent(ws.TypeStreetStarted, ws.StreetStartedPayload{
		Street:    e.gs.Street.String(),
		Community: cardsToStrings(e.gs.Community),
		Pot:       e.gs.TotalPot(),
	}))

	// 检查是否所有人都已全押（跳过行动，进入下一回合）。
	if len(e.gs.CanAct()) == 0 {
		e.gs.ActionSeat = -1
		resetTimer(2 * time.Second) // 短延迟后自动推进
		return
	}

	// 翻牌后：行动从庄家左边开始。
	eligible := e.gs.ActivePlayers()
	e.gs.ActionSeat = e.nextEligibleSeatAfter(e.gs.DealerSeat, eligible)
	e.broadcastActionRequired(resetTimer)
}

// CanAct 返回所有仍可参与下注决策的玩家：未弃牌、未离座、未全押。
// 用于判断当前街道是否还有行动空间，若为空则跳过直接发牌。
func (gs *GameState) CanAct() []*Player {
	var out []*Player
	for _, p := range gs.Seats {
		if p != nil && !p.Folded && !p.SitOut && !p.AllIn {
			out = append(out, p)
		}
	}
	return out
}

// broadcastActionRequired 广播 action_required 事件，告知所有客户端当前需要行动的玩家，
// 并启动行动计时器；若该玩家是 bot，则同步调度 AI 决策。
func (e *Engine) broadcastActionRequired(resetTimer func(time.Duration)) {
	p := e.gs.Seats[e.gs.ActionSeat]
	if p == nil {
		return
	}
	deadline := time.Now().Add(time.Duration(e.gs.Config.ActionTimeSec) * time.Second)

	e.hub.Broadcast(ws.MustEvent(ws.TypeActionRequired, ws.ActionRequiredPayload{
		PlayerID:   p.UserID,
		SeatIndex:  p.SeatIndex,
		DeadlineTs: deadline.UnixMilli(),
		CurrentBet: e.gs.CurrentBet,
		CallAmount: max64(0, e.gs.CurrentBet-p.Bet),
		MinRaise:   e.gs.CurrentBet + e.gs.MinRaise,
		Stack:      p.Stack,
		Pot:        e.gs.TotalPot(),
	}))

	if p.IsBot {
		e.scheduleAIAction(p)
	}

	resetTimer(time.Duration(e.gs.Config.ActionTimeSec) * time.Second)
}

// ---- 摊牌 ----

// runShowdown 执行摊牌流程：
//  1. 将所有未弃牌的手牌通过 showdown 事件公开；
//  2. 调用 BuildPots 计算主池/边池，按手牌强度分配给赢家（平分余数给第一位）；
//  3. 广播 hand_result，异步写入 DB，移除零筹码玩家，
//     进入 StreetIdle 并启动下一手计时器。
func (e *Engine) runShowdown(resetTimer func(time.Duration)) {
	e.gs.Street = StreetShowdown
	e.gs.ActionSeat = -1

	// 展示所有未弃牌的手牌。
	type reveal struct {
		PlayerID  string   `json:"player_id"`
		SeatIndex int      `json:"seat_index"`
		Hole      []string `json:"hole"`
		HandName  string   `json:"hand_name"`
	}
	handNames := map[string]string{} // playerID → handName
	var reveals []reveal
	for _, p := range e.gs.Seats {
		if p != nil && !p.Folded && !p.SitOut && p.Hole[0].Rank == 0 {
			slog.Error("game: showdown player has no hole cards", "player", p.UserID, "seat", p.SeatIndex)
		}
		if p != nil && !p.Folded && !p.SitOut && p.Hole[0].Rank != 0 {
			_, handName := EvaluateHand(p.Hole, e.gs.Community)
			handNames[p.UserID] = handName
			reveals = append(reveals, reveal{
				PlayerID:  p.UserID,
				SeatIndex: p.SeatIndex,
				Hole:      []string{p.Hole[0].String(), p.Hole[1].String()},
				HandName:  handName,
			})
		}
	}

	rawReveals, _ := json.Marshal(reveals)
	e.hub.Broadcast(ws.MustEvent(ws.TypeShowdown, json.RawMessage(rawReveals)))

	// 构建底池并颁发给赢家。
	pots := BuildPots(e.gs.Seats)
	type winEntry struct {
		PlayerID string `json:"player_id"`
		Amount   int64  `json:"amount"`
		HandName string `json:"hand_name,omitempty"`
	}
	var winners []winEntry

	for _, pot := range pots {
		bestRank := uint32(0xFFFFFFFF)
		var bestPlayers []string
		for _, uid := range pot.Eligible {
			p := e.gs.FindPlayer(uid)
			if p == nil || p.Folded {
				continue
			}
			rank, _ := EvaluateHand(p.Hole, e.gs.Community)
			if rank < bestRank {
				bestRank = rank
				bestPlayers = []string{uid}
			} else if rank == bestRank {
				bestPlayers = append(bestPlayers, uid)
			}
		}
		share := pot.Amount / int64(len(bestPlayers))
		remainder := pot.Amount % int64(len(bestPlayers))
		for i, uid := range bestPlayers {
			award := share
			if i == 0 {
				award += remainder
			}
			p := e.gs.FindPlayer(uid)
			if p != nil {
				p.Stack += award
			}
			winners = append(winners, winEntry{
				PlayerID: uid,
				Amount:   award,
				HandName: handNames[uid],
			})
		}
	}

	// 构建 hand_result 座位信息。
	type resultSeat struct {
		PlayerID string `json:"player_id"`
		Stack    int64  `json:"stack"`
	}
	var resultSeats []resultSeat
	for _, p := range e.gs.Seats {
		if p != nil {
			resultSeats = append(resultSeats, resultSeat{p.UserID, p.Stack})
		}
	}

	// 赢家最佳五张牌。
	var bestHand []string
	if len(winners) > 0 {
		wp := e.gs.FindPlayer(winners[0].PlayerID)
		if wp != nil {
			bestHand = BestFiveStrings(wp.Hole, e.gs.Community)
		}
	}

	// 全场玩家摘要（含弃牌者）。
	type playerDetail struct {
		PlayerID    string   `json:"player_id"`
		DisplayName string   `json:"display_name"`
		Hole        []string `json:"hole"`
		HandName    string   `json:"hand_name,omitempty"`
		Folded      bool     `json:"folded"`
		IsWinner    bool     `json:"is_winner"`
	}
	winnerSet := map[string]bool{}
	for _, w := range winners {
		winnerSet[w.PlayerID] = true
	}
	var allPlayers []playerDetail
	for _, p := range e.gs.Seats {
		if p == nil {
			continue
		}
		hole := []string{}
		if !p.Folded {
			hole = []string{p.Hole[0].String(), p.Hole[1].String()}
		}
		allPlayers = append(allPlayers, playerDetail{
			PlayerID:    p.UserID,
			DisplayName: p.DisplayName,
			Hole:        hole,
			HandName:    handNames[p.UserID],
			Folded:      p.Folded,
			IsWinner:    winnerSet[p.UserID],
		})
	}

	rawResult, _ := json.Marshal(struct {
		Winners          []winEntry     `json:"winners"`
		Seats            []resultSeat   `json:"seats"`
		BestHand         []string       `json:"best_hand,omitempty"`
		AllPlayers       []playerDetail `json:"all_players"`
		NextHandDelaySec int            `json:"next_hand_delay_sec"`
	}{winners, resultSeats, bestHand, allPlayers, int(handStartDelay.Seconds())})
	e.hub.Broadcast(ws.MustEvent(ws.TypeHandResult, json.RawMessage(rawResult)))

	slog.Info("game: hand complete", "room", e.room.Code, "hand", e.gs.HandNum)

	e.saveHandHistory(json.RawMessage(rawResult))
	e.kickBrokePlayers()
	e.cleanupDisconnected()

	e.gs.Street = StreetIdle
	e.gs.ActionSeat = -1
	e.scheduleNextHand(resetTimer)
}

// awardUncontested 将底池颁发给最后一个活跃玩家（其他人都弃牌了）。
func (e *Engine) awardUncontested(winner *Player, resetTimer func(time.Duration)) {
	// 如果赢家的下注超过第二高下注，退回未被跟注的部分。
	maxOther := int64(0)
	for _, p := range e.gs.Seats {
		if p != nil && p.UserID != winner.UserID && p.TotalBet > maxOther {
			maxOther = p.TotalBet
		}
	}
	uncalled := winner.TotalBet - maxOther
	if uncalled > 0 {
		winner.Stack += uncalled
		winner.TotalBet -= uncalled
	}

	total := int64(0)
	for _, p := range e.gs.Seats {
		if p != nil {
			total += p.TotalBet
		}
	}
	winner.Stack += total

	type uWinner struct {
		PlayerID string `json:"player_id"`
		Amount   int64  `json:"amount"`
	}
	type uPlayer struct {
		PlayerID    string   `json:"player_id"`
		DisplayName string   `json:"display_name"`
		Hole        []string `json:"hole"`
		Folded      bool     `json:"folded"`
		IsWinner    bool     `json:"is_winner"`
	}
	var uPlayers []uPlayer
	for _, p := range e.gs.Seats {
		if p == nil {
			continue
		}
		uPlayers = append(uPlayers, uPlayer{
			PlayerID:    p.UserID,
			DisplayName: p.DisplayName,
			Hole:        []string{},
			Folded:      p.Folded || p.UserID != winner.UserID,
			IsWinner:    p.UserID == winner.UserID,
		})
	}
	rawResult, _ := json.Marshal(struct {
		Winners          []uWinner `json:"winners"`
		AllPlayers       []uPlayer `json:"all_players"`
		NextHandDelaySec int       `json:"next_hand_delay_sec"`
	}{
		Winners:          []uWinner{{winner.UserID, total}},
		AllPlayers:       uPlayers,
		NextHandDelaySec: int(handStartDelay.Seconds()),
	})
	e.hub.Broadcast(ws.MustEvent(ws.TypeHandResult, json.RawMessage(rawResult)))

	slog.Info("game: uncontested pot", "room", e.room.Code, "winner", winner.UserID, "amount", total)

	e.saveHandHistory(json.RawMessage(rawResult))
	// 移除筹码归零的玩家。
	e.kickBrokePlayers()
	e.cleanupDisconnected()

	e.gs.Street = StreetIdle
	e.gs.ActionSeat = -1
	e.scheduleNextHand(resetTimer)
}

// checkHandOver 检查断线弃牌后手牌是否结束。
func (e *Engine) checkHandOver(resetTimer func(time.Duration), stopTimer func()) {
	active := e.gs.ActivePlayers()
	if len(active) == 1 {
		e.awardUncontested(active[0], resetTimer)
	}
}

// kickBrokePlayers 移除手牌结束后筹码归零的玩家。
// 若有 bot 被踢出，启动 botReplaceDelay 计时器；
// 计时器到期时若仍有人类玩家，则补充新 bot（见 Run() botReplaceC case）。
func (e *Engine) kickBrokePlayers() {
	brokeBotKicked := false
	for _, p := range e.gs.Seats {
		if p == nil || p.Stack > 0 {
			continue
		}
		uid := p.UserID
		seatIdx := p.SeatIndex
		if p.IsBot {
			brokeBotKicked = true
		}
		e.gs.UnseatPlayer(uid)
		slog.Info("game: player broke, unseated", "room", e.room.Code, "player", uid)
		e.hub.Broadcast(ws.MustEvent(ws.TypePlayerLeft, ws.PlayerLeftPayload{
			PlayerID:  uid,
			SeatIndex: seatIdx,
		}))
	}
	e.maybeStartEmptyTimer()

	// bot 破产后启动宽限期计时器，等待真实玩家加入
	if brokeBotKicked && e.hasHumanPlayers() {
		if e.botReplaceTimer != nil {
			e.botReplaceTimer.Stop()
		}
		t := time.NewTimer(botReplaceDelay)
		e.botReplaceTimer = t
		e.botReplaceC = t.C
	}
}

// hasHumanPlayers 返回牌桌上是否还有至少一名真实（非 bot）玩家。
func (e *Engine) hasHumanPlayers() bool {
	for _, p := range e.gs.Seats {
		if p != nil && !IsBotID(p.UserID) {
			return true
		}
	}
	return false
}

// ---- 准备系统 ----

// scheduleNextHand 在手牌结算后安排下一手的开始。
// 若合格玩家 ≥2，广播初始准备状态并启动 handStartDelay 计时器。
// Bot 玩家被视为自动准备好，若此时所有合格玩家都已准备，则缩短到 500ms 直接开始。
func (e *Engine) scheduleNextHand(resetTimer func(time.Duration)) {
	if len(e.gs.EligibleToStart()) < 2 {
		return
	}
	// 重置准备集合，并为所有 bot 自动标记准备。
	e.readyPlayers = make(map[string]bool)
	for _, p := range e.gs.Seats {
		if p != nil && IsBotID(p.UserID) && !p.SitOut && !p.Disconnected && p.Stack > 0 {
			e.readyPlayers[p.UserID] = true
		}
	}
	e.broadcastReadyStatus()
	if e.allEligibleReady() {
		resetTimer(500 * time.Millisecond)
	} else {
		resetTimer(handStartDelay)
	}
}

// handleReady 处理玩家发送的 ready 命令（在结算画面点击"开始下一局"）。
func (e *Engine) handleReady(msg ws.InboundMessage, resetTimer func(time.Duration)) {
	if e.gs.Street != StreetIdle {
		return
	}
	if e.readyPlayers == nil {
		e.readyPlayers = make(map[string]bool)
	}
	e.readyPlayers[msg.SenderID] = true
	e.broadcastReadyStatus()
	if e.allEligibleReady() {
		resetTimer(500 * time.Millisecond)
	}
}

// broadcastReadyStatus 广播当前准备人数给所有客户端。
func (e *Engine) broadcastReadyStatus() {
	eligible := e.gs.EligibleToStart()
	readyCount := 0
	for _, p := range eligible {
		if e.readyPlayers[p.UserID] {
			readyCount++
		}
	}
	e.hub.Broadcast(ws.MustEvent(ws.TypeReadyStatus, struct {
		ReadyCount int `json:"ready_count"`
		TotalCount int `json:"total_count"`
	}{readyCount, len(eligible)}))
}

// allEligibleReady 返回是否所有合格玩家都已准备。
func (e *Engine) allEligibleReady() bool {
	eligible := e.gs.EligibleToStart()
	if len(eligible) < 2 {
		return false
	}
	for _, p := range eligible {
		if !e.readyPlayers[p.UserID] {
			return false
		}
	}
	return true
}

// saveHandHistory 异步将手牌结果写入 DB，不阻塞引擎 goroutine。
func (e *Engine) saveHandHistory(resultJSON json.RawMessage) {
	type playerSnap struct {
		PlayerID    string `json:"player_id"`
		DisplayName string `json:"display_name"`
		SeatIndex   int    `json:"seat_index"`
		Stack       int64  `json:"stack"`
	}
	var players []playerSnap
	for _, p := range e.gs.Seats {
		if p != nil {
			players = append(players, playerSnap{
				PlayerID:    p.UserID,
				DisplayName: p.DisplayName,
				SeatIndex:   p.SeatIndex,
				Stack:       p.Stack,
			})
		}
	}
	playersJSON, _ := json.Marshal(players)
	actionsJSON, _ := json.Marshal(e.handActions)
	roomID := e.room.ID
	handNum := e.gs.HandNum

	go func() {
		err := bizdao.HandHistoryDao.Save(bizmodel.HandHistoryRecord{
			RoomID:      roomID,
			HandNum:     handNum,
			PlayersJSON: playersJSON,
			ActionsJSON: actionsJSON,
			ResultJSON:  resultJSON,
			PlayedAt:    time.Now(),
		})
		if err != nil {
			slog.Error("game: failed to save hand history", "room", e.room.Code, "hand", handNum, "err", err)
		}
	}()
}

// ---- 辅助函数 ----

// nextEligibleSeatAfter 在 eligible 玩家列表中，找到座位号严格大于 from 的下一个座位（循环）。
// from == -1 时直接返回第一个 eligible 玩家的座位；用于庄家/盲注/行动顺序推进。
func (e *Engine) nextEligibleSeatAfter(from int, eligible []*Player) int {
	if from == -1 {
		if len(eligible) > 0 {
			return eligible[0].SeatIndex
		}
		return 0
	}
	for i := 1; i <= 9; i++ {
		idx := (from + i) % 9
		for _, p := range eligible {
			if p.SeatIndex == idx {
				return idx
			}
		}
	}
	return eligible[0].SeatIndex
}

// sendSnapshot 向指定玩家发送 connected 事件，包含当前完整游戏快照和账户余额。
// 重连或首次加入时调用，让客户端恢复到正确的 UI 状态。
func (e *Engine) sendSnapshot(userID string) {
	snap := e.gs.Snapshot(userID)
	payload := ws.ConnectedPayload{
		PlayerID:     userID,
		DisplayName:  e.hub.DisplayName(userID),
		RoomCode:     e.room.Code,
		GameSnapshot: &snap,
	}
	// 查询账户余额（bot 无 int64 ID，跳过）
	if uid, err := strconv.ParseInt(userID, 10, 64); err == nil {
		if u, err := bizdao.UserDao.GetByID(uid); err == nil {
			payload.ChipBalance = u.ChipBalance
		}
	}
	e.hub.SendTo(userID, ws.MustEvent(ws.TypeConnected, payload))
}

// sendError 向指定玩家发送错误事件。
// msgOverride 可选，不传则使用 ErrCode 的默认描述。
func (e *Engine) sendError(userID string, code ws.ErrCode, refSeq int64, msgOverride ...string) {
	msg := code.Message()
	if len(msgOverride) > 0 && msgOverride[0] != "" {
		msg = msgOverride[0]
	}
	e.hub.SendTo(userID, ws.MustEvent(ws.TypeError, ws.ErrorPayload{
		Code: code, Message: msg, RefSeq: refSeq,
	}))
}

// max64 返回两个 int64 中的较大值。
func max64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
