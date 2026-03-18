package game

import (
	"encoding/json"
	"log/slog"
	"time"

	"github.com/allin/server/internal/room"
	"github.com/allin/server/internal/ws"
)

const (
	handStartDelay = 3 * time.Second // delay between hands
)

// Engine drives the game state machine for one room.
// All mutations to GameState happen in the single Run() goroutine.
type Engine struct {
	hub  *ws.Hub
	room *room.Room
	gs   *GameState
	deck []Card
	quit chan struct{}
}

// NewEngine creates an engine for the given hub and room.
func NewEngine(hub *ws.Hub, rm *room.Room) *Engine {
	cfg := rm.Config
	if cfg.ActionTimeSec == 0 {
		cfg.ActionTimeSec = 30
	}
	return &Engine{
		hub:  hub,
		room: rm,
		gs: &GameState{
			Street:     StreetIdle,
			ActionSeat: -1,
			DealerSeat: -1,
			Config:     cfg,
		},
		quit: make(chan struct{}),
	}
}

// Stop signals the engine goroutine to exit.
func (e *Engine) Stop() { close(e.quit) }

// Run is the engine's event loop. Call in a dedicated goroutine.
func (e *Engine) Run() {
	var timerC <-chan time.Time
	var actionTimer *time.Timer

	resetTimer := func(d time.Duration) {
		if actionTimer != nil {
			actionTimer.Stop()
		}
		actionTimer = time.NewTimer(d)
		timerC = actionTimer.C
	}
	stopTimer := func() {
		if actionTimer != nil {
			actionTimer.Stop()
		}
		timerC = nil
	}

	for {
		select {
		case <-e.quit:
			stopTimer()
			return

		case msg := <-e.hub.Inbound:
			e.handleMessage(msg, resetTimer, stopTimer)

		case <-timerC:
			timerC = nil
			e.handleTimeout(resetTimer)
		}
	}
}

func (e *Engine) handleMessage(
	msg ws.InboundMessage,
	resetTimer func(time.Duration),
	stopTimer func(),
) {
	switch msg.Env.Type {
	case ws.CmdJoinRoom:
		e.handleJoinRoom(msg, resetTimer)
	case ws.CmdAction:
		e.handleAction(msg, resetTimer, stopTimer)
	case ws.CmdChat:
		e.handleChat(msg)
	case "disconnect":
		e.handleDisconnect(msg, resetTimer, stopTimer)
	}
}

// ---- Join ----

func (e *Engine) handleJoinRoom(msg ws.InboundMessage, resetTimer func(time.Duration)) {
	// If already seated, ignore.
	if e.gs.FindPlayer(msg.SenderID) != nil {
		return
	}

	if e.gs.SeatedCount() >= e.room.Config.MaxPlayers {
		e.hub.SendTo(msg.SenderID, ws.MustEvent(ws.TypeError, ws.ErrorPayload{
			Code: "room_full", Message: "room is full",
		}))
		return
	}

	p := &Player{
		UserID:      msg.SenderID,
		DisplayName: msg.DisplayName,
		Stack:       e.room.Config.MaxBuyIn,
	}
	e.gs.SeatPlayer(p)

	e.hub.Broadcast(ws.MustEvent(ws.TypePlayerJoined, ws.PlayerJoinedPayload{
		PlayerID:    p.UserID,
		DisplayName: p.DisplayName,
		SeatIndex:   p.SeatIndex,
		Stack:       p.Stack,
		IsReconnect: false,
	}))

	// Send current snapshot to the joining player.
	e.sendSnapshot(msg.SenderID)

	// Auto-start if ≥2 eligible players and no hand running.
	if e.gs.Street == StreetIdle && len(e.gs.EligibleToStart()) >= 2 {
		resetTimer(handStartDelay)
	}
}

// ---- Disconnect ----

func (e *Engine) handleDisconnect(msg ws.InboundMessage, resetTimer func(time.Duration), stopTimer func()) {
	p := e.gs.FindPlayer(msg.SenderID)
	if p == nil {
		return
	}

	e.hub.Broadcast(ws.MustEvent(ws.TypePlayerLeft, ws.PlayerLeftPayload{
		PlayerID:  p.UserID,
		SeatIndex: p.SeatIndex,
	}))

	if e.gs.Street == StreetIdle {
		e.gs.UnseatPlayer(msg.SenderID)
		return
	}

	// In an active hand: fold for them if it's their turn, mark sit-out otherwise.
	if e.gs.ActionSeat == p.SeatIndex {
		ApplyAction(e.gs, p.UserID, ActionFold, 0)
		e.gs.UnseatPlayer(msg.SenderID)
		e.advanceOrEnd(resetTimer, stopTimer)
	} else {
		p.Folded = true
		e.gs.UnseatPlayer(msg.SenderID)
		e.checkHandOver(resetTimer, stopTimer)
	}
}

// ---- Action ----

func (e *Engine) handleAction(
	msg ws.InboundMessage,
	resetTimer func(time.Duration),
	stopTimer func(),
) {
	var cmd ws.ActionCmd
	if err := json.Unmarshal(msg.Env.Payload, &cmd); err != nil {
		e.sendError(msg.SenderID, "bad_payload", "invalid action payload", msg.Env.Seq)
		return
	}

	if err := ValidateAction(e.gs, msg.SenderID, cmd.Action, cmd.Amount); err != nil {
		e.sendError(msg.SenderID, "invalid_action", err.Error(), msg.Env.Seq)
		return
	}

	ApplyAction(e.gs, msg.SenderID, cmd.Action, cmd.Amount)

	p := e.gs.FindPlayer(msg.SenderID)
	var displayAmount int64
	if p != nil {
		displayAmount = p.Bet
	}

	e.hub.Broadcast(ws.MustEvent(ws.TypeActionTaken, ws.ActionTakenPayload{
		PlayerID:  msg.SenderID,
		Action:    cmd.Action,
		Amount:    displayAmount,
		Stack:     p.Stack,
		TotalPot:  e.gs.TotalPot(),
	}))

	stopTimer()
	e.advanceOrEnd(resetTimer, stopTimer)
}

// ---- Chat ----

func (e *Engine) handleChat(msg ws.InboundMessage) {
	var cmd ws.ChatCmd
	if err := json.Unmarshal(msg.Env.Payload, &cmd); err != nil {
		return
	}
	if len(cmd.Text) == 0 || len(cmd.Text) > 200 {
		return
	}
	e.hub.Broadcast(ws.MustEvent(ws.TypeChatMessage, ws.ChatPayload{
		SenderID:    msg.SenderID,
		DisplayName: msg.DisplayName,
		Text:        cmd.Text,
		Ts:          time.Now().UnixMilli(),
	}))
}

// ---- Timeout ----

func (e *Engine) handleTimeout(resetTimer func(time.Duration)) {
	if e.gs.Street == StreetIdle {
		// Hand start timer fired.
		if len(e.gs.EligibleToStart()) >= 2 {
			e.startHand(resetTimer)
		}
		return
	}

	// Action timeout: auto fold or check.
	if e.gs.ActionSeat == -1 {
		return
	}
	p := e.gs.Seats[e.gs.ActionSeat]
	if p == nil {
		return
	}

	action := ActionFold
	if p.Bet >= e.gs.CurrentBet {
		action = ActionCheck
	}

	e.hub.Broadcast(ws.MustEvent(ws.TypeActionTimeout, ws.ActionTimeoutPayload{
		PlayerID: p.UserID,
		Action:   action,
	}))

	ApplyAction(e.gs, p.UserID, action, 0)
	e.advanceOrEnd(resetTimer, func() {})
}

// ---- Hand flow ----

func (e *Engine) startHand(resetTimer func(time.Duration)) {
	e.gs.HandNum++
	e.gs.Street = StreetPreFlop
	e.gs.Community = nil

	eligible := e.gs.EligibleToStart()

	// Advance dealer button.
	e.gs.DealerSeat = e.nextEligibleSeatAfter(e.gs.DealerSeat, eligible)

	// Assign SB and BB.
	if len(eligible) == 2 {
		// Heads-up: dealer = SB, other = BB.
		e.gs.SBSeat = e.gs.DealerSeat
		e.gs.BBSeat = e.nextEligibleSeatAfter(e.gs.DealerSeat, eligible)
	} else {
		e.gs.SBSeat = e.nextEligibleSeatAfter(e.gs.DealerSeat, eligible)
		e.gs.BBSeat = e.nextEligibleSeatAfter(e.gs.SBSeat, eligible)
	}

	// Reset player state.
	for _, p := range e.gs.Seats {
		if p != nil {
			p.Bet = 0
			p.TotalBet = 0
			p.Folded = false
			p.AllIn = false
			p.ActedThisStreet = false
			p.Hole = [2]Card{}
		}
	}

	// Post blinds.
	sb := e.gs.Seats[e.gs.SBSeat]
	bb := e.gs.Seats[e.gs.BBSeat]

	postBlind(sb, e.gs.Config.SmallBlind)
	postBlind(bb, e.gs.Config.BigBlind)

	e.gs.CurrentBet = e.gs.Config.BigBlind
	e.gs.MinRaise = e.gs.Config.BigBlind

	// Shuffle and deal.
	e.deck = newDeck()
	e.dealHoleCards()

	// Broadcast game_started.
	e.hub.Broadcast(ws.MustEvent(ws.TypeGameStarted, ws.GameStartedPayload{
		HandNum:    e.gs.HandNum,
		DealerSeat: e.gs.DealerSeat,
		SBSeat:     e.gs.SBSeat,
		BBSeat:     e.gs.BBSeat,
		SmallBlind: e.gs.Config.SmallBlind,
		BigBlind:   e.gs.Config.BigBlind,
	}))

	// Send private hole cards.
	for _, p := range e.gs.Seats {
		if p == nil || p.SitOut {
			continue
		}
		e.hub.SendTo(p.UserID, ws.MustEvent(ws.TypeHoleCards, ws.HoleCardsPayload{
			PlayerID: p.UserID,
			Hole:     []string{p.Hole[0].String(), p.Hole[1].String()},
		}))
	}

	// Broadcast cards_dealt (opaque — just seat indices).
	var dealtSeats []int
	for _, p := range e.gs.Seats {
		if p != nil && !p.SitOut {
			dealtSeats = append(dealtSeats, p.SeatIndex)
		}
	}
	e.hub.Broadcast(ws.MustEvent(ws.TypeCardsDealt, ws.CardsDealtPayload{Seats: dealtSeats}))

	// Pre-flop: action starts left of BB.
	var firstActor int
	if len(eligible) == 2 {
		firstActor = e.gs.DealerSeat // HU: dealer (SB) acts first preflop
	} else {
		firstActor = e.nextEligibleSeatAfter(e.gs.BBSeat, eligible)
	}
	e.gs.ActionSeat = firstActor

	e.broadcastActionRequired(resetTimer)
}

func postBlind(p *Player, amount int64) {
	if p == nil {
		return
	}
	if amount > p.Stack {
		amount = p.Stack
		p.AllIn = true
	}
	p.Bet += amount
	p.TotalBet += amount
	p.Stack -= amount
}

func (e *Engine) dealHoleCards() {
	idx := 0
	for round := 0; round < 2; round++ {
		for seat := 0; seat < 9; seat++ {
			p := e.gs.Seats[seat]
			if p == nil || p.SitOut {
				continue
			}
			p.Hole[round] = e.deck[idx]
			idx++
		}
	}
	e.deck = e.deck[idx:]
}

func (e *Engine) dealCommunity(n int) {
	for i := 0; i < n; i++ {
		e.gs.Community = append(e.gs.Community, e.deck[0])
		e.deck = e.deck[1:]
	}
}

// advanceOrEnd: after an action, determine next step.
func (e *Engine) advanceOrEnd(resetTimer func(time.Duration), stopTimer func()) {
	active := e.gs.ActivePlayers()

	// Only 1 player left → award pot immediately.
	if len(active) == 1 {
		e.awardUncontested(active[0], resetTimer)
		return
	}

	// Betting round over?
	if e.gs.BettingRoundOver() {
		e.nextStreet(resetTimer)
		return
	}

	// Find next player who can act.
	next := e.gs.nextActableSeat(e.gs.ActionSeat)
	if next == -1 {
		// All remaining players are all-in → run out the board.
		e.nextStreet(resetTimer)
		return
	}
	e.gs.ActionSeat = next
	e.broadcastActionRequired(resetTimer)
}

func (e *Engine) nextStreet(resetTimer func(time.Duration)) {
	// Reset bets for new street.
	for _, p := range e.gs.Seats {
		if p != nil {
			p.Bet = 0
			p.ActedThisStreet = false
		}
	}
	e.gs.CurrentBet = 0
	e.gs.MinRaise = e.gs.Config.BigBlind

	switch e.gs.Street {
	case StreetPreFlop:
		e.gs.Street = StreetFlop
		e.dealCommunity(3)
	case StreetFlop:
		e.gs.Street = StreetTurn
		e.dealCommunity(1)
	case StreetTurn:
		e.gs.Street = StreetRiver
		e.dealCommunity(1)
	case StreetRiver:
		e.runShowdown(resetTimer)
		return
	}

	e.hub.Broadcast(ws.MustEvent(ws.TypeStreetStarted, ws.StreetStartedPayload{
		Street:    e.gs.Street.String(),
		Community: cardsToStrings(e.gs.Community),
		Pot:       e.gs.TotalPot(),
	}))

	// Check if everyone is all-in (skip action, go to next street).
	if len(e.gs.CanAct()) == 0 {
		e.gs.ActionSeat = -1
		resetTimer(2 * time.Second) // short delay then auto-advance
		return
	}

	// Post-flop: action starts left of dealer.
	eligible := e.gs.ActivePlayers()
	e.gs.ActionSeat = e.nextEligibleSeatAfter(e.gs.DealerSeat, eligible)
	e.broadcastActionRequired(resetTimer)
}

// CanAct returns players who are active and not all-in.
func (gs *GameState) CanAct() []*Player {
	var out []*Player
	for _, p := range gs.Seats {
		if p != nil && !p.Folded && !p.SitOut && !p.AllIn {
			out = append(out, p)
		}
	}
	return out
}

func (e *Engine) broadcastActionRequired(resetTimer func(time.Duration)) {
	p := e.gs.Seats[e.gs.ActionSeat]
	if p == nil {
		return
	}
	deadline := time.Now().Add(time.Duration(e.gs.Config.ActionTimeSec) * time.Second)

	e.hub.Broadcast(ws.MustEvent(ws.TypeActionRequired, ws.ActionRequiredPayload{
		PlayerID:    p.UserID,
		SeatIndex:   p.SeatIndex,
		DeadlineTs:  deadline.UnixMilli(),
		CurrentBet:  e.gs.CurrentBet,
		CallAmount:  max64(0, e.gs.CurrentBet-p.Bet),
		MinRaise:    e.gs.CurrentBet + e.gs.MinRaise,
		Stack:       p.Stack,
		Pot:         e.gs.TotalPot(),
	}))

	resetTimer(time.Duration(e.gs.Config.ActionTimeSec) * time.Second)
}

// ---- Showdown ----

func (e *Engine) runShowdown(resetTimer func(time.Duration)) {
	e.gs.Street = StreetShowdown
	e.gs.ActionSeat = -1

	// Reveal all non-folded hands.
	type reveal struct {
		PlayerID    string `json:"player_id"`
		SeatIndex   int    `json:"seat_index"`
		Hole        []string `json:"hole"`
		HandName    string `json:"hand_name"`
	}
	var reveals []reveal
	for _, p := range e.gs.Seats {
		if p != nil && !p.Folded && !p.SitOut {
			_, handName := EvaluateHand(p.Hole, e.gs.Community)
			reveals = append(reveals, reveal{
				PlayerID:  p.UserID,
				SeatIndex: p.SeatIndex,
				Hole:      []string{p.Hole[0].String(), p.Hole[1].String()},
				HandName:  handName,
			})
		}
	}

	rawReveals, _ := json.Marshal(reveals)
	e.hub.Broadcast(ws.MustEvent(ws.TypeShowdown, json.RawMessage(rawReveals)))

	// Build pots and award winners.
	pots := BuildPots(e.gs.Seats)
	type winEntry struct {
		PlayerID string `json:"player_id"`
		Amount   int64  `json:"amount"`
	}
	var winners []winEntry

	for _, pot := range pots {
		// Find best hand among eligible players.
		bestRank := uint32(0xFFFFFFFF)
		var bestPlayers []string
		for _, uid := range pot.Eligible {
			p := e.gs.FindPlayer(uid)
			if p == nil || p.Folded {
				continue
			}
			rank, _ := EvaluateHand(p.Hole, e.gs.Community)
			if rank < bestRank {
				bestRank = rank
				bestPlayers = []string{uid}
			} else if rank == bestRank {
				bestPlayers = append(bestPlayers, uid)
			}
		}
		// Split pot among ties.
		share := pot.Amount / int64(len(bestPlayers))
		remainder := pot.Amount % int64(len(bestPlayers))
		for i, uid := range bestPlayers {
			award := share
			if i == 0 {
				award += remainder // give remainder to first winner
			}
			p := e.gs.FindPlayer(uid)
			if p != nil {
				p.Stack += award
			}
			winners = append(winners, winEntry{PlayerID: uid, Amount: award})
		}
	}

	// Build hand_result seats.
	type resultSeat struct {
		PlayerID string `json:"player_id"`
		Stack    int64  `json:"stack"`
	}
	var resultSeats []resultSeat
	for _, p := range e.gs.Seats {
		if p != nil {
			resultSeats = append(resultSeats, resultSeat{p.UserID, p.Stack})
		}
	}

	rawResult, _ := json.Marshal(struct {
		Winners []winEntry   `json:"winners"`
		Seats   []resultSeat `json:"seats"`
	}{winners, resultSeats})
	e.hub.Broadcast(ws.MustEvent(ws.TypeHandResult, json.RawMessage(rawResult)))

	slog.Info("game: hand complete", "room", e.room.Code, "hand", e.gs.HandNum)

	// Schedule next hand.
	e.gs.Street = StreetIdle
	e.gs.ActionSeat = -1
	if len(e.gs.EligibleToStart()) >= 2 {
		resetTimer(handStartDelay)
	}
}

// awardUncontested gives the pot to the last active player (everyone else folded).
func (e *Engine) awardUncontested(winner *Player, resetTimer func(time.Duration)) {
	// Return uncalled bet if winner's bet exceeds the second-highest bet.
	maxOther := int64(0)
	for _, p := range e.gs.Seats {
		if p != nil && p.UserID != winner.UserID && p.TotalBet > maxOther {
			maxOther = p.TotalBet
		}
	}
	uncalled := winner.TotalBet - maxOther
	if uncalled > 0 {
		winner.Stack += uncalled
		winner.TotalBet -= uncalled
	}

	total := int64(0)
	for _, p := range e.gs.Seats {
		if p != nil {
			total += p.TotalBet
		}
	}
	winner.Stack += total

	e.hub.Broadcast(ws.MustEvent(ws.TypeHandResult, struct {
		Winners []struct {
			PlayerID string `json:"player_id"`
			Amount   int64  `json:"amount"`
		} `json:"winners"`
	}{
		Winners: []struct {
			PlayerID string `json:"player_id"`
			Amount   int64  `json:"amount"`
		}{{winner.UserID, total}},
	}))

	slog.Info("game: uncontested pot", "room", e.room.Code, "winner", winner.UserID, "amount", total)

	e.gs.Street = StreetIdle
	e.gs.ActionSeat = -1
	if len(e.gs.EligibleToStart()) >= 2 {
		resetTimer(handStartDelay)
	}
}

// checkHandOver checks if the hand ended after a disconnect-fold.
func (e *Engine) checkHandOver(resetTimer func(time.Duration), stopTimer func()) {
	active := e.gs.ActivePlayers()
	if len(active) == 1 {
		e.awardUncontested(active[0], resetTimer)
	}
}

// ---- Helpers ----

func (e *Engine) nextEligibleSeatAfter(from int, eligible []*Player) int {
	if from == -1 {
		if len(eligible) > 0 {
			return eligible[0].SeatIndex
		}
		return 0
	}
	for i := 1; i <= 9; i++ {
		idx := (from + i) % 9
		for _, p := range eligible {
			if p.SeatIndex == idx {
				return idx
			}
		}
	}
	return eligible[0].SeatIndex
}

func (e *Engine) sendSnapshot(userID string) {
	snap := e.gs.Snapshot(userID)
	e.hub.SendTo(userID, ws.MustEvent(ws.TypeConnected, ws.ConnectedPayload{
		PlayerID:     userID,
		DisplayName:  e.hub.DisplayName(userID),
		RoomCode:     e.room.Code,
		GameSnapshot: &snap,
	}))
}

func (e *Engine) sendError(userID, code, msg string, refSeq int64) {
	e.hub.SendTo(userID, ws.MustEvent(ws.TypeError, ws.ErrorPayload{
		Code: code, Message: msg, RefSeq: refSeq,
	}))
}

func max64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
