/**
 * 行动轮转演示
 *
 * 自动循环模拟每个玩家的行动回合：
 *   1. 切换到该玩家 → 头像亮起 + 倒计时启动
 *   2. 3 秒后玩家"行动"→ 显示下注徽章
 *   3. 再过 1.5 秒 → 进入下一个玩家
 */
import {useEffect, useRef, useState} from 'react'
import {useGameStore} from '../../../store/game'
import {Street} from '../../../types/enums'
import {makeSeat} from './scenes'

// ── 演示配置 ──────────────────────────────────────────────────

const PLAYERS = [
    {seatIdx: 0, userId: 'player-0', name: '我', stack: 1000, isBot: false},
    {seatIdx: 1, userId: 'bot-1', name: 'Alice', stack: 800, isBot: true},
    {seatIdx: 3, userId: 'bot-2', name: 'Bob', stack: 1200, isBot: true},
    {seatIdx: 5, userId: 'bot-3', name: 'Carol', stack: 600, isBot: true},
    {seatIdx: 7, userId: 'bot-4', name: 'Dave', stack: 900, isBot: true},
]

// 每个玩家的行动：bet > 0 = 下注；bet = 0 = 过牌
const ACTIONS = [
    {bet: 100},
    {bet: 200},
    {bet: 50},
    {bet: 150},
    {bet: 0},  // 过牌
]

const TURN_SECS = 5      // 每轮倒计时秒数
const ACT_AFTER_MS = 3000   // 开始后多久"行动"（显示下注）
const NEXT_AFTER_MS = 4800   // 开始后多久切换到下一玩家

// ── 组件 ──────────────────────────────────────────────────────

export function ActionDemo() {
    const [running, setRunning] = useState(false)
    const [turnIdx, setTurnIdx] = useState(0)
    const betsRef = useRef<number[]>(PLAYERS.map(() => 0))
    const timers = useRef<ReturnType<typeof setTimeout>[]>([])

    // 初始化牌桌
    function setup() {
        betsRef.current = PLAYERS.map(() => 0)
        useGameStore.setState({
            street: Street.PreFlop,
            myUserId: 'player-0',
            dealer_seat: 7,
            action_seat: -1,
            current_bet: 0,
            callAmount: 0,
            minRaiseAmount: 20,
            pot: 30,
            deadlineTs: null,
            myHole: ['Ah', 'Kd'],
            community: [],
            seats: PLAYERS.map((p) => makeSeat(
                p.seatIdx, p.userId, p.name, p.stack,
                // 本地玩家手牌通过 myHole 传递；远程玩家用 '?' 占位显示背面
                {bet: 0, is_bot: p.isBot, hole: p.isBot ? ['?', '?'] : undefined},
            )),
        })
    }

    // 执行第 idx 轮
    function runTurn(idx: number) {
        const player = PLAYERS[idx]
        const action = ACTIONS[idx]

        // 阶段 1：激活该玩家，启动倒计时
        useGameStore.setState({
            action_seat: player.seatIdx,
            deadlineTs: Date.now() + TURN_SECS * 1000,
        })

        // 阶段 2：玩家行动（显示下注）
        const t1 = setTimeout(() => {
            betsRef.current[idx] = action.bet
            useGameStore.setState({
                seats: PLAYERS.map((p, i) => makeSeat(
                    p.seatIdx, p.userId, p.name,
                    p.stack - betsRef.current[i],
                    {bet: betsRef.current[i], is_bot: p.isBot, hole: p.isBot ? ['?', '?'] : undefined},
                )),
            })
        }, ACT_AFTER_MS)

        // 阶段 3：切换下一玩家
        const t2 = setTimeout(() => {
            const next = (idx + 1) % PLAYERS.length
            setTurnIdx(next)
        }, NEXT_AFTER_MS)

        timers.current.push(t1, t2)
    }

    // turnIdx 变化时触发下一轮
    useEffect(() => {
        if (!running) return
        runTurn(turnIdx)
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [running, turnIdx])

    function start() {
        setup()
        timers.current.forEach(clearTimeout)
        timers.current = []
        setTurnIdx(0)
        setRunning(true)
    }

    function stop() {
        timers.current.forEach(clearTimeout)
        timers.current = []
        setRunning(false)
        useGameStore.setState({action_seat: -1, deadlineTs: null})
    }

    // 卸载时清理
    useEffect(() => () => {
        timers.current.forEach(clearTimeout)
    }, [])

    const currentName = PLAYERS[turnIdx].name

    return (
        <div className="flex flex-col gap-2">
            <div className="flex gap-2 items-center">
                <button
                    onClick={running ? stop : start}
                    className={[
                        'flex-1 text-xs py-1.5 rounded border transition-colors',
                        running
                            ? 'bg-red-500/20 text-red-300 border-red-500/40 hover:bg-red-500/30'
                            : 'bg-amber-500/20 text-amber-300 border-amber-500/40 hover:bg-amber-500/30',
                    ].join(' ')}
                >
                    {running ? '■ 停止' : '▶ 开始演示'}
                </button>
            </div>

            {running && (
                <p className="text-[10px] text-white/40 font-mono px-1">
                    轮到 <span className="text-amber-300">{currentName}</span> 行动
                    &nbsp;·&nbsp;{TURN_SECS}s 倒计时
                </p>
            )}

            <p className="text-[10px] text-white/25 leading-relaxed">
                每轮 {TURN_SECS}s，{ACT_AFTER_MS / 1000}s 后玩家行动并显示下注徽章
            </p>
        </div>
    )
}
