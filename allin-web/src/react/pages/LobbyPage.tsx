import { useState, FormEvent, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { roomAPI } from '../../api/http'
import type { RoomConfig } from '../../api/http'
import { useAuthStore } from '../../store/auth'
import { useRoomStore } from '../../store/room'
import styles from './LobbyPage.module.css'

export default function LobbyPage() {
  const navigate = useNavigate()
  const { user, clearAuth } = useAuthStore()
  const setRoom = useRoomStore((s) => s.setRoom)

  // create room form
  const [smallBlind, setSmallBlind] = useState('1')
  const [bigBlind, setBigBlind] = useState('2')
  const [minBuyIn, setMinBuyIn] = useState('40')
  const [maxBuyIn, setMaxBuyIn] = useState('200')
  const [maxPlayers, setMaxPlayers] = useState('9')

  // join room
  const [joinCode, setJoinCode] = useState('')

  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  // auto fill invite code from URL  /join/XXXXXX
  useEffect(() => {
    const match = location.pathname.match(/\/join\/([A-Z0-9]{6})/i)
    if (match) setJoinCode(match[1].toUpperCase())
  }, [])

  async function handleCreate(e: FormEvent) {
    e.preventDefault()
    setError('')
    setLoading(true)
    try {
      const config: RoomConfig = {
        small_blind: Number(smallBlind),
        big_blind: Number(bigBlind),
        min_buy_in: Number(minBuyIn),
        max_buy_in: Number(maxBuyIn),
        max_players: Number(maxPlayers),
      }
      const room = await roomAPI.create(config)
      setRoom(room)
      navigate(`/room/${room.code}`)
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : '创建失败')
    } finally {
      setLoading(false)
    }
  }

  async function handleJoin(e: FormEvent) {
    e.preventDefault()
    setError('')
    const code = joinCode.trim().toUpperCase()
    if (code.length !== 6) {
      setError('房间码应为 6 位')
      return
    }
    setLoading(true)
    try {
      const room = await roomAPI.get(code)
      setRoom(room)
      navigate(`/room/${room.code}`)
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : '加入失败')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className={styles.container}>
      <header className={styles.header}>
        <span className={styles.logo}>AllIn</span>
        <div className={styles.userInfo}>
          <span>{user?.display_name ?? user?.username}</span>
          <span className={styles.chips}>🪙 {user?.chips?.toLocaleString()}</span>
          <button className={styles.logoutBtn} onClick={clearAuth}>
            退出
          </button>
        </div>
      </header>

      <main className={styles.main}>
        {/* Join room */}
        <section className={styles.section}>
          <h2>加入房间</h2>
          <form onSubmit={handleJoin} className={styles.joinForm}>
            <input
              className={styles.input}
              type="text"
              placeholder="输入 6 位房间码"
              value={joinCode}
              onChange={(e) => setJoinCode(e.target.value.toUpperCase())}
              maxLength={6}
            />
            <button className={styles.primaryBtn} type="submit" disabled={loading}>
              加入
            </button>
          </form>
        </section>

        <div className={styles.divider}>— 或 —</div>

        {/* Create room */}
        <section className={styles.section}>
          <h2>创建房间</h2>
          <form onSubmit={handleCreate} className={styles.createForm}>
            <div className={styles.row}>
              <label>
                小盲
                <input
                  className={styles.numInput}
                  type="number"
                  min="1"
                  value={smallBlind}
                  onChange={(e) => setSmallBlind(e.target.value)}
                />
              </label>
              <label>
                大盲
                <input
                  className={styles.numInput}
                  type="number"
                  min="1"
                  value={bigBlind}
                  onChange={(e) => setBigBlind(e.target.value)}
                />
              </label>
              <label>
                最大人数
                <input
                  className={styles.numInput}
                  type="number"
                  min="2"
                  max="9"
                  value={maxPlayers}
                  onChange={(e) => setMaxPlayers(e.target.value)}
                />
              </label>
            </div>
            <div className={styles.row}>
              <label>
                最小买入
                <input
                  className={styles.numInput}
                  type="number"
                  min="1"
                  value={minBuyIn}
                  onChange={(e) => setMinBuyIn(e.target.value)}
                />
              </label>
              <label>
                最大买入
                <input
                  className={styles.numInput}
                  type="number"
                  min="1"
                  value={maxBuyIn}
                  onChange={(e) => setMaxBuyIn(e.target.value)}
                />
              </label>
            </div>
            {error && <p className={styles.error}>{error}</p>}
            <button className={styles.primaryBtn} type="submit" disabled={loading}>
              {loading ? '请稍候...' : '创建房间'}
            </button>
          </form>
        </section>
      </main>
    </div>
  )
}
