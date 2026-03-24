package gmodel

// CardRank 表示扑克牌点数字节（'2'…'9','T','J','Q','K','A'）。
type CardRank byte

// Val 将点数字节转换为整数值（'2'→2 … 'A'→14），未知点数返回 0。
func (r CardRank) Val() int {
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

// Card 是一张扑克牌。
type Card struct {
	Rank CardRank // 点数：'2'…'9','T','J','Q','K','A'
	Suit byte     // 花色：'c'梅花,'d'方块,'h'红心,'s'黑桃
}

// String 返回牌面字符串，例如 "As"（黑桃 A）。
func (c Card) String() string { return string([]byte{byte(c.Rank), c.Suit}) }
