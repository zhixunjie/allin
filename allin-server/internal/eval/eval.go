package eval

import "sort"

// Evaluate7 returns the best 5-card hand rank from 7 cards (lower = better).
// Uses Two Plus Two lookup table if loaded via Load(); otherwise pure Go.
func Evaluate7(cards [7]Card) uint32 {
	if Loaded() {
		return evaluate7Table(cards)
	}
	return evaluate7Pure(cards)
}

// Card is a playing card for the evaluator.
type Card struct {
	Rank byte // '2'…'A'
	Suit byte // 'c','d','h','s'
}

func (c Card) String() string { return string([]byte{c.Rank, c.Suit}) }

// ToEvalInt converts Card to Two Plus Two index (1–52).
func (c Card) ToEvalInt() int {
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

// evaluate7Table uses the Two Plus Two lookup table.
func evaluate7Table(cards [7]Card) uint32 {
	p := int(HR[53+cards[0].ToEvalInt()])
	p = int(HR[p+cards[1].ToEvalInt()])
	p = int(HR[p+cards[2].ToEvalInt()])
	p = int(HR[p+cards[3].ToEvalInt()])
	p = int(HR[p+cards[4].ToEvalInt()])
	p = int(HR[p+cards[5].ToEvalInt()])
	return uint32(HR[p+cards[6].ToEvalInt()])
}

// All C(7,5)=21 index combinations for 7-card best-of-5 selection.
var combos21 = [21][5]int{
	{0, 1, 2, 3, 4}, {0, 1, 2, 3, 5}, {0, 1, 2, 3, 6},
	{0, 1, 2, 4, 5}, {0, 1, 2, 4, 6}, {0, 1, 2, 5, 6},
	{0, 1, 3, 4, 5}, {0, 1, 3, 4, 6}, {0, 1, 3, 5, 6},
	{0, 1, 4, 5, 6}, {0, 2, 3, 4, 5}, {0, 2, 3, 4, 6},
	{0, 2, 3, 5, 6}, {0, 2, 4, 5, 6}, {0, 3, 4, 5, 6},
	{1, 2, 3, 4, 5}, {1, 2, 3, 4, 6}, {1, 2, 3, 5, 6},
	{1, 2, 4, 5, 6}, {1, 3, 4, 5, 6}, {2, 3, 4, 5, 6},
}

func evaluate7Pure(cards [7]Card) uint32 {
	best := uint32(0xFFFFFFFF)
	for _, c := range combos21 {
		var five [5]Card
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

// rankVal maps card rank byte to integer value (2=2…A=14).
func rankVal(r byte) uint8 {
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

// encodeRank encodes a hand to uint32 where lower = better.
// cat: 0=SF…8=HighCard. values: card ranks, highest importance first.
func encodeRank(cat uint8, values ...uint8) uint32 {
	var t uint32
	for i, v := range values {
		if i >= 5 {
			break
		}
		// (15-v) so that higher card value → smaller bit field → lower rank
		t |= uint32(15-v) << uint(4*(4-i))
	}
	return uint32(cat)<<20 | t
}

func evaluateFive(cards [5]Card) uint32 {
	var vals [5]uint8
	var suits [5]byte
	for i, c := range cards {
		vals[i] = rankVal(c.Rank)
		suits[i] = c.Suit
	}

	flush := suits[0] == suits[1] && suits[1] == suits[2] &&
		suits[2] == suits[3] && suits[3] == suits[4]

	sorted := vals
	sort.Slice(sorted[:], func(i, j int) bool { return sorted[i] > sorted[j] })

	// Uniqueness check (needed for straight detection)
	unique := sorted[0] != sorted[1] && sorted[1] != sorted[2] &&
		sorted[2] != sorted[3] && sorted[3] != sorted[4]

	var straight bool
	var straightHigh uint8
	if unique {
		if sorted[0]-sorted[4] == 4 {
			straight = true
			straightHigh = sorted[0]
		}
		// Wheel: A-2-3-4-5
		if sorted[0] == 14 && sorted[1] == 5 && sorted[2] == 4 && sorted[3] == 3 && sorted[4] == 2 {
			straight = true
			straightHigh = 5
		}
	}

	if straight && flush {
		return encodeRank(0, straightHigh)
	}

	// Frequency analysis
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
	// quads/trips/pairs/singles are already sorted descending (we iterate 14→2)

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
