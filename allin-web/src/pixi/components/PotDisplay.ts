import { Container, Graphics, Text } from 'pixi.js'

export class PotDisplay extends Container {
  private bg: Graphics
  private titleText: Text
  private amount: Text

  constructor() {
    super()

    this.bg = new Graphics()
    this.addChild(this.bg)

    this.titleText = new Text({
      text: 'POT',
      style: { fontSize: 11, fill: 0x8a9bb0, letterSpacing: 2 },
    })
    this.titleText.anchor.set(0.5, 0)
    this.titleText.position.set(70, 6)
    this.addChild(this.titleText)

    this.amount = new Text({
      text: '0',
      style: { fontSize: 22, fill: 0xf0c040, fontWeight: 'bold' },
    })
    this.amount.anchor.set(0.5, 0)
    this.amount.position.set(70, 22)
    this.addChild(this.amount)

    this.redraw()
  }

  setPot(value: number) {
    this.amount.text = value > 0 ? value.toLocaleString() : ''
    this.visible = value > 0
    this.redraw()
  }

  private redraw() {
    this.bg.clear()
    this.bg.roundRect(0, 0, 140, 56, 28)
    this.bg.fill({ color: 0x0f1923, alpha: 0.8 })
    this.bg.stroke({ color: 0x2a3545, width: 1 })
  }
}
