/**
 * 快速注入面板：通过按钮直接修改 GameStore 中的单个字段，
 * 用于在不切换整个预设场景的情况下微调某个状态。
 */
import { useGameStore } from '../../../store/game'
import { Street } from '../../../types/enums'

const STREETS = [
  Street.Idle,
  Street.PreFlop,
  Street.Flop,
  Street.Turn,
  Street.River,
  Street.Showdown,
]

const COMMUNITY_DECK = ['As', 'Kh', 'Qd', 'Jc', 'Td', '9s', '8h', '7c']
const POT_VALUES = [0, 100, 500, 1200, 5000]

export function InjectControls() {
  const gs = useGameStore()

  function toggleStreet() {
    const idx = STREETS.indexOf(gs.street as Street)
    useGameStore.setState({ street: STREETS[(idx + 1) % STREETS.length] })
  }

  function addCommunityCard() {
    const current = gs.community ?? []
    if (current.length >= 5) return
    useGameStore.setState({ community: [...current, COMMUNITY_DECK[current.length % COMMUNITY_DECK.length]] })
  }

  function clearCommunity() {
    useGameStore.setState({ community: [] })
  }

  function cyclePot() {
    const idx = POT_VALUES.indexOf(gs.pot)
    useGameStore.setState({ pot: POT_VALUES[(idx + 1) % POT_VALUES.length] })
  }

  function toggleMyTurn() {
    const mySeat = gs.seats.find((s) => s.user_id === gs.myUserId)
    if (!mySeat) return
    const isMyTurn = gs.action_seat === mySeat.seat_index
    useGameStore.setState({
      action_seat: isMyTurn ? -1 : mySeat.seat_index,
      deadlineTs: isMyTurn ? null : Date.now() + 30_000,
    })
  }

  return (
    <>
      <Btn label={`Street: ${gs.street}`} onClick={toggleStreet} />
      <Btn label={`公共牌 +1 (${gs.community?.length ?? 0}/5)`} onClick={addCommunityCard} />
      <Btn label="清空公共牌" onClick={clearCommunity} />
      <Btn label={`底池: $${gs.pot}`} onClick={cyclePot} />
      <Btn label="切换我的行动" onClick={toggleMyTurn} />
    </>
  )
}

function Btn({ label, onClick }: { label: string; onClick: () => void }) {
  return (
    <button
      onClick={onClick}
      className="text-left text-xs px-3 py-1.5 rounded bg-white/5 hover:bg-white/10 text-white/70 hover:text-white transition-colors"
    >
      {label}
    </button>
  )
}
