import { useEffect, useRef, useState, useCallback } from 'react'
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
import { ActionLogPanel } from '../panels/ActionLogPanel'
import { ChatPanel } from '../panels/ChatPanel'
import { Street, WSEventType } from '../../types/enums'
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

  // 手牌历史面板
  const [showHistory, setShowHistory] = useState(false)

  // 本地玩家破产被踢出检测 & 再次买入
  const wasSeated = useRef(false)
  const [brokeOut, setBrokeOut] = useState(false)
  const [showRebuy, setShowRebuy] = useState(false)
  const [rebuyError, setRebuyError] = useState('')

  const minRebuy = gs.config?.min_buy_in ?? 0
  const maxRebuy = Math.min(gs.config?.max_buy_in ?? 0, user?.chip_balance ?? 0)
  const hasOpenSeat = (gs.seats.length) < (gs.config?.max_players ?? 0)
  const canRebuy = maxRebuy >= minRebuy && hasOpenSeat

  const [rebuyAmount, setRebuyAmount] = useState(0)

  useEffect(() => {
    if (gs.mySeat) {
      wasSeated.current = true
      // 重新入座成功后关闭弹窗
      if (brokeOut) { setBrokeOut(false); setShowRebuy(false); setRebuyError('') }
    } else if (wasSeated.current && !brokeOut) {
      setBrokeOut(true)
      setShowRebuy(false)
      setRebuyError('')
      setRebuyAmount(maxRebuy)
    }
  }, [gs.mySeat])

  const handleRebuy = useCallback(() => {
    setRebuyError('')
    // 订阅一次服务端错误，用于捕获满座/余额不足等拒绝原因
    const off = wsClient.on(WSEventType.ServerError, (p: unknown) => {
      const err = p as { message?: string; code?: string }
      setRebuyError(err.message ?? err.code ?? '加入失败')
      off()
    })
    wsClient.send(WSEventType.JoinRoom, { room_code: code, buy_in: rebuyAmount })
    // 3 秒内若入座成功（mySeat 变非 null），上面的 useEffect 会关闭弹窗并移除监听器；
    // 若超时未成功，移除监听器避免泄漏
    setTimeout(() => off(), 3000)
  }, [code, rebuyAmount])

  // 邀请链接复制
  const [copied, setCopied] = useState(false)
  function copyInviteLink() {
    const url = `${window.location.origin}/join/${code}`
    navigator.clipboard.writeText(url).then(() => {
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    })
  }

  useEffect(() => {
    if (!canvasRef.current) return
    let cleanup: (() => void) | null = null
    initPixiApp(canvasRef.current).then((fn) => { cleanup = fn })
    return () => { cleanup?.() }
  }, [])

  return (
    <div className={styles.root}>
      <div className={styles.canvasWrap} ref={canvasRef} />

      <div className={styles.overlay}>

        {/* 破产弹窗：本地玩家筹码归零被踢出时显示 */}
        {brokeOut && (
          <div className="absolute inset-0 z-[50] flex items-center justify-center bg-black/70 backdrop-blur-sm pointer-events-auto">
            <div className="w-full max-w-xs mx-4 rounded-2xl overflow-hidden bg-[#111820] border border-white/10 shadow-[0_24px_80px_rgba(0,0,0,0.8)]">
              <div className="h-0.5 w-full bg-gradient-to-r from-transparent via-red-500/60 to-transparent" />

              {!showRebuy ? (
                /* ── 破产主界面 ── */
                <div className="px-6 py-6 flex flex-col items-center gap-4 text-center">
                  <div className="text-4xl">💸</div>
                  <div>
                    <p className="text-base font-black text-white">筹码已用完</p>
                    <p className="text-xs text-white/40 mt-1">你的桌面筹码归零，已离开牌桌</p>
                  </div>
                  <div className="flex gap-2 w-full">
                    <button
                      onClick={() => navigate('/lobby')}
                      className="flex-1 py-2 rounded-xl border border-white/10 text-white/50 hover:text-white/80 text-sm font-semibold transition-colors"
                    >
                      返回大厅
                    </button>
                    <button
                      onClick={() => { setRebuyAmount(maxRebuy); setShowRebuy(true) }}
                      disabled={!canRebuy}
                      className="flex-1 py-2 rounded-xl bg-amber-500 hover:bg-amber-400 disabled:opacity-30 disabled:cursor-not-allowed text-[#0a0f18] text-sm font-black transition-colors"
                      title={!hasOpenSeat ? '座位已满' : !canRebuy ? '账户余额不足' : ''}
                    >
                      再次买入
                    </button>
                  </div>
                  {!hasOpenSeat && <p className="text-[10px] text-red-400/70">当前座位已满，无法再次入座</p>}
                  {hasOpenSeat && !canRebuy && <p className="text-[10px] text-red-400/70">账户余额不足最低买入 ${minRebuy.toLocaleString()}</p>}
                </div>
              ) : (
                /* ── 再次买入金额选择 ── */
                <div className="px-6 py-5 flex flex-col gap-4">
                  <div>
                    <p className="text-[10px] font-bold tracking-[0.25em] text-amber-500/60 uppercase">再次买入</p>
                    <p className="text-sm font-black text-white mt-0.5">选择买入金额</p>
                  </div>

                  <div className="flex flex-col gap-1.5 rounded-xl bg-white/[0.03] border border-white/8 px-4 py-3 text-sm">
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
                      onClick={() => { setShowRebuy(false); setRebuyError('') }}
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

              <div className="h-0.5 w-full bg-gradient-to-r from-transparent via-red-500/30 to-transparent" />
            </div>
          </div>
        )}

        {/* 顶部栏 */}
        <div className={styles.topBar}>
          <span className={styles.brand}>AllIn</span>

          <div className={styles.tableInfo}>
            <span className={styles.tableInfoLabel}>房间</span>
            <span className={styles.tableInfoValue}>{code}</span>
          </div>
          <button
            className={styles.leaveBtn}
            onClick={copyInviteLink}
            title="复制邀请链接"
            style={{ color: copied ? '#d4af37' : undefined }}
          >
            {copied ? '已复制' : '邀请'}
          </button>

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

        {/* 当局行动日志（常驻右下角，实时滚动） */}
        <ActionLogPanel />

        {/* 手牌历史面板 */}
        {showHistory && code && (
          <HandHistory roomCode={code} onClose={() => setShowHistory(false)} />
        )}

        {/* 聊天面板 */}
        <ChatPanel />

      </div>
    </div>
  )
}
