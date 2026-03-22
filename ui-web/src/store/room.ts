import { create } from 'zustand'
import type { Room } from '../api/http'

interface RoomState {
  currentRoom: Room | null          // 当前所在房间信息，不在房间时为 null
  selectedBuyIn: number             // 玩家选定的买入金额；0 表示使用服务端默认值（max_buy_in）
  setRoom: (room: Room) => void     // 设置当前房间（加入/创建成功后调用）
  clearRoom: () => void             // 清除房间状态（离开房间后调用）
  setSelectedBuyIn: (v: number) => void // 更新选定买入金额（LobbyPage 滑块联动）
}

export const useRoomStore = create<RoomState>()((set) => ({
  currentRoom: null,
  selectedBuyIn: 0,
  setRoom: (room) => set({ currentRoom: room }),
  clearRoom: () => set({ currentRoom: null }),
  setSelectedBuyIn: (v) => set({ selectedBuyIn: v }),
}))
