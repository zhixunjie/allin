import { Container, Graphics, Text } from 'pixi.js'
import { C, FONT_HEADLINE, FONT_BODY } from '../assets'

const POT_W = 220
const POT_H = 64

export class PotDisplay extends Container {
  private bg: Graphics
  private chipStacks: Graphics
  private titleText: Text
  private amount: Text

  constructor() {
    super()

    this.bg = new Graphics()
    this.addChild(this.bg)

    // 左侧筹码堆
    this.chipStacks = new Graphics()
    this.chipStacks.position.set(28, POT_H / 2)
    this.addChild(this.chipStacks)
    this.drawChipStacks()

    // "当前底池" label
    this.titleText = new Text({
      text: '当前底池',
      style: {
        fontFamily: FONT_BODY,
        fontSize: 9,
        fontWeight: '800',
        fill: C.GOLD,
        letterSpacing: 3,
      },
    })
    this.titleText.alpha = 0.6
    this.titleText.anchor.set(0, 0.5)
    this.titleText.position.set(62, POT_H / 2 - 12)
    this.addChild(this.titleText)

    // 金额
    this.amount = new Text({
      text: '0',
      style: {
        fontFamily: FONT_HEADLINE,
        fontSize: 22,
        fontWeight: '900',
        fill: C.GOLD,
        letterSpacing: 1.5,
      },
    })
    this.amount.anchor.set(0, 0.5)
    this.amount.position.set(62, POT_H / 2 + 10)
    this.addChild(this.amount)

    this.redraw()
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
    this.bg.fill({ color: C.GLASS, alpha: 0.8 })

    // 金色边框光晕
    this.bg.roundRect(0, 0, POT_W, POT_H, POT_H / 2)
    this.bg.stroke({ color: C.GOLD, width: 1, alpha: 0.4 })
  }

  private drawChipStacks() {
    const g = this.chipStacks
    g.clear()

    // 并排绘制 3 个筹码堆
    const stacks = [5, 4, 6]
    const stackSpacing = 10
    const chipW = 10
    const chipH = 4
    const chipGap = 3

    for (let s = 0; s < stacks.length; s++) {
      const cx = (s - 1) * stackSpacing
      const count = stacks[s]
      for (let i = 0; i < count; i++) {
        const cy = -i * chipGap
        g.ellipse(cx, cy, chipW / 2, chipH / 2)
        g.fill({ color: C.GOLD })
        g.ellipse(cx, cy, chipW / 2, chipH / 2)
        g.stroke({ color: 0x000000, width: 0.5, alpha: 0.3 })
        // 底部边缘，营造 3D 效果
        if (i === 0) {
          g.ellipse(cx, cy + 2, chipW / 2, chipH / 2)
          g.fill({ color: C.GOLD_DIM })
        }
      }
    }
  }
}
