package engine

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	bizdao "github.com/allin/server/base/biz/dao"
	botpkg "github.com/allin/server/contrib/game/bot"
	"github.com/allin/server/contrib/game/state"
	"github.com/allin/server/contrib/ws"
	"github.com/allin/server/contrib/ws/protocol"
)

// ---- 加入 ----

// handleJoinRoom 处理玩家加入房间请求。
// 若玩家已在座位（断线重连）则直接恢复状态；否则走完整买入入座流程。
func (e *Engine) handleJoinRoom(msg protocol.InboundMessage, resetTimer func(time.Duration)) {
	// 断线重连：若玩家仍在座位（Disconnected=true），直接恢复。
	if existing := e.gs.FindPlayer(msg.SenderID); existing != nil {
		if existing.Disconnected {
			existing.Disconnected = false
			e.sendSnapshot(msg.SenderID)
			e.rc.Broadcast(protocol.MustEvent(protocol.TypePlayerJoined, protocol.PlayerJoinedPayload{
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
	var cmd protocol.JoinRoomCmd
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

	// 从用户账户扣除买入金额（bot 跳过 DB 操作）。
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

	p := &state.Player{
		UserID:      msg.SenderID,
		DisplayName: msg.DisplayName,
		Stack:       buyIn,
	}
	e.gs.SeatPlayer(p)

	e.rc.Broadcast(protocol.MustEvent(protocol.TypePlayerJoined, protocol.PlayerJoinedPayload{
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
	if e.gs.Street == state.StreetIdle && len(e.gs.EligibleToStart()) >= 2 {
		resetTimer(handStartDelay)
	}
}

// seatBots 将 AI 玩家安排到可用座位。
func (e *Engine) seatBots() {
	for i := 0; i < e.room.Config.BotCount; i++ {
		uid := botpkg.GenUserID(e.room.Code, i)
		if e.gs.FindPlayer(uid) != nil {
			continue // 已入座
		}
		p := &state.Player{
			UserID:      uid,
			DisplayName: botpkg.GenUserName(i),
			Stack:       e.room.Config.MaxBuyIn,
			IsBot:       true,
			BotStyle:    string(botpkg.AssignStyle(e.room.Config.BotStyle, i)),
		}
		if !e.gs.SeatPlayer(p) {
			break // 没有更多座位
		}
		e.rc.Broadcast(protocol.MustEvent(protocol.TypePlayerJoined, protocol.PlayerJoinedPayload{
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
func (e *Engine) handleDisconnect(msg protocol.InboundMessage, resetTimer func(time.Duration), stopTimer func()) {
	// Bot ID 从没有真实的 WS 连接；忽略虚假的断开连接消息。
	if botpkg.IsBotID(msg.SenderID) {
		return
	}

	p := e.gs.FindPlayer(msg.SenderID)
	if p == nil {
		return
	}

	if e.gs.Street == state.StreetIdle {
		// 手牌间隙断线：立即离座并返还筹码。
		e.rc.Broadcast(protocol.MustEvent(protocol.TypePlayerLeft, protocol.PlayerLeftPayload{
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
		e.gs.ApplyAction(p.UserID, state.ActionFold, 0)
		stopTimer()
		e.advanceOrEnd(resetTimer, stopTimer)
	}
}

// cashOut 将玩家剩余筹码返还到账户余额（bot 跳过）。
func (e *Engine) cashOut(userID string, stack int64) {
	if botpkg.IsBotID(userID) || stack == 0 {
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

// ---- 离座 ----

// handleSitOut 处理玩家离座/归座请求。
// 离座时若正轮到该玩家行动，自动弃牌；归座时若满足开局条件，启动倒计时。
func (e *Engine) handleSitOut(msg protocol.InboundMessage, resetTimer func(time.Duration), stopTimer func()) {
	var cmd protocol.SitOutCmd
	if err := json.Unmarshal(msg.Env.Payload, &cmd); err != nil {
		return
	}
	p := e.gs.FindPlayer(msg.SenderID)
	if p == nil {
		return
	}
	p.SitOut = cmd.SitOut

	e.rc.Broadcast(protocol.MustEvent(protocol.TypeSitOut, protocol.SitOutPayload{
		PlayerID:  p.UserID,
		SeatIndex: p.SeatIndex,
		SitOut:    p.SitOut,
	}))

	// 离座：若在活跃手牌中且轮到该玩家，自动弃牌。
	if cmd.SitOut && e.gs.Street != state.StreetIdle && e.gs.ActionSeat == p.SeatIndex {
		e.gs.ApplyAction(p.UserID, state.ActionFold, 0)
		stopTimer()
		e.advanceOrEnd(resetTimer, stopTimer)
		return
	}
	// 归座：若处于空闲且满足开局条件，启动计时器。
	if !cmd.SitOut && e.gs.Street == state.StreetIdle && len(e.gs.EligibleToStart()) >= 2 {
		resetTimer(handStartDelay)
	}
}

// ---- 主动离桌 ----

// handleLeaveTable 处理玩家主动离桌请求。
// 只允许在 Idle 状态执行；移除玩家、返还筹码、广播离开事件。
func (e *Engine) handleLeaveTable(msg protocol.InboundMessage) {
	if e.gs.Street != state.StreetIdle {
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
	e.rc.Broadcast(protocol.MustEvent(protocol.TypePlayerLeft, protocol.PlayerLeftPayload{
		PlayerID:  msg.SenderID,
		SeatIndex: seatIdx,
	}))
	e.maybeStartEmptyTimer()
}

// maybeStartEmptyTimer 在所有人类玩家离开时清场 bot 并启动宽限期。
func (e *Engine) maybeStartEmptyTimer() {
	humanCount := 0
	for _, sp := range e.gs.Seats {
		if sp != nil && !botpkg.IsBotID(sp.UserID) {
			humanCount++
		}
	}
	if humanCount == 0 && e.onEmpty != nil {
		for _, sp := range e.gs.Seats {
			if sp != nil && botpkg.IsBotID(sp.UserID) {
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
		e.rc.Broadcast(protocol.MustEvent(protocol.TypePlayerLeft, protocol.PlayerLeftPayload{
			PlayerID:  uid,
			SeatIndex: seatIdx,
		}))
		slog.Info("game: disconnected player cleaned up", "room", e.room.Code, "player", uid)
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
		e.rc.Broadcast(protocol.MustEvent(protocol.TypePlayerLeft, protocol.PlayerLeftPayload{
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
		if p != nil && !botpkg.IsBotID(p.UserID) {
			return true
		}
	}
	return false
}
