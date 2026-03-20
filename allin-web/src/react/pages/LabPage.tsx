/**
 * PixiJS 组件调试实验室
 *
 * 路由：/lab（无需登录）
 * 原理：直接操作 useGameStore 注入模拟状态，TableScene 的 subscribe 回调
 *       会自动响应并重绘所有元素，无需修改任何游戏逻辑代码。
 */
import { useEffect, useRef, useState } from 'react'
import { initPixiApp } from '../../pixi/app'
import { useGameStore } from '../../store/game'
import { Street } from '../../types/enums'
import type { SeatSnapshot } from '../../store/game'

// ── 预设场景 ──────────────────────────────────────────────────────────────

/** 构造一个标准座位快照 */
function makeSeat(
  seatIndex: number,
  userId: string,
  name: string,
  stack: number,
  opts: Partial<SeatSnapshot> = {},
): SeatSnapshot {
  return {
    seat_index: seatIndex,
    user_id: userId,
    display_name: name,
    stack,
    bet: 0,
    folded: false,
    all_in: false,
    sit_out: false,
    is_bot: false,
    ...opts,
  }
}

/** 预设场景列表 */
const SCENES: Record<string, () => void> = {
  '空桌': () => {
    useGameStore.setState({
      street: Street.Idle,
      seats: [],
      community: [],
      pot: 0,
      dealer_seat: -1,
      action_seat: -1,
      myUserId: null,
      myHole: [],
      deadlineTs: null,
    })
  },

  '两人等待': () => {
    useGameStore.setState({
      street: Street.Idle,
      myUserId: 'player-0',
      seats: [
        makeSeat(0, 'player-0', '我', 1000),
        makeSeat(3, 'player-1', 'Alice', 800, { is_bot: false }),
      ],
      community: [],
      pot: 0,
      dealer_seat: -1,
      action_seat: -1,
      myHole: [],
      deadlineTs: null,
    })
  },

  'Preflop（轮到我）': () => {
    useGameStore.setState({
      street: Street.PreFlop,
      myUserId: 'player-0',
      dealer_seat: 3,
      action_seat: 0,
      current_bet: 20,
      callAmount: 10,
      minRaiseAmount: 40,
      pot: 30,
      deadlineTs: Date.now() + 25_000,
      myHole: ['Ah', 'Kd'],
      seats: [
        makeSeat(0, 'player-0', '我', 980, { bet: 10 }),
        makeSeat(3, 'player-1', 'Alice', 780, { bet: 20 }),
        makeSeat(6, 'bot-1', 'Bot#1', 1000, { is_bot: true }),
      ],
      community: [],
    })
  },

  'Flop（Bot 行动）': () => {
    useGameStore.setState({
      street: Street.Flop,
      myUserId: 'player-0',
      dealer_seat: 3,
      action_seat: 3,
      current_bet: 0,
      callAmount: 0,
      minRaiseAmount: 20,
      pot: 180,
      deadlineTs: Date.now() + 20_000,
      myHole: ['Ah', 'Kd'],
      seats: [
        makeSeat(0, 'player-0', '我', 900),
        makeSeat(3, 'player-1', 'Alice', 700),
        makeSeat(6, 'bot-1', 'Bot#1', 900, { is_bot: true }),
      ],
      community: ['Ac', '7d', '2h'],
    })
  },

  'River（有人弃牌）': () => {
    useGameStore.setState({
      street: Street.River,
      myUserId: 'player-0',
      dealer_seat: 3,
      action_seat: 0,
      current_bet: 150,
      callAmount: 150,
      minRaiseAmount: 300,
      pot: 620,
      deadlineTs: Date.now() + 15_000,
      myHole: ['Ah', 'Kd'],
      seats: [
        makeSeat(0, 'player-0', '我', 400, { bet: 100 }),
        makeSeat(3, 'player-1', 'Alice', 0, { folded: true }),
        makeSeat(6, 'bot-1', 'Bot#1', 200, { bet: 150, is_bot: true }),
      ],
      community: ['Ac', '7d', '2h', 'Ks', 'Qc'],
    })
  },

  'Showdown': () => {
    useGameStore.setState({
      street: Street.Showdown,
      myUserId: 'player-0',
      dealer_seat: 3,
      action_seat: -1,
      pot: 1200,
      deadlineTs: null,
      myHole: ['Ah', 'Kd'],
      seats: [
        makeSeat(0, 'player-0', '我', 400, { hole: ['Ah', 'Kd'] }),
        makeSeat(3, 'player-1', 'Alice', 300, { hole: ['Qh', 'Qd'] }),
        makeSeat(6, 'bot-1', 'Bot#1', 0, { all_in: true, hole: ['2c', '7s'], is_bot: true }),
      ],
      community: ['Ac', '7d', '2h', 'Ks', 'Qc'],
    })
  },

  '满座（9人）': () => {
    const names = ['我', 'Alice', 'Bob', 'Carol', 'Dave', 'Eve', 'Frank', 'Grace', 'Hank']
    useGameStore.setState({
      street: Street.PreFlop,
      myUserId: 'player-0',
      dealer_seat: 0,
      action_seat: 3,
      current_bet: 20,
      callAmount: 20,
      minRaiseAmount: 40,
      pot: 30,
      deadlineTs: Date.now() + 30_000,
      myHole: ['Jh', 'Js'],
      seats: names.map((name, i) =>
        makeSeat(i, `player-${i}`, name, 1000, {
          bet: i === 1 ? 10 : i === 2 ? 20 : 0,
          is_bot: i > 0,
        }),
      ),
      community: [],
    })
  },
}

// ── 组件 ─────────────────────────────────────────────────────────────────

export default function LabPage() {
  const canvasRef = useRef<HTMLDivElement>(null)
  const [activeScene, setActiveScene] = useState<string>('')

  // 初始化 PixiJS
  useEffect(() => {
    if (!canvasRef.current) return
    let cleanup: (() => void) | null = null
    initPixiApp(canvasRef.current).then((fn) => {
      cleanup = fn
      // 默认加载第一个场景
      const first = Object.keys(SCENES)[0]
      SCENES[first]()
      setActiveScene(first)
    })
    return () => { cleanup?.() }
  }, [])

  function applyScene(name: string) {
    SCENES[name]()
    setActiveScene(name)
  }

  return (
    <div className="flex flex-col h-screen bg-[#060c14] text-white overflow-hidden">

      {/* 顶部标题栏 */}
      <div className="flex items-center gap-4 px-5 py-2 bg-black/60 border-b border-amber-900/30 shrink-0">
        <span className="text-amber-400 font-bold tracking-widest text-sm uppercase">
          PixiJS Lab
        </span>
        <span className="text-white/20 text-xs">牌桌组件调试实验室</span>
      </div>

      <div className="flex flex-1 overflow-hidden">

        {/* 左侧控制面板 */}
        <aside className="w-56 shrink-0 bg-black/40 border-r border-white/5 flex flex-col overflow-y-auto">

          {/* 预设场景 */}
          <Section title="预设场景">
            {Object.keys(SCENES).map((name) => (
              <SceneBtn
                key={name}
                label={name}
                active={activeScene === name}
                onClick={() => applyScene(name)}
              />
            ))}
          </Section>

          {/* 实时状态注入 */}
          <Section title="快速注入">
            <InjectControls />
          </Section>

        </aside>

        {/* 中央画布区 */}
        <main className="flex-1 flex items-center justify-center overflow-hidden">
          <div ref={canvasRef} className="w-full h-full flex items-center justify-center" />
        </main>

        {/* 右侧状态监视器 */}
        <aside className="w-52 shrink-0 bg-black/40 border-l border-white/5 overflow-y-auto">
          <StateMonitor />
        </aside>

      </div>
    </div>
  )
}

// ── 子组件 ────────────────────────────────────────────────────────────────

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div className="p-3 border-b border-white/5">
      <p className="text-[10px] font-bold text-amber-500/70 uppercase tracking-widest mb-2">
        {title}
      </p>
      <div className="flex flex-col gap-1">{children}</div>
    </div>
  )
}

function SceneBtn({ label, active, onClick }: { label: string; active: boolean; onClick: () => void }) {
  return (
    <button
      onClick={onClick}
      className={[
        'text-left text-xs px-3 py-1.5 rounded transition-colors',
        active
          ? 'bg-amber-500/20 text-amber-300 border border-amber-500/40'
          : 'text-white/60 hover:text-white hover:bg-white/5',
      ].join(' ')}
    >
      {label}
    </button>
  )
}

/** 实时注入控件：直接修改当前 GameStore 中的单个字段 */
function InjectControls() {
  const gs = useGameStore()

  function toggleStreet() {
    const streets = [Street.Idle, Street.PreFlop, Street.Flop, Street.Turn, Street.River, Street.Showdown]
    const idx = streets.indexOf(gs.street as Street)
    useGameStore.setState({ street: streets[(idx + 1) % streets.length] })
  }

  function addCommunityCard() {
    const deck = ['As', 'Kh', 'Qd', 'Jc', 'Td', '9s', '8h', '7c']
    const current = gs.community ?? []
    if (current.length >= 5) return
    const card = deck[current.length % deck.length]
    useGameStore.setState({ community: [...current, card] })
  }

  function clearCommunity() {
    useGameStore.setState({ community: [] })
  }

  function cyclePot() {
    const values = [0, 100, 500, 1200, 5000]
    const idx = values.indexOf(gs.pot)
    useGameStore.setState({ pot: values[(idx + 1) % values.length] })
  }

  function toggleMyTurn() {
    const mySeat = gs.seats.find((s) => s.user_id === gs.myUserId)
    if (!mySeat) return
    const isMyTurn = gs.action_seat === mySeat.seat_index
    useGameStore.setState({
      action_seat: isMyTurn ? -1 : mySeat.seat_index,
      deadlineTs: isMyTurn ? null : Date.now() + 30_000,
    })
  }

  return (
    <>
      <Btn label={`Street: ${gs.street}`} onClick={toggleStreet} />
      <Btn label={`公共牌 +1 (${gs.community?.length ?? 0}/5)`} onClick={addCommunityCard} />
      <Btn label="清空公共牌" onClick={clearCommunity} />
      <Btn label={`底池: $${gs.pot}`} onClick={cyclePot} />
      <Btn label="切换我的行动" onClick={toggleMyTurn} />
    </>
  )
}

function Btn({ label, onClick }: { label: string; onClick: () => void }) {
  return (
    <button
      onClick={onClick}
      className="text-left text-xs px-3 py-1.5 rounded bg-white/5 hover:bg-white/10 text-white/70 hover:text-white transition-colors"
    >
      {label}
    </button>
  )
}

/** 右侧状态监视器：实时显示关键 GameStore 字段 */
function StateMonitor() {
  const gs = useGameStore()

  const rows: [string, string][] = [
    ['street', gs.street],
    ['action_seat', String(gs.action_seat)],
    ['dealer_seat', String(gs.dealer_seat)],
    ['pot', `$${gs.pot}`],
    ['community', gs.community?.join(' ') || '—'],
    ['myHole', gs.myHole?.join(' ') || '—'],
    ['seats', String(gs.seats.length)],
    ['deadlineTs', gs.deadlineTs ? `${Math.ceil((gs.deadlineTs - Date.now()) / 1000)}s` : '—'],
  ]

  return (
    <div className="p-3">
      <p className="text-[10px] font-bold text-amber-500/70 uppercase tracking-widest mb-2">
        Store 状态
      </p>
      <div className="flex flex-col gap-1">
        {rows.map(([k, v]) => (
          <div key={k} className="flex justify-between gap-2 text-[11px]">
            <span className="text-white/40 shrink-0">{k}</span>
            <span className="text-amber-200/80 text-right break-all">{v}</span>
          </div>
        ))}
      </div>

      {/* 座位详情 */}
      {gs.seats.length > 0 && (
        <>
          <p className="text-[10px] font-bold text-amber-500/70 uppercase tracking-widest mt-4 mb-2">
            座位列表
          </p>
          {gs.seats.map((s) => (
            <div key={s.seat_index} className="mb-2 text-[10px] leading-relaxed">
              <span className="text-amber-400 font-bold">#{s.seat_index}</span>
              <span className="text-white/60"> {s.display_name}</span>
              <br />
              <span className="text-white/40">
                ${s.stack} | bet:${s.bet}
                {s.folded ? ' 弃' : ''}
                {s.all_in ? ' AI' : ''}
                {s.is_bot ? ' 🤖' : ''}
              </span>
            </div>
          ))}
        </>
      )}
    </div>
  )
}
