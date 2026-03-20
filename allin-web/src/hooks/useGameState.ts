import { useGameStore } from '../store/game'
import { Street } from '../types/enums'

export function useGameState() {
  const state = useGameStore()

  const myUserId = state.myUserId
  const mySeat = state.seats.find((s) => s.user_id === myUserId) ?? null
  const isMyTurn =
    myUserId !== null &&
    state.action_seat >= 0 &&
    state.seats.find((s) => s.user_id === myUserId)?.seat_index === state.action_seat

  const canCheck = isMyTurn && state.callAmount === 0
  const canCall = isMyTurn && state.callAmount > 0
  const canBet = isMyTurn && state.current_bet === 0
  const canRaise = isMyTurn && state.current_bet > 0

  const street = state.street
  const isActive = street !== Street.Idle && street !== Street.Showdown

  return {
    ...state,
    mySeat,
    isMyTurn,
    canCheck,
    canCall,
    canBet,
    canRaise,
    isActive,
    myHole: state.myHole,
  }
}
