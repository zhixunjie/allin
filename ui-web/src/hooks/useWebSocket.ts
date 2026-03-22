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
import { useRoomStore } from '../store/room'
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
    on(WSEventType.PlayerLeft,     (p) => store().applyPlayerLeft(p as PlayerLeftPayload))     // 玩家离桌（主动离开或踢出）
    on(WSEventType.GameStarted,    (p) => store().applyGameStarted(p as GameStartedPayload))   // 新一手开始：重置座位下注、记录庄/盲位
    on(WSEventType.HoleCards,      (p) => store().applyHoleCards(p as HoleCardsPayload))       // 服务端仅向当事人推送本人手牌
    on(WSEventType.CardsDealt,     (p) => store().applyCardsDealt(p))                          // 广播发牌动作：其他座位显示背面牌占位
    on(WSEventType.StreetStarted,  (p) => store().applyStreetStarted(p as StreetStartedPayload)) // 新街道开始：同步公共牌、底池，清空本街下注
    on(WSEventType.ActionRequired, (p) => store().applyActionRequired(p as ActionRequiredPayload)) // 轮到某玩家行动：启动倒计时并更新跟注/加注参数
    on(WSEventType.ActionTaken,    (p) => store().applyActionTaken(p as ActionTakenPayload))   // 玩家已行动：更新筹码/下注/状态，追加行动日志
    on(WSEventType.ActionTimeout,  (p) => store().applyActionTaken(p as ActionTakenPayload))   // 行动超时：服务端自动弃牌，复用 ActionTaken 路径
    on(WSEventType.Showdown,       (p) => store().applyShowdown(p as ShowdownPayload))         // 摊牌：公开各玩家手牌
    on(WSEventType.HandResult,     (p) => store().applyHandResult(p as HandResultPayload))     // 手牌结算：更新筹码、记录赢家、重置街道为 Idle
    on(WSEventType.SitOutStatus,   (p) => store().applySitOut(p as SitOutPayload))             // 玩家暂离/归座状态变更
    on(WSEventType.ReadyStatus,    (p) => store().applyReadyStatus(p as { ready_count: number; total_count: number })) // 结算间隙准备人数广播

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
      const buyIn = useRoomStore.getState().selectedBuyIn
      wsClient.send(WSEventType.JoinRoom, { room_code: roomCode, ...(buyIn > 0 ? { buy_in: buyIn } : {}) })
    })
    // 若 WS 已经处于 open 状态（如快速重渲染），直接发送
    if (wsClient.isOpen) {
      const buyIn = useRoomStore.getState().selectedBuyIn
      wsClient.send(WSEventType.JoinRoom, { room_code: roomCode, ...(buyIn > 0 ? { buy_in: buyIn } : {}) })
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
