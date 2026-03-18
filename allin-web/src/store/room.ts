import { create } from 'zustand'
import type { Room } from '../api/http'

interface RoomState {
  currentRoom: Room | null
  setRoom: (room: Room) => void
  clearRoom: () => void
}

export const useRoomStore = create<RoomState>()((set) => ({
  currentRoom: null,
  setRoom: (room) => set({ currentRoom: room }),
  clearRoom: () => set({ currentRoom: null }),
}))
