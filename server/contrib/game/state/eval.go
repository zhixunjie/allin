package state

import (
	"fmt"
	"log/slog"

	"github.com/allin/server/contrib/eval"
	"github.com/allin/server/gmodel"
)

// CardsToStrings 将 Card 切片转换为字符串切片，空切片返回 []string{} 而非 nil。
func CardsToStrings(cards []model.Card) []string {
	if len(cards) == 0 {
		return []string{}
	}
	out := make([]string, len(cards))
	for i, c := range cards {
		out[i] = c.String()
	}
	return out
}

// EvaluateHand 返回玩家最佳 7 张牌组合的评估等级（数值越小越好）和手牌名称。
// community 不足 3 张或底牌未发时返回最差等级 model.InvalidHandRank。
func EvaluateHand(hole [2]model.Card, community []model.Card) (uint32, string) {
	if len(community) < 3 {
		slog.Error("game: EvaluateHand called with insufficient community cards", "community_len", len(community))
		return model.InvalidHandRank, ""
	}
	if hole[0].Rank == 0 || hole[1].Rank == 0 {
		slog.Error("game: EvaluateHand called with undealt hole cards", "hole0", hole[0], "hole1", hole[1])
		return model.InvalidHandRank, ""
	}
	cards := [7]eval.Card{cardToEval(hole[0]), cardToEval(hole[1])}
	for i, c := range community {
		if i >= 5 {
			break
		}
		cards[2+i] = cardToEval(c)
	}
	rank := eval.Evaluate7(cards)
	return rank, fmt.Sprintf("%s", eval.Describe(rank))
}

// BestFiveStrings 返回玩家手牌+公共牌中最佳五张的字符串切片。
// community 不足 3 张时返回 nil（翻牌前不评估）。
func BestFiveStrings(hole [2]model.Card, community []model.Card) []string {
	if len(community) < 3 {
		return nil
	}
	cards := [7]eval.Card{cardToEval(hole[0]), cardToEval(hole[1])}
	for i, c := range community {
		if i >= 5 {
			break
		}
		cards[2+i] = cardToEval(c)
	}
	best := eval.BestFive(cards)
	out := make([]string, 5)
	for i, c := range best {
		out[i] = c.String()
	}
	return out
}

// cardToEval 将 Card 转换为 eval 包使用的内部类型。
func cardToEval(c model.Card) eval.Card { return eval.Card{Rank: c.Rank, Suit: c.Suit} }
