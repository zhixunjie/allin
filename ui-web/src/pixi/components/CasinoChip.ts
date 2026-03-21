import { Container, Graphics, Text } from 'pixi.js'

/** 筹码外观配置 */
export interface ChipStyle {
    color: number     // 主体颜色
    spot: number      // 卡槽颜色
    label: string     // 面值文字
}

/** 面额 → 外观映射表（供外部复用） */
export const CHIP_STYLES: { maxAmount: number; style: ChipStyle }[] = [
    { maxAmount:     4, style: { color: 0xdcdce8, spot: 0x888899, label:    '$1' } },
    { maxAmount:    24, style: { color: 0xc0231e, spot: 0xffffff, label:    '$5' } },
    { maxAmount:    99, style: { color: 0x1a7a3a, spot: 0xffffff, label:   '$25' } },
    { maxAmount:   499, style: { color: 0x1a1a1a, spot: 0xd4af37, label:  '$100' } },
    { maxAmount:   999, style: { color: 0x5a1a8a, spot: 0xffffff, label:  '$500' } },
    { maxAmount: Infinity, style: { color: 0xc07800, spot: 0x1a1a1a, label: '$1K' } },
]

/** 根据金额选择合适的筹码外观 */
export function chipStyleForAmount(amount: number): ChipStyle {
    return (CHIP_STYLES.find(e => amount <= e.maxAmount) ?? CHIP_STYLES[CHIP_STYLES.length - 1]).style
}

/**
 * 赌场筹码精灵 — 程序化绘制，以 (0, 0) 为圆心。
 * 通过 `position.set(x, y)` 定位，通过 `scale.set(s)` 缩放。
 *
 * 视觉层次（由外到内）：
 *   阴影 → 金属外环 → 主体色 → 8 个边缘卡槽 → 压边细环
 *   → 24 齿装饰环 → 深色中心 → 高光 → 面值文字 → 轮廓描边
 */
export class CasinoChip extends Container {
    constructor(radius: number, style: ChipStyle) {
        super()
        this._draw(radius, style)
    }

    private _draw(R: number, { color, spot, label }: ChipStyle) {
        const g = new Graphics()
        const cx = 0
        const cy = 0

        // 1. 阴影
        g.moveTo(cx + R + 2, cy)
        g.circle(cx, cy, R + 2).fill({ color: 0x000000, alpha: 0.35 })

        // 2. 金属外环
        g.moveTo(cx + R, cy)
        g.circle(cx, cy, R).fill({ color: 0xc8c8d4 })

        // 3. 主体填充
        g.moveTo(cx + R - 3, cy)
        g.circle(cx, cy, R - 3).fill({ color })

        // 4. 边缘卡槽（8 个，每隔 45°）
        const NOTCH_COUNT = 8
        const NOTCH_ARC = 0.22
        const arcR = R - 5
        for (let i = 0; i < NOTCH_COUNT; i++) {
            const a = (i / NOTCH_COUNT) * Math.PI * 2
            const s = a - NOTCH_ARC / 2
            g.moveTo(cx + arcR * Math.cos(s), cy + arcR * Math.sin(s))
            g.arc(cx, cy, arcR, s, a + NOTCH_ARC / 2)
            g.stroke({ color: spot, width: 8, alpha: 0.92 })
        }

        // 5. 压边细环
        g.moveTo(cx + R - 1, cy)
        g.circle(cx, cy, R - 1).stroke({ color: 0xb0b0bc, width: 1.5, alpha: 0.7 })
        g.moveTo(cx + R - 9, cy)
        g.circle(cx, cy, R - 9).stroke({ color: 0x888898, width: 1, alpha: 0.5 })

        // 6. 24 齿装饰环
        const DECO_COUNT = 24
        const DECO_R = R - 13
        for (let i = 0; i < DECO_COUNT; i++) {
            const a = (i / DECO_COUNT) * Math.PI * 2
            const s = a - Math.PI / DECO_COUNT * 0.6
            g.moveTo(cx + DECO_R * Math.cos(s), cy + DECO_R * Math.sin(s))
            g.arc(cx, cy, DECO_R, s, a + Math.PI / DECO_COUNT * 0.6)
            g.stroke({ color: 0xffffff, width: i % 2 === 0 ? 2.5 : 1, alpha: i % 2 === 0 ? 0.18 : 0.07 })
        }

        // 7. 深色中心区
        const CR = R - 17
        const dark = blendDark(color, 0.35)
        g.moveTo(cx + CR, cy)
        g.circle(cx, cy, CR).fill({ color: dark })
        g.moveTo(cx + CR, cy)
        g.circle(cx, cy, CR).stroke({ color: 0xffffff, width: 1, alpha: 0.12 })

        // 8. 高光
        const gx = cx - CR * 0.2
        const gy = cy - CR * 0.3
        const gr = CR * 0.55
        g.moveTo(gx + gr, gy)
        g.circle(gx, gy, gr).fill({ color: 0xffffff, alpha: 0.04 })

        // 9. 面值文字
        const textColor = isLight(color) ? 0x111111 : 0xffffff
        const fontSize = label.length > 3 ? Math.round(CR * 0.52) : Math.round(CR * 0.62)
        const text = new Text({
            text: label,
            style: {
                fontFamily: 'Georgia, serif',
                fontSize,
                fontWeight: '900',
                fill: textColor,
                letterSpacing: label.length > 3 ? 1 : 2,
            },
        })
        text.anchor.set(0.5)
        g.addChild(text)

        // 10. 轮廓描边
        g.moveTo(cx + R + 1, cy)
        g.circle(cx, cy, R + 1).stroke({ color: 0x000000, width: 1, alpha: 0.4 })

        this.addChild(g)
    }
}

// ── 内部工具函数 ──────────────────────────────────────────────

function blendDark(color: number, factor: number): number {
    const r = ((color >> 16) & 0xff) * (1 - factor)
    const g = ((color >>  8) & 0xff) * (1 - factor)
    const b = ( color        & 0xff) * (1 - factor)
    return (Math.round(r) << 16) | (Math.round(g) << 8) | Math.round(b)
}

function isLight(color: number): boolean {
    const r = (color >> 16) & 0xff
    const g = (color >>  8) & 0xff
    const b =  color        & 0xff
    return r * 0.299 + g * 0.587 + b * 0.114 > 140
}
