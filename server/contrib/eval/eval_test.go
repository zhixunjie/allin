package eval

import (
	"testing"

	"github.com/allin/server/gmodel"
)

func c(r gmodel.CardRank, s gmodel.CardSuit) gmodel.Card { return gmodel.Card{Rank: r, Suit: s} }

func TestRoyalFlush(t *testing.T) {
	hand := [7]gmodel.Card{c('A', 's'), c('K', 's'), c('Q', 's'), c('J', 's'), c('T', 's'), c('2', 'c'), c('3', 'd')}
	rank := Evaluate7(hand)
	if gmodel.HandCategory(rank>>20) != gmodel.CatStraightFlush {
		t.Fatalf("expected Straight Flush (Royal), got category %d, rank %d, desc %q", rank>>20, rank, Describe(rank))
	}
	if Describe(rank) != "Royal Flush" {
		t.Fatalf("expected Royal Flush, got %q", Describe(rank))
	}
}

func TestStraightFlush(t *testing.T) {
	hand := [7]gmodel.Card{c('9', 'h'), c('8', 'h'), c('7', 'h'), c('6', 'h'), c('5', 'h'), c('A', 'c'), c('K', 'd')}
	rank := Evaluate7(hand)
	if gmodel.HandCategory(rank>>20) != gmodel.CatStraightFlush {
		t.Fatalf("expected Straight Flush, got %q (cat %d)", Describe(rank), rank>>20)
	}
}

func TestFourOfAKind(t *testing.T) {
	hand := [7]gmodel.Card{c('A', 's'), c('A', 'h'), c('A', 'd'), c('A', 'c'), c('K', 's'), c('2', 'c'), c('3', 'd')}
	rank := Evaluate7(hand)
	if Describe(rank) != "Four of a Kind" {
		t.Fatalf("expected Four of a Kind, got %q", Describe(rank))
	}
}

func TestFullHouse(t *testing.T) {
	hand := [7]gmodel.Card{c('K', 's'), c('K', 'h'), c('K', 'd'), c('Q', 'c'), c('Q', 's'), c('2', 'c'), c('3', 'd')}
	rank := Evaluate7(hand)
	if Describe(rank) != "Full House" {
		t.Fatalf("expected Full House, got %q", Describe(rank))
	}
}

func TestFlush(t *testing.T) {
	hand := [7]gmodel.Card{c('A', 'c'), c('T', 'c'), c('7', 'c'), c('4', 'c'), c('2', 'c'), c('K', 'd'), c('Q', 's')}
	rank := Evaluate7(hand)
	if Describe(rank) != "Flush" {
		t.Fatalf("expected Flush, got %q", Describe(rank))
	}
}

func TestStraight(t *testing.T) {
	hand := [7]gmodel.Card{c('9', 's'), c('8', 'h'), c('7', 'd'), c('6', 'c'), c('5', 's'), c('A', 'c'), c('K', 'd')}
	rank := Evaluate7(hand)
	if Describe(rank) != "Straight" {
		t.Fatalf("expected Straight, got %q", Describe(rank))
	}
}

func TestWheelStraight(t *testing.T) {
	hand := [7]gmodel.Card{c('A', 's'), c('2', 'h'), c('3', 'd'), c('4', 'c'), c('5', 's'), c('K', 'c'), c('Q', 'd')}
	rank := Evaluate7(hand)
	if Describe(rank) != "Straight" {
		t.Fatalf("wheel: expected Straight, got %q", Describe(rank))
	}
}

func TestThreeOfAKind(t *testing.T) {
	hand := [7]gmodel.Card{c('J', 's'), c('J', 'h'), c('J', 'd'), c('A', 'c'), c('K', 's'), c('2', 'c'), c('3', 'd')}
	rank := Evaluate7(hand)
	if Describe(rank) != "Three of a Kind" {
		t.Fatalf("expected Three of a Kind, got %q", Describe(rank))
	}
}

func TestTwoPair(t *testing.T) {
	hand := [7]gmodel.Card{c('A', 's'), c('A', 'h'), c('K', 'd'), c('K', 'c'), c('Q', 's'), c('2', 'c'), c('3', 'd')}
	rank := Evaluate7(hand)
	if Describe(rank) != "Two Pair" {
		t.Fatalf("expected Two Pair, got %q", Describe(rank))
	}
}

func TestOnePair(t *testing.T) {
	hand := [7]gmodel.Card{c('A', 's'), c('A', 'h'), c('K', 'd'), c('Q', 'c'), c('J', 's'), c('2', 'c'), c('3', 'd')}
	rank := Evaluate7(hand)
	if Describe(rank) != "One Pair" {
		t.Fatalf("expected One Pair, got %q", Describe(rank))
	}
}

func TestHighCard(t *testing.T) {
	hand := [7]gmodel.Card{c('A', 's'), c('K', 'h'), c('Q', 'd'), c('J', 'c'), c('9', 's'), c('2', 'c'), c('4', 'd')}
	rank := Evaluate7(hand)
	if Describe(rank) != "High Card" {
		t.Fatalf("expected High Card, got %q", Describe(rank))
	}
}

// Ordering tests: stronger hand must have lower rank.

func TestOrdering_SFBeatsQuads(t *testing.T) {
	sf := Evaluate7([7]gmodel.Card{c('9', 'h'), c('8', 'h'), c('7', 'h'), c('6', 'h'), c('5', 'h'), c('A', 'c'), c('K', 'd')})
	quads := Evaluate7([7]gmodel.Card{c('A', 's'), c('A', 'h'), c('A', 'd'), c('A', 'c'), c('K', 's'), c('2', 'c'), c('3', 'd')})
	if sf >= quads {
		t.Fatalf("SF rank %d should be < quads rank %d", sf, quads)
	}
}

func TestOrdering_QuadsBeatsFullHouse(t *testing.T) {
	quads := Evaluate7([7]gmodel.Card{c('A', 's'), c('A', 'h'), c('A', 'd'), c('A', 'c'), c('K', 's'), c('2', 'c'), c('3', 'd')})
	fh := Evaluate7([7]gmodel.Card{c('K', 's'), c('K', 'h'), c('K', 'd'), c('Q', 'c'), c('Q', 's'), c('2', 'c'), c('3', 'd')})
	if quads >= fh {
		t.Fatalf("quads rank %d should be < full house rank %d", quads, fh)
	}
}

func TestOrdering_HigherQuadsWin(t *testing.T) {
	acesQuads := Evaluate7([7]gmodel.Card{c('A', 's'), c('A', 'h'), c('A', 'd'), c('A', 'c'), c('K', 's'), c('2', 'c'), c('3', 'd')})
	kingsQuads := Evaluate7([7]gmodel.Card{c('K', 's'), c('K', 'h'), c('K', 'd'), c('K', 'c'), c('A', 's'), c('2', 'c'), c('3', 'd')})
	if acesQuads >= kingsQuads {
		t.Fatalf("aces quads rank %d should be < kings quads rank %d", acesQuads, kingsQuads)
	}
}

func TestOrdering_BestFlush(t *testing.T) {
	aceHighFlush := Evaluate7([7]gmodel.Card{c('A', 'c'), c('K', 'c'), c('Q', 'c'), c('J', 'c'), c('9', 'c'), c('2', 'd'), c('3', 's')})
	kingHighFlush := Evaluate7([7]gmodel.Card{c('K', 'c'), c('Q', 'c'), c('J', 'c'), c('T', 'c'), c('8', 'c'), c('2', 'd'), c('3', 's')})
	if aceHighFlush >= kingHighFlush {
		t.Fatalf("ace-high flush should beat king-high flush")
	}
}
