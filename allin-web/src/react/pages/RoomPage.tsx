import { useEffect, useRef, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { initPixiApp } from '../../pixi/app'
import { useAuthStore } from '../../store/auth'
import { useWebSocket, sendAction } from '../../hooks/useWebSocket'
import { useGameState } from '../../hooks/useGameState'
import { useActionTimer } from '../../hooks/useActionTimer'
import styles from './RoomPage.module.css'

export default function RoomPage() {
  const { code } = useParams<{ code: string }>()
  const navigate = useNavigate()
  const { token } = useAuthStore()
  const canvasRef = useRef<HTMLDivElement>(null)

  const [betAmount, setBetAmount] = useState(0)
  const [showBetSlider, setShowBetSlider] = useState(false)

  // WS connection
  useWebSocket(code, token)

  // Derived game state
  const gs = useGameState()
  const secondsLeft = useActionTimer()

  // PixiJS mount
  useEffect(() => {
    if (!canvasRef.current) return
    let cleanup: (() => void) | null = null

    initPixiApp(canvasRef.current).then((fn) => {
      cleanup = fn
    })

    return () => {
      cleanup?.()
    }
  }, [])

  // Sync bet amount default to min-raise
  useEffect(() => {
    if (gs.isMyTurn) {
      setBetAmount(gs.minRaiseAmount || gs.config?.big_blind || 2)
    }
  }, [gs.isMyTurn, gs.minRaiseAmount])

  function handleFold() { sendAction('fold') }
  function handleCheck() { sendAction('check') }
  function handleCall() { sendAction('call') }
  function handleBet() {
    sendAction('bet', betAmount)
    setShowBetSlider(false)
  }
  function handleRaise() {
    sendAction('raise', betAmount)
    setShowBetSlider(false)
  }
  function handleAllIn() {
    sendAction('all_in')
    setShowBetSlider(false)
  }

  const myStack = gs.mySeat?.stack ?? 0
  const maxBet = myStack + (gs.mySeat?.bet ?? 0)
  const minBet = gs.current_bet > 0 ? gs.minRaiseAmount : (gs.config?.big_blind ?? 2)

  return (
    <div className={styles.root}>
      {/* PixiJS canvas */}
      <div className={styles.canvasWrap} ref={canvasRef} />

      {/* React overlay */}
      <div className={styles.overlay}>
        {/* Top bar */}
        <div className={styles.topBar}>
          <span className={styles.roomCode}>房间 {code}</span>
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

        {/* Waiting message */}
        {gs.street === 'idle' && gs.seats.length < 2 && (
          <div className={styles.waiting}>等待其他玩家加入…</div>
        )}
        {gs.street === 'idle' && gs.seats.length >= 2 && (
          <div className={styles.waiting}>准备开始下一手…</div>
        )}

        {/* Bet slider */}
        {showBetSlider && gs.isMyTurn && (
          <div className={styles.sliderPanel}>
            <div className={styles.sliderRow}>
              <button className={styles.smallBtn} onClick={() => setBetAmount(Math.max(minBet, betAmount - (gs.config?.big_blind ?? 2)))}>−</button>
              <input
                type="range"
                min={minBet}
                max={maxBet}
                step={gs.config?.big_blind ?? 2}
                value={betAmount}
                onChange={(e) => setBetAmount(Number(e.target.value))}
                className={styles.slider}
              />
              <button className={styles.smallBtn} onClick={() => setBetAmount(Math.min(maxBet, betAmount + (gs.config?.big_blind ?? 2)))}>+</button>
            </div>
            <div className={styles.sliderPresets}>
              <button className={styles.presetBtn} onClick={() => setBetAmount(Math.round(gs.pot * 0.33))}>1/3</button>
              <button className={styles.presetBtn} onClick={() => setBetAmount(Math.round(gs.pot * 0.5))}>1/2</button>
              <button className={styles.presetBtn} onClick={() => setBetAmount(Math.round(gs.pot * 0.75))}>3/4</button>
              <button className={styles.presetBtn} onClick={() => setBetAmount(gs.pot)}>Pot</button>
              <span className={styles.betValue}>{betAmount.toLocaleString()}</span>
            </div>
          </div>
        )}

        {/* Action buttons */}
        {gs.isMyTurn && (
          <div className={styles.actionPanel}>
            <button className={`${styles.actionBtn} ${styles.foldBtn}`} onClick={handleFold}>
              弃牌
            </button>

            {gs.canCheck && (
              <button className={`${styles.actionBtn} ${styles.checkBtn}`} onClick={handleCheck}>
                看牌
              </button>
            )}

            {gs.canCall && (
              <button className={`${styles.actionBtn} ${styles.callBtn}`} onClick={handleCall}>
                跟注 {gs.callAmount.toLocaleString()}
              </button>
            )}

            {(gs.canBet || gs.canRaise) && (
              <>
                <button
                  className={`${styles.actionBtn} ${styles.raiseBtn}`}
                  onClick={() => setShowBetSlider((v) => !v)}
                >
                  {gs.canBet ? '下注' : '加注'} ▲
                </button>
                {showBetSlider && (
                  <button
                    className={`${styles.actionBtn} ${styles.confirmBtn}`}
                    onClick={gs.canBet ? handleBet : handleRaise}
                  >
                    确认 {betAmount.toLocaleString()}
                  </button>
                )}
              </>
            )}

            <button className={`${styles.actionBtn} ${styles.allInBtn}`} onClick={handleAllIn}>
              All-In
            </button>
          </div>
        )}

        {/* My hole cards (React layer, shown at bottom) */}
        {gs.myHole.length === 2 && (
          <div className={styles.myCards}>
            {gs.myHole.map((card, i) => (
              <div key={i} className={`${styles.card} ${isRedSuit(card) ? styles.red : styles.black}`}>
                <span className={styles.cardRank}>{card[0]}</span>
                <span className={styles.cardSuit}>{suitSymbol(card[1])}</span>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}

function isRedSuit(card: string) {
  return card[1] === 'h' || card[1] === 'd'
}

function suitSymbol(s: string) {
  return { c: '♣', d: '♦', h: '♥', s: '♠' }[s] ?? s
}
