import styles from './ActionPanel.module.css'
import { sendAction } from '../../hooks/useWebSocket'
import { useGameState } from '../../hooks/useGameState'
type UseGameReturn = ReturnType<typeof useGameState>
import { BetSlider } from './BetSlider'
import { useState, useEffect } from 'react'

interface Props {
  gs: UseGameReturn
}

export function ActionPanel({ gs }: Props) {
  const [betAmount, setBetAmount] = useState(0)
  const [showSlider, setShowSlider] = useState(false)

  useEffect(() => {
    if (gs.isMyTurn) {
      setBetAmount(gs.minRaiseAmount || gs.config?.big_blind || 2)
      setShowSlider(false)
    }
  }, [gs.isMyTurn, gs.minRaiseAmount])

  if (!gs.isMyTurn) return null

  function fold()  { sendAction('fold') }
  function check() { sendAction('check') }
  function call()  { sendAction('call') }
  function bet()   { sendAction('bet', betAmount);   setShowSlider(false) }
  function raise() { sendAction('raise', betAmount); setShowSlider(false) }
  function allIn() { sendAction('all_in');            setShowSlider(false) }

  return (
    <div className={styles.root}>
      {showSlider && (
        <BetSlider
          pot={gs.pot}
          min={gs.current_bet > 0 ? gs.minRaiseAmount : (gs.config?.big_blind ?? 2)}
          max={(gs.mySeat?.stack ?? 0) + (gs.mySeat?.bet ?? 0)}
          step={gs.config?.big_blind ?? 2}
          value={betAmount}
          onChange={setBetAmount}
        />
      )}

      <div className={styles.buttons}>
        <button className={`${styles.btn} ${styles.fold}`}  onClick={fold}>弃牌</button>

        {gs.canCheck && (
          <button className={`${styles.btn} ${styles.check}`} onClick={check}>看牌</button>
        )}
        {gs.canCall && (
          <button className={`${styles.btn} ${styles.call}`}  onClick={call}>
            跟注&nbsp;{gs.callAmount.toLocaleString()}
          </button>
        )}
        {(gs.canBet || gs.canRaise) && (
          <>
            <button
              className={`${styles.btn} ${styles.raise} ${showSlider ? styles.active : ''}`}
              onClick={() => setShowSlider(v => !v)}
            >
              {gs.canBet ? '下注' : '加注'} {showSlider ? '▼' : '▲'}
            </button>
            {showSlider && (
              <button className={`${styles.btn} ${styles.confirm}`} onClick={gs.canBet ? bet : raise}>
                确认&nbsp;{betAmount.toLocaleString()}
              </button>
            )}
          </>
        )}

        <button className={`${styles.btn} ${styles.allIn}`} onClick={allIn}>All-In</button>
      </div>
    </div>
  )
}
