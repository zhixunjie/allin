package state

import (
	"testing"

	"github.com/allin/server/gmodel"
)

func mkPlayer(id string, totalBet int64, folded bool) *model.Player {
	return &model.Player{UserID: id, TotalBet: totalBet, Folded: folded}
}

func seats(players ...*model.Player) [9]*model.Player {
	var s [9]*model.Player
	for i, p := range players {
		s[i] = p
	}
	return s
}

// --- Basic pot tests ---

func TestPot_TwoWay(t *testing.T) {
	s := seats(mkPlayer("A", 100, false), mkPlayer("B", 100, false))
	pots := BuildPots(s)
	if len(pots) != 1 {
		t.Fatalf("expected 1 pot, got %d", len(pots))
	}
	if pots[0].Amount != 200 {
		t.Fatalf("expected pot 200, got %d", pots[0].Amount)
	}
	if len(pots[0].Eligible) != 2 {
		t.Fatalf("expected 2 eligible, got %d", len(pots[0].Eligible))
	}
}

func TestPot_ThreeWay(t *testing.T) {
	s := seats(mkPlayer("A", 200, false), mkPlayer("B", 200, false), mkPlayer("C", 200, false))
	pots := BuildPots(s)
	if len(pots) != 1 || pots[0].Amount != 600 {
		t.Fatalf("expected 1 pot of 600, got %v", pots)
	}
}

func TestPot_AllInSidePot(t *testing.T) {
	// A all-in 100, B has 200, C has 200
	s := seats(mkPlayer("A", 100, false), mkPlayer("B", 200, false), mkPlayer("C", 200, false))
	pots := BuildPots(s)
	if len(pots) != 2 {
		t.Fatalf("expected 2 pots, got %d: %v", len(pots), pots)
	}
	// Main pot: 100*3 = 300
	if pots[0].Amount != 300 {
		t.Fatalf("main pot: expected 300, got %d", pots[0].Amount)
	}
	if len(pots[0].Eligible) != 3 {
		t.Fatalf("main pot: expected 3 eligible, got %d", len(pots[0].Eligible))
	}
	// Side pot: 100*2 = 200
	if pots[1].Amount != 200 {
		t.Fatalf("side pot: expected 200, got %d", pots[1].Amount)
	}
	if len(pots[1].Eligible) != 2 {
		t.Fatalf("side pot: expected 2 eligible, got %d", len(pots[1].Eligible))
	}
	// A not in side pot
	for _, uid := range pots[1].Eligible {
		if uid == "A" {
			t.Fatal("A should not be eligible for side pot")
		}
	}
}

func TestPot_FoldedPlayerContributes(t *testing.T) {
	// A folded (contributed 50), B and C each 200
	s := seats(mkPlayer("A", 50, true), mkPlayer("B", 200, false), mkPlayer("C", 200, false))
	pots := BuildPots(s)
	total := int64(0)
	for _, p := range pots {
		total += p.Amount
	}
	if total != 450 {
		t.Fatalf("expected total 450, got %d", total)
	}
	// A is not eligible for any pot
	for _, pot := range pots {
		for _, uid := range pot.Eligible {
			if uid == "A" {
				t.Fatal("folded player A should not be eligible")
			}
		}
	}
}

func TestPot_TwoAllIns(t *testing.T) {
	// A: 50, B: 100, C: 200
	s := seats(mkPlayer("A", 50, false), mkPlayer("B", 100, false), mkPlayer("C", 200, false))
	pots := BuildPots(s)
	if len(pots) != 3 {
		t.Fatalf("expected 3 pots, got %d: %v", len(pots), pots)
	}
	// Pot 1: 50*3=150, eligible: A,B,C
	if pots[0].Amount != 150 {
		t.Fatalf("pot1: expected 150, got %d", pots[0].Amount)
	}
	// Pot 2: 50*2=100, eligible: B,C
	if pots[1].Amount != 100 {
		t.Fatalf("pot2: expected 100, got %d", pots[1].Amount)
	}
	// Pot 3: 100*1=100, eligible: C
	if pots[2].Amount != 100 {
		t.Fatalf("pot3: expected 100, got %d", pots[2].Amount)
	}
	if len(pots[2].Eligible) != 1 || pots[2].Eligible[0] != "C" {
		t.Fatalf("pot3 eligible should be [C], got %v", pots[2].Eligible)
	}
}

func TestPot_EqualBets(t *testing.T) {
	s := seats(mkPlayer("A", 500, false), mkPlayer("B", 500, false), mkPlayer("C", 500, false), mkPlayer("D", 500, false))
	pots := BuildPots(s)
	if len(pots) != 1 || pots[0].Amount != 2000 {
		t.Fatalf("expected 1 pot of 2000, got %v", pots)
	}
}

func TestPot_SomeFolded(t *testing.T) {
	// A:100 folded, B:100 folded, C:100 not folded, D:100 not folded
	s := seats(mkPlayer("A", 100, true), mkPlayer("B", 100, true), mkPlayer("C", 100, false), mkPlayer("D", 100, false))
	pots := BuildPots(s)
	if len(pots) != 1 {
		t.Fatalf("expected 1 pot, got %d", len(pots))
	}
	if pots[0].Amount != 400 {
		t.Fatalf("expected 400, got %d", pots[0].Amount)
	}
	if len(pots[0].Eligible) != 2 {
		t.Fatalf("expected 2 eligible, got %d", len(pots[0].Eligible))
	}
}

func TestPot_HeadsUp_AllIn(t *testing.T) {
	// A: 200 (all-in), B: 500
	s := seats(mkPlayer("A", 200, false), mkPlayer("B", 500, false))
	pots := BuildPots(s)
	if len(pots) != 2 {
		t.Fatalf("expected 2 pots, got %d: %v", len(pots), pots)
	}
	if pots[0].Amount != 400 {
		t.Fatalf("main pot: expected 400, got %d", pots[0].Amount)
	}
	if pots[1].Amount != 300 {
		t.Fatalf("side pot: expected 300, got %d", pots[1].Amount)
	}
	if len(pots[1].Eligible) != 1 || pots[1].Eligible[0] != "B" {
		t.Fatalf("side pot eligible should be [B], got %v", pots[1].Eligible)
	}
}

func TestPot_SinglePlayer(t *testing.T) {
	s := seats(mkPlayer("A", 100, false))
	pots := BuildPots(s)
	if len(pots) != 1 || pots[0].Amount != 100 {
		t.Fatalf("expected 1 pot of 100, got %v", pots)
	}
}

func TestPot_Empty(t *testing.T) {
	var s [9]*model.Player
	pots := BuildPots(s)
	if len(pots) != 0 {
		t.Fatalf("expected 0 pots, got %d", len(pots))
	}
}

func TestPot_TotalAmountConsistent(t *testing.T) {
	// Complex: A:50, B:150, C:300, D:300 (B folded)
	s := seats(mkPlayer("A", 50, false), mkPlayer("B", 150, true), mkPlayer("C", 300, false), mkPlayer("D", 300, false))
	pots := BuildPots(s)
	total := int64(0)
	for _, p := range pots {
		total += p.Amount
	}
	if total != 800 {
		t.Fatalf("expected total 800, got %d (pots: %v)", total, pots)
	}
}

func TestPot_FiveWay(t *testing.T) {
	s := seats(
		mkPlayer("A", 400, false),
		mkPlayer("B", 400, false),
		mkPlayer("C", 400, false),
		mkPlayer("D", 400, false),
		mkPlayer("E", 400, false),
	)
	pots := BuildPots(s)
	if len(pots) != 1 || pots[0].Amount != 2000 {
		t.Fatalf("expected 1 pot of 2000, got %v", pots)
	}
}

func TestPot_ThreeAllin_Ordered(t *testing.T) {
	// A: 100 all-in, B: 200 all-in, C: 400 all-in, D: 400
	s := seats(
		mkPlayer("A", 100, false),
		mkPlayer("B", 200, false),
		mkPlayer("C", 400, false),
		mkPlayer("D", 400, false),
	)
	pots := BuildPots(s)
	if len(pots) != 3 {
		t.Fatalf("expected 3 pots, got %d: %v", len(pots), pots)
	}
	// Pot1: 100*4 = 400, all eligible
	if pots[0].Amount != 400 || len(pots[0].Eligible) != 4 {
		t.Fatalf("pot1: got amount=%d eligible=%d", pots[0].Amount, len(pots[0].Eligible))
	}
	// Pot2: 100*3 = 300, B,C,D eligible
	if pots[1].Amount != 300 || len(pots[1].Eligible) != 3 {
		t.Fatalf("pot2: got amount=%d eligible=%d", pots[1].Amount, len(pots[1].Eligible))
	}
	// Pot3: 200*2 = 400, C,D eligible
	if pots[2].Amount != 400 || len(pots[2].Eligible) != 2 {
		t.Fatalf("pot3: got amount=%d eligible=%d", pots[2].Amount, len(pots[2].Eligible))
	}
}

func TestPot_AllFolded(t *testing.T) {
	s := seats(mkPlayer("A", 100, true), mkPlayer("B", 100, true))
	pots := BuildPots(s)
	// No eligible players
	for _, p := range pots {
		if len(p.Eligible) > 0 {
			t.Fatalf("no one should be eligible when all folded")
		}
	}
}

func TestPot_ZeroBet(t *testing.T) {
	s := seats(mkPlayer("A", 0, false), mkPlayer("B", 100, false))
	pots := BuildPots(s)
	// A with 0 bet doesn't contribute; B only eligible
	total := int64(0)
	for _, p := range pots {
		total += p.Amount
	}
	if total != 100 {
		t.Fatalf("expected total 100, got %d", total)
	}
}

func TestPot_ManyPlayers_Consistent(t *testing.T) {
	// 9 players, varying bets
	bets := []int64{50, 100, 150, 200, 200, 300, 300, 400, 500}
	s := seats(
		mkPlayer("P1", bets[0], false),
		mkPlayer("P2", bets[1], false),
		mkPlayer("P3", bets[2], false),
		mkPlayer("P4", bets[3], false),
		mkPlayer("P5", bets[4], true), // folded
		mkPlayer("P6", bets[5], false),
		mkPlayer("P7", bets[6], true), // folded
		mkPlayer("P8", bets[7], false),
		mkPlayer("P9", bets[8], false),
	)
	pots := BuildPots(s)
	total := int64(0)
	for _, p := range pots {
		total += p.Amount
	}
	expectedTotal := int64(0)
	for _, b := range bets {
		expectedTotal += b
	}
	if total != expectedTotal {
		t.Fatalf("total mismatch: expected %d, got %d (pots: %v)", expectedTotal, total, pots)
	}
}

func TestPot_UncalledSingleBig(t *testing.T) {
	// A bet 1000 but only B called 200 (B had 200 total bet, A had 1000 total)
	// This simulates AFTER uncalled chips returned to A (A TotalBet = 200 too)
	s := seats(mkPlayer("A", 200, false), mkPlayer("B", 200, false))
	pots := BuildPots(s)
	if pots[0].Amount != 400 {
		t.Fatalf("expected 400, got %d", pots[0].Amount)
	}
}

func TestPot_SixWayComplex(t *testing.T) {
	s := seats(
		mkPlayer("A", 100, false),  // all-in
		mkPlayer("B", 250, false),  // all-in
		mkPlayer("C", 250, true),   // folded
		mkPlayer("D", 500, false),
		mkPlayer("E", 500, false),
		mkPlayer("F", 500, false),
	)
	pots := BuildPots(s)
	total := int64(0)
	for _, p := range pots {
		total += p.Amount
	}
	if total != 100+250+250+500+500+500 {
		t.Fatalf("total mismatch, got %d (pots: %v)", total, pots)
	}
}

func TestPot_AmountNeverNegative(t *testing.T) {
	s := seats(
		mkPlayer("A", 1, false),
		mkPlayer("B", 2, false),
		mkPlayer("C", 3, false),
	)
	pots := BuildPots(s)
	for _, p := range pots {
		if p.Amount < 0 {
			t.Fatalf("negative pot amount %d", p.Amount)
		}
	}
}
