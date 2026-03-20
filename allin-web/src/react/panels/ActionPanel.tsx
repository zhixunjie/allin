import styles from './ActionPanel.module.css'
import { sendAction } from '../../hooks/useWebSocket'
import { useGameState } from '../../hooks/useGameState'
import { PlayerAction } from '../../types/enums'
type UseGameReturn = ReturnType<typeof useGameState>
import { useState, useEffect } from 'react'

interface Props {
  gs: UseGameReturn
}

export function ActionPanel({ gs }: Props) {
  const [betAmount, setBetAmount] = useState(0)

  const minBet = gs.current_bet > 0 ? gs.minRaiseAmount : (gs.config?.big_blind ?? 2)
  const maxBet = (gs.mySeat?.stack ?? 0) + (gs.mySeat?.bet ?? 0)
  const step = gs.config?.big_blind ?? 2

  useEffect(() => {
    if (gs.isMyTurn) {
      setBetAmount(gs.minRaiseAmount || gs.config?.big_blind || 2)
    }
  }, [gs.isMyTurn, gs.minRaiseAmount])

  if (!gs.isMyTurn) return null

  function fold()  { sendAction(PlayerAction.Fold) }
  function check() { sendAction(PlayerAction.Check) }
  function call()  { sendAction(PlayerAction.Call) }
  function doRaise() {
    if (gs.canBet) {
      sendAction(PlayerAction.Bet, betAmount)
    } else {
      sendAction(PlayerAction.Raise, betAmount)
    }
  }
  function allIn() { sendAction(PlayerAction.AllIn) }

  const clamp = (v: number) => Math.max(minBet, Math.min(maxBet, v))

  return (
    <div className={styles.root}>
      <div className={styles.dock}>
        {/* 弃牌 */}
        <button className={`${styles.actionBtn} ${styles.fold}`} onClick={fold}>
          <span className={styles.icon}>✕</span>
          <span className={styles.label}>弃牌</span>
        </button>

        {/* 过牌 */}
        {gs.canCheck && (
          <button className={`${styles.actionBtn} ${styles.check}`} onClick={check}>
            <span className={styles.icon}>✓</span>
            <span className={styles.label}>过牌</span>
          </button>
        )}

        {/* 跟注 */}
        {gs.canCall && (
          <button className={`${styles.actionBtn} ${styles.call}`} onClick={call}>
            <span className={styles.icon}>💰</span>
            <span className={styles.label}>跟注 ${gs.callAmount.toLocaleString()}</span>
          </button>
        )}

        {/* 加注 — 内联 -/+ 控件 */}
        {(gs.canBet || gs.canRaise) && (
          <div className={styles.raiseGroup}>
            <button
              className={styles.adjBtn}
              onClick={() => setBetAmount(clamp(betAmount - step))}
            >
              −
            </button>
            <button className={styles.raiseCenter} onClick={doRaise}>
              <span className={styles.icon}>↗</span>
              <span className={styles.raiseLabel}>
                {gs.canBet ? '下注' : '加注'} ${betAmount.toLocaleString()}
              </span>
            </button>
            <button
              className={styles.adjBtn}
              onClick={() => setBetAmount(clamp(betAmount + step))}
            >
              +
            </button>
          </div>
        )}

        {/* 全押 */}
        <button className={`${styles.actionBtn} ${styles.allIn}`} onClick={allIn}>
          <span className={styles.icon}>★</span>
          <span className={styles.label}>全押</span>
        </button>
      </div>
    </div>
  )
}
