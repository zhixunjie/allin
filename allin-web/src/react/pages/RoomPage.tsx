import { useEffect, useRef } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { initPixiApp } from '../../pixi/app'
import { useAuthStore } from '../../store/auth'
import { useWebSocket } from '../../hooks/useWebSocket'
import { useGameState } from '../../hooks/useGameState'
import { useActionTimer } from '../../hooks/useActionTimer'
import { ActionPanel } from '../panels/ActionPanel'
import { ChatPanel } from '../panels/ChatPanel'
import { HandHistory } from '../panels/HandHistory'
import { RoomInfo } from '../panels/RoomInfo'
import { ConnectionBanner } from '../components/ConnectionBanner'
import styles from './RoomPage.module.css'

export default function RoomPage() {
  const { code } = useParams<{ code: string }>()
  const navigate = useNavigate()
  const { token } = useAuthStore()
  const canvasRef = useRef<HTMLDivElement>(null)

  useWebSocket(code, token)

  const gs = useGameState()
  const secondsLeft = useActionTimer()

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
        {/* Top bar */}
        <div className={styles.topBar}>
          <RoomInfo code={code ?? ''} />
          <span className={styles.streetBadge}>{gs.street !== 'idle' ? gs.street.toUpperCase() : ''}</span>
          <button className={styles.leaveBtn} onClick={() => navigate('/lobby')}>离开</button>
        </div>

        {/* Action timer */}
        {gs.isMyTurn && secondsLeft > 0 && (
          <div className={styles.timerBanner}>
            <span className={styles.timerNum} style={{ color: secondsLeft <= 5 ? '#ff5252' : '#f0c040' }}>
              {secondsLeft}s
            </span>
            &nbsp;轮到你行动
          </div>
        )}

        {/* Waiting messages */}
        {gs.street === 'idle' && gs.seats.length < 2 && (
          <div className={styles.waiting}>等待其他玩家加入…</div>
        )}
        {gs.street === 'idle' && gs.seats.length >= 2 && (
          <div className={styles.waiting}>准备开始下一手…</div>
        )}

        {/* Action panel */}
        <div className={styles.actionArea}>
          <ActionPanel gs={gs} />
        </div>

        {/* Connection status */}
        <ConnectionBanner />

        {/* Chat */}
        <ChatPanel />

        {/* Hand result overlay */}
        <HandHistory />
      </div>
    </div>
  )
}

