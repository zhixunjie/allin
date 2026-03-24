import { create } from 'zustand'
import { Street, PlayerAction } from '../types/enums'

// ── WebSocket payload 类型定义（与服务端消息结构一一对应） ──

export interface ConnectedPayload {
  player_id: string        // 本次连接对应的用户 ID
  chip_balance?: number    // 买入后账户余额
  game_snapshot?: GameSnapshot // 若房间已在进行中，附带当前完整快照
}

export interface GameStartedPayload {
  hand_num: number    // 本手编号（从 1 递增）
  dealer_seat: number // 庄家座位（服务端索引）
  sb_seat: number     // 小盲座位
  bb_seat: number     // 大盲座位
  small_blind: number // 小盲金额
  big_blind: number   // 大盲金额
}

export interface HoleCardsPayload {
  player_id: string // 收牌玩家（只发给当事人）
  hole: string[]    // 手牌，如 ['Ah', 'Kd']
}

export interface StreetStartedPayload {
  street: Street    // 新街道名称
  community: string[] // 当前公共牌（flop=3张，turn=4张，river=5张）
  pot: number       // 本街开始时的底池
}

export interface ActionRequiredPayload {
  player_id: string  // 需要行动的玩家
  seat_index: number // 其座位索引
  deadline_ts: number // 行动截止时间（Unix 毫秒）
  current_bet: number // 本轮最大下注额
  call_amount: number // 跟注所需金额
  min_raise: number   // 最小加注额
  stack: number       // 玩家当前筹码
  pot: number         // 当前底池
}

export interface ActionTakenPayload {
  player_id: string  // 行动玩家
  action: PlayerAction // 执行的操作
  amount: number     // 操作金额（check/fold 为 0）
  stack: number      // 行动后玩家剩余筹码
  total_pot: number  // 行动后总底池
}

export type ShowdownPayload = Array<{
  player_id: string  // 玩家 ID
  seat_index: number // 座位索引
  hole: string[]     // 公开的手牌
  hand_name: string  // 牌型名称，如 "Full House"
}>

export interface HandResultPayload {
  winners: Array<{ player_id: string; amount: number; hand_name?: string }>
  seats?: Array<{ player_id: string; stack: number }>
  best_hand?: string[]
  all_players?: Array<{
    player_id: string
    display_name: string
    hole: string[]
    hand_name?: string
    folded: boolean
    is_winner: boolean
  }>
  next_hand_delay_sec?: number // 下一手开始前的等待秒数（服务端 handStartDelay）
}

export interface SitOutPayload {
  player_id: string
  seat_index: number
  sit_out: boolean
}

export interface PlayerJoinedPayload {
  player_id: string    // 加入的玩家 ID
  display_name: string // 显示名
  seat_index: number   // 分配的座位（-1 表示旁观）
  stack: number        // 携带筹码
  is_reconnect: boolean // 是否为重连（而非首次加入）
  is_bot?: boolean     // 是否为 AI 机器人
}

export interface PlayerLeftPayload {
  player_id: string  // 离开的玩家 ID
  seat_index: number // 其座位索引
}

// ── 游戏数据结构 ──

/** 单个座位的实时快照 */
export interface SeatSnapshot {
  seat_index: number    // 服务端座位索引（0-8）
  user_id: string       // 玩家 ID
  display_name: string  // 显示名
  stack: number         // 当前筹码
  bet: number           // 本街已下注金额
  folded: boolean       // 是否已弃牌
  all_in: boolean       // 是否全押
  sit_out: boolean      // 是否暂离
  disconnected?: boolean // 是否断线（手牌中保留座位）
  is_bot?: boolean      // 是否 AI 机器人
  hole?: string[]       // 手牌（showdown 时服务端下发）
  avatar?: string       // 玩家头像图片链接
}

/** 房间配置 */
export interface GameConfig {
  small_blind: number    // 小盲注
  big_blind: number      // 大盲注
  min_buy_in: number     // 最小买入
  max_buy_in: number     // 最大买入
  max_players: number    // 最大玩家数
  action_time_sec: number // 每人行动时限（秒）
}

/** 完整游戏快照（连接/重连时同步） */
export interface GameSnapshot {
  street: Street        // 当前街道
  hand_num: number      // 手牌编号
  community: string[]   // 公共牌
  seats: SeatSnapshot[] // 所有座位
  pot: number           // 当前底池
  dealer_seat: number   // 庄家座位（-1 表示未开始）
  action_seat: number   // 当前行动座位（-1 表示无）
  current_bet: number   // 本轮最大下注额
  min_raise: number     // 最小加注额
  config: GameConfig | null // 房间配置（首次连接后下发）
}

/** 单个赢家信息 */
export interface HandWinner {
  player_id: string    // 赢家 ID
  display_name: string // 显示名
  amount: number       // 赢得筹码
  hand_name?: string   // 牌型，如 "Flush"
}

/** 结算弹窗中每位玩家的详情 */
export interface RoundPlayerDetail {
  player_id: string    // 玩家唯一 ID
  display_name: string // 显示名称
  hole: string[]       // 两张手牌（弃牌者为 []，未公开者为 ['?','?']）
  hand_name?: string   // 英文牌型名（如 "Two Pair"）；弃牌者为空
  folded: boolean      // 是否在本手牌中弃牌
  is_winner: boolean   // 是否为本手赢家
}

/** 本手结算结果（用于展示结算弹窗） */
export interface LastHandResult {
  winners: HandWinner[]
  bestHand?: string[]              // 赢家最佳五张（用于顶部展示）
  allPlayers?: RoundPlayerDetail[] // 全场玩家手牌汇总
  nextHandDelaySec: number         // 倒计时总秒数（来自服务端 next_hand_delay_sec）
}

// ── 行动日志（当局实时记录） ──

export interface ActionLogEntry {
  player_id: string    // 执行行动的玩家 ID
  display_name: string // 玩家显示名称（用于日志展示）
  action: PlayerAction // 行动类型：fold / check / call / bet / raise / all_in
  amount: number       // 行动金额；fold / check 为 0
  street: Street       // 行动所在街道（preflop / flop / turn / river）
}

// ── Store 状态定义 ──

interface GameStoreState extends GameSnapshot {
  myUserId: string | null      // 本客户端对应的玩家 ID
  myHole: string[]             // 本人手牌（服务端单独下发，不经过 seats）
  deadlineTs: number | null    // 当前行动截止时间（Unix 毫秒），驱动倒计时
  callAmount: number           // 跟注所需金额（用于 ActionPanel 显示）
  minRaiseAmount: number       // 最小加注额（用于 BetSlider 下限）
  lastHandResult: LastHandResult | null // 最近一手结算结果，null 表示无需展示
  actionLog: ActionLogEntry[]  // 当局行动流水（game_started 时清空）
  readyCount: number           // 结算间隙已准备的玩家数
  readyTotal: number           // 结算间隙合格玩家总数

  setMyUserId: (id: string) => void
  applyConnected: (payload: ConnectedPayload) => void
  applyGameStarted: (payload: GameStartedPayload) => void
  applyHoleCards: (payload: HoleCardsPayload) => void
  applyCardsDealt: (payload: unknown) => void
  applyStreetStarted: (payload: StreetStartedPayload) => void
  applyActionRequired: (payload: ActionRequiredPayload) => void
  applyActionTaken: (payload: ActionTakenPayload) => void
  applyShowdown: (payload: ShowdownPayload) => void
  applyHandResult: (payload: HandResultPayload) => void
  applyPlayerJoined: (payload: PlayerJoinedPayload) => void
  applyPlayerLeft: (payload: PlayerLeftPayload) => void
  applySitOut: (payload: SitOutPayload) => void
  applyReadyStatus: (payload: { ready_count: number; total_count: number }) => void
  reset: () => void
}

const defaultSnapshot: GameSnapshot = {
  street: Street.Idle,
  hand_num: 0,
  community: [],
  seats: [],
  pot: 0,
  dealer_seat: -1,
  action_seat: -1,
  current_bet: 0,
  min_raise: 0,
  config: null,
}

export const useGameStore = create<GameStoreState>()((set, get) => ({
  ...defaultSnapshot,
  myUserId: null,
  myHole: [],
  deadlineTs: null,
  callAmount: 0,
  minRaiseAmount: 0,
  lastHandResult: null,
  actionLog: [],
  readyCount: 0,
  readyTotal: 0,

  setMyUserId: (id) => set({ myUserId: id }),

  // 连接成功：记录本人 ID，若附带快照则直接同步整局状态
  applyConnected: (payload) => {
    set({ myUserId: payload.player_id })
    if (payload.game_snapshot) {
      const snap = payload.game_snapshot
      const myHole = snap.seats.find((s) => s.user_id === payload.player_id)?.hole ?? []
      set({ ...snap, myHole })
    }
  },

  // 新一手开始：重置所有座位状态，保留玩家列表，清空行动日志
  applyGameStarted: (payload) => {
    set({
      hand_num: payload.hand_num,
      dealer_seat: payload.dealer_seat,
      street: Street.PreFlop,
      community: [],
      pot: 0,
      action_seat: -1,
      current_bet: payload.big_blind,
      myHole: [],
      deadlineTs: null,
      actionLog: [],
      lastHandResult: null,
      readyCount: 0,
      readyTotal: 0,
      seats: get().seats.map((s) => {
        let bet = 0
        if (s.seat_index === payload.sb_seat) bet = Math.min(payload.small_blind, s.stack)
        if (s.seat_index === payload.bb_seat) bet = Math.min(payload.big_blind, s.stack)
        return { ...s, bet, folded: false, all_in: false, hole: undefined }
      }),
    })
  },

  // 收到手牌：仅更新本人（服务端只向当事人发送）
  applyHoleCards: (payload) => {
    if (payload.player_id === get().myUserId) {
      set({ myHole: payload.hole })
    }
  },

  // 发牌通知：为所有已发牌座位展示背面牌（? 占位），showdown 时替换为正面
  applyCardsDealt: (payload) => {
    const p = payload as { seats: number[] }
    set((state) => ({
      seats: state.seats.map((s) =>
        p.seats.includes(s.seat_index) ? { ...s, hole: ['?', '?'] } : s
      ),
    }))
  },

  // 新街道：同步公共牌、底池，重置本街下注
  applyStreetStarted: (payload) => {
    set({
      street: payload.street,
      community: payload.community,
      pot: payload.pot,
      current_bet: 0,
      action_seat: -1,
      deadlineTs: null,
      seats: get().seats.map((s) => ({ ...s, bet: 0 })),
    })
  },

  // 轮到某玩家行动：更新行动座位、倒计时及跟注/加注参数
  applyActionRequired: (payload) => {
    set({
      action_seat: payload.seat_index,
      current_bet: payload.current_bet,
      pot: payload.pot,
      deadlineTs: payload.deadline_ts,
      callAmount: payload.call_amount,
      minRaiseAmount: payload.min_raise,
    })
  },

  // 玩家已行动：更新底池、该玩家的筹码/下注/状态，追加行动日志
  applyActionTaken: (payload) => {
    set((state) => {
      const seat = state.seats.find((s) => s.user_id === payload.player_id)
      const entry: ActionLogEntry = {
        player_id: payload.player_id,
        display_name: seat?.display_name ?? payload.player_id,
        action: payload.action,
        amount: payload.amount,
        street: state.street,
      }
      return {
        pot: payload.total_pot,
        deadlineTs: null,
        actionLog: [...state.actionLog, entry],
        seats: state.seats.map((s) => {
          if (s.user_id !== payload.player_id) return s
          return {
            ...s,
            stack: payload.stack,
            bet: payload.action === PlayerAction.Fold ? s.bet : payload.amount,
            folded: payload.action === PlayerAction.Fold ? true : s.folded,
            all_in: payload.action === PlayerAction.AllIn ? true : s.all_in,
          }
        }),
      }
    })
  },

  // 摊牌：公开各玩家手牌
  applyShowdown: (payload) => {
    set((state) => ({
      street: Street.Showdown,
      seats: state.seats.map((s) => {
        const r = payload.find((x) => x.player_id === s.user_id)
        return r ? { ...s, hole: r.hole } : s
      }),
    }))
  },

  // 本手结算：更新各玩家筹码，记录赢家/手牌信息，重置街道为 idle
  applyHandResult: (payload) => {
    set((state) => {
      // 同一玩家可能赢多个底池（主池+边池），按 player_id 合并后只展示一行
      const mergedWinners = payload.winners.reduce<HandWinner[]>((acc, w) => {
        const existing = acc.find((e) => e.player_id === w.player_id)
        if (existing) {
          existing.amount += w.amount
        } else {
          acc.push({
            player_id: w.player_id,
            display_name: state.seats.find((s) => s.user_id === w.player_id)?.display_name ?? w.player_id,
            amount: w.amount,
            hand_name: w.hand_name,
          })
        }
        return acc
      }, [])
      const lastHandResult: LastHandResult = {
        winners: mergedWinners,
        bestHand: payload.best_hand,
        nextHandDelaySec: payload.next_hand_delay_sec ?? 10,
        allPlayers: payload.all_players?.map((p) => ({
          player_id: p.player_id,
          display_name: p.display_name,
          hole: p.hole,
          hand_name: p.hand_name,
          folded: p.folded,
          is_winner: p.is_winner,
        })),
      }
      return {
        street: Street.Idle,
        action_seat: -1,
        deadlineTs: null,
        lastHandResult,
        seats: state.seats.map((s) => {
          const rs = payload.seats?.find((x) => x.player_id === s.user_id)
          return {
            ...s,
            stack: rs ? rs.stack : s.stack,
            bet: 0,
            folded: false,
            all_in: false,
            hole: undefined,
          }
        }),
      }
    })
  },

  // 玩家加入：追加到座位列表；重连时清除断线标记
  applyPlayerJoined: (payload) => {
    if (payload.seat_index < 0) return
    set((state) => {
      const existing = state.seats.find((s) => s.user_id === payload.player_id)
      if (existing) {
        // 重连：清除可能因快照遗留的 disconnected 标记
        if (payload.is_reconnect) {
          return {
            seats: state.seats.map((s) =>
              s.user_id === payload.player_id ? { ...s, disconnected: false } : s
            ),
          }
        }
        return {}
      }
      const newSeat: SeatSnapshot = {
        seat_index: payload.seat_index,
        user_id: payload.player_id,
        display_name: payload.display_name,
        stack: payload.stack,
        bet: 0,
        folded: false,
        all_in: false,
        sit_out: false,
        is_bot: payload.is_bot ?? false,
      }
      return { seats: [...state.seats, newSeat] }
    })
  },

  // 玩家离开：从座位列表移除
  applyPlayerLeft: (payload) => {
    set((state) => ({
      seats: state.seats.filter((s) => s.user_id !== payload.player_id),
    }))
  },

  // 玩家离座/归座状态变更
  applySitOut: (payload) => {
    set((state) => ({
      seats: state.seats.map((s) =>
        s.user_id === payload.player_id ? { ...s, sit_out: payload.sit_out } : s
      ),
    }))
  },

  // 结算间隙准备状态更新
  applyReadyStatus: (payload) => {
    set({ readyCount: payload.ready_count, readyTotal: payload.total_count })
  },

  // 离开房间时完整重置
  reset: () =>
    set({
      ...defaultSnapshot,
      myHole: [],
      deadlineTs: null,
      callAmount: 0,
      minRaiseAmount: 0,
      lastHandResult: null,
      actionLog: [],
      readyCount: 0,
      readyTotal: 0,
    }),
}))
