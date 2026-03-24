package engine

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	bizdao "github.com/allin/server/base/biz/dao"
	botpkg "github.com/allin/server/contrib/game/bot"
	"github.com/allin/server/contrib/ws/protocol"
	"github.com/allin/server/gmodel"
)

// ---- 加入 ----

// handleJoinRoom 处理玩家加入房间请求。
// 若玩家已在座位（断线重连）则直接恢复状态；否则走完整买入入座流程。
func (e *Engine) handleJoinRoom(msg protocol.InboundMessage) {
	// 断线重连：若玩家仍在座位（Disconnected=true），直接恢复。
	if existing := e.gs.FindPlayer(msg.SenderID); existing != nil {
		if existing.Disconnected {
			existing.Disconnected = false
			e.sendSnapshot(msg.SenderID)
			e.rc.Broadcast(protocol.MustNewEnvelope(protocol.TypePlayerJoined, protocol.PlayerJoinedPayload{
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
		e.sendError(msg.SenderID, gmodel.SkErrRoomFull, msg.Env.Seq)
		return
	}

	// 解析带入金额（0 表示默认使用 MaxBuyIn）。
	var cmd protocol.JoinRoomCmd
	if err := json.Unmarshal(msg.Env.Payload, &cmd); err != nil {
		slog.Warn("game: failed to parse join_room payload, using defaults", "user", msg.SenderID, "err", err)
	}
	buyIn := cmd.BuyIn
	if buyIn == 0 {
		buyIn = e.room.Config.MaxBuyIn
	}
	minBuyIn := e.room.Config.MinBuyIn
	maxBuyIn := e.room.Config.MaxBuyIn
	if buyIn < minBuyIn || buyIn > maxBuyIn {
		e.sendError(msg.SenderID, gmodel.SkErrInvalidBuyIn, msg.Env.Seq,
			fmt.Sprintf("buy_in must be between %d and %d", minBuyIn, maxBuyIn))
		return
	}

	// 从用户账户扣除买入金额（bot 跳过 DB 操作）。
	senderIntID, parseErr := strconv.ParseInt(msg.SenderID, 10, 64)
	if parseErr != nil {
		e.sendError(msg.SenderID, gmodel.SkErrUserNotFound, msg.Env.Seq)
		return
	}
	u, err := bizdao.UserDao.GetByID(senderIntID)
	if err != nil {
		e.sendError(msg.SenderID, gmodel.SkErrUserNotFound, msg.Env.Seq)
		return
	}
	if u.ChipBalance < buyIn {
		e.sendError(msg.SenderID, gmodel.SkErrInsufficientChips, msg.Env.Seq,
			fmt.Sprintf("insufficient chips: need $%d, have $%d", buyIn, u.ChipBalance))
		return
	}
	if err := bizdao.UserDao.AdjustChips(senderIntID, -buyIn, "buy_in", e.room.Code); err != nil {
		slog.Error("game: failed to deduct buy-in", "user", msg.SenderID, "err", err)
		e.sendError(msg.SenderID, gmodel.SkErrServerError, msg.Env.Seq, "failed to process buy-in")
		return
	}

	p := &gmodel.Player{
		UserID:      msg.SenderID,
		DisplayName: msg.DisplayName,
		Stack:       buyIn,
	}
	e.gs.SeatPlayer(p)

	e.rc.Broadcast(protocol.MustNewEnvelope(protocol.TypePlayerJoined, protocol.PlayerJoinedPayload{
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
	if e.gs.Street == gmodel.StreetIdle && len(e.gs.EligibleToStart()) >= 2 {
		e.resetTimer(handStartDelay)
	}
}

// seatBots 将 AI 玩家安排到可用座位。
func (e *Engine) seatBots() {
	for i := 0; i < e.room.Config.BotCount; i++ {
		uid := botpkg.GenUserID(e.room.Code, i)
		if e.gs.FindPlayer(uid) != nil {
			continue // 已入座
		}
		p := &gmodel.Player{
			UserID:      uid,
			DisplayName: botpkg.GenUserName(i),
			Stack:       e.room.Config.MaxBuyIn,
			IsBot:       true,
			BotStyle:    gmodel.AssignStyle(e.room.Config.BotType, i),
		}
		if !e.gs.SeatPlayer(p) {
			break // 没有更多座位
		}
		e.rc.Broadcast(protocol.MustNewEnvelope(protocol.TypePlayerJoined, protocol.PlayerJoinedPayload{
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
func (e *Engine) handleDisconnect(msg protocol.InboundMessage) {
	// Bot 没有真实 WS 连接，hub 不会产生 bot 的断线消息；防御性过滤
	if botpkg.IsBotID(msg.SenderID) {
		return
	}

	p := e.gs.FindPlayer(msg.SenderID)
	if p == nil {
		// 玩家尚未入座（连接建立后断开但未坐下），无需处理
		return
	}

	if e.gs.Street == gmodel.StreetIdle {
		// 手牌间隙断线：视同主动离桌，先广播再离座，保证客户端先收到事件
		e.rc.Broadcast(protocol.MustNewEnvelope(protocol.TypePlayerLeft, protocol.PlayerLeftPayload{
			PlayerID:  p.UserID,
			SeatIndex: p.SeatIndex,
		}))
		stack := p.Stack
		e.gs.UnseatPlayer(msg.SenderID)
		e.cashOut(msg.SenderID, stack) // 将剩余筹码归还账户
		e.room.Touch()                 // 刷新活跃时间，防止 GC 误回收
		e.maybeStartEmptyTimer()       // 若人类玩家已全部离开，启动空桌宽限期
		return
	}

	// 手牌进行中断线：保留座位给重连使用，仅标记断线状态
	p.Disconnected = true
	if e.gs.ActionSeat == p.SeatIndex {
		// 当前正轮到断线玩家行动，立即代为弃牌，避免全桌等待
		e.gs.ApplyAction(p.UserID, gmodel.ActionFold, 0)
		e.stopTimer()
		e.advanceOrEnd()
	}
	// 若不是行动位，不做干预：等轮到他时超时逻辑会自动弃牌，
	// 这段时间内玩家仍可重连并恢复正常参与
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
func (e *Engine) handleSitOut(msg protocol.InboundMessage) {
	var cmd protocol.SitOutCmd
	if err := json.Unmarshal(msg.Env.Payload, &cmd); err != nil {
		return
	}
	p := e.gs.FindPlayer(msg.SenderID)
	if p == nil {
		return
	}
	p.SitOut = cmd.SitOut

	e.rc.Broadcast(protocol.MustNewEnvelope(protocol.TypeSitOut, protocol.SitOutPayload{
		PlayerID:  p.UserID,
		SeatIndex: p.SeatIndex,
		SitOut:    p.SitOut,
	}))

	// 离座：若在活跃手牌中且轮到该玩家，自动弃牌。
	if cmd.SitOut && e.gs.Street != gmodel.StreetIdle && e.gs.ActionSeat == p.SeatIndex {
		e.gs.ApplyAction(p.UserID, gmodel.ActionFold, 0)
		e.stopTimer()
		e.advanceOrEnd()
		return
	}
	// 归座：若处于空闲且满足开局条件，启动计时器。
	if !cmd.SitOut && e.gs.Street == gmodel.StreetIdle && len(e.gs.EligibleToStart()) >= 2 {
		e.resetTimer(handStartDelay)
	}
}

// ---- 主动离桌 ----

// handleLeaveTable 处理玩家主动离桌请求。
// 只允许在 Idle（手牌间隙）状态执行；手牌进行中拒绝并返回错误。
// 流程：校验状态 → 起身 → 返还筹码 → 广播离开事件 → 检查空桌定时器。
func (e *Engine) handleLeaveTable(msg protocol.InboundMessage) {
	// 手牌进行中不允许离桌，告知客户端稍后再试
	if e.gs.Street != gmodel.StreetIdle {
		e.sendError(msg.SenderID, gmodel.SkErrHandInProgress, msg.Env.Seq)
		return
	}
	p := e.gs.FindPlayer(msg.SenderID)
	if p == nil {
		// 玩家已不在座位（重复请求或竞态），忽略
		return
	}
	// 提前保存 stack 和 seatIdx，UnseatPlayer 执行后指针不再有效
	stack := p.Stack
	seatIdx := p.SeatIndex
	e.gs.UnseatPlayer(msg.SenderID)
	// 将桌面筹码归还到玩家账户
	e.cashOut(msg.SenderID, stack)
	// 更新房间最后活跃时间，防止 GC 误回收
	e.room.Touch()
	// 通知所有客户端该座位已清空
	e.rc.Broadcast(protocol.MustNewEnvelope(protocol.TypePlayerLeft, protocol.PlayerLeftPayload{
		PlayerID:  msg.SenderID,
		SeatIndex: seatIdx,
	}))
	// 若人类玩家已全部离开，触发 bot 清场与空桌宽限期
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

// cleanupDisconnected 在每手牌结束后统一清理仍处于断线状态的玩家。
// 断线玩家在手牌进行中保留座位（见 handleDisconnect），手牌结束时才在此统一处理，
// 避免中途离座影响边底池计算和摊牌逻辑。
func (e *Engine) cleanupDisconnected() {
	for _, p := range e.gs.Seats {
		if p == nil || !p.Disconnected {
			continue
		}
		// 提前保存，UnseatPlayer 执行后指针字段已清空
		uid := p.UserID
		seatIdx := p.SeatIndex
		stack := p.Stack
		e.gs.UnseatPlayer(uid)
		e.cashOut(uid, stack) // 将剩余筹码归还账户
		// 通知所有客户端该座位已清空，保持前端状态一致
		e.rc.Broadcast(protocol.MustNewEnvelope(protocol.TypePlayerLeft, protocol.PlayerLeftPayload{
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
		e.rc.Broadcast(protocol.MustNewEnvelope(protocol.TypePlayerLeft, protocol.PlayerLeftPayload{
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
