// WebSocket singleton + simple event bus

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

  connect(roomCode: string, token: string): void {
    if (this.socket) this.disconnect()

    const protocol = location.protocol === 'https:' ? 'wss' : 'ws'
    const url = `${protocol}://${location.host}/api/ws?room=${roomCode}&token=${token}`
    this.socket = new WebSocket(url)

    this.socket.onopen = () => {
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
      if (handlers) {
        handlers.forEach((fn) => fn(env.payload))
      }
      // wildcard listeners
      const all = this.listeners.get('*')
      if (all) all.forEach((fn) => fn(env))
    }

    this.socket.onclose = () => {
      const handlers = this.listeners.get('__close__')
      if (handlers) handlers.forEach((fn) => fn(null))
    }

    this.socket.onerror = () => {
      const handlers = this.listeners.get('__error__')
      if (handlers) handlers.forEach((fn) => fn(null))
    }
  }

  disconnect(): void {
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
