import { useEffect, useRef } from 'react'
import { wsClient } from '../api/ws'
import { useGameStore } from '../store/game'

export function useWebSocket(roomCode: string | undefined, token: string | null) {
  const store = useGameStore.getState
  const connectedRef = useRef(false)

  useEffect(() => {
    if (!roomCode || !token) return

    wsClient.connect(roomCode, token)
    connectedRef.current = true

    const offs: Array<() => void> = []

    const on = (type: string, fn: (p: unknown) => void) => {
      offs.push(wsClient.on(type, fn))
    }

    on('connected', (p) => store().applyConnected(p))
    on('player_joined', (p) => store().applyPlayerJoined(p))
    on('player_left', (p) => store().applyPlayerLeft(p))
    on('game_started', (p) => store().applyGameStarted(p))
    on('hole_cards', (p) => store().applyHoleCards(p))
    on('cards_dealt', (p) => store().applyCardsDealt(p))
    on('street_started', (p) => store().applyStreetStarted(p))
    on('action_required', (p) => store().applyActionRequired(p))
    on('action_taken', (p) => store().applyActionTaken(p))
    on('action_timeout', (p) => store().applyActionTaken(p))
    on('showdown', (p) => store().applyShowdown(p))
    on('hand_result', (p) => store().applyHandResult(p))

    // Send join_room after connecting
    on('__open__', () => {
      wsClient.send('join_room', { room_code: roomCode })
    })

    // Immediately try to join (ws might already be open if reconnecting)
    if (wsClient.isOpen) {
      wsClient.send('join_room', { room_code: roomCode })
    }

    return () => {
      offs.forEach((off) => off())
      wsClient.disconnect()
      store().reset()
      connectedRef.current = false
    }
  }, [roomCode, token])
}

export function sendAction(action: string, amount = 0) {
  wsClient.send('action', { action, amount })
}
