import { useState, FormEvent, useEffect, useCallback } from 'react'
import { useNavigate, Link } from 'react-router-dom'
import { roomAPI } from '../../api/http'
import type { RoomConfig } from '../../api/http'
import { useAuthStore } from '../../store/auth'
import { useRoomStore } from '../../store/room'
import { BotStyle } from '../../types/enums'
import styles from './LobbyPage.module.css'

// ── 确认弹窗 ────────────────────────────────────────────
interface ConfirmDialogProps {
  buyIn: number
  balance: number
  action: '创建房间' | '加入房间'
  onConfirm: () => void
  onCancel: () => void
}

function ConfirmDialog({ buyIn, balance, action, onConfirm, onCancel }: ConfirmDialogProps) {
  const remaining = balance - buyIn
  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm"
      onClick={onCancel}
    >
      <div
        className="w-full max-w-sm mx-4 rounded-2xl overflow-hidden bg-[#111820] border border-white/10 shadow-[0_24px_80px_rgba(0,0,0,0.8)] animate-[fadeSlideUp_0.25s_ease]"
        onClick={(e) => e.stopPropagation()}
      >
        {/* 顶部金色装饰线 */}
        <div className="h-0.5 w-full bg-gradient-to-r from-transparent via-amber-400/60 to-transparent" />

        <div className="px-6 py-5 flex flex-col gap-4">
          {/* 标题 */}
          <div>
            <p className="text-[10px] font-bold tracking-[0.25em] text-amber-500/60 uppercase">确认操作</p>
            <h2 className="text-base font-black text-white mt-0.5">{action}</h2>
          </div>

          {/* 金额明细 */}
          <div className="flex flex-col gap-1.5 rounded-xl bg-white/[0.03] border border-white/8 px-4 py-3 text-sm">
            <div className="flex justify-between items-center">
              <span className="text-white/50">买入筹码</span>
              <span className="font-bold text-amber-300">${buyIn.toLocaleString()}</span>
            </div>
            <div className="flex justify-between items-center">
              <span className="text-white/50">当前余额</span>
              <span className="font-semibold text-white/80">${balance.toLocaleString()}</span>
            </div>
            <div className="h-px bg-white/8 my-0.5" />
            <div className="flex justify-between items-center">
              <span className="text-white/50">买入后剩余</span>
              <span className={`font-bold ${remaining < 0 ? 'text-red-400' : 'text-white/80'}`}>
                ${remaining.toLocaleString()}
              </span>
            </div>
          </div>

          {/* 按钮 */}
          <div className="flex gap-3">
            <button
              onClick={onCancel}
              className="flex-1 py-2 rounded-xl border border-white/10 text-white/50 hover:text-white/80 hover:border-white/20 text-sm font-semibold transition-colors"
            >
              取消
            </button>
            <button
              onClick={onConfirm}
              className="flex-1 py-2 rounded-xl bg-amber-500 hover:bg-amber-400 active:bg-amber-600 text-[#0a0f18] text-sm font-black transition-colors"
            >
              确认{action}
            </button>
          </div>
        </div>

        <div className="h-0.5 w-full bg-gradient-to-r from-transparent via-amber-400/30 to-transparent" />
      </div>

      <style>{`
        @keyframes fadeSlideUp {
          from { opacity: 0; transform: translateY(16px); }
          to   { opacity: 1; transform: translateY(0); }
        }
      `}</style>
    </div>
  )
}

export default function LobbyPage() {
  const navigate = useNavigate()
  const { user, clearAuth } = useAuthStore()
  const setRoom = useRoomStore((s) => s.setRoom)

  // 创建房间表单
  const [smallBlind,     setSmallBlind]     = useState('1')
  const [minBuyIn,       setMinBuyIn]       = useState('20')
  const [maxBuyIn,       setMaxBuyIn]       = useState('200')
  const [maxPlayers,     setMaxPlayers]     = useState('3')
  const [actionTimeSec,  setActionTimeSec]  = useState('30')
  const [botCount,       setBotCount]       = useState('2')
  const [botStyle,       setBotStyle]       = useState<BotStyle>(BotStyle.Mixed)

  // 大盲 = 小盲 × 2（自动联动）
  const bigBlind = Number(smallBlind) * 2

  // 加入房间
  const [joinCode, setJoinCode] = useState('')

  const [error,   setError]   = useState('')
  const [loading, setLoading] = useState(false)

  // 确认弹窗
  const [confirmProps, setConfirmProps] = useState<Omit<ConfirmDialogProps, 'onConfirm' | 'onCancel'> | null>(null)
  const [pendingAction, setPendingAction] = useState<(() => void) | null>(null)

  const askConfirm = useCallback(
    (props: Omit<ConfirmDialogProps, 'onConfirm' | 'onCancel'>, onConfirm: () => void) => {
      setConfirmProps(props)
      setPendingAction(() => onConfirm)
    },
    []
  )

  function closeConfirm() {
    setConfirmProps(null)
    setPendingAction(null)
  }

  // 从 URL /join/XXXXXX 自动填充邀请码
  useEffect(() => {
    const match = location.pathname.match(/\/join\/([A-Z0-9]{6})/i)
    if (match) setJoinCode(match[1].toUpperCase())
  }, [])

  async function handleCreate(e: FormEvent) {
    e.preventDefault()
    setError('')
    if (Number(botCount) >= Number(maxPlayers)) {
      setError('AI 玩家数必须小于最大人数，至少留 1 个座位给真人')
      return
    }
    const buyIn = Number(maxBuyIn)
    const balance = user?.chip_balance ?? 0
    if (balance < buyIn) {
      setError(`余额不足：需要 $${buyIn.toLocaleString()}，当前余额 $${balance.toLocaleString()}`)
      return
    }
    askConfirm({ buyIn, balance, action: '创建房间' }, async () => {
      closeConfirm()
      setLoading(true)
      try {
        const config: RoomConfig = {
          small_blind:     Number(smallBlind),
          big_blind:       bigBlind,
          min_buy_in:      Number(minBuyIn),
          max_buy_in:      Number(maxBuyIn),
          max_players:     Number(maxPlayers),
          action_time_sec: Number(actionTimeSec),
          bot_count:       Number(botCount),
          bot_style:       botStyle,
        }
        const room = await roomAPI.create(config)
        setRoom(room)
        navigate(`/room/${room.code}`)
      } catch (err: unknown) {
        setError(err instanceof Error ? err.message : '创建失败')
      } finally {
        setLoading(false)
      }
    })
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
      const buyIn = room.config.max_buy_in
      const balance = user?.chip_balance ?? 0
      if (balance < buyIn) {
        setError(`余额不足：需要 $${buyIn.toLocaleString()}，当前余额 $${balance.toLocaleString()}`)
        return
      }
      askConfirm({ buyIn, balance, action: '加入房间' }, () => {
        closeConfirm()
        setRoom(room)
        navigate(`/room/${room.code}`)
      })
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : '加入失败')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className={styles.container}>
      {confirmProps && (
        <ConfirmDialog
          {...confirmProps}
          onConfirm={() => { pendingAction?.() }}
          onCancel={closeConfirm}
        />
      )}
      <header className={styles.header}>
        <span className={styles.logo}>AllIn</span>
        <div className={styles.userInfo}>
          <Link to="/lab" className={styles.logoutBtn} style={{ textDecoration: 'none' }}>Lab</Link>
          <span className={styles.userName}>{user?.display_name ?? user?.username}</span>
          <span className={styles.chips}>
            <span className={styles.chipIcon}>$</span>
            {user?.chip_balance?.toLocaleString() ?? '—'}
          </span>
          <button className={styles.logoutBtn} onClick={clearAuth}>退出</button>
        </div>
      </header>

      <main className={styles.main}>

        {/* ── 加入房间 ── */}
        <section className={styles.section}>
          <h2 className={styles.sectionTitle}>加入房间</h2>
          <form onSubmit={handleJoin} className={styles.joinForm}>
            <input
              className={styles.codeInput}
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

        {/* ── 创建房间 ── */}
        <section className={styles.section}>
          <h2 className={styles.sectionTitle}>创建房间</h2>
          <form onSubmit={handleCreate} className={styles.createForm}>

            {/* 盲注 */}
            <div className={styles.fieldGroup}>
              <p className={styles.groupLabel}>盲注</p>
              <div className={styles.row}>
                <label className={styles.field}>
                  <span className={styles.fieldLabel}>小盲</span>
                  <div className={styles.moneyInput}>
                    <span className={styles.dollar}>$</span>
                    <input
                      className={styles.numInput}
                      type="number" min="1"
                      value={smallBlind}
                      onChange={(e) => setSmallBlind(e.target.value)}
                    />
                  </div>
                </label>
                <label className={styles.field}>
                  <span className={styles.fieldLabel}>大盲 <span className={styles.hint}>（= 小盲 × 2）</span></span>
                  <div className={`${styles.moneyInput} ${styles.readOnly}`}>
                    <span className={styles.dollar}>$</span>
                    <input className={styles.numInput} type="number" value={bigBlind} readOnly />
                  </div>
                </label>
              </div>
            </div>

            {/* 买入 */}
            <div className={styles.fieldGroup}>
              <p className={styles.groupLabel}>买入金额</p>
              <div className={styles.row}>
                <label className={styles.field}>
                  <span className={styles.fieldLabel}>最低买入 <span className={styles.hint}>（≥ 大盲 × 10）</span></span>
                  <div className={styles.moneyInput}>
                    <span className={styles.dollar}>$</span>
                    <input
                      className={styles.numInput}
                      type="number" min="1"
                      value={minBuyIn}
                      onChange={(e) => setMinBuyIn(e.target.value)}
                    />
                  </div>
                </label>
                <label className={styles.field}>
                  <span className={styles.fieldLabel}>最高买入 <span className={styles.hint}>（入桌初始筹码）</span></span>
                  <div className={styles.moneyInput}>
                    <span className={styles.dollar}>$</span>
                    <input
                      className={styles.numInput}
                      type="number" min="1"
                      value={maxBuyIn}
                      onChange={(e) => setMaxBuyIn(e.target.value)}
                    />
                  </div>
                </label>
              </div>
            </div>

            {/* 房间设置 */}
            <div className={styles.fieldGroup}>
              <p className={styles.groupLabel}>房间设置</p>
              <div className={styles.row}>
                <label className={styles.field}>
                  <span className={styles.fieldLabel}>最大人数</span>
                  <input
                    className={styles.numInput}
                    type="number" min="2" max="9"
                    value={maxPlayers}
                    onChange={(e) => {
                      const v = e.target.value
                      setMaxPlayers(v)
                      setBotCount(String(Math.max(0, Number(v) - 1)))
                    }}
                  />
                </label>
                <label className={styles.field}>
                  <span className={styles.fieldLabel}>行动时限</span>
                  <div className={styles.suffixInput}>
                    <input
                      className={styles.numInput}
                      type="number" min="5" max="120"
                      value={actionTimeSec}
                      onChange={(e) => setActionTimeSec(e.target.value)}
                    />
                    <span className={styles.suffix}>秒</span>
                  </div>
                </label>
              </div>
            </div>

            {/* AI 设置 */}
            <div className={styles.fieldGroup}>
              <p className={styles.groupLabel}>AI 玩家</p>
              <div className={styles.row}>
                <label className={styles.field}>
                  <span className={styles.fieldLabel}>数量 <span className={styles.hint}>（0 = 纯真人）</span></span>
                  <input
                    className={styles.numInput}
                    type="number" min="0" max={Number(maxPlayers) - 1}
                    value={botCount}
                    onChange={(e) => setBotCount(e.target.value)}
                  />
                  {Number(botCount) >= Number(maxPlayers) && (
                    <span className={styles.fieldError}>AI 数不能占满所有座位</span>
                  )}
                </label>
                <label className={styles.field}>
                  <span className={styles.fieldLabel}>风格</span>
                  <select
                    className={styles.numInput}
                    value={botStyle}
                    onChange={(e) => setBotStyle(e.target.value as BotStyle)}
                  >
                    <option value={BotStyle.Mixed}>混合</option>
                    <option value={BotStyle.Aggressive}>激进</option>
                    <option value={BotStyle.Passive}>被动</option>
                    <option value={BotStyle.Random}>随机</option>
                  </select>
                </label>
              </div>
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
