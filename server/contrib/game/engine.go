package game

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	bizdao "github.com/allin/server/base/biz/dao"
	"github.com/allin/server/contrib/room"
	"github.com/allin/server/contrib/ws"
)

const (
	handStartDelay = 5 * time.Second // 手牌之间的延迟
)

const chatRateLimit = time.Second // 每个玩家聊天消息的最小间隔

const emptyGracePeriod = 30 * time.Second

// Engine 驱动一个房间的游戏状态机。
// 所有对 GameState 的修改都在单一的 Run() goroutine 中发生。
type Engine struct {
	hub         *ws.Hub
	room        *room.Room
	gs          *GameState
	deck        []Card
	quit        chan struct{}
	chatLimiter map[string]time.Time // 每个 userID 的上次聊天时间
	registry    *Registry
	onEmpty     func() // 所有玩家离开时调用
	emptyTimer  *time.Timer
	botsSeated  bool              // 首次有人类玩家加入时安排 bot 入座
	handActions []actionLogEntry  // 当前手牌行动序列（用于写入历史）
}

// actionLogEntry 记录单次行动，用于写入 hand_history.actions_json。
type actionLogEntry struct {
	PlayerID string `json:"player_id"`
	Action   string `json:"action"`
	Amount   int64  `json:"amount"`
	Street   string `json:"street"`
}

// NewEngine 为给定的 hub 和房间创建引擎。
func NewEngine(hub *ws.Hub, rm *room.Room, registry *Registry) *Engine {
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
		}
	}
}

func (e *Engine) handleMessage(
	msg ws.InboundMessage,
	resetTimer func(time.Duration),
	stopTimer func(),
) {
	switch msg.Env.Type {
	case ws.CmdJoinRoom:
		e.handleJoinRoom(msg, resetTimer)
	case ws.CmdAction:
		e.handleAction(msg, resetTimer, stopTimer)
	case ws.CmdChat:
		e.handleChat(msg)
	case ws.CmdAddChips:
		e.handleAddChips(msg)
	case ws.CmdSitOut:
		e.handleSitOut(msg, resetTimer, stopTimer)
	case ws.CmdLeaveTable:
		e.handleLeaveTable(msg)
	case ws.CmdDisconnect:
		e.handleDisconnect(msg, resetTimer, stopTimer)
	}
}

// ---- 加入 ----

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
		e.hub.SendTo(msg.SenderID, ws.MustEvent(ws.TypeError, ws.ErrorPayload{
			Code: "room_full", Message: "room is full",
		}))
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
		e.hub.SendTo(msg.SenderID, ws.MustEvent(ws.TypeError, ws.ErrorPayload{
			Code:    "invalid_buy_in",
			Message: fmt.Sprintf("buy_in must be between %d and %d", minBuyIn, maxBuyIn),
		}))
		return
	}

	// 从用户账户扣除买入金额（bot 跳过 DB 操作）
	senderIntID, parseErr := strconv.ParseInt(msg.SenderID, 10, 64)
	if parseErr != nil {
		e.hub.SendTo(msg.SenderID, ws.MustEvent(ws.TypeError, ws.ErrorPayload{
			Code: "user_not_found", Message: "user not found",
		}))
		return
	}
	u, err := bizdao.UserDao.GetByID(senderIntID)
	if err != nil {
		e.hub.SendTo(msg.SenderID, ws.MustEvent(ws.TypeError, ws.ErrorPayload{
			Code: "user_not_found", Message: "user not found",
		}))
		return
	}
	if u.ChipBalance < buyIn {
		e.hub.SendTo(msg.SenderID, ws.MustEvent(ws.TypeError, ws.ErrorPayload{
			Code: "insufficient_chips",
			Message: fmt.Sprintf("insufficient chips: need $%d, have $%d", buyIn, u.ChipBalance),
		}))
		return
	}
	if err := bizdao.UserDao.AdjustChips(senderIntID, -buyIn, "buy_in", e.room.Code); err != nil {
		slog.Error("game: failed to deduct buy-in", "user", msg.SenderID, "err", err)
		e.hub.SendTo(msg.SenderID, ws.MustEvent(ws.TypeError, ws.ErrorPayload{
			Code: "server_error", Message: "failed to process buy-in",
		}))
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

	// 活跃手牌中断线：保留座位，标记断线，自动弃牌。
	p.Disconnected = true
	if e.gs.ActionSeat == p.SeatIndex {
		ApplyAction(e.gs, p.UserID, ActionFold, 0)
		stopTimer()
		e.advanceOrEnd(resetTimer, stopTimer)
	} else {
		p.Folded = true
		e.checkHandOver(resetTimer, stopTimer)
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

func (e *Engine) handleAction(
	msg ws.InboundMessage,
	resetTimer func(time.Duration),
	stopTimer func(),
) {
	var cmd ws.ActionCmd
	if err := json.Unmarshal(msg.Env.Payload, &cmd); err != nil {
		e.sendError(msg.SenderID, "bad_payload", "invalid action payload", msg.Env.Seq)
		return
	}

	if err := ValidateAction(e.gs, msg.SenderID, Action(cmd.Action), cmd.Amount); err != nil {
		e.sendError(msg.SenderID, "invalid_action", err.Error(), msg.Env.Seq)
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
		PlayerID:  msg.SenderID,
		Action:    cmd.Action,
		Amount:    displayAmount,
		Stack:     p.Stack,
		TotalPot:  e.gs.TotalPot(),
	}))

	stopTimer()
	e.advanceOrEnd(resetTimer, stopTimer)
}

// ---- 聊天 ----

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

// ---- 加注筹码 ----

func (e *Engine) handleAddChips(msg ws.InboundMessage) {
	if e.gs.Street != StreetIdle {
		e.sendError(msg.SenderID, "hand_in_progress", "can only add chips between hands", msg.Env.Seq)
		return
	}
	var cmd ws.AddChipsCmd
	if err := json.Unmarshal(msg.Env.Payload, &cmd); err != nil {
		e.sendError(msg.SenderID, "bad_payload", "invalid add_chips payload", msg.Env.Seq)
		return
	}
	p := e.gs.FindPlayer(msg.SenderID)
	if p == nil {
		e.sendError(msg.SenderID, "not_seated", "you are not seated", msg.Env.Seq)
		return
	}
	if cmd.Amount <= 0 {
		e.sendError(msg.SenderID, "invalid_amount", "amount must be positive", msg.Env.Seq)
		return
	}
	newStack := p.Stack + cmd.Amount
	if newStack > e.gs.Config.MaxBuyIn {
		newStack = e.gs.Config.MaxBuyIn
	}
	added := newStack - p.Stack
	if added == 0 {
		return
	}

	// 从 DB 账户余额扣除（bot 跳过）。
	uid, err := strconv.ParseInt(msg.SenderID, 10, 64)
	if err != nil {
		e.sendError(msg.SenderID, "server_error", "invalid user id", msg.Env.Seq)
		return
	}
	u, err := bizdao.UserDao.GetByID(uid)
	if err != nil {
		e.sendError(msg.SenderID, "user_not_found", "user not found", msg.Env.Seq)
		return
	}
	if u.ChipBalance < added {
		e.sendError(msg.SenderID, "insufficient_chips",
			fmt.Sprintf("insufficient chips: need $%d, have $%d", added, u.ChipBalance), msg.Env.Seq)
		return
	}
	if err := bizdao.UserDao.AdjustChips(uid, -added, "add_chips", e.room.Code); err != nil {
		slog.Error("game: failed to deduct add_chips", "user", msg.SenderID, "err", err)
		e.sendError(msg.SenderID, "server_error", "failed to process add chips", msg.Env.Seq)
		return
	}

	p.Stack = newStack
	e.hub.Broadcast(ws.MustEvent(ws.TypeStackUpdated, ws.StackUpdatedPayload{
		PlayerID: p.UserID,
		Stack:    p.Stack,
		Delta:    added,
	}))
	slog.Info("game: chips added", "room", e.room.Code, "player", p.UserID, "added", added, "stack", p.Stack)
}

// ---- 离座 ----

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

func (e *Engine) handleLeaveTable(msg ws.InboundMessage) {
	if e.gs.Street != StreetIdle {
		e.sendError(msg.SenderID, "hand_in_progress", "can only leave between hands", msg.Env.Seq)
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

	e.hub.Broadcast(ws.MustEvent(ws.TypeActionTimeout, ws.ActionTimeoutPayload{
		PlayerID: p.UserID,
		Action:   string(action),
	}))

	ApplyAction(e.gs, p.UserID, action, 0)
	e.handActions = append(e.handActions, actionLogEntry{
		PlayerID: p.UserID,
		Action:   string(action),
		Amount:   0,
		Street:   e.gs.Street.String(),
	})
	e.advanceOrEnd(resetTimer, func() {})
}

// ---- 手牌流程 ----

func (e *Engine) startHand(resetTimer func(time.Duration)) {
	e.handActions = e.handActions[:0] // 清空行动日志
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

func (e *Engine) dealCommunity(n int) {
	for i := 0; i < n; i++ {
		e.gs.Community = append(e.gs.Community, e.deck[0])
		e.deck = e.deck[1:]
	}
}

// advanceOrEnd：行动后，确定下一步。
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

// CanAct 返回活跃且未全押的玩家。
func (gs *GameState) CanAct() []*Player {
	var out []*Player
	for _, p := range gs.Seats {
		if p != nil && !p.Folded && !p.SitOut && !p.AllIn {
			out = append(out, p)
		}
	}
	return out
}

func (e *Engine) broadcastActionRequired(resetTimer func(time.Duration)) {
	p := e.gs.Seats[e.gs.ActionSeat]
	if p == nil {
		return
	}
	deadline := time.Now().Add(time.Duration(e.gs.Config.ActionTimeSec) * time.Second)

	e.hub.Broadcast(ws.MustEvent(ws.TypeActionRequired, ws.ActionRequiredPayload{
		PlayerID:    p.UserID,
		SeatIndex:   p.SeatIndex,
		DeadlineTs:  deadline.UnixMilli(),
		CurrentBet:  e.gs.CurrentBet,
		CallAmount:  max64(0, e.gs.CurrentBet-p.Bet),
		MinRaise:    e.gs.CurrentBet + e.gs.MinRaise,
		Stack:       p.Stack,
		Pot:         e.gs.TotalPot(),
	}))

	if p.IsBot {
		e.scheduleAIAction(p)
	}

	resetTimer(time.Duration(e.gs.Config.ActionTimeSec) * time.Second)
}

// ---- 摊牌 ----

func (e *Engine) runShowdown(resetTimer func(time.Duration)) {
	e.gs.Street = StreetShowdown
	e.gs.ActionSeat = -1

	// 展示所有未弃牌的手牌。
	type reveal struct {
		PlayerID    string `json:"player_id"`
		SeatIndex   int    `json:"seat_index"`
		Hole        []string `json:"hole"`
		HandName    string `json:"hand_name"`
	}
	handNames := map[string]string{} // playerID → handName
	var reveals []reveal
	for _, p := range e.gs.Seats {
		if p != nil && !p.Folded && !p.SitOut {
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
		Winners    []winEntry     `json:"winners"`
		Seats      []resultSeat   `json:"seats"`
		BestHand   []string       `json:"best_hand,omitempty"`
		AllPlayers []playerDetail `json:"all_players"`
	}{winners, resultSeats, bestHand, allPlayers})
	e.hub.Broadcast(ws.MustEvent(ws.TypeHandResult, json.RawMessage(rawResult)))

	slog.Info("game: hand complete", "room", e.room.Code, "hand", e.gs.HandNum)

	e.saveHandHistory(json.RawMessage(rawResult))
	e.kickBrokePlayers()
	e.cleanupDisconnected()

	e.gs.Street = StreetIdle
	e.gs.ActionSeat = -1
	if len(e.gs.EligibleToStart()) >= 2 {
		resetTimer(handStartDelay)
	}
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
		Winners    []uWinner `json:"winners"`
		AllPlayers []uPlayer `json:"all_players"`
	}{
		Winners:    []uWinner{{winner.UserID, total}},
		AllPlayers: uPlayers,
	})
	e.hub.Broadcast(ws.MustEvent(ws.TypeHandResult, json.RawMessage(rawResult)))

	slog.Info("game: uncontested pot", "room", e.room.Code, "winner", winner.UserID, "amount", total)

	e.saveHandHistory(json.RawMessage(rawResult))
	// 移除筹码归零的玩家。
	e.kickBrokePlayers()
	e.cleanupDisconnected()

	e.gs.Street = StreetIdle
	e.gs.ActionSeat = -1
	if len(e.gs.EligibleToStart()) >= 2 {
		resetTimer(handStartDelay)
	}
}

// checkHandOver 检查断线弃牌后手牌是否结束。
func (e *Engine) checkHandOver(resetTimer func(time.Duration), stopTimer func()) {
	active := e.gs.ActivePlayers()
	if len(active) == 1 {
		e.awardUncontested(active[0], resetTimer)
	}
}

// kickBrokePlayers 移除手牌结束后筹码归零的玩家。
func (e *Engine) kickBrokePlayers() {
	for _, p := range e.gs.Seats {
		if p == nil || p.Stack > 0 {
			continue
		}
		uid := p.UserID
		seatIdx := p.SeatIndex
		e.gs.UnseatPlayer(uid)
		slog.Info("game: player broke, unseated", "room", e.room.Code, "player", uid)
		e.hub.Broadcast(ws.MustEvent(ws.TypePlayerLeft, ws.PlayerLeftPayload{
			PlayerID:  uid,
			SeatIndex: seatIdx,
		}))
	}
	e.maybeStartEmptyTimer()
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
		err := bizdao.HandHistoryDao.Save(bizdao.HandHistoryRecord{
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

func (e *Engine) sendError(userID, code, msg string, refSeq int64) {
	e.hub.SendTo(userID, ws.MustEvent(ws.TypeError, ws.ErrorPayload{
		Code: code, Message: msg, RefSeq: refSeq,
	}))
}

func max64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
