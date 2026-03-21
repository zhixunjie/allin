import { useEffect, useState } from 'react'
import { useGameStore } from '../../store/game'
import type { RoundPlayerDetail } from '../../store/game'
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
      <span className={`absolute top-0.5 left-1 font-black leading-none ${color} ${small ? 'text-[9px]' : 'text-xs'}`}>
        {RANK_DISPLAY[rank] ?? rank}
      </span>
      <span className={`${color} ${small ? 'text-base' : 'text-2xl'} leading-none mt-1`}>
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
      'flex items-center gap-3 px-3 py-2 rounded-xl border transition-all',
      player.is_winner
        ? 'bg-amber-500/10 border-amber-500/40'
        : player.folded
          ? 'bg-white/[0.02] border-white/5 opacity-50'
          : 'bg-white/[0.03] border-white/8',
    ].join(' ')}>

      {/* 头像 */}
      <div className={`avatar flex-shrink-0 ${player.is_winner ? 'ring-2 ring-amber-400 ring-offset-1 ring-offset-[#0a1018]' : ''} rounded-full`}>
        <div className="w-8 h-8 rounded-full overflow-hidden bg-white/10">
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
          'badge badge-sm flex-shrink-0 font-semibold border-0',
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
export function RoundResultModal() {
  const lastResult = useGameStore((s) => s.lastHandResult)
  const [visible, setVisible] = useState(false)
  const [countdown, setCountdown] = useState(10)

  useEffect(() => {
    if (!lastResult) { setVisible(false); return }
    setVisible(true)
    setCountdown(10)
    const tick = setInterval(() => setCountdown((n) => n - 1), 1000)
    const hide = setTimeout(() => setVisible(false), 10000)
    return () => { clearInterval(tick); clearTimeout(hide) }
  }, [lastResult])

  if (!visible || !lastResult) return null

  const winner = lastResult.winners[0]
  const bestHand = lastResult.bestHand ?? []
  const allPlayers = lastResult.allPlayers ?? []

  return (
    /* backdrop */
    <div className="absolute inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm">

      {/* modal box */}
      <div className="relative w-full max-w-md mx-4 rounded-2xl overflow-hidden
                      bg-[#0d1520] border border-white/10
                      shadow-[0_24px_80px_rgba(0,0,0,0.8)]
                      animate-[fadeSlideUp_0.3s_ease]">

        {/* 顶部金色装饰线 */}
        <div className="h-0.5 w-full bg-gradient-to-r from-transparent via-amber-400/60 to-transparent" />

        <div className="px-6 pt-5 pb-6 flex flex-col gap-5 max-h-[85vh] overflow-y-auto">

          {/* ── 标题 ── */}
          <div className="text-center">
            <h2 className="text-xs font-bold tracking-[0.3em] text-amber-500/70 uppercase">Round Results</h2>
            <p className="text-lg font-black text-white mt-0.5">本局结算</p>
          </div>

          {/* ── 赢家区域 ── */}
          {winner && (
            <div className="flex flex-col items-center gap-3 py-4 px-4
                            rounded-xl bg-amber-500/5 border border-amber-500/20">
              <div className="text-3xl">👑</div>
              <div className="text-xl font-black text-amber-300">{winner.display_name}</div>

              {winner.hand_name && (
                <div className="badge bg-amber-500/20 text-amber-300 border-amber-500/30 font-bold px-3 py-2 text-sm">
                  {handZh(winner.hand_name)}
                  <span className="ml-1.5 opacity-60 text-xs font-normal">{winner.hand_name}</span>
                </div>
              )}

              {winner.amount > 0 && (
                <div className="text-sm text-amber-400/80 font-semibold">
                  +{winner.amount.toLocaleString()} 筹码
                </div>
              )}

              {/* 最佳五张 */}
              {bestHand.length > 0 && (
                <div className="flex gap-1.5 mt-1">
                  {bestHand.map((c, i) => (
                    <PokerCard key={i} code={c} highlight={i < 2} />
                  ))}
                </div>
              )}
            </div>
          )}

          {/* ── 全场手牌总览 ── */}
          {allPlayers.length > 0 && (
            <div className="flex flex-col gap-2">
              <p className="text-[10px] font-bold tracking-widest text-white/30 uppercase">All Players</p>
              {allPlayers.map((p) => (
                <PlayerRow key={p.player_id} player={p} />
              ))}
            </div>
          )}

          {/* ── 底部 ── */}
          <div className="flex flex-col items-center gap-2 pt-1">
            <button
              onClick={() => setVisible(false)}
              className="btn w-full bg-amber-500/15 hover:bg-amber-500/25 text-amber-300
                         border border-amber-500/30 hover:border-amber-500/50
                         font-bold tracking-wide"
            >
              开始下一局
            </button>
            <p className="text-xs text-white/25">{countdown}s 后自动开始</p>
          </div>

        </div>

        {/* 底部金色装饰线 */}
        <div className="h-0.5 w-full bg-gradient-to-r from-transparent via-amber-400/40 to-transparent" />
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
