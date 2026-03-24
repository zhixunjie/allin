package gmodel

// CardRank 表示扑克牌点数字节。
type CardRank byte

// 点数枚举，值与 ASCII 字节一致，可直接用于字符串拼接和网络传输。
const (
	CardRankNone CardRank = 0   // 空牌/未发牌哨兵值
	CardRank2    CardRank = '2'
	CardRank3 CardRank = '3'
	CardRank4 CardRank = '4'
	CardRank5 CardRank = '5'
	CardRank6 CardRank = '6'
	CardRank7 CardRank = '7'
	CardRank8 CardRank = '8'
	CardRank9 CardRank = '9'
	CardRankT CardRank = 'T'
	CardRankJ CardRank = 'J'
	CardRankQ CardRank = 'Q'
	CardRankK CardRank = 'K'
	CardRankA CardRank = 'A'
)

// Val 将点数转换为整数值（CardRank2→2 … CardRankA→14），未知点数返回 0。
func (r CardRank) Val() int {
	switch r {
	case CardRank2:
		return 2
	case CardRank3:
		return 3
	case CardRank4:
		return 4
	case CardRank5:
		return 5
	case CardRank6:
		return 6
	case CardRank7:
		return 7
	case CardRank8:
		return 8
	case CardRank9:
		return 9
	case CardRankT:
		return 10
	case CardRankJ:
		return 11
	case CardRankQ:
		return 12
	case CardRankK:
		return 13
	case CardRankA:
		return 14
	}
	return 0
}

// CardSuit 表示扑克牌花色字节。
type CardSuit byte

// 花色枚举，值与 ASCII 字节一致。
const (
	CardSuitC CardSuit = 'c' // 梅花 Clubs
	CardSuitD CardSuit = 'd' // 方块 Diamonds
	CardSuitH CardSuit = 'h' // 红心 Hearts
	CardSuitS CardSuit = 's' // 黑桃 Spades
)

// Card 是一张扑克牌。
type Card struct {
	Rank CardRank // 点数，见 CardRank 枚举
	Suit CardSuit // 花色，见 CardSuit 枚举
}

// String 返回牌面字符串，例如 "As"（黑桃 A）。
func (c Card) String() string { return string([]byte{byte(c.Rank), byte(c.Suit)}) }
