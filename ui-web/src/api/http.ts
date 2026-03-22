const BASE = '/api'

function getToken(): string | null {
  return localStorage.getItem('token')
}

async function request<T>(
  method: string,
  path: string,
  body?: unknown,
  auth = false,
): Promise<T> {
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
  }
  if (auth) {
    const token = getToken()
    if (token) headers['Authorization'] = `Bearer ${token}`
  }

  const res = await fetch(`${BASE}${path}`, {
    method,
    headers,
    body: body !== undefined ? JSON.stringify(body) : undefined,
  })

  const data = await res.json().catch(() => ({}))
  if (!res.ok) {
    throw new Error(data.error ?? data.message ?? `HTTP ${res.status}`)
  }
  return data as T
}

export const http = {
  get: <T>(path: string, auth = false) => request<T>('GET', path, undefined, auth),
  post: <T>(path: string, body: unknown, auth = false) => request<T>('POST', path, body, auth),
}

// --- 类型化的 API 调用 ---

export interface User {
  id: string
  username: string
  display_name: string
  chip_balance: number
}

export interface AuthResponse {
  token: string
  user: User
}

export interface RoomConfig {
  small_blind: number
  big_blind: number
  min_buy_in: number
  max_buy_in: number
  max_players: number
  action_time_sec?: number
  bot_count?: number
  bot_style?: string
}

export interface Room {
  code: string
  config: RoomConfig
  player_count: number
  created_at: string
}

export const authAPI = {
  register: (username: string, password: string, display_name: string) =>
    http.post<AuthResponse>('/auth/register', { username, password, display_name }),
  login: (username: string, password: string) =>
    http.post<AuthResponse>('/auth/login', { username, password }),
  me: () => http.get<User>('/me', true),
}

export interface HandWinner {
  player_id: string   // 赢家玩家 ID
  amount: number      // 本手获得的筹码总额（已合并主池+边池）
  hand_name?: string  // 获胜牌型名称（如 "Full House"），无竞争时为空
}

export interface HandPlayer {
  player_id: string    // 玩家唯一 ID
  display_name: string // 显示名称
  hole: string[]       // 手牌字符串数组（如 ["Ac","Kd"]）；弃牌者为空数组
  hand_name?: string   // 最终牌型名称；弃牌者或未到摊牌阶段时为空
  folded: boolean      // 是否在本手牌中弃牌
  is_winner: boolean   // 是否为本手赢家（可能多人平分）
}

export interface HandAction {
  player_id: string // 执行行动的玩家 ID
  action: string    // 行动类型：fold / check / call / bet / raise / all_in
  amount: number    // 行动金额；fold / check 为 0
  street: string    // 行动所在街道：preflop / flop / turn / river
}

export interface HandEntry {
  hand_num: number
  result: {
    winners: HandWinner[]
    all_players: HandPlayer[]
    best_hand?: string[]
    seats?: Array<{ player_id: string; stack: number }>
  }
  actions: HandAction[]
  played_at: string
}

export const chipsAPI = {
  claim: () => http.post<{ chip_balance: number }>('/chips/claim', {}, true),
}

export const roomAPI = {
  create: (config: RoomConfig) => http.post<Room>('/rooms', config, true),
  get: (code: string) => http.get<Room>(`/rooms/${code}`),
  getHands: (code: string) => http.get<HandEntry[]>(`/rooms/${code}/hands`, true),
}
