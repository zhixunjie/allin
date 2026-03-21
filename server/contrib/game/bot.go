package game

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/allin/server/contrib/ws"
)

var botNames = []string{
	"Alice", "Bob", "Charlie", "Diana", "Eve",
	"Frank", "Grace", "Henry", "Ivy", "Jack",
}

// botUserID 根据房间码和序号生成确定性的 bot 用户 ID。
func botUserID(roomCode string, i int) string {
	return fmt.Sprintf("bot_%s_%d", roomCode, i)
}

// botDisplayName 返回第 i 个 bot 的显示名。
func botDisplayName(i int) string {
	return botNames[i%len(botNames)]
}

// IsBotID 判断 userID 是否属于 bot。
func IsBotID(userID string) bool {
	return strings.HasPrefix(userID, "bot_")
}

// ---- 风格人设 ----

// BotStyle 表示 bot 的风格流派（值必须与 room.RoomConfig.BotStyle 的 JSON 字段匹配）。
type BotStyle string

const (
	BotStyleTag        BotStyle = "tag"       // 紧凶（TAG）
	BotStyleLag        BotStyle = "lag"       // 松凶（LAG）
	BotStyleStation    BotStyle = "station"   // 松被动（Calling Station）
	BotStyleRock       BotStyle = "rock"      // 紧被动（Rock）
	BotStyleMixed      BotStyle = "mixed"     // 混合（循环分配四种风格）
	BotStyleAggressive BotStyle = "aggressive" // 激进主题（TAG+LAG 交替）
	BotStylePassive    BotStyle = "passive"   // 被动主题（Rock+Station 交替）
	BotStyleRandom     BotStyle = "random"    // 随机（每个 bot 独立随机）
)

// BotPersonality 定义某种风格 bot 的决策阈值。
type BotPersonality struct {
	PreflopEnterThreshold float64 // 主动入局所需最低 preflop 强度
	PreflopRaiseThreshold float64 // preflop 选择加注而非跟注的门槛
	PostflopBetThreshold  float64 // postflop 主动下注所需最低强度
	PostflopFoldThreshold float64 // postflop 面对下注时弃牌的强度上限
	BluffRate             float64 // 手牌偏弱时仍激进行动的概率
}

// personalities 存储四种风格的参数：
//
//	TAG（紧凶）：高选择性入局，入局即施压
//	LAG（松凶）：宽松入局，频繁加注/虚张声势
//	Station（松被动）：几乎不弃牌，喜欢跟注，很少主动下注
//	Rock（紧被动）：极度保守，只玩超强牌，见注则缩
var personalities = map[BotStyle]BotPersonality{
	BotStyleTag:     {0.65, 0.80, 0.55, 0.30, 0.08},
	BotStyleLag:     {0.35, 0.50, 0.35, 0.15, 0.22},
	BotStyleStation: {0.30, 0.85, 0.70, 0.05, 0.02},
	BotStyleRock:    {0.78, 0.92, 0.72, 0.48, 0.02},
}

// styleOrder 用于按序号循环分配风格（混合主题）。
var styleOrder = []BotStyle{BotStyleTag, BotStyleLag, BotStyleStation, BotStyleRock}

// assignBotStyle 根据房间风格主题和 bot 序号，返回具体风格名。
//
//	mixed（默认）: TAG→LAG→Station→Rock 循环
//	aggressive:    TAG→LAG 交替
//	passive:       Rock→Station 交替
//	random:        每个 bot 独立随机
func assignBotStyle(roomStyle string, index int) BotStyle {
	switch roomStyle {
	case string(BotStyleAggressive):
		return []BotStyle{BotStyleTag, BotStyleLag}[index%2]
	case string(BotStylePassive):
		return []BotStyle{BotStyleRock, BotStyleStation}[index%2]
	case string(BotStyleRandom):
		return styleOrder[rand.Intn(len(styleOrder))]
	default: // "mixed" 或空字符串
		return styleOrder[index%len(styleOrder)]
	}
}

// ---- 手牌强度评估 ----

// rankVal 将牌面字节转换为整数（2→2 … A→14）。
func rankVal(r byte) int {
	switch r {
	case '2':
		return 2
	case '3':
		return 3
	case '4':
		return 4
	case '5':
		return 5
	case '6':
		return 6
	case '7':
		return 7
	case '8':
		return 8
	case '9':
		return 9
	case 'T':
		return 10
	case 'J':
		return 11
	case 'Q':
		return 12
	case 'K':
		return 13
	case 'A':
		return 14
	}
	return 0
}

// preflopStrength 在没有公共牌时估算底牌强度，返回 0.0–1.0。
//
//   - 对子：0.5 + (rank-2)/24.0×0.5（22≈0.54，AA=1.0）
//   - 非对子：按点数和计算基础值，同花 +0.04，连牌 +0.02
func preflopStrength(hole [2]Card) float64 {
	r1 := rankVal(hole[0].Rank)
	r2 := rankVal(hole[1].Rank)
	suited := hole[0].Suit == hole[1].Suit

	if r1 == r2 {
		// 口袋对子
		return 0.5 + float64(r1-2)/24.0*0.5
	}
	// 非对子：取高低牌计算基础值
	hi, lo := r1, r2
	if lo > hi {
		hi, lo = lo, hi
	}
	base := float64(hi+lo-4) / float64(14+13-4)
	if suited {
		base += 0.04
	}
	if hi-lo == 1 {
		base += 0.02
	}
	if base > 1.0 {
		base = 1.0
	}
	return base
}

// catStrength 将成牌类别（rank >> 20）映射为强度值，0=同花顺 … 8=高牌。
var catStrength = [9]float64{1.0, 0.95, 0.88, 0.78, 0.70, 0.60, 0.45, 0.30, 0.15}

// postflopStrength 在有公共牌时评估最佳成牌强度，返回 0.0–1.0。
//
// EvaluateHand 内部构造 [7]eval.Card 数组，不足 7 张时用零值填充。
// 翻牌（3张）和转牌（4张）时零牌会使 evaluateFive 越界 panic，
// 因此只有公共牌达到 5 张（河牌后）才做完整评估，否则退回 preflop 估算。
func postflopStrength(hole [2]Card, community []Card) float64 {
	if len(community) < 5 {
		// 翻牌/转牌阶段：公共牌不足，零值填充会引发 panic，退回 preflop 强度
		return preflopStrength(hole)
	}
	rank, _ := EvaluateHand(hole, community)
	if rank == 0xFFFFFFFF {
		return preflopStrength(hole)
	}
	cat := rank >> 20
	if cat >= 9 {
		return 0.05
	}
	return catStrength[cat]
}

// handStrength 按当前街道分派到 preflop 或 postflop 强度计算。
func handStrength(street Street, hole [2]Card, community []Card) float64 {
	if street == StreetPreFlop || len(community) < 3 {
		return preflopStrength(hole)
	}
	return postflopStrength(hole, community)
}

// ---- 决策逻辑 ----

// decideBotAction 根据风格人设和当前局面返回行动及金额。
//
// Preflop 无需跟注（BB 过牌或 limp）：
//
//	强度 >= RaiseThreshold → raise(3×BB)
//	否则 → check
//
// Preflop 需跟注：
//
//	强度 < EnterThreshold 且非虚张声势 → fold
//	强度 >= RaiseThreshold → raise(3×BB)
//	否则 → call
//
// Postflop 无人下注：
//
//	强度 >= BetThreshold 或触发 BluffRate → bet(0.6×pot)
//	否则 → check
//
// Postflop 面对下注：
//
//	强度 < FoldThreshold 且非虚张声势 → fold
//	强度 >= BetThreshold×1.3 → raise(2.5× currentBet)
//	否则 → call
func decideBotAction(
	p *BotPersonality,
	street Street,
	hole [2]Card,
	community []Card,
	currentBet, playerBet, stack, bigBlind, pot int64,
) (Action, int64) {
	strength := handStrength(street, hole, community)
	toCall := currentBet - playerBet
	r := rand.Float64()

	if street == StreetPreFlop {
		if toCall <= 0 {
			// 无需入池成本（BB 的免费过牌或 limp）
			if strength >= p.PreflopRaiseThreshold {
				return safeRaise(currentBet, playerBet, stack, bigBlind*3, bigBlind)
			}
			return ActionCheck, 0
		}
		// 面对 preflop 下注
		if strength < p.PreflopEnterThreshold && r > p.BluffRate {
			return ActionFold, 0
		}
		if strength >= p.PreflopRaiseThreshold {
			return safeRaise(currentBet, playerBet, stack, bigBlind*3, bigBlind)
		}
		if toCall >= stack {
			return ActionAllIn, 0
		}
		return ActionCall, 0
	}

	// Postflop：无人下注，先手
	if toCall <= 0 {
		if strength >= p.PostflopBetThreshold || r < p.BluffRate {
			betAmt := pot * 6 / 10 // 0.6 × pot
			if betAmt < bigBlind {
				betAmt = bigBlind
			}
			if betAmt >= stack {
				return ActionAllIn, 0
			}
			return ActionBet, betAmt
		}
		return ActionCheck, 0
	}

	// Postflop：面对下注
	if strength < p.PostflopFoldThreshold && r > p.BluffRate {
		return ActionFold, 0
	}
	raiseThresh := p.PostflopBetThreshold * 1.3
	if raiseThresh > 1.0 {
		raiseThresh = 1.0
	}
	if strength >= raiseThresh {
		return safeRaise(currentBet, playerBet, stack, currentBet*5/2, bigBlind)
	}
	if toCall >= stack {
		return ActionAllIn, 0
	}
	return ActionCall, 0
}

// safeRaise 计算合法的 raise 总额：不低于 minRaise 增量，不超过 stack 则 all-in。
func safeRaise(currentBet, playerBet, stack, target, minRaise int64) (Action, int64) {
	minTotal := currentBet + minRaise
	if target < minTotal {
		target = minTotal
	}
	needed := target - playerBet
	if needed >= stack {
		return ActionAllIn, 0
	}
	return ActionRaise, target
}

// ---- AI 行动调度 ----

// scheduleAIAction 启动 goroutine，在随机延迟后向引擎注入 bot 行动。
// 只读取快照变量，写入经 buffered channel，线程安全。
func (e *Engine) scheduleAIAction(p *Player) {
	botID := p.UserID

	// 确定风格人设
	style := p.BotStyle
	if style == "" {
		style = BotStyleTag
	}
	personality, ok := personalities[style]
	if !ok {
		personality = personalities[BotStyleTag]
	}

	// 捕获局面快照（不可变）
	street := e.gs.Street
	hole := p.Hole
	community := make([]Card, len(e.gs.Community))
	copy(community, e.gs.Community)
	currentBet := e.gs.CurrentBet
	playerBet := p.Bet
	stack := p.Stack
	bigBlind := e.gs.Config.BigBlind
	pot := e.gs.TotalPot()

	go func() {
		// 模拟思考时间：1–3 秒随机延迟
		delay := time.Duration(1000+rand.Intn(2000)) * time.Millisecond
		time.Sleep(delay)

		action, amount := decideBotAction(
			&personality, street, hole, community,
			currentBet, playerBet, stack, bigBlind, pot,
		)
		payload, _ := json.Marshal(ws.ActionCmd{Action: string(action), Amount: amount})

		select {
		case e.hub.Inbound <- ws.InboundMessage{
			SenderID: botID,
			Env: ws.CmdEnvelope{
				Type:    ws.CmdAction,
				Payload: payload,
			},
		}:
		default:
			// channel 已满，ValidateAction 会拒绝过期行动，静默丢弃
		}
	}()
}
