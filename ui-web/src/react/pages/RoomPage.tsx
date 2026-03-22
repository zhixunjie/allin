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
import { ActionLogPanel } from '../panels/ActionLogPanel'
import { ChatPanel } from '../panels/ChatPanel'
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

  // 手牌历史面板
  const [showHistory, setShowHistory] = useState(false)

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

        {/* 本局结算弹窗（含破产再次买入） */}
        <RoundResultModal code={code ?? ''} onLobby={() => navigate('/lobby')} />

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
