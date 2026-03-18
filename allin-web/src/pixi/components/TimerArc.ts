import { Container, Graphics, Text } from 'pixi.js'

const R = 28
const TWO_PI = Math.PI * 2

export class TimerArc extends Container {
  private arc: Graphics
  private numText: Text
  private totalSecs: number
  private deadlineTs: number | null = null

  constructor() {
    super()

    this.arc = new Graphics()
    this.addChild(this.arc)

    this.numText = new Text({
      text: '',
      style: { fontSize: 14, fill: 0xffffff, fontWeight: 'bold' },
    })
    this.numText.anchor.set(0.5)
    this.addChild(this.numText)

    this.totalSecs = 30
    this.visible = false
  }

  start(deadlineTs: number, totalSecs: number) {
    this.deadlineTs = deadlineTs
    this.totalSecs = totalSecs
    this.visible = true
  }

  stop() {
    this.deadlineTs = null
    this.visible = false
  }

  tick() {
    if (!this.deadlineTs) return
    const remaining = Math.max(0, (this.deadlineTs - Date.now()) / 1000)
    const fraction = remaining / this.totalSecs

    const color = fraction > 0.4 ? 0x4caf50 : fraction > 0.2 ? 0xf0c040 : 0xff5252

    this.arc.clear()

    // Background circle
    this.arc.circle(0, 0, R)
    this.arc.fill({ color: 0x1a2535 })

    // Arc fill
    if (fraction > 0) {
      const startAngle = -Math.PI / 2
      const endAngle = startAngle + TWO_PI * fraction

      this.arc.moveTo(0, 0)
      this.arc.arc(0, 0, R, startAngle, endAngle)
      this.arc.lineTo(0, 0)
      this.arc.fill({ color, alpha: 0.85 })
    }

    // Border
    this.arc.circle(0, 0, R)
    this.arc.stroke({ color: 0x3a4555, width: 2 })

    this.numText.text = Math.ceil(remaining).toString()
    this.numText.style.fill = color

    if (remaining <= 0) this.stop()
  }
}
