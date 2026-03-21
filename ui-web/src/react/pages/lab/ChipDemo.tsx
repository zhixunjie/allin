/**
 * 赌场筹码视觉演示 — 使用 CasinoChip 组件渲染 6 种面额筹码
 */
import { getApp, getFreeLayer } from '../../../pixi/app'
import { CasinoChip, CHIP_STYLES } from '../../../pixi/components/CasinoChip'

const DEMO_RADIUS = 52

export function ChipDemo() {
    function show() {
        const layer = getFreeLayer()
        const app = getApp()
        if (!layer || !app) return
        layer.removeChildren()

        const cols = CHIP_STYLES.length
        const totalW = cols * (DEMO_RADIUS * 2 + 24) - 24
        const startX = (app.screen.width - totalW) / 2 + DEMO_RADIUS
        const cy = app.screen.height / 2

        CHIP_STYLES.forEach(({ style }, i) => {
            const chip = new CasinoChip(DEMO_RADIUS, style)
            chip.position.set(startX + i * (DEMO_RADIUS * 2 + 24), cy)
            layer.addChild(chip)
        })
    }

    function clear() {
        getFreeLayer()?.removeChildren()
    }

    return (
        <div className="flex flex-col gap-2">
            <p className="text-[10px] text-white/30 leading-relaxed">
                程序化绘制 6 种面额赌场筹码
            </p>
            <div className="flex gap-2">
                <button
                    onClick={show}
                    className="flex-1 text-xs py-1.5 rounded bg-amber-500/20 hover:bg-amber-500/30 text-amber-300 border border-amber-500/40 transition-colors"
                >
                    显示筹码
                </button>
                <button
                    onClick={clear}
                    className="text-xs px-3 py-1.5 rounded bg-white/5 hover:bg-white/10 text-white/50 hover:text-white/80 transition-colors"
                >
                    清空
                </button>
            </div>
        </div>
    )
}
