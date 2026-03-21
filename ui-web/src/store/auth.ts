import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import type { User } from '../api/http'

interface AuthState {
  token: string | null
  user: User | null
  setAuth: (token: string, user: User) => void
  clearAuth: () => void
  isLoggedIn: () => boolean
  updateChipBalance: (balance: number) => void
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set, get) => ({
      token: null,
      user: null,
      setAuth: (token, user) => {
        localStorage.setItem('token', token)
        set({ token, user })
      },
      clearAuth: () => {
        localStorage.removeItem('token')
        set({ token: null, user: null })
      },
      isLoggedIn: () => get().token !== null,
      updateChipBalance: (balance) => {
        const u = get().user
        if (u) set({ user: { ...u, chip_balance: balance } })
      },
    }),
    {
      name: 'allin-auth',
      partialize: (state) => ({ token: state.token, user: state.user }),
    },
  ),
)
