import { useEffect, useRef } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { initPixiApp } from '../../pixi/app'
import { useAuthStore } from '../../store/auth'
import { useWebSocket } from '../../hooks/useWebSocket'
import { useGameState } from '../../hooks/useGameState'
// import { useActionTimer } from '../../hooks/useActionTimer'
import { ActionPanel } from '../panels/ActionPanel'
// import { ChatPanel } from '../panels/ChatPanel'
import { RoundResultModal } from '../panels/RoundResultModal'
import { ConnectionBanner } from '../components/ConnectionBanner'
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
  const { code } = useParams<{ code: string }>()  // 用于 WebSocket 连接房间
  const { token, user } = useAuthStore()
  const navigate = useNavigate()
  const canvasRef = useRef<HTMLDivElement>(null)

  useWebSocket(code, token)

  const gs = useGameState()
  // const secondsLeft = useActionTimer()

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
                <span className={styles.tableInfoLabel}>筹码</span>
                <span className={styles.tableInfoValue}>${gs.mySeat.stack.toLocaleString()}</span>
              </div>
            )}
            <span className={styles.navLink}>{user?.display_name ?? user?.username}</span>
            <button className={styles.leaveBtn} onClick={() => navigate('/lobby')}>
              离开牌桌
            </button>
          </div>
        </div>

        {/* 行动计时器 临时屏蔽*/}
        {/*{gs.isMyTurn && secondsLeft > 0 && (*/}
        {/*  <div className={styles.timerBanner}>*/}
        {/*    <span className={styles.timerNum} style={{ color: secondsLeft <= 5 ? '#ff5252' : '#d4af37' }}>*/}
        {/*      {secondsLeft}s*/}
        {/*    </span>*/}
        {/*    &nbsp;轮到你行动*/}
        {/*  </div>*/}
        {/*)}*/}

        {/* 等待消息 */}
        {gs.street === Street.Idle && (() => {
          const eligible = gs.seats.filter(s => !s.sit_out && s.stack > 0).length
          return eligible < 2
            ? <div className={styles.waiting}>等待其他玩家加入…</div>
            : <div className={styles.waiting}>准备开始下一手…</div>
        })()}

        {/* 操作面板 */}
        <div className={styles.actionArea}>
          <ActionPanel gs={gs} />
        </div>

        {/* 连接状态 */}
        <ConnectionBanner />

        {/* 聊天（暂时屏蔽）*/}
        {/* <ChatPanel /> */}

        {/* 本局结算弹窗 */}
        <RoundResultModal />
      </div>
    </div>
  )
}
