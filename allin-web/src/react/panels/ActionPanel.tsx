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

  const pot    = gs.pot ?? 0
  const minBet = gs.current_bet > 0 ? gs.minRaiseAmount : (gs.config?.big_blind ?? 2)
  const maxBet = (gs.mySeat?.stack ?? 0) + (gs.mySeat?.bet ?? 0)
  const step   = gs.config?.big_blind ?? 2

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
    sendAction(gs.canBet ? PlayerAction.Bet : PlayerAction.Raise, betAmount)
  }
  function allIn() { sendAction(PlayerAction.AllIn) }

  const clamp    = (v: number) => Math.max(minBet, Math.min(maxBet, v))
  const presetAmt = (ratio: number) => clamp(Math.round(pot * ratio))
  const preset   = (ratio: number) => setBetAmount(presetAmt(ratio))
  const fmt      = (v: number) => v >= 1000 ? `${(v / 1000).toFixed(1)}K` : String(v)

  return (
    <div className={styles.root}>
      <div className={styles.dock}>

        {/* ── 弃牌 ───────────────────── */}
        <button className={`${styles.actionBtn} ${styles.fold}`} onClick={fold}>
          <span className={styles.icon}>✕</span>
          <span className={styles.label}>弃牌</span>
        </button>

        {/* ── 过牌 ───────────────────── */}
        {gs.canCheck && (
          <button className={`${styles.actionBtn} ${styles.check}`} onClick={check}>
            <span className={styles.icon}>✓</span>
            <span className={styles.label}>过牌</span>
          </button>
        )}

        {/* ── 下注 / 加注 ─────────────── */}
        {(gs.canBet || gs.canRaise) && (
          <div className={styles.raiseGroup}>
            {/* 底池比例快捷：同时显示实际金额 */}
            <div className={styles.presets}>
              <button className={styles.preset} onClick={() => preset(0.33)}>
                <span className={styles.presetLabel}>1/3</span>
                <span className={styles.presetAmt}>${fmt(presetAmt(0.33))}</span>
              </button>
              <button className={styles.preset} onClick={() => preset(0.5)}>
                <span className={styles.presetLabel}>1/2</span>
                <span className={styles.presetAmt}>${fmt(presetAmt(0.5))}</span>
              </button>
              <button className={styles.preset} onClick={() => preset(0.75)}>
                <span className={styles.presetLabel}>3/4</span>
                <span className={styles.presetAmt}>${fmt(presetAmt(0.75))}</span>
              </button>
              <button className={styles.preset} onClick={() => preset(1)}>
                <span className={styles.presetLabel}>底池</span>
                <span className={styles.presetAmt}>${fmt(presetAmt(1))}</span>
              </button>
            </div>
            {/* 滑条 + 微调 + 确认（合并一行） */}
            <div className={styles.raiseRow}>
              <button className={styles.adjBtn} onClick={() => setBetAmount(clamp(betAmount - step))}>−</button>
              <input
                type="range"
                className={styles.slider}
                min={minBet}
                max={maxBet}
                step={step}
                value={betAmount}
                style={{ '--fill': `${((betAmount - minBet) / Math.max(1, maxBet - minBet) * 100).toFixed(1)}%` } as React.CSSProperties}
                onChange={(e) => setBetAmount(Number(e.target.value))}
              />
              <button className={styles.adjBtn} onClick={() => setBetAmount(clamp(betAmount + step))}>+</button>
              <button className={styles.raiseCenter} onClick={doRaise}>
                <span className={styles.raiseAction}>{gs.canBet ? '下注' : '加注'}</span>
                <span className={styles.raiseAmount}>${betAmount.toLocaleString()}</span>
              </button>
            </div>
          </div>
        )}

        {/* ── 跟注 ───────────────────── */}
        {gs.canCall && (
          <button className={`${styles.actionBtn} ${styles.call}`} onClick={call}>
            <span className={styles.callAmount}>${gs.callAmount.toLocaleString()}</span>
            <span className={styles.label}>跟注</span>
          </button>
        )}

        {/* ── 全押 ───────────────────── */}
        <button className={`${styles.actionBtn} ${styles.allIn}`} onClick={allIn}>
          <span className={styles.icon}>★</span>
          <span className={styles.label}>全押</span>
        </button>

      </div>
    </div>
  )
}
