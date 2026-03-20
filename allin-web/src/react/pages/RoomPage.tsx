import { useEffect, useRef } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { initPixiApp } from '../../pixi/app'
import { useAuthStore } from '../../store/auth'
import { useWebSocket } from '../../hooks/useWebSocket'
import { useGameState } from '../../hooks/useGameState'
// import { useActionTimer } from '../../hooks/useActionTimer'
import { ActionPanel } from '../panels/ActionPanel'
// import { ChatPanel } from '../panels/ChatPanel'
import { HandHistory } from '../panels/HandHistory'
import { ConnectionBanner } from '../components/ConnectionBanner'
import { Street } from '../../types/enums'
import styles from './RoomPage.module.css'

export default function RoomPage() {
  const { code } = useParams<{ code: string }>()
  const navigate = useNavigate()
  const { token } = useAuthStore()
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

  const blindsText = gs.config
    ? `${gs.config.small_blind}/${gs.config.big_blind}`
    : ''

  return (
    <div className={styles.root}>
      <div className={styles.canvasWrap} ref={canvasRef} />

      <div className={styles.overlay}>
        {/* 顶部信息栏 */}
        <div className={styles.topBar}>
          <span className={styles.brand}>GALACTIC ACES</span>

          <div className={styles.topBarRight}>
            {blindsText && (
              <div className={styles.tableInfo}>
                <span className={styles.tableInfoLabel}>牌桌 #{code ?? ''}</span>
                <span className={styles.tableInfoValue}>盲注 {blindsText}</span>
              </div>
            )}
            {gs.street !== Street.Idle && (
              <span className={styles.streetBadge}>{gs.street.toUpperCase()}</span>
            )}
            <button className={styles.leaveBtn} onClick={() => navigate('/lobby')}>
              ← 离开
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
        {gs.street === Street.Idle && gs.seats.length < 2 && (
          <div className={styles.waiting}>等待其他玩家加入…</div>
        )}
        {gs.street === Street.Idle && gs.seats.length >= 2 && (
          <div className={styles.waiting}>准备开始下一手…</div>
        )}

        {/* 操作面板 */}
        <div className={styles.actionArea}>
          <ActionPanel gs={gs} />
        </div>

        {/* 连接状态 */}
        <ConnectionBanner />

        {/* 聊天（暂时屏蔽）*/}
        {/* <ChatPanel /> */}

        {/* 本手结果浮层 */}
        <HandHistory />
      </div>
    </div>
  )
}
