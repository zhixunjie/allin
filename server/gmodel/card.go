package gmodel

// Card 是一张扑克牌。
type Card struct {
	Rank byte // 点数：'2'…'9','T','J','Q','K','A'
	Suit byte // 花色：'c'梅花,'d'方块,'h'红心,'s'黑桃
}

// String 返回牌面字符串，例如 "As"（黑桃 A）。
func (c Card) String() string { return string([]byte{c.Rank, c.Suit}) }
