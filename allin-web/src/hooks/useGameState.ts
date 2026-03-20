import { useGameStore } from '../store/game'
import { Street } from '../types/enums'

/**
 * 对原始 GameStore 状态做二次加工，派生出 UI 直接可用的布尔标志。
 *
 * 使用场景：ActionPanel、BetSlider 等需要判断"当前是否轮到我"以及
 * "我能做哪些操作"的组件，通过这个 hook 而非直接读 store，
 * 避免在每个组件里重复相同的派生逻辑。
 */
export function useGameState() {
  const state = useGameStore()

  const myUserId = state.myUserId

  // 找到本人的座位快照（未入座时为 null）
  const mySeat = state.seats.find((s) => s.user_id === myUserId) ?? null

  // 判断是否轮到本人行动：
  // 1. 已登录（myUserId 非空）
  // 2. 存在待行动座位（action_seat >= 0）
  // 3. 本人座位索引等于当前待行动座位
  const isMyTurn =
    myUserId !== null &&
    state.action_seat >= 0 &&
    state.seats.find((s) => s.user_id === myUserId)?.seat_index === state.action_seat

  // 可用操作标志（仅在轮到自己时为 true）
  const canCheck = isMyTurn && state.callAmount === 0  // 本轮无人下注，可过牌
  const canCall  = isMyTurn && state.callAmount > 0   // 本轮有人下注，可跟注
  const canBet   = isMyTurn && state.current_bet === 0 // 本轮尚无下注，可首次下注
  const canRaise = isMyTurn && state.current_bet > 0  // 本轮已有下注，可加注

  const street = state.street

  // 牌局进行中（排除 idle 等待阶段和 showdown 结算阶段）
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
