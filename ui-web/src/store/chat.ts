import { create } from 'zustand'

export interface ChatMessage {
  id: string
  senderId: string
  displayName: string
  text: string
  ts: number
}

interface ChatStore {
  messages: ChatMessage[]
  addMessage: (msg: Omit<ChatMessage, 'id'>) => void
  clear: () => void
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
