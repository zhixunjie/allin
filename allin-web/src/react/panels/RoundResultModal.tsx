import { useEffect, useState } from 'react'
import { useGameStore } from '../../store/game'
import type { RoundPlayerDetail } from '../../store/game'
import { RANK_DISPLAY, SUIT_SYMBOLS } from '../../pixi/config/card'
import styles from './RoundResultModal.module.css'

// ── 牌型中文名映射 ───────────────────────────────────────
const HAND_ZH: Record<string, string> = {
  'High Card':       '高牌',
  'One Pair':        '一对',
  'Two Pair':        '两对',
  'Three of a Kind': '三条',
  'Straight':        '顺子',
  'Flush':           '同花',
  'Full House':      '葫芦',
  'Four of a Kind':  '四条',
  'Straight Flush':  '同花顺',
  'Royal Flush':     '皇家同花顺',
}

function handZh(name?: string) {
  if (!name) return ''
  return HAND_ZH[name] ?? name
}

// ── 单张扑克牌 ──────────────────────────────────────────
interface CardProps {
  code: string        // e.g. 'Ah', 'Kd', '?' for hidden
  highlight?: boolean // 金色边框（赢家手牌）
  small?: boolean     // 列表行里的小牌
}

function PokerCard({ code, highlight, small }: CardProps) {
  if (!code || code === '?') {
    return <div className={`${styles.card} ${styles.cardHidden} ${small ? styles.cardSmall : ''}`} />
  }
  const rank = code.slice(0, -1)
  const suit = code.slice(-1)
  const isRed = suit === 'h' || suit === 'd'
  const rankDisplay = RANK_DISPLAY[rank] ?? rank

  return (
    <div className={[
      styles.card,
      highlight ? styles.cardHole : styles.cardCommunity,
      small ? styles.cardSmall : '',
      isRed ? styles.cardRed : styles.cardBlack,
    ].join(' ')}>
      <span className={styles.cardRank}>{rankDisplay}</span>
      <span className={styles.cardSuit}>{SUIT_SYMBOLS[suit] ?? suit}</span>
    </div>
  )
}

// ── 玩家行 ──────────────────────────────────────────────
function PlayerRow({ player }: { player: RoundPlayerDetail }) {
  const avatarUrl = `https://api.dicebear.com/9.x/toon-head/svg?seed=${encodeURIComponent(player.player_id)}`
  const handLabel = player.folded
    ? '弃牌 (FOLDED)'
    : player.hand_name
      ? `${handZh(player.hand_name)} (${player.hand_name.toUpperCase()})`
      : ''

  return (
    <div className={`${styles.playerRow} ${player.is_winner ? styles.playerRowWinner : ''}`}>
      <img className={styles.avatar} src={avatarUrl} alt={player.display_name} />
      <span className={styles.playerName}>{player.display_name}</span>
      <div className={styles.holeCards}>
        {player.folded
          ? [0, 1].map((i) => <PokerCard key={i} code="?" small />)
          : player.hole.map((c, i) => <PokerCard key={i} code={c} small highlight={player.is_winner} />)
        }
      </div>
      <span className={`${styles.handLabel} ${player.folded ? styles.handLabelFolded : ''}`}>
        {handLabel}
      </span>
    </div>
  )
}

// ── 主组件 ──────────────────────────────────────────────
export function RoundResultModal() {
  const lastResult = useGameStore((s) => s.lastHandResult)
  const [visible, setVisible] = useState(false)
  const [countdown, setCountdown] = useState(5)

  useEffect(() => {
    if (!lastResult) { setVisible(false); return }
    setVisible(true)
    setCountdown(5)
    const tick = setInterval(() => setCountdown((n) => n - 1), 1000)
    const hide = setTimeout(() => setVisible(false), 5000)
    return () => { clearInterval(tick); clearTimeout(hide) }
  }, [lastResult])

  if (!visible || !lastResult) return null

  const winner = lastResult.winners[0]
  const bestHand = lastResult.bestHand ?? []
  const allPlayers = lastResult.allPlayers ?? []

  return (
    <div className={styles.overlay}>
      <div className={styles.panel}>

        {/* ── 标题 ── */}
        <h1 className={styles.title}>
          本局结算 <span className={styles.titleEn}>(ROUND RESULTS)</span>
        </h1>
        <div className={styles.titleDivider} />

        {/* ── 赢家区域 ── */}
        {winner && (
          <div className={styles.winnerSection}>
            <span className={styles.crown}>👑</span>
            <div className={styles.winnerName}>{winner.display_name}</div>
            {winner.hand_name && (
              <div className={styles.winnerHand}>
                {handZh(winner.hand_name)}{' '}
                <span className={styles.winnerHandEn}>({winner.hand_name})</span>
              </div>
            )}
            {bestHand.length > 0 && (
              <div className={styles.bestHand}>
                {bestHand.map((c, i) => (
                  <PokerCard key={i} code={c} highlight={i < 2} />
                ))}
              </div>
            )}
          </div>
        )}

        {/* ── 全场手牌总览 ── */}
        {allPlayers.length > 0 && (
          <>
            <div className={styles.sectionLabel}>全场手牌总览</div>
            <div className={styles.playerList}>
              {allPlayers.map((p) => (
                <PlayerRow key={p.player_id} player={p} />
              ))}
            </div>
          </>
        )}

        {/* ── 底部按钮 ── */}
        <div className={styles.footer}>
          <button className={styles.nextBtn} onClick={() => setVisible(false)}>
            开始下一局 (Next Round)
          </button>
          <div className={styles.autoNote}>系统将于 {countdown}S 后自动发牌</div>
        </div>

      </div>
    </div>
  )
}
