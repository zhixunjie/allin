import { useEffect, useRef } from 'react'
import { useGameStore } from '../../store/game'
import { PlayerAction, Street } from '../../types/enums'

const ACTION_LABEL: Record<string, string> = {
  [PlayerAction.Fold]:  '弃牌',
  [PlayerAction.Check]: '过牌',
  [PlayerAction.Call]:  '跟注',
  [PlayerAction.Bet]:   '下注',
  [PlayerAction.Raise]: '加注',
  [PlayerAction.AllIn]: '全押',
}

const STREET_LABEL: Record<string, string> = {
  [Street.PreFlop]: '翻前',
  [Street.Flop]:    '翻牌',
  [Street.Turn]:    '转牌',
  [Street.River]:   '河牌',
}

type Item =
  | { type: 'street'; street: string; key: string }
  | { type: 'action'; entry: ReturnType<typeof useGameStore.getState>['actionLog'][0]; key: string }

export function ActionLogPanel() {
  const actionLog = useGameStore((s) => s.actionLog)
  const containerRef = useRef<HTMLDivElement>(null)

  // 新行动到来时自动滚到底部（直接操作容器 scrollTop，避免 scrollIntoView 冒泡到 document）
  useEffect(() => {
    const el = containerRef.current
    if (el) el.scrollTop = el.scrollHeight
  }, [actionLog.length])

  if (actionLog.length === 0) return null

  // 行动列表中插入街道分隔标签
  const items: Item[] = []
  let lastStreet = ''
  actionLog.forEach((e, i) => {
    if (e.street !== lastStreet) {
      items.push({ type: 'street', street: e.street, key: `s-${e.street}-${i}` })
      lastStreet = e.street
    }
    items.push({ type: 'action', entry: e, key: `a-${i}` })
  })

  return (
    <div
      ref={containerRef}
      className="absolute bottom-40 right-4 w-44 max-h-56 overflow-y-auto flex flex-col gap-px"
      style={{ pointerEvents: 'none' }}
    >
      {items.map((item) =>
        item.type === 'street' ? (
          <div
            key={item.key}
            className="text-[9px] font-bold text-amber-500/35 tracking-widest uppercase mt-1.5 mb-0.5 first:mt-0"
          >
            — {STREET_LABEL[item.street] ?? item.street} —
          </div>
        ) : (
          <div key={item.key} className="flex items-baseline gap-1.5">
            <span className="text-[11px] text-white/45 truncate max-w-[68px] shrink-0 leading-snug">
              {item.entry.display_name}
            </span>
            <span className={`text-[11px] font-semibold shrink-0 leading-snug ${actionColor(item.entry.action)}`}>
              {ACTION_LABEL[item.entry.action] ?? item.entry.action}
            </span>
            {item.entry.amount > 0 && (
              <span className="text-[11px] text-amber-400/65 shrink-0 leading-snug">
                ${item.entry.amount.toLocaleString()}
              </span>
            )}
          </div>
        )
      )}
    </div>
  )
}

function actionColor(action: string): string {
  if (action === PlayerAction.Fold)  return 'text-white/28'
  if (action === PlayerAction.Check) return 'text-white/48'
  if (action === PlayerAction.Call)  return 'text-sky-400/70'
  if (action === PlayerAction.AllIn) return 'text-red-400/85'
  return 'text-amber-400/70' // bet / raise
}
