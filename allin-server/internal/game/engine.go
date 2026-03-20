package game

import (
	"encoding/json"
	"log/slog"
	"time"

	"github.com/allin/server/internal/room"
	"github.com/allin/server/internal/ws"
)

const (
	handStartDelay = 3 * time.Second // 手牌之间的延迟
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
	botsSeated  bool // 首次有人类玩家加入时安排 bot 入座
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
	case "disconnect":
		e.handleDisconnect(msg, resetTimer, stopTimer)
	}
}

// ---- 加入 ----

func (e *Engine) handleJoinRoom(msg ws.InboundMessage, resetTimer func(time.Duration)) {
	// 如果已入座，忽略。
	if e.gs.FindPlayer(msg.SenderID) != nil {
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

	p := &Player{
		UserID:      msg.SenderID,
		DisplayName: msg.DisplayName,
		Stack:       e.room.Config.MaxBuyIn,
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

	e.hub.Broadcast(ws.MustEvent(ws.TypePlayerLeft, ws.PlayerLeftPayload{
		PlayerID:  p.UserID,
		SeatIndex: p.SeatIndex,
	}))

	if e.gs.Street == StreetIdle {
		e.gs.UnseatPlayer(msg.SenderID)
		e.room.Touch()

		// 统计剩余的人类玩家。
		humanCount := 0
		for _, sp := range e.gs.Seats {
			if sp != nil && !IsBotID(sp.UserID) {
				humanCount++
			}
		}
		if humanCount == 0 && e.onEmpty != nil {
			// 移除 bot 座位，以便人类玩家重新连接时房间重新开始。
			for _, sp := range e.gs.Seats {
				if sp != nil && IsBotID(sp.UserID) {
					e.gs.UnseatPlayer(sp.UserID)
				}
			}
			e.botsSeated = false
			// 宽限期：给玩家 30 秒时间重新连接，之后关闭房间。
			e.emptyTimer = time.AfterFunc(emptyGracePeriod, e.onEmpty)
		}
		return
	}

	// 在活跃手牌中：如果轮到他们则代为弃牌，否则标记为离座。
	if e.gs.ActionSeat == p.SeatIndex {
		ApplyAction(e.gs, p.UserID, ActionFold, 0)
		e.gs.UnseatPlayer(msg.SenderID)
		e.advanceOrEnd(resetTimer, stopTimer)
	} else {
		p.Folded = true
		e.gs.UnseatPlayer(msg.SenderID)
		e.checkHandOver(resetTimer, stopTimer)
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

	e.hub.Broadcast(ws.MustEvent(ws.TypePlayerJoined, ws.PlayerJoinedPayload{
		PlayerID:    p.UserID,
		DisplayName: p.DisplayName,
		SeatIndex:   p.SeatIndex,
		Stack:       p.Stack,
		IsReconnect: true,
	}))

	// 如果在活跃手牌中离座且轮到他们，自动弃牌。
	if cmd.SitOut && e.gs.Street != StreetIdle && e.gs.ActionSeat == p.SeatIndex {
		ApplyAction(e.gs, p.UserID, ActionFold, 0)
		stopTimer()
		e.advanceOrEnd(resetTimer, stopTimer)
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

	// 行动超时：自动弃牌或过牌。
	if e.gs.ActionSeat == -1 {
		return
	}
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
	e.advanceOrEnd(resetTimer, func() {})
}

// ---- 手牌流程 ----

func (e *Engine) startHand(resetTimer func(time.Duration)) {
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
	var reveals []reveal
	for _, p := range e.gs.Seats {
		if p != nil && !p.Folded && !p.SitOut {
			_, handName := EvaluateHand(p.Hole, e.gs.Community)
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
	}
	var winners []winEntry

	for _, pot := range pots {
		// 在有资格的玩家中找到最佳手牌。
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
		// 平局时平分底池。
		share := pot.Amount / int64(len(bestPlayers))
		remainder := pot.Amount % int64(len(bestPlayers))
		for i, uid := range bestPlayers {
			award := share
			if i == 0 {
				award += remainder // 将余数给第一个赢家
			}
			p := e.gs.FindPlayer(uid)
			if p != nil {
				p.Stack += award
			}
			winners = append(winners, winEntry{PlayerID: uid, Amount: award})
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

	rawResult, _ := json.Marshal(struct {
		Winners []winEntry   `json:"winners"`
		Seats   []resultSeat `json:"seats"`
	}{winners, resultSeats})
	e.hub.Broadcast(ws.MustEvent(ws.TypeHandResult, json.RawMessage(rawResult)))

	slog.Info("game: hand complete", "room", e.room.Code, "hand", e.gs.HandNum)

	// 安排下一手牌。
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

	e.hub.Broadcast(ws.MustEvent(ws.TypeHandResult, struct {
		Winners []struct {
			PlayerID string `json:"player_id"`
			Amount   int64  `json:"amount"`
		} `json:"winners"`
	}{
		Winners: []struct {
			PlayerID string `json:"player_id"`
			Amount   int64  `json:"amount"`
		}{{winner.UserID, total}},
	}))

	slog.Info("game: uncontested pot", "room", e.room.Code, "winner", winner.UserID, "amount", total)

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
	e.hub.SendTo(userID, ws.MustEvent(ws.TypeConnected, ws.ConnectedPayload{
		PlayerID:     userID,
		DisplayName:  e.hub.DisplayName(userID),
		RoomCode:     e.room.Code,
		GameSnapshot: &snap,
	}))
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
