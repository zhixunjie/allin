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
  id: number
  username: string
  display_name: string
  chips: number
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

export const roomAPI = {
  create: (config: RoomConfig) => http.post<Room>('/rooms', config, true),
  get: (code: string) => http.get<Room>(`/rooms/${code}`),
}
