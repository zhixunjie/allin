// WebSocket singleton + simple event bus with auto-reconnect

import { useConnectionStore } from '../store/connection'

type Listener = (payload: unknown) => void

interface Envelope {
  type: string
  seq: number
  ts: number
  payload: unknown
}

class WSClient {
  private socket: WebSocket | null = null
  private listeners: Map<string, Set<Listener>> = new Map()
  private seq = 0

  // Reconnect state
  private roomCode = ''
  private token = ''
  private manualClose = false
  private reconnectAttempts = 0
  private readonly maxReconnectAttempts = 10
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null

  connect(roomCode: string, token: string): void {
    this.roomCode = roomCode
    this.token = token
    this.manualClose = false
    this.reconnectAttempts = 0
    this._connect()
  }

  private _connect(): void {
    if (this.socket) {
      // Prevent old socket from triggering reconnect
      this.socket.onclose = null
      this.socket.close()
      this.socket = null
    }

    const protocol = location.protocol === 'https:' ? 'wss' : 'ws'
    const url = `${protocol}://${location.host}/api/ws?room=${this.roomCode}&token=${this.token}`
    this.socket = new WebSocket(url)

    this.socket.onopen = () => {
      this.reconnectAttempts = 0
      useConnectionStore.getState().set('connected')
      const handlers = this.listeners.get('__open__')
      if (handlers) handlers.forEach((fn) => fn(null))
    }

    this.socket.onmessage = (e) => {
      let env: Envelope
      try {
        env = JSON.parse(e.data)
      } catch {
        return
      }
      const handlers = this.listeners.get(env.type)
      if (handlers) handlers.forEach((fn) => fn(env.payload))
      const all = this.listeners.get('*')
      if (all) all.forEach((fn) => fn(env))
    }

    this.socket.onerror = () => {
      const handlers = this.listeners.get('__error__')
      if (handlers) handlers.forEach((fn) => fn(null))
    }

    this.socket.onclose = () => {
      if (this.manualClose) {
        useConnectionStore.getState().set('disconnected')
        const handlers = this.listeners.get('__close__')
        if (handlers) handlers.forEach((fn) => fn(null))
        return
      }

      // Auto-reconnect with exponential backoff (1s, 2s, 4s … capped at 30s)
      if (this.reconnectAttempts < this.maxReconnectAttempts) {
        const delay = Math.min(1000 * Math.pow(2, this.reconnectAttempts), 30_000)
        this.reconnectAttempts++
        useConnectionStore.getState().set('reconnecting', this.reconnectAttempts)
        this.reconnectTimer = setTimeout(() => this._connect(), delay)
      } else {
        useConnectionStore.getState().set('disconnected')
      }
    }
  }

  disconnect(): void {
    this.manualClose = true
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer)
      this.reconnectTimer = null
    }
    this.socket?.close()
    this.socket = null
  }

  send(type: string, payload: unknown = {}): void {
    if (!this.socket || this.socket.readyState !== WebSocket.OPEN) return
    const env: Envelope = { type, seq: ++this.seq, ts: Date.now(), payload }
    this.socket.send(JSON.stringify(env))
  }

  on(type: string, fn: Listener): () => void {
    if (!this.listeners.has(type)) this.listeners.set(type, new Set())
    this.listeners.get(type)!.add(fn)
    return () => this.listeners.get(type)?.delete(fn)
  }

  off(type: string, fn: Listener): void {
    this.listeners.get(type)?.delete(fn)
  }

  get isOpen(): boolean {
    return this.socket?.readyState === WebSocket.OPEN
  }
}

export const wsClient = new WSClient()
