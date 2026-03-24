package eval

import (
	"sort"

	"github.com/allin/server/gmodel"
)

// Evaluate7 从 7 张牌中返回最佳 5 张牌的手牌等级（值越小越好）。
// 如果通过 Load() 加载了 Two Plus Two 查找表则使用快速路径，否则使用纯 Go 实现。
func Evaluate7(cards [7]gmodel.Card) uint32 {
	if Loaded() {
		return evaluate7Table(cards)
	}
	return evaluate7Pure(cards)
}

// toEvalInt 将 Card 转换为 Two Plus Two 索引（1–52）。
func toEvalInt(c gmodel.Card) int {
	var rank int
	switch c.Rank {
	case '2':
		rank = 0
	case '3':
		rank = 1
	case '4':
		rank = 2
	case '5':
		rank = 3
	case '6':
		rank = 4
	case '7':
		rank = 5
	case '8':
		rank = 6
	case '9':
		rank = 7
	case 'T':
		rank = 8
	case 'J':
		rank = 9
	case 'Q':
		rank = 10
	case 'K':
		rank = 11
	case 'A':
		rank = 12
	}
	var suit int
	switch c.Suit {
	case 'c':
		suit = 0
	case 'd':
		suit = 1
	case 'h':
		suit = 2
	case 's':
		suit = 3
	}
	return rank*4 + suit + 1
}

// evaluate7Table 使用 Two Plus Two 查找表进行评估。
func evaluate7Table(cards [7]gmodel.Card) uint32 {
	p := int(HR[53+toEvalInt(cards[0])])
	p = int(HR[p+toEvalInt(cards[1])])
	p = int(HR[p+toEvalInt(cards[2])])
	p = int(HR[p+toEvalInt(cards[3])])
	p = int(HR[p+toEvalInt(cards[4])])
	p = int(HR[p+toEvalInt(cards[5])])
	return uint32(HR[p+toEvalInt(cards[6])])
}

// 所有 C(7,5)=21 种索引组合，用于从 7 张牌中选出最佳 5 张。
var combos21 = [21][5]int{
	{0, 1, 2, 3, 4}, {0, 1, 2, 3, 5}, {0, 1, 2, 3, 6},
	{0, 1, 2, 4, 5}, {0, 1, 2, 4, 6}, {0, 1, 2, 5, 6},
	{0, 1, 3, 4, 5}, {0, 1, 3, 4, 6}, {0, 1, 3, 5, 6},
	{0, 1, 4, 5, 6}, {0, 2, 3, 4, 5}, {0, 2, 3, 4, 6},
	{0, 2, 3, 5, 6}, {0, 2, 4, 5, 6}, {0, 3, 4, 5, 6},
	{1, 2, 3, 4, 5}, {1, 2, 3, 4, 6}, {1, 2, 3, 5, 6},
	{1, 2, 4, 5, 6}, {1, 3, 4, 5, 6}, {2, 3, 4, 5, 6},
}

func evaluate7Pure(cards [7]gmodel.Card) uint32 {
	best := gmodel.InvalidHandRank
	for _, c := range combos21 {
		var five [5]gmodel.Card
		for i, idx := range c {
			five[i] = cards[idx]
		}
		r := evaluateFive(five)
		if r < best {
			best = r
		}
	}
	return best
}


// encodeRank 将手牌编码为 uint32，值越小越好。
// cat: 0=同花顺…8=高牌。values: 牌面值，按重要性从高到低排列。
func encodeRank(cat uint8, values ...uint8) uint32 {
	var t uint32
	for i, v := range values {
		if i >= 5 {
			break
		}
		// (15-v) 使得牌面值越大 → 位域越小 → 等级越低（越好）
		t |= uint32(15-v) << uint(4*(4-i))
	}
	return uint32(cat)<<20 | t
}

func evaluateFive(cards [5]gmodel.Card) uint32 {
	var vals [5]uint8
	var suits [5]byte
	for i, c := range cards {
		vals[i] = uint8(c.Rank.Val())
		suits[i] = c.Suit
	}

	flush := suits[0] == suits[1] && suits[1] == suits[2] &&
		suits[2] == suits[3] && suits[3] == suits[4]

	sorted := vals
	sort.Slice(sorted[:], func(i, j int) bool { return sorted[i] > sorted[j] })

	// 唯一性检查（顺子检测所需）
	unique := sorted[0] != sorted[1] && sorted[1] != sorted[2] &&
		sorted[2] != sorted[3] && sorted[3] != sorted[4]

	var straight bool
	var straightHigh uint8
	if unique {
		if sorted[0]-sorted[4] == 4 {
			straight = true
			straightHigh = sorted[0]
		}
		// 最小顺子（轮子）：A-2-3-4-5
		if sorted[0] == 14 && sorted[1] == 5 && sorted[2] == 4 && sorted[3] == 3 && sorted[4] == 2 {
			straight = true
			straightHigh = 5
		}
	}

	if straight && flush {
		return encodeRank(0, straightHigh)
	}

	// 频率分析
	freq := [15]int{}
	for _, v := range vals {
		freq[v]++
	}
	var quads, trips, pairs, singles []uint8
	for v := uint8(14); v >= 2; v-- {
		switch freq[v] {
		case 4:
			quads = append(quads, v)
		case 3:
			trips = append(trips, v)
		case 2:
			pairs = append(pairs, v)
		case 1:
			singles = append(singles, v)
		}
	}
	// quads/trips/pairs/singles 已按降序排列（我们从 14 迭代到 2）

	if len(quads) > 0 {
		kicker := singles[0]
		return encodeRank(1, quads[0], kicker)
	}
	if len(trips) > 0 && len(pairs) > 0 {
		return encodeRank(2, trips[0], pairs[0])
	}
	if flush {
		return encodeRank(3, sorted[0], sorted[1], sorted[2], sorted[3], sorted[4])
	}
	if straight {
		return encodeRank(4, straightHigh)
	}
	if len(trips) > 0 {
		return encodeRank(5, trips[0], singles[0], singles[1])
	}
	if len(pairs) >= 2 {
		kicker := singles[0]
		return encodeRank(6, pairs[0], pairs[1], kicker)
	}
	if len(pairs) == 1 {
		return encodeRank(7, pairs[0], singles[0], singles[1], singles[2])
	}
	return encodeRank(8, sorted[0], sorted[1], sorted[2], sorted[3], sorted[4])
}

// BestFive 从 7 张牌中找出构成最佳手牌的 5 张，返回这 5 张牌。
// 遍历全部 C(7,5)=21 种组合，取等级最低（最好）的一组。
func BestFive(cards [7]gmodel.Card) [5]gmodel.Card {
	bestRank := gmodel.InvalidHandRank
	var bestIdx [5]int
	for _, combo := range combos21 {
		var five [5]gmodel.Card
		for i, idx := range combo {
			five[i] = cards[idx]
		}
		if r := evaluateFive(five); r < bestRank {
			bestRank = r
			bestIdx = combo
		}
	}
	var result [5]gmodel.Card
	for i, idx := range bestIdx {
		result[i] = cards[idx]
	}
	return result
}
