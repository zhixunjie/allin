package engine

import (
	"time"

	"github.com/allin/server/contrib/game/bot"
	"github.com/allin/server/contrib/game/state"
	"github.com/allin/server/contrib/room"
	"github.com/allin/server/contrib/ws"
	"github.com/allin/server/contrib/ws/protocol"
	"github.com/allin/server/gmodel"
)

const (
	handStartDelay   = 10 * time.Second // 两手牌之间的等待时长，给玩家留出准备时间
	chatRateLimit    = time.Second      // 聊天消息发送频率上限：每人每秒最多 1 条
	emptyGracePeriod = 30 * time.Second // 所有人类玩家离开后，房间被回收前的宽限期
	botReplaceDelay  = 8 * time.Second  // bot 破产离桌后，等待真实玩家加入的宽限期；超时则补充新 bot
)

// Engine 驱动一个房间的游戏状态机。
// 所有对 GameStateMachine 的修改都严格发生在单一的 Run() goroutine 中，
// 无需任何锁即可保证并发安全。
type Engine struct {
	rc              *ws.RoomConn            // 该房间的 WebSocket 消息总线
	room            *room.Room              // 房间元数据与配置
	gs              *state.GameStateMachine // 游戏状态（街道、座位、下注等）
	deck            []gmodel.Card           // 当前手牌使用的洗牌牌组（每手重置）
	bot             *bot.Bot                // AI 行动调度器，持有 RoomConn 引用
	quit            chan struct{}           // 关闭此 channel 通知 Run() 退出
	chatLimiter     map[string]time.Time    // 各玩家最近一次聊天时间，用于频率限制
	registry        *Registry               // 全局引擎注册表，为 nil 时不注册
	onEmpty         func()                  // 所有人类玩家离开后触发的回调（用于回收房间）
	emptyTimer      *time.Timer             // 宽限期计时器，到期后执行 onEmpty
	botsSeated      bool                    // 标记 bot 是否已入座（首位人类玩家加入时触发一次）
	handActions     []actionLogEntry        // 当前手牌的行动序列，手牌结束后写入历史表
	botReplaceTimer *time.Timer             // bot 破产后等待补充的计时器
	botReplaceC     <-chan time.Time        // botReplaceTimer 对应的 channel，nil 时不触发
	readyPlayers    map[string]bool         // 在结算画面点击"开始下一局"的玩家集合
	resetTimer      func(time.Duration)     // 重置行动倒计时（停旧 timer，启新 timer）
	stopTimer       func()                  // 停止行动倒计时（timer.Stop + timerC 置 nil）
}

// NewEngine 为给定的 RoomConn 和房间创建引擎。
func NewEngine(rc *ws.RoomConn, rm *room.Room, registry *Registry) *Engine {
	cfg := rm.Config
	if cfg.ActionTimeSec == 0 {
		cfg.ActionTimeSec = 30
	}
	e := &Engine{
		rc:          rc,
		room:        rm,
		bot:         bot.New(rc),
		chatLimiter: make(map[string]time.Time),
		gs: &state.GameStateMachine{
			Street:     gmodel.StreetIdle,
			ActionSeat: gmodel.NoSeat,
			DealerSeat: gmodel.NoSeat,
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

	e.resetTimer = func(d time.Duration) {
		if actionTimer != nil {
			actionTimer.Stop()
		}
		actionTimer = time.NewTimer(d)
		timerC = actionTimer.C
	}
	e.stopTimer = func() {
		if actionTimer != nil {
			actionTimer.Stop()
		}
		timerC = nil
	}

	for {
		select {
		case <-e.quit:
			e.stopTimer()
			return

		case msg := <-e.rc.Inbound:
			e.handleMessage(msg)

		case <-timerC:
			timerC = nil
			e.handleTimeout()

		case <-e.botReplaceC:
			// bot 破产宽限期结束：若仍有人类玩家且 bot 数量不足，补充新 bot
			e.botReplaceC = nil
			if e.hasHumanPlayers() {
				e.seatBots()
				if e.gs.Street == gmodel.StreetIdle && len(e.gs.EligibleToStart()) >= 2 {
					e.resetTimer(handStartDelay)
				}
			}
		}
	}
}

// handleMessage 将入站消息按命令类型路由到对应处理函数。
func (e *Engine) handleMessage(msg protocol.InboundMessage) {
	switch msg.Env.Type {
	// 玩家加入或重连：分配座位、扣除买入、广播状态
	case protocol.CmdJoinRoom:
		e.handleJoinRoom(msg)
	// 玩家行动（fold/check/call/bet/raise/all_in）：校验合法性、推进状态机
	case protocol.CmdAction:
		e.handleAction(msg)
	// 聊天消息：限速后广播给房间内所有人
	case protocol.CmdChat:
		e.handleChat(msg)
	// 离座/归座切换：更新 SitOut 标志，影响下一手参与资格
	case protocol.CmdSitOut:
		e.handleSitOut(msg)
	// 主动离桌：仅限手牌间隙，归还筹码并移除座位
	case protocol.CmdLeaveTable:
		e.handleLeaveTable(msg)
	// 客户端优雅断开：手牌中保留座位并标记 Disconnected，间隙则直接离桌
	case protocol.CmdDisconnect:
		e.handleDisconnect(msg)
	// 结算画面点击"准备"：全员准备后提前 500ms 开始下一手
	case protocol.CmdReady:
		e.handleReady(msg)
	}
}
