package eval

// Category 从纯 Go 等级值返回 1–9（1=皇家同花顺/同花顺，…，9=高牌）。
func Category(rank uint32) int {
	return int(rank>>20) + 1
}

// Describe 根据纯 Go 等级值返回手牌类别名称。
func Describe(rank uint32) string {
	cat := rank >> 20
	if cat == 0 {
		// 皇家同花顺：A 为最高的同花顺 → encodeRank(0, 14) → 最高半字节 = 15-14=1
		if (rank>>16)&0xF == 1 {
			return "Royal Flush"
		}
		return "Straight Flush"
	}
	names := [9]string{
		"Straight Flush", // 0 (handled above)
		"Four of a Kind", // 1
		"Full House",     // 2
		"Flush",          // 3
		"Straight",       // 4
		"Three of a Kind", // 5
		"Two Pair",       // 6
		"One Pair",       // 7
		"High Card",      // 8
	}
	if cat < 9 {
		return names[cat]
	}
	return "Unknown"
}
