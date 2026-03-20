import { useEffect, useRef } from 'react'
import { wsClient } from '../api/ws'
import { useGameStore } from '../store/game'
import type {
  ConnectedPayload,
  GameStartedPayload,
  HoleCardsPayload,
  StreetStartedPayload,
  ActionRequiredPayload,
  ActionTakenPayload,
  ShowdownPayload,
  HandResultPayload,
  PlayerJoinedPayload,
  PlayerLeftPayload,
} from '../store/game'
import { useChatStore } from '../store/chat'
import { PlayerAction, WSEventType, WSInternalEvent } from '../types/enums'

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

    on(WSEventType.Connected,       (p) => store().applyConnected(p as ConnectedPayload))
    on(WSEventType.PlayerJoined,    (p) => store().applyPlayerJoined(p as PlayerJoinedPayload))
    on(WSEventType.PlayerLeft,      (p) => store().applyPlayerLeft(p as PlayerLeftPayload))
    on(WSEventType.GameStarted,     (p) => store().applyGameStarted(p as GameStartedPayload))
    on(WSEventType.HoleCards,       (p) => store().applyHoleCards(p as HoleCardsPayload))
    on(WSEventType.CardsDealt,      (p) => store().applyCardsDealt(p))
    on(WSEventType.StreetStarted,   (p) => store().applyStreetStarted(p as StreetStartedPayload))
    on(WSEventType.ActionRequired,  (p) => store().applyActionRequired(p as ActionRequiredPayload))
    on(WSEventType.ActionTaken,     (p) => store().applyActionTaken(p as ActionTakenPayload))
    on(WSEventType.ActionTimeout,   (p) => store().applyActionTaken(p as ActionTakenPayload))
    on(WSEventType.Showdown,        (p) => store().applyShowdown(p as ShowdownPayload))
    on(WSEventType.HandResult,      (p) => store().applyHandResult(p as HandResultPayload))
    on(WSEventType.ChatMessage, (p) => {
      const m = p as { sender_id: string; display_name: string; text: string; ts: number }
      useChatStore.getState().addMessage({
        senderId: m.sender_id,
        displayName: m.display_name,
        text: m.text,
        ts: m.ts,
      })
    })

    // 连接建立后发送 join_room
    on(WSInternalEvent.Open, () => {
      wsClient.send(WSEventType.JoinRoom, { room_code: roomCode })
    })

    // 立即尝试加入（重连时 ws 可能已经打开）
    if (wsClient.isOpen) {
      wsClient.send(WSEventType.JoinRoom, { room_code: roomCode })
    }

    return () => {
      offs.forEach((off) => off())
      wsClient.disconnect()
      store().reset()
      useChatStore.getState().clear()
      connectedRef.current = false
    }
  }, [roomCode, token])
}

export function sendAction(action: PlayerAction, amount = 0) {
  wsClient.send(WSEventType.Action, { action, amount })
}
