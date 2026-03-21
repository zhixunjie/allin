import {Container, Graphics, Text} from 'pixi.js'
import {C, FONT_HEADLINE} from '../assets'

const POT_W = 200   // 药丸总宽度
const POT_H = 36    // 药丸高度

/**
 * 底池金额显示组件 — 固定居中在公共牌下方。
 *
 * 视觉结构（左→右）：
 *   bg         — 药丸形玻璃背景 + 金色描边
 *   chipStacks — 左侧三列侧视角堆叠筹码（纯视觉装饰，不随金额变化）
 *   titleText  — "POT" 标签
 *   amount     — 底池金额数字（setPot 时动态更新）
 *
 * 使用方式：
 *   potDisplay.setPot(1200)  // 显示 "$1,200"
 *   potDisplay.setPot(0)     // 隐藏整个组件
 */
export class PotDisplay extends Container {
    /** 向外金色晕光层（多层同心 roundRect，最先绘制置于底层） */
    private glow!: Graphics
    /** 药丸形玻璃底板（半透明毛玻璃 + 金色边框） */
    private bg!: Graphics
    /** 左侧装饰性筹码堆（三列，从左到右：红/青/金，高度不等增加层次感） */
    private chipStacks!: Graphics
    /** "POT" 固定标签 */
    private titleText!: Text
    /** 动态金额文字 */
    private amount!: Text

    constructor() {
        super()
        this.buildGlow()
        this.buildBackground()
        this.buildChipStacks()
        this.buildTitleText()
        this.buildAmountText()
        this.redraw()
    }

    // ─── 构造分解 ──────────────────────────────────────────────

    private buildGlow() {
        this.glow = new Graphics()
        this.addChild(this.glow)
        this.drawGlow()
    }

    private buildBackground() {
        this.bg = new Graphics()
        this.addChild(this.bg)
    }

    private buildChipStacks() {
        this.chipStacks = new Graphics()
        // x=18 让筹码堆距左侧圆角有足够内边距
        this.chipStacks.position.set(18, POT_H / 2)
        this.addChild(this.chipStacks)
        this.drawChipStacks()
    }

    private buildTitleText() {
        this.titleText = new Text({
            text: 'POT',
            style: {
                fontFamily: FONT_HEADLINE,
                fontSize: 11,
                fontWeight: '900',
                fill: C.GOLD_DIM,
                letterSpacing: 1,
            },
        })
        this.titleText.anchor.set(0, 0.5)
        this.titleText.position.set(58, POT_H / 2)
        this.addChild(this.titleText)
    }

    private buildAmountText() {
        this.amount = new Text({
            text: '0',
            style: {
                fontFamily: FONT_HEADLINE,
                fontSize: 16,
                fontWeight: '900',
                fill: C.GOLD,
                letterSpacing: 1.5,
            },
        })
        this.amount.anchor.set(0, 0.5)
        this.amount.position.set(98, POT_H / 2)
        this.addChild(this.amount)
    }

    // ─── 公开接口 ──────────────────────────────────────────────

    /** 更新底池金额；value=0 时隐藏整个组件 */
    setPot(value: number) {
        this.amount.text = value > 0 ? `$${value.toLocaleString()}` : ''
        this.titleText.visible = value > 0
        this.visible = value > 0
        this.redraw()
    }

    // ─── 内部绘制 ──────────────────────────────────────────────

    /** 重绘药丸背景（每次 setPot 触发，保持与 visible 同步） */
    private redraw() {
        this.drawGlow()
        this.bg.clear()
        // 半透明玻璃填充
        this.bg.roundRect(0, 0, POT_W, POT_H, POT_H / 2)
        this.bg.fill({color: C.GLASS, alpha: 0.8})
        // 金色描边
        this.bg.roundRect(0, 0, POT_W, POT_H, POT_H / 2)
        this.bg.stroke({color: C.GOLD, width: 1, alpha: 0.4})
    }

    /**
     * 向外扩散的多层金色晕光：
     * 从最外层（扩散 28px，alpha 最低）到最内层（扩散 4px，alpha 最高），
     * 共 6 层叠加，产生平滑的 glow 渐变效果。
     */
    private drawGlow() {
        this.glow.clear()
        const r = POT_H / 2
        const layers = [
            { expand: 28, alpha: 0.025 },
            { expand: 20, alpha: 0.045 },
            { expand: 14, alpha: 0.07  },
            { expand: 9,  alpha: 0.10  },
            { expand: 5,  alpha: 0.14  },
            { expand: 2,  alpha: 0.18  },
        ]
        for (const { expand, alpha } of layers) {
            const x = -expand
            const y = -expand
            const w = POT_W + expand * 2
            const h = POT_H + expand * 2
            this.glow.roundRect(x, y, w, h, r + expand)
            this.glow.fill({ color: C.GOLD, alpha })
        }
    }

    /**
     * 绘制侧视角堆叠筹码（仅构造时调用一次，静态装饰）。
     *
     * 每枚筹码由 4 层组成：
     *   1. 厚度层（偏下的同色椭圆，产生 3D 柱体厚度感）
     *   2. 顶面层（主色椭圆）
     *   3. 顶面轮廓（细黑描边，增强立体感）
     *   4. 中心白点（模拟赌场筹码的品牌贴纸 decal）
     */
    private drawChipStacks() {
        const g = this.chipStacks
        g.clear()

        // 三列筹码配置：count=枚数，top=顶面色，edge=厚度层色
        const stackCols = [
            { count: 5, top: 0xff4d4f, edge: 0xcf1322 }, // 红（$5）
            { count: 4, top: 0x36cfc9, edge: 0x08979c }, // 青（装饰色）
            { count: 6, top: 0xffc53d, edge: 0xd48806 }, // 金（$25）
        ]

        const stackSpacing = 10  // 每列筹码的水平间距
        const chipW = 10         // 椭圆长轴直径（正视宽度）
        const chipH = 4          // 椭圆短轴直径（透视压缩高度）
        const chipGap = 2        // 每枚叠上去后露出的厚度（堆叠间距）
        const chipThickness = 2  // 厚度层向下偏移量（模拟侧面厚度）

        // 计算最高列的总高度，用于整体垂直居中
        const maxStackH = (Math.max(...stackCols.map(s => s.count)) - 1) * chipGap
        const yOffset = maxStackH / 2

        for (let s = 0; s < stackCols.length; s++) {
            const cx = s * stackSpacing
            const col = stackCols[s]
            for (let i = 0; i < col.count; i++) {
                // i=0 在底部，i 越大越靠上（y 越小）
                const cy = -i * chipGap + yOffset

                // 1. 厚度层（向下偏移，产生伪 3D 侧面）
                g.ellipse(cx, cy + chipThickness, chipW / 2, chipH / 2)
                g.fill({color: col.edge})

                // 2. 顶面主色
                g.ellipse(cx, cy, chipW / 2, chipH / 2)
                g.fill({color: col.top})

                // 3. 顶面轮廓暗线
                g.ellipse(cx, cy, chipW / 2, chipH / 2)
                g.stroke({color: 0x000000, width: 0.5, alpha: 0.25})

                // 4. 中心白色贴纸 decal
                g.ellipse(cx, cy, chipW / 3.5, chipH / 3.5)
                g.fill({color: 0xffffff, alpha: 0.85})
                g.ellipse(cx, cy, chipW / 3.5, chipH / 3.5)
                g.stroke({color: 0x000000, width: 0.5, alpha: 0.15})
            }
        }
    }
}
