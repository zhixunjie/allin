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
	// names 按 category 值（0–8）映射到手牌类型名称。
	// category = rank >> 20，值越小表示手牌越强。
	// index 0 在函数开头已单独处理（区分同花顺与皇家同花顺）。
	names := [9]string{
		"Straight Flush",  // 0: 同花顺 — 五张同花色的连续牌；A 高为皇家同花顺（已在上方单独处理）
		"Four of a Kind",  // 1: 四条 — 四张点数相同的牌 + 一张踢脚牌
		"Full House",      // 2: 葫芦 — 三张相同点数 + 一对
		"Flush",           // 3: 同花 — 五张同花色但不连续的牌
		"Straight",        // 4: 顺子 — 五张连续点数但非同花色的牌；A-2-3-4-5 为最小顺子（轮子）
		"Three of a Kind", // 5: 三条 — 三张点数相同的牌 + 两张单张踢脚牌
		"Two Pair",        // 6: 两对 — 两组不同点数的对子 + 一张踢脚牌
		"One Pair",        // 7: 一对 — 两张点数相同的牌 + 三张踢脚牌
		"High Card",       // 8: 高牌 — 无任何有效组合，以最高牌比大小
	}
	if cat < 9 {
		return names[cat]
	}
	return "Unknown"
}
