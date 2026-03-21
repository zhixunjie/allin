import { useEffect, useRef } from 'react'
import { useParams } from 'react-router-dom'
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

export default function RoomPage() {
  const { code } = useParams<{ code: string }>()  // 用于 WebSocket 连接房间
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

  return (
    <div className={styles.root}>
      <div className={styles.canvasWrap} ref={canvasRef} />

      <div className={styles.overlay}>
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

        {/* 本局结算弹窗 */}
        <RoundResultModal />
      </div>
    </div>
  )
}
