package eval

// HandCategory 手牌类别（rank >> 20），值越小手牌越强。
type HandCategory uint32

const (
	CatStraightFlush HandCategory = iota // 0: 同花顺 / 皇家同花顺
	CatFourOfAKind                       // 1: 四条
	CatFullHouse                         // 2: 葫芦
	CatFlush                             // 3: 同花
	CatStraight                          // 4: 顺子
	CatThreeOfAKind                      // 5: 三条
	CatTwoPair                           // 6: 两对
	CatOnePair                           // 7: 一对
	CatHighCard                          // 8: 高牌
)

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

// Category 从纯 Go 等级值返回 1–9（1=皇家同花顺/同花顺，…，9=高牌）。
func Category(rank uint32) int {
	return int(rank>>20) + 1
}

// Describe 根据纯 Go 等级值返回手牌类别名称。
func Describe(rank uint32) string {
	cat := HandCategory(rank >> 20)
	if cat == CatStraightFlush {
		// 皇家同花顺：A 为最高的同花顺 → encodeRank(0, 14) → 最高半字节 = 15-14=1
		if (rank>>16)&0xF == 1 {
			return "Royal Flush"
		}
		return "Straight Flush"
	}
	if cat < CatHighCard+1 {
		return categoryNames[cat]
	}
	return "Unknown"
}
