package model

// InvalidHandRank 是 EvaluateHand 在无法评估时返回的哨兵值（uint32 最大值）。
const InvalidHandRank uint32 = 0xFFFFFFFF

// HandCategory 手牌类别（由评估等级右移 20 位得到），值越小手牌越强。
type HandCategory uint32

const (
	CatStraightFlush HandCategory = iota // 0 (0000): 同花顺 / 皇家同花顺
	CatFourOfAKind                       // 1 (0001): 四条
	CatFullHouse                         // 2 (0010): 葫芦
	CatFlush                             // 3 (0011): 同花
	CatStraight                          // 4 (0100): 顺子
	CatThreeOfAKind                      // 5 (0101): 三条
	CatTwoPair                           // 6 (0110): 两对
	CatOnePair                           // 7 (0111): 一对
	CatHighCard                          // 8 (1000): 高牌
)
