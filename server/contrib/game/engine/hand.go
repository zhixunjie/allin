package engine

import (
	"encoding/json"
	"log/slog"
	"time"

	botpkg "github.com/allin/server/contrib/game/bot"
	"github.com/allin/server/contrib/game/state"
	"github.com/allin/server/contrib/ws/protocol"
	"github.com/allin/server/gmodel"
)

// ---- 超时 ----

// handleTimeout 处理计时器到期事件，分三种情况：
//  1. Idle 状态：手牌间隔计时结束，若仍满足条件则开始新手牌。
//  2. ActionSeat == -1：全员全押，无人可行动，自动推进到下一街道。
//  3. 正常行动超时：对当前行动玩家执行自动弃牌（有下注则 fold，否则 check）。
func (e *Engine) handleTimeout() {
	if e.gs.Street == gmodel.StreetIdle {
		if len(e.gs.EligibleToStart()) >= 2 {
			e.startHand()
		}
		return
	}

	// 全员全押自动推进：ActionSeat==-1 表示本街无人可行动，继续发牌。
	if e.gs.ActionSeat == gmodel.NoSeat {
		e.nextStreet()
		return
	}

	// 行动超时：自动弃牌或过牌。
	p := e.gs.Seats[e.gs.ActionSeat]
	if p == nil {
		return
	}

	action := gmodel.ActionFold
	if p.Bet >= e.gs.CurrentBet {
		action = gmodel.ActionCheck
	}

	e.gs.ApplyAction(p.UserID, action, 0)
	e.rc.Broadcast(protocol.MustNewEnvelope(protocol.TypeActionTimeout, protocol.ActionTimeoutPayload{
		PlayerID: p.UserID,
		Action:   action,
		Stack:    p.Stack,
		TotalPot: e.gs.TotalPot(),
	}))
	e.handActions = append(e.handActions, actionLogEntry{
		PlayerID: p.UserID,
		Action:   action,
		Amount:   0,
		Street:   e.gs.Street.String(),
	})
	e.advanceOrEnd()
}

// ---- 行动 ----

// handleAction 处理玩家行动（fold/check/call/bet/raise/all_in）。
// 先校验合法性，通过后应用行动、记录日志、广播结果，再推进牌局。
func (e *Engine) handleAction(msg protocol.InboundMessage) {
	var cmd protocol.ActionCmd
	if err := json.Unmarshal(msg.Env.Payload, &cmd); err != nil {
		e.sendError(msg.SenderID, protocol.SkErrBadPayload, msg.Env.Seq)
		return
	}

	if err := e.gs.ValidateAction(msg.SenderID, cmd.Action, cmd.Amount); err != nil {
		e.sendError(msg.SenderID, protocol.SkErrInvalidAction, msg.Env.Seq, err.Error())
		return
	}

	e.gs.ApplyAction(msg.SenderID, cmd.Action, cmd.Amount)

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

	e.rc.Broadcast(protocol.MustNewEnvelope(protocol.TypeActionTaken, protocol.ActionTakenPayload{
		PlayerID: msg.SenderID,
		Action:   cmd.Action,
		Amount:   displayAmount,
		Stack:    p.Stack,
		TotalPot: e.gs.TotalPot(),
	}))

	e.stopTimer()
	e.advanceOrEnd()
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
	e.rc.Broadcast(protocol.MustNewEnvelope(protocol.TypeChatMessage, protocol.ChatPayload{
		SenderID:    msg.SenderID,
		DisplayName: msg.DisplayName,
		Text:        cmd.Text,
		Ts:          time.Now().UnixMilli(),
	}))
}

// ---- 手牌流程 ----

// startHand 开始新一手牌：递增手牌编号、移动庄家按钮、分配盲注、洗牌发牌，
// 并广播 game_started / hole_cards / cards_dealt 事件，最后提示第一个行动玩家。
func (e *Engine) startHand() {
	e.handActions = e.handActions[:0]
	e.readyPlayers = nil
	e.gs.HandNum++
	e.gs.Street = gmodel.StreetPreFlop
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
			p.Hole = [2]gmodel.Card{}
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

	e.rc.Broadcast(protocol.MustNewEnvelope(protocol.TypeGameStarted, protocol.GameStartedPayload{
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
		e.rc.SendTo(p.UserID, protocol.MustNewEnvelope(protocol.TypeHoleCards, protocol.HoleCardsPayload{
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
	e.rc.Broadcast(protocol.MustNewEnvelope(protocol.TypeCardsDealt, protocol.CardsDealtPayload{Seats: dealtSeats}))

	var firstActor int
	if len(eligible) == 2 {
		firstActor = e.gs.DealerSeat
	} else {
		firstActor = e.nextEligibleSeatAfter(e.gs.BBSeat, eligible)
	}
	e.gs.ActionSeat = firstActor

	e.broadcastActionRequired()
}

// postBlind 强制玩家下盲注：若筹码不足则全押，更新 Bet / TotalBet / Stack。
func postBlind(p *gmodel.Player, amount int64) {
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
func (e *Engine) advanceOrEnd() {
	active := e.gs.ActivePlayers()

	if len(active) == 1 {
		e.awardUncontested(active[0])
		return
	}

	if e.gs.BettingRoundOver() {
		e.nextStreet()
		return
	}

	next := e.gs.NextActableSeat(e.gs.ActionSeat)
	if next == gmodel.NoSeat {
		e.nextStreet()
		return
	}
	e.gs.ActionSeat = next
	e.broadcastActionRequired()
}

// nextStreet 将游戏推进到下一个下注街道。
func (e *Engine) nextStreet() {
	for _, p := range e.gs.Seats {
		if p != nil {
			p.Bet = 0
			p.ActedThisStreet = false
		}
	}
	e.gs.CurrentBet = 0
	e.gs.MinRaise = e.gs.Config.BigBlind

	switch e.gs.Street {
	case gmodel.StreetPreFlop:
		e.gs.Street = gmodel.StreetFlop
		e.dealCommunity(3)
	case gmodel.StreetFlop:
		e.gs.Street = gmodel.StreetTurn
		e.dealCommunity(1)
	case gmodel.StreetTurn:
		e.gs.Street = gmodel.StreetRiver
		e.dealCommunity(1)
	case gmodel.StreetRiver:
		e.runShowdown()
		return
	}

	e.rc.Broadcast(protocol.MustNewEnvelope(protocol.TypeStreetStarted, protocol.StreetStartedPayload{
		Street:    e.gs.Street.String(),
		Community: state.CardsToStrings(e.gs.Community),
		Pot:       e.gs.TotalPot(),
	}))

	if len(e.gs.CanAct()) == 0 {
		e.gs.ActionSeat = gmodel.NoSeat
		e.resetTimer(2 * time.Second)
		return
	}

	eligible := e.gs.ActivePlayers()
	e.gs.ActionSeat = e.nextEligibleSeatAfter(e.gs.DealerSeat, eligible)
	e.broadcastActionRequired()
}

// broadcastActionRequired 广播 action_required 事件，并启动行动计时器。
// 若当前玩家是 bot，同步调度 AI 决策。
func (e *Engine) broadcastActionRequired() {
	p := e.gs.Seats[e.gs.ActionSeat]
	if p == nil {
		return
	}
	deadline := time.Now().Add(time.Duration(e.gs.Config.ActionTimeSec) * time.Second)

	e.rc.Broadcast(protocol.MustNewEnvelope(protocol.TypeActionRequired, protocol.ActionRequiredPayload{
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

	e.resetTimer(time.Duration(e.gs.Config.ActionTimeSec) * time.Second)
}

// ---- 摊牌 ----

// runShowdown 执行摊牌流程：展示手牌、分配底池、广播结果、写入历史、启动下一手。
func (e *Engine) runShowdown() {
	e.gs.Street = gmodel.StreetShowdown
	e.gs.ActionSeat = gmodel.NoSeat

	handNames := map[string]string{}
	var reveals []revealEntry
	for _, p := range e.gs.Seats {
		if p != nil && !p.Folded && !p.SitOut && p.Hole[0].Rank == 0 {
			slog.Error("game: showdown player has no hole cards", "player", p.UserID, "seat", p.SeatIndex)
		}
		if p != nil && !p.Folded && !p.SitOut && p.Hole[0].Rank != 0 {
			_, handName := state.EvaluateHand(p.Hole, e.gs.Community)
			handNames[p.UserID] = handName
			reveals = append(reveals, revealEntry{
				PlayerID:  p.UserID,
				SeatIndex: p.SeatIndex,
				Hole:      []string{p.Hole[0].String(), p.Hole[1].String()},
				HandName:  handName,
			})
		}
	}

	rawReveals, err := json.Marshal(reveals)
	if err != nil {
		slog.Error("game: failed to marshal showdown reveals", "room", e.room.Code, "err", err)
		return
	}
	e.rc.Broadcast(protocol.MustNewEnvelope(protocol.TypeShowdown, json.RawMessage(rawReveals)))

	pots := state.BuildPots(e.gs.Seats)
	var winners []handResultWinner

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
			winners = append(winners, handResultWinner{
				PlayerID: uid,
				Amount:   award,
				HandName: handNames[uid],
			})
		}
	}

	var resultSeats []handResultSeat
	for _, p := range e.gs.Seats {
		if p != nil {
			resultSeats = append(resultSeats, handResultSeat{p.UserID, p.Stack})
		}
	}

	var bestHand []string
	if len(winners) > 0 {
		wp := e.gs.FindPlayer(winners[0].PlayerID)
		if wp != nil {
			bestHand = state.BestFiveStrings(wp.Hole, e.gs.Community)
		}
	}

	winnerSet := map[string]bool{}
	for _, w := range winners {
		winnerSet[w.PlayerID] = true
	}
	var allPlayers []handResultPlayer
	for _, p := range e.gs.Seats {
		if p == nil {
			continue
		}
		hole := []string{}
		if !p.Folded {
			hole = []string{p.Hole[0].String(), p.Hole[1].String()}
		}
		allPlayers = append(allPlayers, handResultPlayer{
			PlayerID:    p.UserID,
			DisplayName: p.DisplayName,
			Hole:        hole,
			HandName:    handNames[p.UserID],
			Folded:      p.Folded,
			IsWinner:    winnerSet[p.UserID],
		})
	}

	rawResult, err := json.Marshal(handResultPayload{
		Winners:          winners,
		Seats:            resultSeats,
		BestHand:         bestHand,
		AllPlayers:       allPlayers,
		NextHandDelaySec: int(handStartDelay.Seconds()),
	})
	if err != nil {
		slog.Error("game: failed to marshal hand result", "room", e.room.Code, "err", err)
		return
	}
	e.rc.Broadcast(protocol.MustNewEnvelope(protocol.TypeHandResult, json.RawMessage(rawResult)))

	slog.Info("game: hand complete", "room", e.room.Code, "hand", e.gs.HandNum)

	e.saveHandHistory(json.RawMessage(rawResult))
	e.kickBrokePlayers()
	e.cleanupDisconnected()

	e.gs.Street = gmodel.StreetIdle
	e.gs.ActionSeat = gmodel.NoSeat
	e.scheduleNextHand()
}

// awardUncontested 将底池颁发给最后一个活跃玩家（其他人都弃牌了）。
func (e *Engine) awardUncontested(winner *gmodel.Player) {
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

	var uPlayers []handResultPlayer
	for _, p := range e.gs.Seats {
		if p == nil {
			continue
		}
		uPlayers = append(uPlayers, handResultPlayer{
			PlayerID:    p.UserID,
			DisplayName: p.DisplayName,
			Hole:        []string{},
			Folded:      p.Folded || p.UserID != winner.UserID,
			IsWinner:    p.UserID == winner.UserID,
		})
	}
	rawResult, err := json.Marshal(handResultPayload{
		Winners:          []handResultWinner{{PlayerID: winner.UserID, Amount: total}},
		AllPlayers:       uPlayers,
		NextHandDelaySec: int(handStartDelay.Seconds()),
	})
	if err != nil {
		slog.Error("game: failed to marshal uncontested result", "room", e.room.Code, "err", err)
		return
	}
	e.rc.Broadcast(protocol.MustNewEnvelope(protocol.TypeHandResult, json.RawMessage(rawResult)))

	slog.Info("game: uncontested pot", "room", e.room.Code, "winner", winner.UserID, "amount", total)

	e.saveHandHistory(json.RawMessage(rawResult))
	e.kickBrokePlayers()
	e.cleanupDisconnected()

	e.gs.Street = gmodel.StreetIdle
	e.gs.ActionSeat = gmodel.NoSeat
	e.scheduleNextHand()
}

// checkHandOver 检查断线弃牌后手牌是否结束。
func (e *Engine) checkHandOver() {
	active := e.gs.ActivePlayers()
	if len(active) == 1 {
		e.awardUncontested(active[0])
	}
}

// ---- 准备系统 ----

// scheduleNextHand 在手牌结算后安排下一手的开始。
func (e *Engine) scheduleNextHand() {
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
		e.resetTimer(500 * time.Millisecond)
	} else {
		e.resetTimer(handStartDelay)
	}
}

// handleReady 处理玩家发送的 ready 命令（在结算画面点击"开始下一局"）。
func (e *Engine) handleReady(msg protocol.InboundMessage) {
	if e.gs.Street != gmodel.StreetIdle {
		return
	}
	if e.readyPlayers == nil {
		e.readyPlayers = make(map[string]bool)
	}
	e.readyPlayers[msg.SenderID] = true
	e.broadcastReadyStatus()
	if e.allEligibleReady() {
		e.resetTimer(500 * time.Millisecond)
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
	e.rc.Broadcast(protocol.MustNewEnvelope(protocol.TypeReadyStatus, struct {
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
