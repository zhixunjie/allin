import { PokerCard } from '../../components/PokerCard'

const RANKS = ['A', 'K', 'Q', 'J', 'T', '9', '8', '7', '6', '5', '4', '3', '2']
const SUITS = [
  { key: 'h', label: '♥ Hearts',   color: '#e03434' },
  { key: 'd', label: '♦ Diamonds', color: '#e03434' },
  { key: 'c', label: '♣ Clubs',    color: '#ccc' },
  { key: 's', label: '♠ Spades',   color: '#ccc' },
]

export function CardGallery({ onClose }: { onClose: () => void }) {
  return (
    <div style={{
      position: 'absolute', inset: 0, zIndex: 55,
      background: 'rgba(4,8,16,0.92)', backdropFilter: 'blur(8px)',
      display: 'flex', flexDirection: 'column',
      alignItems: 'center', padding: '24px 32px', overflowY: 'auto',
    }}>
      {/* 标题栏 */}
      <div style={{ display: 'flex', alignItems: 'center', gap: 16, marginBottom: 24, width: '100%', maxWidth: 900 }}>
        <span style={{ fontFamily: 'Space Grotesk', fontSize: 18, fontWeight: 900, color: '#d4af37', letterSpacing: '0.08em' }}>
          CARD GALLERY — 52 张扑克牌
        </span>
        <button
          onClick={onClose}
          style={{
            marginLeft: 'auto', background: 'none', border: '1px solid rgba(255,255,255,0.15)',
            borderRadius: 8, color: 'rgba(255,255,255,0.5)', cursor: 'pointer',
            padding: '4px 12px', fontSize: 12,
          }}
        >
          ✕ 关闭
        </button>
      </div>

      {/* 按花色分行 */}
      <div style={{ display: 'flex', flexDirection: 'column', gap: 20, width: '100%', maxWidth: 900 }}>
        {SUITS.map(suit => (
          <div key={suit.key}>
            <div style={{
              fontFamily: 'Manrope', fontSize: 11, fontWeight: 700,
              color: suit.color, letterSpacing: '0.1em', marginBottom: 8, opacity: 0.8,
            }}>
              {suit.label}
            </div>
            <div style={{ display: 'flex', gap: 6, flexWrap: 'wrap' }}>
              {RANKS.map(rank => (
                <PokerCard key={rank + suit.key} code={rank + suit.key} />
              ))}
            </div>
          </div>
        ))}

        {/* 牌背 */}
        <div>
          <div style={{
            fontFamily: 'Manrope', fontSize: 11, fontWeight: 700,
            color: 'rgba(255,255,255,0.3)', letterSpacing: '0.1em', marginBottom: 8,
          }}>
            BACK / HIDDEN
          </div>
          <div style={{ display: 'flex', gap: 6 }}>
            <PokerCard code="?" />
            <PokerCard code="?" small />
          </div>
        </div>
      </div>
    </div>
  )
}
