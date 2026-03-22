import { useEffect, useState } from 'react'
import { roomAPI } from '../../api/http'
import type { HandEntry } from '../../api/http'

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
      className="fixed top-14 right-4 z-40 w-72 max-h-[70vh] flex flex-col rounded-2xl overflow-hidden
                 bg-[#0d1520]/95 backdrop-blur-md border border-white/10 shadow-[0_24px_80px_rgba(0,0,0,0.7)]"
    >
      {/* 顶部装饰线 */}
      <div className="h-0.5 w-full bg-gradient-to-r from-transparent via-amber-400/50 to-transparent" />

      {/* 标题栏 */}
      <div className="flex items-center justify-between px-4 py-3 border-b border-white/8">
        <span className="text-xs font-black tracking-[0.2em] text-amber-400/80 uppercase">手牌历史</span>
        <button
          onClick={onClose}
          className="text-white/40 hover:text-white/80 text-lg leading-none transition-colors"
        >
          ✕
        </button>
      </div>

      {/* 内容区 */}
      <div className="flex-1 overflow-y-auto py-2">
        {loading && (
          <p className="text-center text-white/30 text-xs py-8">加载中…</p>
        )}
        {error && (
          <p className="text-center text-red-400/60 text-xs py-8">{error}</p>
        )}
        {!loading && !error && hands.length === 0 && (
          <p className="text-center text-white/30 text-xs py-8">暂无手牌记录</p>
        )}
        {!loading && hands.map((hand) => (
          <HandRow key={hand.hand_num} hand={hand} />
        ))}
      </div>

      {/* 底部装饰线 */}
      <div className="h-0.5 w-full bg-gradient-to-r from-transparent via-amber-400/20 to-transparent" />
    </div>
  )
}

function HandRow({ hand }: { hand: HandEntry }) {
  const winners = hand.result ?? []
  const time = new Date(hand.played_at).toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })

  return (
    <div className="px-4 py-2.5 border-b border-white/5 last:border-0 hover:bg-white/[0.03] transition-colors">
      <div className="flex items-center justify-between mb-1">
        <span className="text-[10px] font-bold text-amber-500/50 tracking-widest">第 {hand.hand_num} 手</span>
        <span className="text-[10px] text-white/25">{time}</span>
      </div>
      {winners.map((w, i) => (
        <div key={i} className="flex items-center justify-between">
          <span className="text-xs text-white/70 truncate max-w-[120px]">{w.player_id}</span>
          <div className="flex items-center gap-2">
            {w.hand_name && (
              <span className="text-[10px] text-white/35 italic">{w.hand_name}</span>
            )}
            <span className="text-xs font-bold text-amber-400">+{w.amount.toLocaleString()}</span>
          </div>
        </div>
      ))}
    </div>
  )
}
