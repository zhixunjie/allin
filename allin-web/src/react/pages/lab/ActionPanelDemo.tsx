/**
 * 行动面板演示 — 注入不同的回合状态，让 ActionPanel 显示在画布底部。
 */
import { useGameStore } from '../../../store/game'
import { Street } from '../../../types/enums'
import { makeSeat } from './scenes'

const MY_USER = 'lab-player'
const MY_SEAT = 0

const BASE_SEATS = [
    makeSeat(MY_SEAT, MY_USER, '我', 3000, { bet: 0 }),
    makeSeat(1, 'bot-1', 'Alice', 1200, { bet: 100, is_bot: true }),
    makeSeat(3, 'bot-2', 'Bob',    900, { bet: 0,   is_bot: true }),
]

type Scenario = 'hidden' | 'check_bet' | 'call_raise'

const SCENARIOS: { key: Scenario; label: string; desc: string }[] = [
    { key: 'hidden',     label: '隐藏',   desc: '非我的回合' },
    { key: 'check_bet',  label: '过牌/下注', desc: 'callAmount=0，无人下注' },
    { key: 'call_raise', label: '跟注/加注', desc: 'callAmount=100，有人下注' },
]

export function ActionPanelDemo({ active, onToggle }: { active: Scenario; onToggle: (s: Scenario) => void }) {
    function apply(key: Scenario) {
        onToggle(key)
        if (key === 'hidden') {
            useGameStore.setState({ action_seat: -1, deadlineTs: null })
            return
        }
        const isCallRaise = key === 'call_raise'
        useGameStore.setState({
            street: Street.PreFlop,
            myUserId: MY_USER,
            dealer_seat: 3,
            action_seat: MY_SEAT,
            current_bet: isCallRaise ? 100 : 0,
            callAmount:  isCallRaise ? 100 : 0,
            minRaiseAmount: isCallRaise ? 200 : 20,
            pot: isCallRaise ? 250 : 30,
            deadlineTs: Date.now() + 30_000,
            myHole: ['Ah', 'Kd'],
            community: [],
            seats: BASE_SEATS,
        })
    }

    return (
        <div className="flex flex-col gap-2">
            <p className="text-[10px] text-white/30 leading-relaxed">
                注入状态让行动面板出现在画布底部
            </p>
            <div className="flex flex-col gap-1">
                {SCENARIOS.map(s => (
                    <button
                        key={s.key}
                        onClick={() => apply(s.key)}
                        className={[
                            'text-left text-xs px-3 py-1.5 rounded border transition-colors',
                            active === s.key
                                ? 'bg-amber-500/20 text-amber-300 border-amber-500/40'
                                : 'text-white/60 border-white/10 hover:text-white hover:bg-white/5',
                        ].join(' ')}
                    >
                        <span className="font-bold">{s.label}</span>
                        <span className="text-white/30 ml-1.5 text-[10px]">{s.desc}</span>
                    </button>
                ))}
            </div>
        </div>
    )
}
