import { RANK_DISPLAY, SUIT_SYMBOLS } from '../../pixi/config/card'
import styles from './PokerCard.module.css'

interface PokerCardProps {
  code: string       // e.g. 'Ah', 'Kd', '?' for hidden/back
  highlight?: boolean // 金色边框
  small?: boolean     // 小尺寸（列表行用）
  dim?: boolean       // 半透明（弃牌）
}

export function PokerCard({ code, highlight, small, dim }: PokerCardProps) {
  if (!code || code === '?') {
    return (
      <div className={[styles.card, styles.hidden, small ? styles.small : ''].join(' ')}>
        <div className={styles.hiddenInner}>♠</div>
      </div>
    )
  }
  const rank = code.slice(0, -1)
  const suit = code.slice(-1)
  const isRed = suit === 'h' || suit === 'd'

  return (
    <div className={[
      styles.card,
      highlight ? styles.gold : styles.normal,
      small  ? styles.small  : '',
      isRed  ? styles.red    : styles.black,
      dim    ? styles.dim    : '',
    ].join(' ')}>
      <span className={styles.rank}>{RANK_DISPLAY[rank] ?? rank}</span>
      <span className={styles.suit}>{SUIT_SYMBOLS[suit] ?? suit}</span>
    </div>
  )
}
