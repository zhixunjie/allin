/**
 * 右侧状态监视器：实时展示 GameStore 中的关键字段，
 * 辅助调试时观察状态变化。
 */
import { useGameStore } from '../../../store/game'

export function StateMonitor() {
  const gs = useGameStore()

  const rows: [string, string][] = [
    ['street',      gs.street],
    ['action_seat', String(gs.action_seat)],
    ['dealer_seat', String(gs.dealer_seat)],
    ['pot',         `$${gs.pot}`],
    ['community',   gs.community?.join(' ') || '—'],
    ['myHole',      gs.myHole?.join(' ') || '—'],
    ['seats',       String(gs.seats.length)],
    ['deadlineTs',  gs.deadlineTs ? `${Math.ceil((gs.deadlineTs - Date.now()) / 1000)}s` : '—'],
  ]

  return (
    <div className="p-3">
      <p className="text-[10px] font-bold text-amber-500/70 uppercase tracking-widest mb-2">
        Store 状态
      </p>
      <div className="flex flex-col gap-1">
        {rows.map(([k, v]) => (
          <div key={k} className="flex justify-between gap-2 text-[11px]">
            <span className="text-white/40 shrink-0">{k}</span>
            <span className="text-amber-200/80 text-right break-all">{v}</span>
          </div>
        ))}
      </div>

      {gs.seats.length > 0 && (
        <>
          <p className="text-[10px] font-bold text-amber-500/70 uppercase tracking-widest mt-4 mb-2">
            座位列表
          </p>
          {gs.seats.map((s) => (
            <div key={s.seat_index} className="mb-2 text-[10px] leading-relaxed">
              <span className="text-amber-400 font-bold">#{s.seat_index}</span>
              <span className="text-white/60"> {s.display_name}</span>
              <br />
              <span className="text-white/40">
                ${s.stack} | bet:${s.bet}
                {s.folded ? ' 弃' : ''}
                {s.all_in ? ' AI' : ''}
                {s.is_bot ? ' 🤖' : ''}
              </span>
            </div>
          ))}
        </>
      )}
    </div>
  )
}
