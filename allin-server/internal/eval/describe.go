package eval

// Category returns 1–9 (1=Straight Flush/Royal, …, 9=High Card) from a pure-Go rank.
func Category(rank uint32) int {
	return int(rank>>20) + 1
}

// Describe returns the hand category name for a pure-Go rank.
func Describe(rank uint32) string {
	cat := rank >> 20
	if cat == 0 {
		// Royal Flush: A-high straight flush → encodeRank(0, 14) → top nibble = 15-14=1
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
