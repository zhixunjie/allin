import { create } from 'zustand'
import { ConnectionStatus } from '../types/enums'

export { ConnectionStatus }

interface ConnectionStore {
  status: ConnectionStatus
  attempt: number
  set: (status: ConnectionStatus, attempt?: number) => void
}

export const useConnectionStore = create<ConnectionStore>()((set) => ({
  status: ConnectionStatus.Connected,
  attempt: 0,
  set: (status, attempt = 0) => set({ status, attempt }),
}))
