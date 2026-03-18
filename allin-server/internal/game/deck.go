package game

import "math/rand/v2"

var ranks = []byte{'2', '3', '4', '5', '6', '7', '8', '9', 'T', 'J', 'Q', 'K', 'A'}
var suits = []byte{'c', 'd', 'h', 's'}

// newDeck returns a freshly shuffled 52-card deck.
func newDeck() []Card {
	deck := make([]Card, 0, 52)
	for _, r := range ranks {
		for _, s := range suits {
			deck = append(deck, Card{Rank: r, Suit: s})
		}
	}
	rand.Shuffle(len(deck), func(i, j int) {
		deck[i], deck[j] = deck[j], deck[i]
	})
	return deck
}
