import { create } from 'zustand'
import { ConnectionStatus } from '../types/enums'

export { ConnectionStatus }

interface ConnectionStore {
  status: ConnectionStatus                                    // WebSocket 当前连接状态
  attempt: number                                            // 当前重连尝试次数（首次连接时为 0）
  set: (status: ConnectionStatus, attempt?: number) => void  // 更新连接状态和重连次数
}

export const useConnectionStore = create<ConnectionStore>()((set) => ({
  status: ConnectionStatus.Connected,
  attempt: 0,
  set: (status, attempt = 0) => set({ status, attempt }),
}))
