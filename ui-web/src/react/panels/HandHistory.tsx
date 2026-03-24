import { useEffect, useState } from 'react'
import { roomAPI } from '../../api/http'
import type { HandEntry, HandAction } from '../../api/http'

interface Props {
  roomCode: string
  onClose: () => void
}

export function HandHistory({ roomCode, onClose }: Props) {
  const [hands, setHands] = useState<HandEntry[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  useEffect(() => {
    setLoading(true)
    roomAPI.getHands(roomCode)
      .then((data) => setHands(data))
      .catch(() => setError('加载失败'))
      .finally(() => setLoading(false))
  }, [roomCode])

  return (
    <div
      className="fixed top-14 right-4 z-40 w-96 max-h-[70vh] flex flex-col rounded-2xl overflow-hidden
                 bg-[#0d1520]/95 backdrop-blur-md border border-white/10 shadow-[0_24px_80px_rgba(0,0,0,0.7)]
                 pointer-events-auto"
    >
      <div className="h-0.5 w-full bg-gradient-to-r from-transparent via-amber-400/50 to-transparent" />

      <div className="flex items-center justify-between px-4 py-3 border-b border-white/8">
        <span className="text-xs font-black tracking-[0.2em] text-amber-400/80 uppercase">手牌历史</span>
        <button
          onClick={onClose}
          className="text-white/40 hover:text-white/80 text-lg leading-none transition-colors"
        >
          ✕
        </button>
      </div>

      <div className="flex-1 overflow-y-auto py-2">
        {loading && <p className="text-center text-white/30 text-xs py-8">加载中…</p>}
        {error   && <p className="text-center text-red-400/60 text-xs py-8">{error}</p>}
        {!loading && !error && hands.length === 0 && (
          <p className="text-center text-white/30 text-xs py-8">暂无手牌记录</p>
        )}
        {!loading && hands.map((hand) => (
          <HandRow key={hand.hand_num} hand={hand} />
        ))}
      </div>

      <div className="h-0.5 w-full bg-gradient-to-r from-transparent via-amber-400/20 to-transparent" />
    </div>
  )
}

const ACTION_LABEL: Record<string, string> = {
  fold: '弃牌', check: '过牌', call: '跟注', bet: '下注', raise: '加注', all_in: '全押',
}
const STREET_LABEL: Record<string, string> = {
  preflop: '翻前', flop: '翻牌', turn: '转牌', river: '河牌',
}
const STREET_ORDER = ['preflop', 'flop', 'turn', 'river']

function actionColor(action: string): string {
  if (action === 'fold')   return 'text-white/35'
  if (action === 'check')  return 'text-white/55'
  if (action === 'call')   return 'text-sky-400/80'
  if (action === 'all_in') return 'text-red-400/90'
  return 'text-amber-400/80'
}

function HandRow({ hand }: { hand: HandEntry }) {
  const [expanded, setExpanded] = useState(false)

  const winners = hand.result?.winners ?? []
  const playerMap = Object.fromEntries(
    (hand.result?.all_players ?? []).map((p) => [p.player_id, p.display_name])
  )
  const time = new Date(hand.played_at).toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })

  const actions = hand.actions ?? []
  const grouped = STREET_ORDER.reduce<Record<string, HandAction[]>>((acc, s) => {
    const entries = actions.filter((a) => a.street === s)
    if (entries.length > 0) acc[s] = entries
    return acc
  }, {})
  const hasActions = actions.length > 0

  return (
    <div className="border-b border-white/5 last:border-0">
      {/* 摘要行 */}
      <div
        className="px-4 py-2.5 hover:bg-white/[0.03] transition-colors cursor-pointer select-none"
        onClick={() => hasActions && setExpanded((v) => !v)}
      >
        <div className="flex items-center justify-between mb-1">
          <span className="text-[10px] font-bold text-amber-500/50 tracking-widest">第 {hand.hand_num} 手</span>
          <div className="flex items-center gap-2">
            <span className="text-[10px] text-white/50">{time}</span>
            {hasActions && (
              <span className="text-[10px] text-white/25">{expanded ? '▲' : '▼'}</span>
            )}
          </div>
        </div>
        {winners.map((w, i) => (
          <div key={i} className="flex items-center justify-between">
            <span className="text-xs text-white/70 truncate max-w-[120px]">
              {playerMap[w.player_id] ?? w.player_id}
            </span>
            <div className="flex items-center gap-2">
              {w.hand_name && <span className="text-[10px] text-white/35 italic">{w.hand_name}</span>}
              <span className="text-xs font-bold text-amber-400">+{w.amount.toLocaleString()}</span>
            </div>
          </div>
        ))}
      </div>

      {/* 行动明细（展开） */}
      {expanded && (
        <div className="pb-2 bg-white/[0.015]">
          {Object.entries(grouped).map(([street, entries]) => (
            <div key={street}>
              <div className="px-4 pt-2 pb-0.5">
                <span className="text-[9px] font-bold text-amber-500/35 tracking-widest uppercase">
                  {STREET_LABEL[street] ?? street}
                </span>
              </div>
              {entries.map((a, i) => (
                <div key={i} className="px-4 py-0.5 flex items-center justify-between">
                  <span className="text-[11px] text-white/55 truncate max-w-[110px]">
                    {playerMap[a.player_id] ?? a.player_id}
                  </span>
                  <div className="flex items-center gap-1.5">
                    <span className={`text-[11px] font-medium ${actionColor(a.action)}`}>
                      {ACTION_LABEL[a.action] ?? a.action}
                    </span>
                    {a.amount > 0 && (
                      <span className="text-[11px] text-amber-400/70">${a.amount.toLocaleString()}</span>
                    )}
                  </div>
                </div>
              ))}
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
