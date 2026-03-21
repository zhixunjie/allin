import { useState, useEffect } from 'react'
import { useGameStore } from '../../store/game'
import styles from './HandHistory.module.css'

export function HandHistory() {
  const lastResult = useGameStore((s) => s.lastHandResult)
  const [visible, setVisible] = useState(false)

  useEffect(() => {
    if (lastResult) {
      setVisible(true)
      const t = setTimeout(() => setVisible(false), 5000)
      return () => clearTimeout(t)
    }
  }, [lastResult])

  if (!visible || !lastResult) return null

  return (
    <div className={styles.root}>
      <div className={styles.header}>本手结果</div>
      {lastResult.winners.map((w, i) => (
        <div key={i} className={styles.row}>
          <span className={styles.name}>{w.display_name}</span>
          <span className={styles.amount}>+{w.amount.toLocaleString()}</span>
          {w.hand_name && <span className={styles.hand}>{w.hand_name}</span>}
        </div>
      ))}
      <button className={styles.close} onClick={() => setVisible(false)}>✕</button>
    </div>
  )
}
