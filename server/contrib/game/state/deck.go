package state

import (
	"math/rand/v2"

	"github.com/allin/server/contrib/game/model"
)

var ranks = []byte{'2', '3', '4', '5', '6', '7', '8', '9', 'T', 'J', 'Q', 'K', 'A'}
var suits = []byte{'c', 'd', 'h', 's'}

// NewShuffledDeck 返回一副新洗好的 52 张牌。
func NewShuffledDeck() []model.Card {
	deck := make([]model.Card, 0, 52)
	for _, r := range ranks {
		for _, s := range suits {
			deck = append(deck, model.Card{Rank: r, Suit: s})
		}
	}
	rand.Shuffle(len(deck), func(i, j int) {
		deck[i], deck[j] = deck[j], deck[i]
	})
	return deck
}
