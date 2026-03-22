import { create } from 'zustand'

export interface ChatMessage {
  id: string          // 前端自增序号，用于 React key
  senderId: string    // 发送者玩家 ID
  displayName: string // 发送者显示名
  text: string        // 消息内容
  ts: number          // 发送时间戳（Unix 毫秒）
}

interface ChatStore {
  messages: ChatMessage[]                            // 聊天消息列表（最多保留 200 条，超出时丢弃最旧一条）
  addMessage: (msg: Omit<ChatMessage, 'id'>) => void // 追加一条新消息，id 由 store 内部自增分配
  clear: () => void                                  // 清空消息列表（离开房间时调用）
}

let _seq = 0

export const useChatStore = create<ChatStore>()((set) => ({
  messages: [],
  addMessage: (msg) =>
    set((s) => ({
      messages: [...s.messages.slice(-199), { ...msg, id: String(++_seq) }],
    })),
  clear: () => set({ messages: [] }),
}))
