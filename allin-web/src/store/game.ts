import { create } from 'zustand'

export interface SeatSnapshot {
  seat_index: number
  user_id: string
  display_name: string
  stack: number
  bet: number
  folded: boolean
  all_in: boolean
  sit_out: boolean
  hole?: string[]
}

export interface GameConfig {
  small_blind: number
  big_blind: number
  min_buy_in: number
  max_buy_in: number
  max_players: number
  action_time_sec: number
}

export interface GameSnapshot {
  street: string
  hand_num: number
  community: string[]
  seats: SeatSnapshot[]
  pot: number
  dealer_seat: number
  action_seat: number
  current_bet: number
  min_raise: number
  config: GameConfig | null
}

interface GameStoreState extends GameSnapshot {
  myUserId: string | null
  myHole: string[]
  deadlineTs: number | null
  callAmount: number
  minRaiseAmount: number

  setMyUserId: (id: string) => void
  applyConnected: (payload: unknown) => void
  applyGameStarted: (payload: unknown) => void
  applyHoleCards: (payload: unknown) => void
  applyCardsDealt: (payload: unknown) => void
  applyStreetStarted: (payload: unknown) => void
  applyActionRequired: (payload: unknown) => void
  applyActionTaken: (payload: unknown) => void
  applyShowdown: (payload: unknown) => void
  applyHandResult: (payload: unknown) => void
  applyPlayerJoined: (payload: unknown) => void
  applyPlayerLeft: (payload: unknown) => void
  reset: () => void
}

const defaultSnapshot: GameSnapshot = {
  street: 'idle',
  hand_num: 0,
  community: [],
  seats: [],
  pot: 0,
  dealer_seat: -1,
  action_seat: -1,
  current_bet: 0,
  min_raise: 0,
  config: null,
}

export const useGameStore = create<GameStoreState>()((set, get) => ({
  ...defaultSnapshot,
  myUserId: null,
  myHole: [],
  deadlineTs: null,
  callAmount: 0,
  minRaiseAmount: 0,

  setMyUserId: (id) => set({ myUserId: id }),

  applyConnected: (payload) => {
    const p = payload as {
      player_id: string
      game_snapshot?: GameSnapshot
    }
    set({ myUserId: p.player_id })
    if (p.game_snapshot) {
      const snap = p.game_snapshot
      const myHole = snap.seats.find((s) => s.user_id === p.player_id)?.hole ?? []
      set({ ...snap, myHole })
    }
  },

  applyGameStarted: (payload) => {
    const p = payload as {
      hand_num: number
      dealer_seat: number
      sb_seat: number
      bb_seat: number
    }
    set({
      hand_num: p.hand_num,
      dealer_seat: p.dealer_seat,
      street: 'preflop',
      community: [],
      pot: 0,
      action_seat: -1,
      current_bet: 0,
      myHole: [],
      deadlineTs: null,
      // Reset seat bets
      seats: get().seats.map((s) => ({ ...s, bet: 0, folded: false, all_in: false, hole: undefined })),
    })
  },

  applyHoleCards: (payload) => {
    const p = payload as { player_id: string; hole: string[] }
    if (p.player_id === get().myUserId) {
      set({ myHole: p.hole })
    }
  },

  applyCardsDealt: (_payload) => {
    // Visual cue only — seat list already updated by applyGameStarted
  },

  applyStreetStarted: (payload) => {
    const p = payload as { street: string; community: string[]; pot: number }
    set({
      street: p.street,
      community: p.community,
      pot: p.pot,
      current_bet: 0,
      action_seat: -1,
      deadlineTs: null,
      seats: get().seats.map((s) => ({ ...s, bet: 0 })),
    })
  },

  applyActionRequired: (payload) => {
    const p = payload as {
      player_id: string
      seat_index: number
      deadline_ts: number
      current_bet: number
      call_amount: number
      min_raise: number
      stack: number
      pot: number
    }
    set({
      action_seat: p.seat_index,
      current_bet: p.current_bet,
      pot: p.pot,
      deadlineTs: p.deadline_ts,
      callAmount: p.call_amount,
      minRaiseAmount: p.min_raise,
    })
  },

  applyActionTaken: (payload) => {
    const p = payload as {
      player_id: string
      action: string
      amount: number
      stack: number
      total_pot: number
    }
    set((state) => ({
      pot: p.total_pot,
      deadlineTs: null,
      seats: state.seats.map((s) => {
        if (s.user_id !== p.player_id) return s
        return {
          ...s,
          stack: p.stack,
          bet: p.action === 'fold' ? s.bet : p.amount,
          folded: p.action === 'fold' ? true : s.folded,
          all_in: p.action === 'all_in' ? true : s.all_in,
        }
      }),
    }))
  },

  applyShowdown: (payload) => {
    const reveals = payload as Array<{
      player_id: string
      seat_index: number
      hole: string[]
      hand_name: string
    }>
    set((state) => ({
      street: 'showdown',
      seats: state.seats.map((s) => {
        const r = reveals.find((x) => x.player_id === s.user_id)
        return r ? { ...s, hole: r.hole } : s
      }),
    }))
  },

  applyHandResult: (payload) => {
    const p = payload as {
      winners: Array<{ player_id: string; amount: number }>
      seats: Array<{ player_id: string; stack: number }>
    }
    if (!p.seats) return
    set((state) => ({
      street: 'idle',
      action_seat: -1,
      deadlineTs: null,
      seats: state.seats.map((s) => {
        const rs = p.seats?.find((x) => x.player_id === s.user_id)
        return rs ? { ...s, stack: rs.stack } : s
      }),
    }))
  },

  applyPlayerJoined: (payload) => {
    const p = payload as {
      player_id: string
      display_name: string
      seat_index: number
      stack: number
      is_reconnect: boolean
    }
    if (p.seat_index < 0) return
    set((state) => {
      const existing = state.seats.find((s) => s.user_id === p.player_id)
      if (existing) return {}
      const newSeat: SeatSnapshot = {
        seat_index: p.seat_index,
        user_id: p.player_id,
        display_name: p.display_name,
        stack: p.stack,
        bet: 0,
        folded: false,
        all_in: false,
        sit_out: false,
      }
      return { seats: [...state.seats, newSeat] }
    })
  },

  applyPlayerLeft: (payload) => {
    const p = payload as { player_id: string; seat_index: number }
    set((state) => ({
      seats: state.seats.filter((s) => s.user_id !== p.player_id),
    }))
  },

  reset: () =>
    set({
      ...defaultSnapshot,
      myHole: [],
      deadlineTs: null,
      callAmount: 0,
      minRaiseAmount: 0,
    }),
}))
