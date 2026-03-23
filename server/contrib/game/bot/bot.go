package bot

import (
	"encoding/json"
	"math/rand"
	"time"

	"github.com/allin/server/contrib/game/model"
	"github.com/allin/server/contrib/game/state"
	"github.com/allin/server/contrib/ws"
	"github.com/allin/server/contrib/ws/protocol"
)

// Bot 封装了 AI 行动调度逻辑，持有写入引擎 Inbound channel 所需的 RoomConn。
type Bot struct {
	rc *ws.RoomConn // 该房间的 WebSocket 消息总线，用于注入 bot 行动
}

// New 创建一个与指定 RoomConn 绑定的 Bot。
func New(rc *ws.RoomConn) *Bot {
	return &Bot{rc: rc}
}

// ScheduleAction 启动 goroutine，在随机延迟后向引擎注入 bot 行动。
// 只读取快照变量，写入经 buffered channel，线程安全。
func (b *Bot) ScheduleAction(gs *state.GameStateMachine, p *model.Player) {
	botID := p.UserID

	// 确定 bot 性格
	personality, ok := personalities[p.BotStyle]
	if !ok {
		personality = personalities[model.BotStyleTag]
	}

	situation := state.NewBotSituation(gs, p)

	go func() {
		// 模拟思考时间：1–3 秒随机延迟
		delay := time.Duration(1000+rand.Intn(2000)) * time.Millisecond
		time.Sleep(delay)

		action, amount := personality.decide(situation)
		payload, _ := json.Marshal(protocol.ActionCmd{Action: string(action), Amount: amount})

		select {
		case b.rc.Inbound <- protocol.InboundMessage{
			SenderID: botID,
			Env: protocol.CmdEnvelope{
				Type:    protocol.CmdAction,
				Payload: payload,
			},
		}:
		default:
			// channel 已满，ValidateAction 会拒绝过期行动，静默丢弃
		}
	}()
}
