import { create } from 'zustand'
import type { Room } from '../api/http'

interface RoomState {
  currentRoom: Room | null
  selectedBuyIn: number  // 0 = use server default (max_buy_in)
  setRoom: (room: Room) => void
  clearRoom: () => void
  setSelectedBuyIn: (v: number) => void
}

export const useRoomStore = create<RoomState>()((set) => ({
  currentRoom: null,
  selectedBuyIn: 0,
  setRoom: (room) => set({ currentRoom: room }),
  clearRoom: () => set({ currentRoom: null }),
  setSelectedBuyIn: (v) => set({ selectedBuyIn: v }),
}))
