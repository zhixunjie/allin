import { useEffect } from 'react'
import { wsClient } from '../api/ws'
import { useGameStore } from '../store/game'
import { useAuthStore } from '../store/auth'
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
  SitOutPayload,
  StackUpdatedPayload,
} from '../store/game'
import { useChatStore } from '../store/chat'
import { authAPI } from '../api/http'
import { PlayerAction, WSEventType, WSInternalEvent } from '../types/enums'

function refreshBalance() {
  authAPI.me()
    .then((u) => useAuthStore.getState().updateChipBalance(u.chip_balance))
    .catch(() => {})
}

/**
 * 管理 WebSocket 连接的生命周期，并将服务端消息路由到对应的 store。
 *
 * - roomCode / token 变化时重新建立连接
 * - 组件卸载时断开连接并重置所有游戏状态
 * - 所有 `as XxxPayload` 强制转换集中在此处（WS 边界），store 内部不再做断言
 */
export function useWebSocket(roomCode: string | undefined, token: string | null) {
  // 使用 getState 而非 subscribe，避免 hook 内部产生额外渲染
  const store = useGameStore.getState

  useEffect(() => {
    if (!roomCode || !token) return

    wsClient.connect(roomCode, token)

    // offs 收集所有事件监听的取消函数，cleanup 时统一注销
    const offs: Array<() => void> = []
    const on = (type: string, fn: (p: unknown) => void) => {
      offs.push(wsClient.on(type, fn))
    }

    // ── 服务端 → 客户端（游戏状态更新） ─────────────────────────────
    on(WSEventType.Connected, (p) => {
      const payload = p as ConnectedPayload
      store().applyConnected(payload)
      if (payload.chip_balance != null) {
        useAuthStore.getState().updateChipBalance(payload.chip_balance)
      }
    })
    on(WSEventType.PlayerJoined, (p) => {
      const payload = p as PlayerJoinedPayload
      store().applyPlayerJoined(payload)
      // 自己首次入桌：买入已从账户扣除，刷新余额
      if (payload.player_id === store().myUserId && !payload.is_reconnect && !payload.is_bot) {
        refreshBalance()
      }
    })
    on(WSEventType.PlayerLeft,     (p) => store().applyPlayerLeft(p as PlayerLeftPayload))
    on(WSEventType.GameStarted,    (p) => store().applyGameStarted(p as GameStartedPayload))
    on(WSEventType.HoleCards,      (p) => store().applyHoleCards(p as HoleCardsPayload))
    on(WSEventType.CardsDealt,     (p) => store().applyCardsDealt(p))
    on(WSEventType.StreetStarted,  (p) => store().applyStreetStarted(p as StreetStartedPayload))
    on(WSEventType.ActionRequired, (p) => store().applyActionRequired(p as ActionRequiredPayload))
    on(WSEventType.ActionTaken,    (p) => store().applyActionTaken(p as ActionTakenPayload))
    on(WSEventType.ActionTimeout,  (p) => store().applyActionTaken(p as ActionTakenPayload)) // 超时视同自动行动
    on(WSEventType.Showdown,       (p) => store().applyShowdown(p as ShowdownPayload))
    on(WSEventType.HandResult,     (p) => store().applyHandResult(p as HandResultPayload))
    on(WSEventType.SitOutStatus,   (p) => store().applySitOut(p as SitOutPayload))
    on('stack_updated', (p) => {
      const payload = p as StackUpdatedPayload
      store().applyStackUpdated(payload)
      // 自己补充筹码：账户余额已扣除，刷新显示
      if (payload.player_id === store().myUserId) {
        refreshBalance()
      }
    })

    // ── 服务端 → 客户端（聊天） ──────────────────────────────────────
    on(WSEventType.ChatMessage, (p) => {
      const m = p as { sender_id: string; display_name: string; text: string; ts: number }
      useChatStore.getState().addMessage({
        senderId: m.sender_id,
        displayName: m.display_name,
        text: m.text,
        ts: m.ts,
      })
    })

    // ── 连接建立后加入房间 ────────────────────────────────────────────
    // Open 事件：WS 握手完成时触发（包括断线重连后）
    on(WSInternalEvent.Open, () => {
      wsClient.send(WSEventType.JoinRoom, { room_code: roomCode })
    })
    // 若 WS 已经处于 open 状态（如快速重渲染），直接发送
    if (wsClient.isOpen) {
      wsClient.send(WSEventType.JoinRoom, { room_code: roomCode })
    }

    // ── 清理 ──────────────────────────────────────────────────────────
    return () => {
      offs.forEach((off) => off())
      wsClient.disconnect()
      store().reset()
      useChatStore.getState().clear()
    }
  }, [roomCode, token])
}

/** 向服务端发送玩家操作（fold / check / call / bet / raise / all_in）。 */
export function sendAction(action: PlayerAction, amount = 0) {
  wsClient.send(WSEventType.Action, { action, amount })
}
