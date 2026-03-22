import { useCallback, useEffect, useRef, useState } from 'react'
import { useGameStore } from '../../store/game'
import type { RoundPlayerDetail } from '../../store/game'
import { useAuthStore } from '../../store/auth'
import { useGameState } from '../../hooks/useGameState'
import { wsClient } from '../../api/ws'
import { WSEventType } from '../../types/enums'
import { RANK_DISPLAY, SUIT_SYMBOLS } from '../../pixi/config/card'

// ── 牌型中文名映射 ───────────────────────────────────────
const HAND_ZH: Record<string, string> = {
  'High Card':       '高牌',
  'One Pair':        '一对',
  'Two Pair':        '两对',
  'Three of a Kind': '三条',
  'Straight':        '顺子',
  'Flush':           '同花',
  'Full House':      '葫芦',
  'Four of a Kind':  '四条',
  'Straight Flush':  '同花顺',
  'Royal Flush':     '皇家同花顺',
}

function handZh(name?: string) {
  if (!name) return ''
  return HAND_ZH[name] ?? name
}

// ── 单张扑克牌 ──────────────────────────────────────────
function PokerCard({ code, highlight, small }: {
  code: string
  highlight?: boolean
  small?: boolean
}) {
  const base = small
    ? 'w-8 h-11 rounded text-[9px]'
    : 'w-14 h-20 rounded-lg text-sm'

  if (!code || code === '?') {
    return (
      <div className={`${base} bg-[#1a2a5e] border border-[#2a3f8f] flex items-center justify-center text-[#4060cc]/40 flex-shrink-0`}>
        ♠
      </div>
    )
  }

  const rank = code.slice(0, -1)
  const suit = code.slice(-1)
  const isRed = suit === 'h' || suit === 'd'
  const color = isRed ? 'text-red-500' : 'text-gray-900'
  const border = highlight
    ? 'border-2 border-amber-400 shadow-[0_0_10px_rgba(212,175,55,0.5)]'
    : 'border border-gray-300/70 shadow-sm'

  return (
    <div className={`${base} ${border} bg-white flex flex-col items-center justify-center relative flex-shrink-0`}>
      <span className={`absolute top-0.5 left-1 font-black leading-none ${color} ${small ? 'text-[10px]' : 'text-sm'}`}>
        {RANK_DISPLAY[rank] ?? rank}
      </span>
      <span className={`${color} ${small ? 'text-lg' : 'text-3xl'} leading-none mt-1`}>
        {SUIT_SYMBOLS[suit] ?? suit}
      </span>
    </div>
  )
}

// ── 玩家行 ──────────────────────────────────────────────
function PlayerRow({ player }: { player: RoundPlayerDetail }) {
  const avatarUrl = `https://api.dicebear.com/9.x/toon-head/svg?seed=${encodeURIComponent(player.player_id)}`
  const handLabel = player.folded
    ? '弃牌'
    : player.hand_name
      ? `${handZh(player.hand_name)}`
      : ''

  return (
    <div className={[
      'flex items-center gap-4 px-4 py-3 rounded-xl border transition-all',
      player.is_winner
        ? 'bg-amber-500/10 border-amber-500/40'
        : player.folded
          ? 'bg-white/[0.02] border-white/5 opacity-40'
          : 'bg-white/[0.03] border-white/8',
    ].join(' ')}>

      {/* 头像（含皇冠） */}
      <div className="relative flex-shrink-0">
        {player.is_winner && (
          <span className="absolute -top-3 left-1/2 -translate-x-1/2 text-base leading-none">👑</span>
        )}
        <div className={`w-9 h-9 rounded-full overflow-hidden bg-white/10 ${player.is_winner ? 'ring-2 ring-amber-400 ring-offset-1 ring-offset-[#0a1018]' : ''}`}>
          <img src={avatarUrl} alt={player.display_name} className="w-full h-full object-cover" />
        </div>
      </div>

      {/* 名字 */}
      <span className={`flex-1 text-sm font-bold truncate ${player.is_winner ? 'text-amber-300' : 'text-white/80'}`}>
        {player.display_name}
      </span>

      {/* 手牌 */}
      <div className="flex gap-1">
        {player.folded
          ? [0, 1].map((i) => <PokerCard key={i} code="?" small />)
          : player.hole.map((c, i) => <PokerCard key={i} code={c} small highlight={player.is_winner} />)
        }
      </div>

      {/* 牌型标签 */}
      {handLabel && (
        <span className={[
          'badge badge-sm flex-shrink-0 font-semibold border-0 text-xs',
          player.folded
            ? 'bg-white/10 text-white/40'
            : player.is_winner
              ? 'bg-amber-500/30 text-amber-300'
              : 'bg-white/10 text-white/60',
        ].join(' ')}>
          {handLabel}
        </span>
      )}
    </div>
  )
}

// ── 主组件 ──────────────────────────────────────────────
export function RoundResultModal({
  duration,
  code,
  onLobby,
}: {
  duration?: number
  code: string        // 房间码，再次买入时发送 join_room 使用
  onLobby: () => void // 返回大厅回调
}) {
  const lastResult = useGameStore((s) => s.lastHandResult)
  const readyCount = useGameStore((s) => s.readyCount)
  const readyTotal = useGameStore((s) => s.readyTotal)
  const gs = useGameState()
  const { user } = useAuthStore()
  const [visible, setVisible] = useState(false)
  const [countdown, setCountdown] = useState(0)

  // ── 破产再次买入 ──
  const wasSeated = useRef(false)
  const [brokeOut, setBrokeOut] = useState(false)
  const [showRebuyPanel, setShowRebuyPanel] = useState(false)
  const [rebuyError, setRebuyError] = useState('')
  const minRebuy = gs.config?.min_buy_in ?? 0
  const maxRebuy = Math.min(gs.config?.max_buy_in ?? 0, user?.chip_balance ?? 0)
  const hasOpenSeat = gs.seats.length < (gs.config?.max_players ?? 0)
  const canRebuy = maxRebuy >= minRebuy && hasOpenSeat
  const [rebuyAmount, setRebuyAmount] = useState(0)

  // 检测本地玩家从有座位变为无座位（破产被踢）
  useEffect(() => {
    if (gs.mySeat) {
      wasSeated.current = true
      if (brokeOut) {
        // 重新入座成功，关闭弹窗
        setBrokeOut(false)
        setShowRebuyPanel(false)
        setRebuyError('')
        setVisible(false)
      }
    } else if (wasSeated.current && !brokeOut) {
      setBrokeOut(true)
      setShowRebuyPanel(false)
      setRebuyError('')
      setRebuyAmount(maxRebuy)
      setVisible(true) // 确保弹窗可见
    }
  }, [gs.mySeat])

  // 结算结果出现/消失时控制弹窗显隐；破产时不自动关闭弹窗
  useEffect(() => {
    if (lastResult) {
      const dur = duration ?? lastResult.nextHandDelaySec
      setVisible(true)
      setCountdown(dur)
      const tick = setInterval(() => setCountdown((n) => Math.max(0, n - 1)), 1000)
      return () => clearInterval(tick)
    } else if (!brokeOut) {
      setVisible(false)
    }
  }, [lastResult, brokeOut])

  const handleRebuy = useCallback(() => {
    setRebuyError('')
    const off = wsClient.on(WSEventType.ServerError, (p: unknown) => {
      const err = p as { message?: string; code?: string }
      setRebuyError(err.message ?? err.code ?? '加入失败')
      off()
    })
    wsClient.send(WSEventType.JoinRoom, { room_code: code, buy_in: rebuyAmount })
    // 3 秒内未入座则移除监听，避免泄漏
    setTimeout(() => off(), 3000)
  }, [code, rebuyAmount])

  if (!visible) return null

  const winners = lastResult?.winners ?? []
  const bestHand = lastResult?.bestHand ?? []
  const allPlayers = lastResult?.allPlayers ?? []

  return (
    /* backdrop */
    <div className="absolute inset-0 z-[70] flex items-center justify-center bg-black/60 backdrop-blur-sm pointer-events-auto">

      {/* modal box */}
      <div className="relative w-full max-w-2xl mx-6 rounded-2xl overflow-hidden
                      bg-[#0d1520] border border-white/10
                      shadow-[0_24px_80px_rgba(0,0,0,0.8)]
                      animate-[fadeSlideUp_0.3s_ease]">

        {/* 顶部装饰线：破产时红色，正常时金色 */}
        <div className={`h-0.5 w-full bg-gradient-to-r from-transparent ${brokeOut ? 'via-red-500/60' : 'via-amber-400/60'} to-transparent`} />

        <div className="px-8 pt-6 pb-6 flex flex-col gap-5">

          {/* ── 标题 ── */}
          <div className="text-center">
            {brokeOut ? (
              <div className="flex flex-col items-center gap-1">
                <span className="text-3xl">💸</span>
                <p className="text-base font-black text-white">筹码已用完</p>
                <p className="text-xs text-white/40">你的桌面筹码归零，已离开牌桌</p>
              </div>
            ) : (
              <p className="text-sm font-black tracking-[0.25em] text-amber-500/70 uppercase">本局结算 · Round Results</p>
            )}
          </div>

          {/* ── 赢家区域（有结算数据时显示） ── */}
          {winners.length > 0 && (
            <div className="flex flex-col gap-2">
              {winners.map((winner, idx) => (
                <div key={winner.player_id} className="flex items-center gap-4 py-3.5 px-4
                                rounded-xl bg-amber-500/5 border border-amber-500/20">
                  <div className="text-3xl">👑</div>
                  <div className="flex-1 min-w-0">
                    <div className="text-lg font-black text-amber-300 truncate">{winner.display_name}</div>
                    <div className="flex items-center gap-2 mt-1">
                      {winner.hand_name && (
                        <span className="badge badge-sm bg-amber-500/20 text-amber-300 border-amber-500/30 font-bold border-0 text-xs">
                          {handZh(winner.hand_name)}
                        </span>
                      )}
                      {winner.amount > 0 && (
                        <span className="text-sm text-amber-400/80 font-semibold">+{winner.amount.toLocaleString()} 筹码</span>
                      )}
                    </div>
                  </div>
                  {/* 最佳五张仅第一位赢家显示 */}
                  {idx === 0 && bestHand.length > 0 && (
                    <div className="flex gap-1 flex-shrink-0">
                      {bestHand.map((c, i) => (
                        <PokerCard key={i} code={c} highlight={i < 2} />
                      ))}
                    </div>
                  )}
                </div>
              ))}
            </div>
          )}

          {/* ── 全场手牌总览（有结算数据时显示） ── */}
          {allPlayers.length > 0 && (
            <div className="flex flex-col gap-2">
              <p className="text-xs font-bold tracking-widest text-white/30 uppercase">All Players</p>
              {allPlayers.map((p) => (
                <PlayerRow key={p.player_id} player={p} />
              ))}
            </div>
          )}

          {/* ── 再次买入面板（破产时展开） ── */}
          {brokeOut && showRebuyPanel && (
            <div className="flex flex-col gap-3 rounded-xl bg-white/[0.03] border border-white/10 px-4 py-3">
              <div className="flex flex-col gap-1.5 text-sm">
                <div className="flex justify-between">
                  <span className="text-white/50">账户余额</span>
                  <span className="font-semibold text-white/80">${(user?.chip_balance ?? 0).toLocaleString()}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-white/50">买入金额</span>
                  <span className="font-bold text-amber-300">${rebuyAmount.toLocaleString()}</span>
                </div>
                <div className="h-px bg-white/8 my-0.5" />
                <div className="flex justify-between">
                  <span className="text-white/50">买入后余额</span>
                  <span className="font-bold text-white/80">${((user?.chip_balance ?? 0) - rebuyAmount).toLocaleString()}</span>
                </div>
              </div>
              <input
                type="range"
                min={minRebuy}
                max={maxRebuy}
                step={gs.config?.big_blind ?? 1}
                value={rebuyAmount}
                onChange={(e) => setRebuyAmount(Number(e.target.value))}
                className="w-full accent-amber-400"
              />
              <div className="flex justify-between text-[10px] text-white/30">
                <span>最低 ${minRebuy.toLocaleString()}</span>
                <span>最高 ${maxRebuy.toLocaleString()}</span>
              </div>
              {rebuyError && <p className="text-xs text-red-400 text-center">{rebuyError}</p>}
              <div className="flex gap-2">
                <button
                  onClick={() => { setShowRebuyPanel(false); setRebuyError('') }}
                  className="flex-1 py-2 rounded-xl border border-white/10 text-white/50 hover:text-white/80 text-sm font-semibold transition-colors"
                >
                  返回
                </button>
                <button
                  onClick={handleRebuy}
                  className="flex-1 py-2 rounded-xl bg-amber-500 hover:bg-amber-400 text-[#0a0f18] text-sm font-black transition-colors"
                >
                  确认买入
                </button>
              </div>
            </div>
          )}

          {/* ── 底部操作栏 ── */}
          <div className="flex items-center gap-3 pt-1">
            {brokeOut && !showRebuyPanel ? (
              /* 破产状态：返回大厅 / 再次买入 */
              <>
                <button
                  onClick={onLobby}
                  className="flex-1 py-2 rounded-xl border border-white/10 text-white/50 hover:text-white/80 text-sm font-semibold transition-colors"
                >
                  返回大厅
                </button>
                <button
                  onClick={() => { setRebuyAmount(maxRebuy); setShowRebuyPanel(true) }}
                  disabled={!canRebuy}
                  className="flex-1 py-2 rounded-xl bg-amber-500 hover:bg-amber-400 disabled:opacity-30 disabled:cursor-not-allowed text-[#0a0f18] text-sm font-black transition-colors"
                  title={!hasOpenSeat ? '座位已满' : !canRebuy ? '账户余额不足' : ''}
                >
                  再次买入
                </button>
                {!hasOpenSeat && (
                  <p className="text-[10px] text-red-400/70 flex-shrink-0">座位已满</p>
                )}
              </>
            ) : !brokeOut ? (
              /* 正常状态：开始下一局 */
              <>
                <button
                  onClick={() => {
                    wsClient.send(WSEventType.Ready, {})
                    setVisible(false)
                  }}
                  className="btn btn-sm flex-1 bg-amber-500/15 hover:bg-amber-500/25 text-amber-300
                             border border-amber-500/30 hover:border-amber-500/50 font-bold tracking-wide text-sm"
                >
                  开始下一局
                </button>
                {readyTotal > 0 && (
                  <p className="text-sm text-white/40 flex-shrink-0">{readyCount}/{readyTotal}</p>
                )}
                <p className="text-sm text-white/25 flex-shrink-0">
                  {countdown > 0 ? `${countdown}s` : '开局中…'}
                </p>
              </>
            ) : null}
          </div>

        </div>

        {/* 底部装饰线 */}
        <div className={`h-0.5 w-full bg-gradient-to-r from-transparent ${brokeOut ? 'via-red-500/30' : 'via-amber-400/40'} to-transparent`} />
      </div>

      <style>{`
        @keyframes fadeSlideUp {
          from { opacity: 0; transform: translateY(20px); }
          to   { opacity: 1; transform: translateY(0); }
        }
      `}</style>
    </div>
  )
}