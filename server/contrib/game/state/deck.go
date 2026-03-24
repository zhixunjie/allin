package state

import (
	"math/rand/v2"

	"github.com/allin/server/gmodel"
)

var ranks = []gmodel.CardRank{
	gmodel.CardRank2, gmodel.CardRank3, gmodel.CardRank4, gmodel.CardRank5,
	gmodel.CardRank6, gmodel.CardRank7, gmodel.CardRank8, gmodel.CardRank9,
	gmodel.CardRankT, gmodel.CardRankJ, gmodel.CardRankQ, gmodel.CardRankK, gmodel.CardRankA,
}
var suits = []gmodel.CardSuit{gmodel.CardSuitC, gmodel.CardSuitD, gmodel.CardSuitH, gmodel.CardSuitS}

// NewShuffledDeck 返回一副新洗好的 52 张牌。
func NewShuffledDeck() []gmodel.Card {
	deck := make([]gmodel.Card, 0, 52)
	for _, r := range ranks {
		for _, s := range suits {
			deck = append(deck, gmodel.Card{Rank: r, Suit: s})
		}
	}
	rand.Shuffle(len(deck), func(i, j int) {
		deck[i], deck[j] = deck[j], deck[i]
	})
	return deck
}
