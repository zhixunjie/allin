import {Container, Graphics, Text} from 'pixi.js'
import {C, FONT_HEADLINE} from '../assets'

const POT_W = 200     // 约等于 3 张牌宽度，不显得过于修长
const POT_H = 36      // 高度减小，让底池药丸显得更加修长雅致，不与公共牌抢戏

/**
 * 底池金额显示组件。
 *
 * 视觉结构（左→右）：
 *   bg          — 药丸形玻璃背景 + 金色描边
 *   chipStacks  — 左侧堆叠金色筹码
 *   titleText   — "POT" 标签
 *   amount      — 底池金额数字
 */
export class PotDisplay extends Container {
    /** 药丸形玻璃底板容器（提供半透明毛玻璃质感与边缘光晕） */
    private bg!: Graphics
    /** 左侧装饰性筹码堆图示（三个筹码塔，纯视觉装饰，不随当前金额变化） */
    private chipStacks!: Graphics
    /** "POT" 固定文字标签（位于筹码右侧，用于标示当前区域身份） */
    private titleText!: Text
    /** 动态更新的底池详细金额数字（位于整体最右侧） */
    private amount!: Text

    constructor() {
        super()
        this.buildBackground()
        this.buildChipStacks()
        this.buildTitleText()
        this.buildAmountText()

        // 初始绘制默认状态（空底池隐藏由 setPot 触发，或者默认隐藏）
        this.redraw()
    }

    // ─── 构造分解 ──────────────────────────────────────────────

    private buildBackground() {
        this.bg = new Graphics()
        this.addChild(this.bg)
    }

    private buildChipStacks() {
        this.chipStacks = new Graphics()
        // 左边距留足 18px (刚好是圆角直径的一半往内)，避免画到圆角外发生越界跳出
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
        // 筹码往右推了，文字也相对右移保持距离
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
        // 从 TitleText 顺延往右偏移
        this.amount.position.set(98, POT_H / 2)
        this.addChild(this.amount)
    }

    setPot(value: number) {
        this.amount.text = value > 0 ? `$${value.toLocaleString()}` : ''
        this.titleText.visible = value > 0
        this.visible = value > 0
        this.redraw()
    }

    private redraw() {
        this.bg.clear()

        // 药丸形背景，带玻璃效果
        this.bg.roundRect(0, 0, POT_W, POT_H, POT_H / 2)
        this.bg.fill({color: C.GLASS, alpha: 0.8})

        // 金色边框光晕
        this.bg.roundRect(0, 0, POT_W, POT_H, POT_H / 2)
        this.bg.stroke({color: C.GOLD, width: 1, alpha: 0.4})
    }

    private drawChipStacks() {
        const g = this.chipStacks
        g.clear()

        // 运用真实赌场的红蓝金明亮发牌配色（避免在深色桌布上显得过于暗沉）
        const stackCols = [
            { count: 5, top: 0xff4d4f, edge: 0xcf1322 }, // 亮采红
            { count: 4, top: 0x36cfc9, edge: 0x08979c }, // 霓虹青
            { count: 6, top: 0xffc53d, edge: 0xd48806 }, // 香槟金
        ]

        const stackSpacing = 10
        const chipW = 10
        const chipH = 4
        // chipGap 代表每一枚筹码叠上去的露出厚度（稍小于真实厚度能产生堆叠感）
        const chipGap = 2
        // chipThickness 代表筹码的柱体厚度
        const chipThickness = 2

        const maxStackHeight = (Math.max(...stackCols.map(s => s.count)) - 1) * chipGap
        const yOffset = maxStackHeight / 2 // 向下偏移以完美垂直居中

        for (let s = 0; s < stackCols.length; s++) {
            const cx = s * stackSpacing
            const colData = stackCols[s]

            for (let i = 0; i < colData.count; i++) {
                const cy = -i * chipGap + yOffset

                // 1. 绘制筹码厚度层 (产生伪全 3D 柱体效果，且每一枚都有厚度)
                g.ellipse(cx, cy + chipThickness, chipW / 2, chipH / 2)
                g.fill({color: colData.edge})

                // 2. 绘制筹码顶面主力颜色
                g.ellipse(cx, cy, chipW / 2, chipH / 2)
                g.fill({color: colData.top})
                
                // 3. 画出顶面与厚度之间的圆角轮廓暗线，极度增强立体感
                g.ellipse(cx, cy, chipW / 2, chipH / 2)
                g.stroke({color: 0x000000, width: 0.5, alpha: 0.25})

                // 4. 画一个白色的小内圈，模拟赌场筹码中心的品牌贴纸 (Decal)
                g.ellipse(cx, cy, chipW / 3.5, chipH / 3.5)
                g.fill({color: 0xffffff, alpha: 0.85})
                // 贴纸外再套一层极细的微透黑圈表示贴纸边界
                g.ellipse(cx, cy, chipW / 3.5, chipH / 3.5)
                g.stroke({color: 0x000000, width: 0.5, alpha: 0.15})
            }
        }
    }
}
