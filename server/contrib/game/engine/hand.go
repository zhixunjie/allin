package engine

import (
	"encoding/json"
	"log/slog"
	"time"

	botpkg "github.com/allin/server/contrib/game/bot"
	"github.com/allin/server/contrib/game/state"
	"github.com/allin/server/contrib/ws"
	"github.com/allin/server/contrib/ws/protocol"
)

// ---- 超时 ----

// handleTimeout 处理计时器到期事件，分三种情况：
//  1. Idle 状态：手牌间隔计时结束，若仍满足条件则开始新手牌。
//  2. ActionSeat == -1：全员全押，无人可行动，自动推进到下一街道。
//  3. 正常行动超时：对当前行动玩家执行自动弃牌（有下注则 fold，否则 check）。
func (e *Engine) handleTimeout(resetTimer func(time.Duration)) {
	if e.gs.Street == state.StreetIdle {
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

	action := state.ActionFold
	if p.Bet >= e.gs.CurrentBet {
		action = state.ActionCheck
	}

	e.gs.ApplyAction(p.UserID, action, 0)
	e.rc.Broadcast(protocol.MustEvent(protocol.TypeActionTimeout, protocol.ActionTimeoutPayload{
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

// ---- 行动 ----

// handleAction 处理玩家行动（fold/check/call/bet/raise/all_in）。
// 先校验合法性，通过后应用行动、记录日志、广播结果，再推进牌局。
func (e *Engine) handleAction(
	msg protocol.InboundMessage,
	resetTimer func(time.Duration),
	stopTimer func(),
) {
	var cmd protocol.ActionCmd
	if err := json.Unmarshal(msg.Env.Payload, &cmd); err != nil {
		e.sendError(msg.SenderID, ws.ErrBadPayload, msg.Env.Seq)
		return
	}

	if err := e.gs.ValidateAction(msg.SenderID, state.Action(cmd.Action), cmd.Amount); err != nil {
		e.sendError(msg.SenderID, ws.ErrInvalidAction, msg.Env.Seq, err.Error())
		return
	}

	e.gs.ApplyAction(msg.SenderID, state.Action(cmd.Action), cmd.Amount)

	p := e.gs.FindPlayer(msg.SenderID)
	var displayAmount int64
	if p != nil {
		displayAmount = p.Bet
	}

	e.handActions = append(e.handActions, actionLogEntry{
		PlayerID: msg.SenderID,
		Action:   cmd.Action,
		Amount:   cmd.Amount,
		Street:   e.gs.Street.String(),
	})

	e.rc.Broadcast(protocol.MustEvent(protocol.TypeActionTaken, protocol.ActionTakenPayload{
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
func (e *Engine) handleChat(msg protocol.InboundMessage) {
	if last, ok := e.chatLimiter[msg.SenderID]; ok && time.Since(last) < chatRateLimit {
		return
	}
	e.chatLimiter[msg.SenderID] = time.Now()

	var cmd protocol.ChatCmd
	if err := json.Unmarshal(msg.Env.Payload, &cmd); err != nil {
		return
	}
	if len(cmd.Text) == 0 || len(cmd.Text) > 200 {
		return
	}
	e.rc.Broadcast(protocol.MustEvent(protocol.TypeChatMessage, protocol.ChatPayload{
		SenderID:    msg.SenderID,
		DisplayName: msg.DisplayName,
		Text:        cmd.Text,
		Ts:          time.Now().UnixMilli(),
	}))
}

// ---- 手牌流程 ----

// startHand 开始新一手牌：递增手牌编号、移动庄家按钮、分配盲注、洗牌发牌，
// 并广播 game_started / hole_cards / cards_dealt 事件，最后提示第一个行动玩家。
func (e *Engine) startHand(resetTimer func(time.Duration)) {
	e.handActions = e.handActions[:0]
	e.readyPlayers = nil
	e.gs.HandNum++
	e.gs.Street = state.StreetPreFlop
	e.gs.Community = nil

	eligible := e.gs.EligibleToStart()

	e.gs.DealerSeat = e.nextEligibleSeatAfter(e.gs.DealerSeat, eligible)

	if len(eligible) == 2 {
		e.gs.SBSeat = e.gs.DealerSeat
		e.gs.BBSeat = e.nextEligibleSeatAfter(e.gs.DealerSeat, eligible)
	} else {
		e.gs.SBSeat = e.nextEligibleSeatAfter(e.gs.DealerSeat, eligible)
		e.gs.BBSeat = e.nextEligibleSeatAfter(e.gs.SBSeat, eligible)
	}

	for _, p := range e.gs.Seats {
		if p != nil {
			p.Bet = 0
			p.TotalBet = 0
			p.Folded = false
			p.AllIn = false
			p.ActedThisStreet = false
			p.Hole = [2]state.Card{}
		}
	}

	sb := e.gs.Seats[e.gs.SBSeat]
	bb := e.gs.Seats[e.gs.BBSeat]
	postBlind(sb, e.gs.Config.SmallBlind)
	postBlind(bb, e.gs.Config.BigBlind)

	e.gs.CurrentBet = e.gs.Config.BigBlind
	e.gs.MinRaise = e.gs.Config.BigBlind

	e.deck = state.NewShuffledDeck()
	e.dealHoleCards()

	e.rc.Broadcast(protocol.MustEvent(protocol.TypeGameStarted, protocol.GameStartedPayload{
		HandNum:    e.gs.HandNum,
		DealerSeat: e.gs.DealerSeat,
		SBSeat:     e.gs.SBSeat,
		BBSeat:     e.gs.BBSeat,
		SmallBlind: e.gs.Config.SmallBlind,
		BigBlind:   e.gs.Config.BigBlind,
	}))

	for _, p := range e.gs.Seats {
		if p == nil || p.SitOut {
			continue
		}
		e.rc.SendTo(p.UserID, protocol.MustEvent(protocol.TypeHoleCards, protocol.HoleCardsPayload{
			PlayerID: p.UserID,
			Hole:     []string{p.Hole[0].String(), p.Hole[1].String()},
		}))
	}

	var dealtSeats []int
	for _, p := range e.gs.Seats {
		if p != nil && !p.SitOut {
			dealtSeats = append(dealtSeats, p.SeatIndex)
		}
	}
	e.rc.Broadcast(protocol.MustEvent(protocol.TypeCardsDealt, protocol.CardsDealtPayload{Seats: dealtSeats}))

	var firstActor int
	if len(eligible) == 2 {
		firstActor = e.gs.DealerSeat
	} else {
		firstActor = e.nextEligibleSeatAfter(e.gs.BBSeat, eligible)
	}
	e.gs.ActionSeat = firstActor

	e.broadcastActionRequired(resetTimer)
}

// postBlind 强制玩家下盲注：若筹码不足则全押，更新 Bet / TotalBet / Stack。
func postBlind(p *state.Player, amount int64) {
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
func (e *Engine) dealCommunity(n int) {
	for i := 0; i < n; i++ {
		e.gs.Community = append(e.gs.Community, e.deck[0])
		e.deck = e.deck[1:]
	}
}

// advanceOrEnd 在玩家行动或超时弃牌后决定下一步。
func (e *Engine) advanceOrEnd(resetTimer func(time.Duration), stopTimer func()) {
	active := e.gs.ActivePlayers()

	if len(active) == 1 {
		e.awardUncontested(active[0], resetTimer)
		return
	}

	if e.gs.BettingRoundOver() {
		e.nextStreet(resetTimer)
		return
	}

	next := e.gs.NextActableSeat(e.gs.ActionSeat)
	if next == -1 {
		e.nextStreet(resetTimer)
		return
	}
	e.gs.ActionSeat = next
	e.broadcastActionRequired(resetTimer)
}

// nextStreet 将游戏推进到下一个下注街道。
func (e *Engine) nextStreet(resetTimer func(time.Duration)) {
	for _, p := range e.gs.Seats {
		if p != nil {
			p.Bet = 0
			p.ActedThisStreet = false
		}
	}
	e.gs.CurrentBet = 0
	e.gs.MinRaise = e.gs.Config.BigBlind

	switch e.gs.Street {
	case state.StreetPreFlop:
		e.gs.Street = state.StreetFlop
		e.dealCommunity(3)
	case state.StreetFlop:
		e.gs.Street = state.StreetTurn
		e.dealCommunity(1)
	case state.StreetTurn:
		e.gs.Street = state.StreetRiver
		e.dealCommunity(1)
	case state.StreetRiver:
		e.runShowdown(resetTimer)
		return
	}

	e.rc.Broadcast(protocol.MustEvent(protocol.TypeStreetStarted, protocol.StreetStartedPayload{
		Street:    e.gs.Street.String(),
		Community: state.CardsToStrings(e.gs.Community),
		Pot:       e.gs.TotalPot(),
	}))

	if len(e.gs.CanAct()) == 0 {
		e.gs.ActionSeat = -1
		resetTimer(2 * time.Second)
		return
	}

	eligible := e.gs.ActivePlayers()
	e.gs.ActionSeat = e.nextEligibleSeatAfter(e.gs.DealerSeat, eligible)
	e.broadcastActionRequired(resetTimer)
}

// broadcastActionRequired 广播 action_required 事件，并启动行动计时器。
// 若当前玩家是 bot，同步调度 AI 决策。
func (e *Engine) broadcastActionRequired(resetTimer func(time.Duration)) {
	p := e.gs.Seats[e.gs.ActionSeat]
	if p == nil {
		return
	}
	deadline := time.Now().Add(time.Duration(e.gs.Config.ActionTimeSec) * time.Second)

	e.rc.Broadcast(protocol.MustEvent(protocol.TypeActionRequired, protocol.ActionRequiredPayload{
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
		e.bot.ScheduleAction(e.gs, p)
	}

	resetTimer(time.Duration(e.gs.Config.ActionTimeSec) * time.Second)
}

// ---- 摊牌 ----

// runShowdown 执行摊牌流程：展示手牌、分配底池、广播结果、写入历史、启动下一手。
func (e *Engine) runShowdown(resetTimer func(time.Duration)) {
	e.gs.Street = state.StreetShowdown
	e.gs.ActionSeat = -1

	type reveal struct {
		PlayerID  string   `json:"player_id"`
		SeatIndex int      `json:"seat_index"`
		Hole      []string `json:"hole"`
		HandName  string   `json:"hand_name"`
	}
	handNames := map[string]string{}
	var reveals []reveal
	for _, p := range e.gs.Seats {
		if p != nil && !p.Folded && !p.SitOut && p.Hole[0].Rank == 0 {
			slog.Error("game: showdown player has no hole cards", "player", p.UserID, "seat", p.SeatIndex)
		}
		if p != nil && !p.Folded && !p.SitOut && p.Hole[0].Rank != 0 {
			_, handName := state.EvaluateHand(p.Hole, e.gs.Community)
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
	e.rc.Broadcast(protocol.MustEvent(protocol.TypeShowdown, json.RawMessage(rawReveals)))

	pots := state.BuildPots(e.gs.Seats)
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
			rank, _ := state.EvaluateHand(p.Hole, e.gs.Community)
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

	var bestHand []string
	if len(winners) > 0 {
		wp := e.gs.FindPlayer(winners[0].PlayerID)
		if wp != nil {
			bestHand = state.BestFiveStrings(wp.Hole, e.gs.Community)
		}
	}

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
	e.rc.Broadcast(protocol.MustEvent(protocol.TypeHandResult, json.RawMessage(rawResult)))

	slog.Info("game: hand complete", "room", e.room.Code, "hand", e.gs.HandNum)

	e.saveHandHistory(json.RawMessage(rawResult))
	e.kickBrokePlayers()
	e.cleanupDisconnected()

	e.gs.Street = state.StreetIdle
	e.gs.ActionSeat = -1
	e.scheduleNextHand(resetTimer)
}

// awardUncontested 将底池颁发给最后一个活跃玩家（其他人都弃牌了）。
func (e *Engine) awardUncontested(winner *state.Player, resetTimer func(time.Duration)) {
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
	e.rc.Broadcast(protocol.MustEvent(protocol.TypeHandResult, json.RawMessage(rawResult)))

	slog.Info("game: uncontested pot", "room", e.room.Code, "winner", winner.UserID, "amount", total)

	e.saveHandHistory(json.RawMessage(rawResult))
	e.kickBrokePlayers()
	e.cleanupDisconnected()

	e.gs.Street = state.StreetIdle
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

// ---- 准备系统 ----

// scheduleNextHand 在手牌结算后安排下一手的开始。
func (e *Engine) scheduleNextHand(resetTimer func(time.Duration)) {
	if len(e.gs.EligibleToStart()) < 2 {
		return
	}
	e.readyPlayers = make(map[string]bool)
	for _, p := range e.gs.Seats {
		if p != nil && botpkg.IsBotID(p.UserID) && !p.SitOut && !p.Disconnected && p.Stack > 0 {
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
func (e *Engine) handleReady(msg protocol.InboundMessage, resetTimer func(time.Duration)) {
	if e.gs.Street != state.StreetIdle {
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
	e.rc.Broadcast(protocol.MustEvent(protocol.TypeReadyStatus, struct {
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
