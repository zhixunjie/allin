package bot

import (
	"math/rand/v2"

	"github.com/allin/server/contrib/game/model"
	"github.com/allin/server/contrib/game/state"
)

// BotStyle 表示单个 bot 的风格流派（用于决策阈值映射）。
type BotStyle string

const (
	BotStyleTag     BotStyle = "tag"     // 紧凶（TAG）：选牌严格，有牌强打
	BotStyleLag     BotStyle = "lag"     // 松凶（LAG）：宽泛入局，频繁加注
	BotStyleStation BotStyle = "station" // 松被动（Calling Station）：喜欢跟注，很少主动
	BotStyleRock    BotStyle = "rock"    // 紧被动（Rock）：极度保守，只玩超强牌
)

// personalities 存储四种风格的参数：
//
//	TAG（紧凶）：高选择性入局，入局即施压
//	LAG（松凶）：宽松入局，频繁加注/虚张声势
//	Station（松被动）：几乎不弃牌，喜欢跟注，很少主动下注
//	Rock（紧被动）：极度保守，只玩超强牌，见注则缩
var personalities = map[BotStyle]Personality{
	BotStyleTag:     {0.65, 0.80, 0.55, 0.30, 0.08},
	BotStyleLag:     {0.35, 0.50, 0.35, 0.15, 0.22},
	BotStyleStation: {0.30, 0.85, 0.70, 0.05, 0.02},
	BotStyleRock:    {0.78, 0.92, 0.72, 0.48, 0.02},
}

// styleOrder 用于按序号循环分配风格（混合主题）。
var styleOrder = []BotStyle{
	BotStyleTag,
	BotStyleLag,
	BotStyleStation,
	BotStyleRock,
}

// SetBotType 表示房间 bot 的整体风格主题。
type SetBotType string

const (
	SetBotTypeMixed      SetBotType = "mixed"      // 混合（TAG→LAG→Station→Rock 循环分配）
	SetBotTypeAggressive SetBotType = "aggressive" // 激进主题（TAG 与 LAG 交替）
	SetBotTypePassive    SetBotType = "passive"    // 被动主题（Rock 与 Station 交替）
	SetBotTypeRandom     SetBotType = "random"     // 随机（每个 bot 独立随机选取风格）
)

var AllSetBotType = []SetBotType{SetBotTypeMixed, SetBotTypeAggressive, SetBotTypePassive, SetBotTypeRandom}

// Personality 定义某种风格 bot 的决策阈值。
type Personality struct {
	PreflopEnterThreshold float64 // 主动入局所需最低 preflop 强度
	PreflopRaiseThreshold float64 // preflop 选择加注而非跟注的门槛
	PostflopBetThreshold  float64 // postflop 主动下注所需最低强度
	PostflopFoldThreshold float64 // postflop 面对下注时弃牌的强度上限
	BluffRate             float64 // 手牌偏弱时仍激进行动的概率
}

// decide 根据风格人设和当前局面返回行动及金额。
func (p *Personality) decide(sit state.BotSituation) (state.Action, int64) {
	strength := handStrength(sit.Street, sit.Hole, sit.Community)
	toCall := sit.CurrentBet - sit.PlayerBet
	r := rand.Float64()

	if sit.Street == model.StreetPreFlop {
		if toCall <= 0 {
			if strength >= p.PreflopRaiseThreshold {
				return safeRaise(sit.CurrentBet, sit.PlayerBet, sit.Stack, sit.BigBlind*3, sit.BigBlind)
			}
			return state.ActionCheck, 0
		}
		if strength < p.PreflopEnterThreshold && r > p.BluffRate {
			return state.ActionFold, 0
		}
		if strength >= p.PreflopRaiseThreshold {
			return safeRaise(sit.CurrentBet, sit.PlayerBet, sit.Stack, sit.BigBlind*3, sit.BigBlind)
		}
		if toCall >= sit.Stack {
			return state.ActionAllIn, 0
		}
		return state.ActionCall, 0
	}

	// Postflop：无人下注，先手
	if toCall <= 0 {
		if strength >= p.PostflopBetThreshold || r < p.BluffRate {
			betAmt := sit.Pot * 6 / 10
			if betAmt < sit.BigBlind {
				betAmt = sit.BigBlind
			}
			if betAmt >= sit.Stack {
				return state.ActionAllIn, 0
			}
			return state.ActionBet, betAmt
		}
		return state.ActionCheck, 0
	}

	// Postflop：面对下注
	if strength < p.PostflopFoldThreshold && r > p.BluffRate {
		return state.ActionFold, 0
	}
	raiseThresh := p.PostflopBetThreshold * 1.3
	if raiseThresh > 1.0 {
		raiseThresh = 1.0
	}
	if strength >= raiseThresh {
		return safeRaise(sit.CurrentBet, sit.PlayerBet, sit.Stack, sit.CurrentBet*5/2, sit.BigBlind)
	}
	if toCall >= sit.Stack {
		return state.ActionAllIn, 0
	}
	return state.ActionCall, 0
}

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
func preflopStrength(hole [2]model.Card) float64 {
	r1 := rankVal(hole[0].Rank)
	r2 := rankVal(hole[1].Rank)
	suited := hole[0].Suit == hole[1].Suit

	if r1 == r2 {
		return 0.5 + float64(r1-2)/24.0*0.5
	}
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
func postflopStrength(hole [2]model.Card, community []model.Card) float64 {
	if len(community) < 5 {
		return preflopStrength(hole)
	}
	rank, _ := state.EvaluateHand(hole, community)
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
func handStrength(street model.Street, hole [2]model.Card, community []model.Card) float64 {
	if street == model.StreetPreFlop || len(community) < 3 {
		return preflopStrength(hole)
	}
	return postflopStrength(hole, community)
}

// safeRaise 计算合法的 raise 总额：不低于 minRaise 增量，超过 stack 则 all-in。
func safeRaise(currentBet, playerBet, stack, target, minRaise int64) (state.Action, int64) {
	minTotal := currentBet + minRaise
	if target < minTotal {
		target = minTotal
	}
	needed := target - playerBet
	if needed >= stack {
		return state.ActionAllIn, 0
	}
	return state.ActionRaise, target
}
