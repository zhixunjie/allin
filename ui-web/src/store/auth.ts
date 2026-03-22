import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import type { User } from '../api/http'

interface AuthState {
  token: string | null                              // JWT 认证令牌，未登录时为 null
  user: User | null                                 // 当前登录用户信息，未登录时为 null
  setAuth: (token: string, user: User) => void      // 登录成功后保存 token 和用户信息
  clearAuth: () => void                             // 登出，清除 token 和用户信息
  isLoggedIn: () => boolean                         // 判断当前是否已登录
  updateChipBalance: (balance: number) => void      // 刷新账户余额（/api/me 轮询后调用）
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set, get) => ({
      token: null as string | null,
      user: null as User | null,
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
