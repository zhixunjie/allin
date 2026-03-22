import { useEffect, useRef, useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { initPixiApp } from '../../pixi/app'
import { useAuthStore } from '../../store/auth'
import { useWebSocket } from '../../hooks/useWebSocket'
import { wsClient } from '../../api/ws'
import { useGameState } from '../../hooks/useGameState'
import { ActionPanel } from '../panels/ActionPanel'
import { RoundResultModal } from '../panels/RoundResultModal'
import { ConnectionBanner } from '../components/ConnectionBanner'
import { HandHistory } from '../panels/HandHistory'
import { Street } from '../../types/enums'
import styles from './RoomPage.module.css'

const STREET_LABEL: Record<string, string> = {
  [Street.PreFlop]: '翻牌前',
  [Street.Flop]:    '翻牌',
  [Street.Turn]:    '转牌',
  [Street.River]:   '河牌',
  [Street.Showdown]:'摊牌',
}

export default function RoomPage() {
  const { code } = useParams<{ code: string }>()
  const { token, user } = useAuthStore()
  const navigate = useNavigate()
  const canvasRef = useRef<HTMLDivElement>(null)

  useWebSocket(code, token)

  const gs = useGameState()

  // 补充筹码弹窗
  const [showAddChips, setShowAddChips] = useState(false)
  const maxAdd = Math.max(0, (gs.config?.max_buy_in ?? 0) - (gs.mySeat?.stack ?? 0))
  const [addAmount, setAddAmount] = useState(0)

  // 手牌历史面板
  const [showHistory, setShowHistory] = useState(false)

  useEffect(() => {
    if (!canvasRef.current) return
    let cleanup: (() => void) | null = null
    initPixiApp(canvasRef.current).then((fn) => { cleanup = fn })
    return () => { cleanup?.() }
  }, [])

  // 打开补充筹码弹窗时重置金额为最大可补额
  function openAddChips() {
    setAddAmount(Math.min(maxAdd, user?.chip_balance ?? 0))
    setShowAddChips(true)
  }

  function confirmAddChips() {
    if (addAmount > 0) {
      wsClient.send('add_chips', { amount: addAmount })
    }
    setShowAddChips(false)
  }

  const canAddChips =
    gs.street === Street.Idle &&
    gs.mySeat != null &&
    maxAdd > 0 &&
    (user?.chip_balance ?? 0) > 0

  return (
    <div className={styles.root}>
      <div className={styles.canvasWrap} ref={canvasRef} />

      <div className={styles.overlay}>
        {/* 顶部栏 */}
        <div className={styles.topBar}>
          <span className={styles.brand}>AllIn</span>

          <div className={styles.tableInfo}>
            <span className={styles.tableInfoLabel}>房间</span>
            <span className={styles.tableInfoValue}>{code}</span>
          </div>

          {gs.config && (
            <div className={styles.tableInfo}>
              <span className={styles.tableInfoLabel}>盲注</span>
              <span className={styles.tableInfoValue}>
                ${gs.config.small_blind} / ${gs.config.big_blind}
              </span>
            </div>
          )}

          {gs.street !== Street.Idle && (
            <span className={styles.streetBadge}>{STREET_LABEL[gs.street] ?? gs.street}</span>
          )}

          <div className={styles.topBarRight}>
            {gs.mySeat && (
              <div className={styles.tableInfo}>
                <span className={styles.tableInfoLabel}>桌面筹码</span>
                <span className={styles.tableInfoValue}>${gs.mySeat.stack.toLocaleString()}</span>
              </div>
            )}
            {user?.chip_balance != null && (
              <div className={styles.tableInfo}>
                <span className={styles.tableInfoLabel}>账户余额</span>
                <span className={styles.tableInfoValue}>${user.chip_balance.toLocaleString()}</span>
              </div>
            )}
            {canAddChips && (
              <button className={styles.addChipsBtn} onClick={openAddChips}>
                补充筹码
              </button>
            )}
            <button
              className={styles.leaveBtn}
              onClick={() => setShowHistory((v) => !v)}
              style={{ background: showHistory ? 'rgba(212,175,55,0.2)' : undefined }}
            >
              历史
            </button>
            <span className={styles.navLink}>{user?.display_name ?? user?.username}</span>
            <button className={styles.leaveBtn} onClick={() => {
              wsClient.send('leave_table', {})
              navigate('/lobby')
            }}>
              离开牌桌
            </button>
          </div>
        </div>

        {/* 等待消息 */}
        {gs.street === Street.Idle && (() => {
          const eligible = gs.seats.filter(s => !s.sit_out && s.stack > 0).length
          const isWaiting = eligible < 2
          return (
            <div className="absolute bottom-44 left-1/2 -translate-x-1/2 pointer-events-none">
              <div className={[
                'flex items-center gap-2.5 px-5 py-2.5 rounded-full',
                'bg-[#040810]/85 backdrop-blur-md',
                'border text-sm font-semibold whitespace-nowrap',
                isWaiting
                  ? 'border-white/10 text-white/50'
                  : 'border-amber-500/25 text-amber-400/80',
              ].join(' ')}>
                {isWaiting ? (
                  <>
                    <span className="w-1.5 h-1.5 rounded-full bg-white/30 animate-pulse" />
                    等待其他玩家加入…
                  </>
                ) : (
                  <>
                    <span className="w-1.5 h-1.5 rounded-full bg-amber-400 animate-ping" />
                    准备开始下一手…
                  </>
                )}
              </div>
            </div>
          )
        })()}

        {/* 操作面板 */}
        <div className={styles.actionArea}>
          <ActionPanel gs={gs} />
        </div>

        {/* 连接状态 */}
        <ConnectionBanner />

        {/* 本局结算弹窗 */}
        <RoundResultModal />

        {/* 手牌历史面板 */}
        {showHistory && code && (
          <HandHistory roomCode={code} onClose={() => setShowHistory(false)} />
        )}

        {/* 补充筹码弹窗 */}
        {showAddChips && (
          <div
            className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm"
            onClick={() => setShowAddChips(false)}
          >
            <div
              className="w-80 rounded-2xl overflow-hidden bg-[#111820] border border-white/10 shadow-[0_24px_80px_rgba(0,0,0,0.8)]"
              onClick={(e) => e.stopPropagation()}
            >
              <div className="h-0.5 w-full bg-gradient-to-r from-transparent via-amber-400/60 to-transparent" />
              <div className="px-6 py-5 flex flex-col gap-4">
                <div>
                  <p className="text-[10px] font-bold tracking-[0.25em] text-amber-500/60 uppercase">补充筹码</p>
                  <p className="text-xs text-white/40 mt-1">
                    当前桌面 ${gs.mySeat?.stack.toLocaleString()} · 最高买入 ${gs.config?.max_buy_in.toLocaleString()} · 账户余额 ${user?.chip_balance?.toLocaleString()}
                  </p>
                </div>
                <div className="flex flex-col gap-2">
                  <input
                    type="number"
                    min={1}
                    max={Math.min(maxAdd, user?.chip_balance ?? 0)}
                    value={addAmount}
                    onChange={(e) => setAddAmount(Math.max(1, Math.min(Math.min(maxAdd, user?.chip_balance ?? 0), Number(e.target.value))))}
                    className="w-full bg-white/5 border border-white/10 rounded-xl px-4 py-2 text-white text-sm focus:outline-none focus:border-amber-500/50"
                  />
                  <div className="flex gap-2">
                    {[0.25, 0.5, 1].map((r) => {
                      const v = Math.min(Math.round(maxAdd * r), user?.chip_balance ?? 0)
                      return (
                        <button
                          key={r}
                          onClick={() => setAddAmount(v)}
                          className="flex-1 py-1 text-xs rounded-lg bg-white/5 hover:bg-white/10 text-white/60 hover:text-white/90 border border-white/8 transition-colors"
                        >
                          {r === 1 ? '全补' : `${r * 100}%`}
                        </button>
                      )
                    })}
                  </div>
                </div>
                <div className="flex gap-3">
                  <button
                    onClick={() => setShowAddChips(false)}
                    className="flex-1 py-2 rounded-xl border border-white/10 text-white/50 hover:text-white/80 text-sm font-semibold transition-colors"
                  >
                    取消
                  </button>
                  <button
                    onClick={confirmAddChips}
                    disabled={addAmount <= 0}
                    className="flex-1 py-2 rounded-xl bg-amber-500 hover:bg-amber-400 disabled:opacity-40 text-[#0a0f18] text-sm font-black transition-colors"
                  >
                    确认补充 ${addAmount.toLocaleString()}
                  </button>
                </div>
              </div>
              <div className="h-0.5 w-full bg-gradient-to-r from-transparent via-amber-400/30 to-transparent" />
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
