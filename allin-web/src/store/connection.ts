import { create } from 'zustand'

export type ConnectionStatus = 'connected' | 'reconnecting' | 'disconnected'

interface ConnectionStore {
  status: ConnectionStatus
  attempt: number
  set: (status: ConnectionStatus, attempt?: number) => void
}

export const useConnectionStore = create<ConnectionStore>()((set) => ({
  status: 'connected',
  attempt: 0,
  set: (status, attempt = 0) => set({ status, attempt }),
}))
