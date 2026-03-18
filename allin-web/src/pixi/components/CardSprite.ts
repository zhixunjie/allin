import { Container, Graphics, Text } from 'pixi.js'
import { CARD_H, CARD_RADIUS, CARD_W, SUIT_COLORS, SUIT_SYMBOLS } from '../assets'

export class CardSprite extends Container {
  private bg: Graphics
  private rankText: Text
  private suitText: Text
  private faceDown = false

  constructor() {
    super()

    this.bg = new Graphics()
    this.addChild(this.bg)

    this.rankText = new Text({
      text: '',
      style: { fontSize: 15, fontWeight: 'bold', fill: 0x111111 },
    })
    this.rankText.position.set(4, 2)
    this.addChild(this.rankText)

    this.suitText = new Text({
      text: '',
      style: { fontSize: 22, fill: 0x111111 },
    })
    this.suitText.anchor.set(0.5)
    this.suitText.position.set(CARD_W / 2, CARD_H / 2 + 4)
    this.addChild(this.suitText)

    this.drawBack()
  }

  setCard(card: string) {
    const rank = card[0]
    const suit = card[1] as keyof typeof SUIT_SYMBOLS
    const color = SUIT_COLORS[suit] ?? 0x111111
    const symbol = SUIT_SYMBOLS[suit] ?? '?'

    this.faceDown = false
    this.drawFace()

    this.rankText.text = rank
    this.rankText.style.fill = color
    this.suitText.text = symbol
    this.suitText.style.fill = color
    this.rankText.visible = true
    this.suitText.visible = true
  }

  setFaceDown() {
    this.faceDown = true
    this.rankText.visible = false
    this.suitText.visible = false
    this.drawBack()
  }

  private drawFace() {
    this.bg.clear()
    this.bg.roundRect(0, 0, CARD_W, CARD_H, CARD_RADIUS)
    this.bg.fill({ color: 0xffffff })
    this.bg.stroke({ color: 0xaaaaaa, width: 1 })
  }

  private drawBack() {
    this.bg.clear()
    this.bg.roundRect(0, 0, CARD_W, CARD_H, CARD_RADIUS)
    this.bg.fill({ color: 0x1a3a8f })
    this.bg.stroke({ color: 0x0d1f5c, width: 1 })
    // Simple crosshatch pattern
    this.bg.roundRect(4, 4, CARD_W - 8, CARD_H - 8, 3)
    this.bg.stroke({ color: 0x2a5ac8, width: 1 })
  }

  get isFaceDown() {
    return this.faceDown
  }
}
