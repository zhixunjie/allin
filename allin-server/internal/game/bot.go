package game

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/allin/server/internal/ws"
)

var botNames = []string{
	"Alice", "Bob", "Charlie", "Diana", "Eve",
	"Frank", "Grace", "Henry", "Ivy", "Jack",
}

// botUserID generates a deterministic bot user ID for a given room and index.
func botUserID(roomCode string, i int) string {
	return fmt.Sprintf("bot_%s_%d", roomCode, i)
}

// botDisplayName returns a display name for bot index i.
func botDisplayName(i int) string {
	return botNames[i%len(botNames)]
}

// IsBotID returns true if the userID belongs to a bot.
func IsBotID(userID string) bool {
	return strings.HasPrefix(userID, "bot_")
}

// scheduleAIAction starts a goroutine that injects a bot action after a short delay.
// It captures only immutable snapshot values — the write goes through the buffered Inbound channel.
func (e *Engine) scheduleAIAction(p *Player) {
	botID := p.UserID
	currentBet := e.gs.CurrentBet
	playerBet := p.Bet
	stack := p.Stack
	bigBlind := e.gs.Config.BigBlind

	go func() {
		delay := time.Duration(1000+rand.Intn(2000)) * time.Millisecond
		time.Sleep(delay)

		action, amount := decideBotAction(currentBet, playerBet, stack, bigBlind)
		payload, _ := json.Marshal(ws.ActionCmd{Action: action, Amount: amount})

		select {
		case e.hub.Inbound <- ws.InboundMessage{
			SenderID: botID,
			Env: ws.Envelope{
				Type:    ws.CmdAction,
				Payload: payload,
			},
		}:
		default:
			// Inbound buffer full; ValidateAction will reject the stale action anyway.
		}
	}()
}

// decideBotAction returns a simple poker decision for a bot.
//
// Strategy:
//   - No bet to call: 80% check, 20% bet(2×BB)
//   - Facing a bet:   5% fold, 20% raise(2× current), 75% call
func decideBotAction(currentBet, playerBet, stack, bigBlind int64) (string, int64) {
	toCall := currentBet - playerBet
	r := rand.Float64()

	if toCall <= 0 {
		if r < 0.80 {
			return ActionCheck, 0
		}
		betAmt := bigBlind * 2
		if betAmt >= stack {
			return ActionAllIn, 0
		}
		return ActionBet, betAmt
	}

	if r < 0.05 {
		return ActionFold, 0
	}
	if r < 0.25 {
		raiseTotal := currentBet * 2
		if raiseTotal >= playerBet+stack {
			return ActionAllIn, 0
		}
		return ActionRaise, raiseTotal
	}
	if toCall >= stack {
		return ActionAllIn, 0
	}
	return ActionCall, 0
}
