/**
 * 赌场筹码视觉演示
 * 使用 PixiJS Graphics API 程序化绘制仿真筹码，展示在自由区图层上。
 */
import { getApp, getFreeLayer } from '../../../pixi/app'
import { Graphics, Text } from 'pixi.js'

// ── 筹码规格表 ────────────────────────────────────────────────
const CHIPS: { value: number; color: number; spot: number; label: string }[] = [
    { value:     1, color: 0xdcdce8, spot: 0x888899, label:    '$1' },
    { value:     5, color: 0xc0231e, spot: 0xffffff, label:    '$5' },
    { value:    25, color: 0x1a7a3a, spot: 0xffffff, label:   '$25' },
    { value:   100, color: 0x1a1a1a, spot: 0xd4af37, label:  '$100' },
    { value:   500, color: 0x5a1a8a, spot: 0xffffff, label:  '$500' },
    { value:  1000, color: 0xc07800, spot: 0x1a1a1a, label: '$1000' },
]

/** 在指定坐标绘制一枚赌场筹码 */
function drawChip(
    layer: ReturnType<typeof getFreeLayer>,
    cx: number,
    cy: number,
    R: number,
    chipColor: number,
    spotColor: number,
    denomination: string,
) {
    if (!layer) return
    const g = new Graphics()

    // 1. 金属外沿阴影
    g.circle(cx, cy, R + 2).fill({ color: 0x000000, alpha: 0.35 })

    // 2. 金属外环（银色）
    g.circle(cx, cy, R).fill({ color: 0xc8c8d4 })

    // 3. 主体填充
    g.circle(cx, cy, R - 3).fill({ color: chipColor })

    // 4. 边缘卡槽（8个，每隔45°一个）— 凹陷白色矩形块感
    const NOTCH_COUNT = 8
    const NOTCH_ARC = 0.22        // 每个 notch 的弧长（弧度）
    const NOTCH_R_OUTER = R - 1
    const NOTCH_R_INNER = R - 9
    for (let i = 0; i < NOTCH_COUNT; i++) {
        const angle = (i / NOTCH_COUNT) * Math.PI * 2
        const arcR = (NOTCH_R_OUTER + NOTCH_R_INNER) / 2
        const startAngle = angle - NOTCH_ARC / 2
        g.moveTo(cx + arcR * Math.cos(startAngle), cy + arcR * Math.sin(startAngle))
        g.arc(cx, cy, arcR, startAngle, angle + NOTCH_ARC / 2)
        g.stroke({ color: spotColor, width: NOTCH_R_OUTER - NOTCH_R_INNER, alpha: 0.92 })
    }

    // 5. notch 外侧金属细环（压住 notch 边缘）
    g.moveTo(cx + R - 1, cy)
    g.circle(cx, cy, R - 1).stroke({ color: 0xb0b0bc, width: 1.5, alpha: 0.7 })
    g.moveTo(cx + R - 9, cy)
    g.circle(cx, cy, R - 9).stroke({ color: 0x888898, width: 1,   alpha: 0.5 })

    // 6. 内侧装饰环（齿轮花纹感）
    const DECO_COUNT = 24
    const DECO_R = R - 13
    for (let i = 0; i < DECO_COUNT; i++) {
        const angle = (i / DECO_COUNT) * Math.PI * 2
        const isEven = i % 2 === 0
        const startAngle = angle - Math.PI / DECO_COUNT * 0.6
        g.moveTo(cx + DECO_R * Math.cos(startAngle), cy + DECO_R * Math.sin(startAngle))
        g.arc(cx, cy, DECO_R, startAngle, angle + Math.PI / DECO_COUNT * 0.6)
        g.stroke({ color: 0xffffff, width: isEven ? 2.5 : 1, alpha: isEven ? 0.18 : 0.07 })
    }

    // 7. 内圈背景（深色中心区）
    const CENTER_R = R - 17
    const centerDark = blendDark(chipColor, 0.35)
    g.moveTo(cx + CENTER_R, cy)
    g.circle(cx, cy, CENTER_R).fill({ color: centerDark })
    g.moveTo(cx + CENTER_R, cy)
    g.circle(cx, cy, CENTER_R).stroke({ color: 0xffffff, width: 1, alpha: 0.12 })

    // 8. 中心光泽（径向高光模拟）
    const glowX = cx - CENTER_R * 0.2
    const glowY = cy - CENTER_R * 0.3
    const glowR = CENTER_R * 0.55
    g.moveTo(glowX + glowR, glowY)
    g.circle(glowX, glowY, glowR).fill({ color: 0xffffff, alpha: 0.04 })

    // 9. 面值文字
    const textColor = isLight(chipColor) ? 0x111111 : 0xffffff
    const fontSize = denomination.length > 3 ? Math.round(CENTER_R * 0.52) : Math.round(CENTER_R * 0.62)
    const label = new Text({
        text: denomination,
        style: {
            fontFamily: 'Georgia, serif',
            fontSize,
            fontWeight: '900',
            fill: textColor,
            letterSpacing: denomination.length > 3 ? 1 : 2,
        },
    })
    label.anchor.set(0.5)
    label.position.set(cx, cy)
    g.addChild(label)

    // 10. 最外一圈细描边（勾勒轮廓）
    g.circle(cx, cy, R + 1).stroke({ color: 0x000000, width: 1, alpha: 0.4 })

    layer.addChild(g)
}

/** 将颜色调暗（混入黑色） */
function blendDark(color: number, factor: number): number {
    const r = ((color >> 16) & 0xff) * (1 - factor)
    const g = ((color >>  8) & 0xff) * (1 - factor)
    const b = ( color        & 0xff) * (1 - factor)
    return (Math.round(r) << 16) | (Math.round(g) << 8) | Math.round(b)
}

/** 判断颜色是否偏亮（决定文字用黑/白） */
function isLight(color: number): boolean {
    const r = (color >> 16) & 0xff
    const g = (color >>  8) & 0xff
    const b =  color        & 0xff
    return (r * 0.299 + g * 0.587 + b * 0.114) > 140
}

// ── 组件 ──────────────────────────────────────────────────────

export function ChipDemo() {
    function show() {
        const layer = getFreeLayer()
        const app = getApp()
        if (!layer || !app) return
        layer.removeChildren()

        const R = 52
        const cols = CHIPS.length
        const totalW = cols * (R * 2 + 24) - 24
        const startX = (app.screen.width - totalW) / 2 + R
        const cy = app.screen.height / 2

        CHIPS.forEach((chip, i) => {
            const cx = startX + i * (R * 2 + 24)
            drawChip(layer, cx, cy, R, chip.color, chip.spot, chip.label)
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
