/**
 * 调试实验室的预设场景数据。
 * 每个场景是一个函数，调用时直接覆写 GameStore 状态，
 * TableScene 的 subscribe 会自动响应并重绘。
 */
import { useGameStore } from '../../../store/game'
import { Street } from '../../../types/enums'
import type { SeatSnapshot } from '../../../store/game'

/** 构造一个标准座位快照，opts 可覆盖任意字段 */
export function makeSeat(
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

/** 预设场景列表（名称 → 注入函数） */
export const SCENES: Record<string, () => void> = {
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
        makeSeat(3, 'player-1', 'Alice', 800, { is_bot: true }),
        makeSeat(4, 'player-2', 'Alice', 800, { is_bot: true }),
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

  '特殊行动': () => {
    // 9 人满座，每个玩家展示一种特殊状态，方便对比所有效果
    useGameStore.setState({
      street: Street.River,
      myUserId: 'player-0',
      dealer_seat: 4,
      action_seat: 0,
      current_bet: 300,
      callAmount: 300,
      minRaiseAmount: 600,
      pot: 1800,
      deadlineTs: Date.now() + 20_000,
      myHole: ['Ah', 'Kd'],
      community: ['Ac', '7d', '2h', 'Ks', 'Qc'],
      seats: [
        makeSeat(0, 'player-0', '我',      500,  { bet: 200, hole: ['Ah', 'Kd'] }),              // 行动中
        makeSeat(1, 'bot-1',    'Alice',   0,    { bet: 300, all_in: true, is_bot: true, hole: ['?', '?'] }),  // ALL-IN
        makeSeat(2, 'bot-2',    'Bob',     800,  { folded: true, is_bot: true }),                 // 已弃牌
        makeSeat(3, 'bot-3',    'Carol',   1200, { bet: 100, is_bot: true, hole: ['?', '?'] }),   // 有下注
        makeSeat(4, 'bot-4',    'Dave',    600,  { is_bot: true, hole: ['?', '?'] }),             // 普通等待
        makeSeat(5, 'bot-5',    'Eve',     0,    { all_in: true, bet: 500, is_bot: true, hole: ['?', '?'] }), // ALL-IN+下注
        makeSeat(6, 'bot-6',    'Frank',   900,  { folded: true, is_bot: true }),                 // 已弃牌
        makeSeat(7, 'bot-7',    'Grace',   400,  { bet: 50, is_bot: true, hole: ['?', '?'] }),    // 有下注
        makeSeat(8, 'bot-8',    'Hank',    750,  { is_bot: true, hole: ['?', '?'] }),             // 普通等待
      ],
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
