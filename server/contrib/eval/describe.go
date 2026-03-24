package eval

import "github.com/allin/server/gmodel"

// categoryNames 按 HandCategory 值（0–8）映射到手牌类型名称。
var categoryNames = [9]string{
	"Straight Flush",  // 0: CatStraightFlush — 五张同花色的连续牌；A 高为皇家同花顺
	"Four of a Kind",  // 1: CatFourOfAKind   — 四张点数相同的牌 + 一张踢脚牌
	"Full House",      // 2: CatFullHouse      — 三张相同点数 + 一对
	"Flush",           // 3: CatFlush          — 五张同花色但不连续的牌
	"Straight",        // 4: CatStraight       — 五张连续点数但非同花色；A-2-3-4-5 为最小顺子（轮子）
	"Three of a Kind", // 5: CatThreeOfAKind   — 三张点数相同的牌 + 两张踢脚牌
	"Two Pair",        // 6: CatTwoPair        — 两组不同点数的对子 + 一张踢脚牌
	"One Pair",        // 7: CatOnePair        — 两张点数相同的牌 + 三张踢脚牌
	"High Card",       // 8: CatHighCard       — 无任何有效组合，以最高牌比大小
}

// Describe 根据纯 Go 等级值返回手牌类别名称。
func Describe(rank uint32) string {
	cat := gmodel.HandCategory(rank >> 20)
	if cat == gmodel.CatStraightFlush {
		// 皇家同花顺：A 为最高的同花顺 → encodeRank(0, 14) → 最高半字节 = 15-14=1
		if (rank>>16)&0xF == 1 {
			return "Royal Flush"
		}
		return "Straight Flush"
	}
	if cat < gmodel.CatHighCard+1 {
		return categoryNames[cat]
	}
	return "Unknown"
}
