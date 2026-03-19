import { Container, Graphics, Text } from 'pixi.js'
import type { SeatSnapshot } from '../../store/game'
import { CardSprite } from './CardSprite'

const SEAT_W = 120
const SEAT_H = 80
const AVATAR_R = 26
const CARD_W = 60
const CARD_H = 84
const CARD_Y = -CARD_H - AVATAR_R * 2 - 10

export class SeatSprite extends Container {
  private bg: Graphics
  private avatar: Graphics
  private nameText: Text
  private stackText: Text
  private betText: Text
  private statusText: Text
  private card0: CardSprite
  private card1: CardSprite
  private isLocal: boolean

  constructor(isLocal = false) {
    super()
    this.isLocal = isLocal

    // Seat background
    this.bg = new Graphics()
    this.addChild(this.bg)

    // Avatar circle
    this.avatar = new Graphics()
    this.avatar.position.set(SEAT_W / 2, -AVATAR_R - 6)
    this.addChild(this.avatar)

    // Name label
    this.nameText = new Text({
      text: '',
      style: { fontSize: 12, fill: 0xe0e8f0, fontWeight: 'bold' },
    })
    this.nameText.anchor.set(0.5, 0)
    this.nameText.position.set(SEAT_W / 2, 6)
    this.addChild(this.nameText)

    // Stack
    this.stackText = new Text({
      text: '',
      style: { fontSize: 12, fill: 0xf0c040 },
    })
    this.stackText.anchor.set(0.5, 0)
    this.stackText.position.set(SEAT_W / 2, 24)
    this.addChild(this.stackText)

    // Current bet (shown below seat)
    this.betText = new Text({
      text: '',
      style: { fontSize: 13, fill: 0xffffff, fontWeight: 'bold' },
    })
    this.betText.anchor.set(0.5, 0)
    this.betText.position.set(SEAT_W / 2, SEAT_H + 4)
    this.addChild(this.betText)

    // Status overlay (FOLD / ALL-IN)
    this.statusText = new Text({
      text: '',
      style: { fontSize: 14, fill: 0xff6b6b, fontWeight: 'bold' },
    })
    this.statusText.anchor.set(0.5)
    this.statusText.position.set(SEAT_W / 2, SEAT_H / 2)
    this.addChild(this.statusText)

    // Hole cards
    const cardX0 = SEAT_W / 2 - CARD_W - 2
    const cardX1 = SEAT_W / 2 + 2
    this.card0 = new CardSprite()
    this.card1 = new CardSprite()
    this.card0.position.set(cardX0, CARD_Y)
    this.card1.position.set(cardX1, CARD_Y)
    this.card0.visible = false
    this.card1.visible = false
    this.addChild(this.card0, this.card1)

    this.drawEmpty()
  }

  update(seat: SeatSnapshot | null, isActive: boolean, myHole?: string[]) {
    if (!seat) {
      this.drawEmpty()
      return
    }

    const activeTurn = isActive
    this.drawFilled(seat.user_id, activeTurn, seat.is_bot ?? false)

    const prefix = seat.is_bot ? '🤖 ' : ''
    const name = prefix + seat.display_name.slice(0, 10)
    this.nameText.text = name
    this.stackText.text = `🪙 ${seat.stack.toLocaleString()}`
    this.betText.text = seat.bet > 0 ? `${seat.bet}` : ''

    if (seat.folded) {
      this.statusText.text = 'FOLD'
      this.statusText.style.fill = 0x666666
    } else if (seat.all_in) {
      this.statusText.text = 'ALL-IN'
      this.statusText.style.fill = 0xf0c040
    } else {
      this.statusText.text = ''
    }

    // Hole cards — local player uses myHole; other players only reveal at showdown via seat.hole.
    const hole = this.isLocal ? myHole : seat.hole
    if (hole && hole.length === 2) {
      this.card0.setCard(hole[0])
      this.card1.setCard(hole[1])
      this.card0.visible = true
      this.card1.visible = true
    } else {
      this.card0.visible = false
      this.card1.visible = false
    }
  }

  private drawEmpty() {
    this.nameText.text = 'Empty'
    this.nameText.style.fill = 0x4a5565
    this.stackText.text = ''
    this.betText.text = ''
    this.statusText.text = ''
    this.card0.visible = false
    this.card1.visible = false

    this.bg.clear()
    this.bg.roundRect(0, 0, SEAT_W, SEAT_H, 10)
    this.bg.fill({ color: 0x1a2535, alpha: 0.5 })
    this.bg.stroke({ color: 0x2a3545, width: 1, alpha: 0.6 })

    this.avatar.clear()
    this.avatar.circle(0, 0, AVATAR_R)
    this.avatar.fill({ color: 0x2a3545 })
  }

  private drawFilled(userId: string, isActive: boolean, isBot: boolean) {
    this.nameText.style.fill = 0xe0e8f0

    const bgColor = isActive ? 0x253545 : 0x1a2535
    const borderColor = isActive ? 0xf0c040 : (isBot ? 0x4060a0 : 0x3a4555)
    const borderWidth = isActive ? 2 : 1

    this.bg.clear()
    this.bg.roundRect(0, 0, SEAT_W, SEAT_H, 10)
    this.bg.fill({ color: bgColor })
    this.bg.stroke({ color: borderColor, width: borderWidth })

    const hue = isBot ? 0x4060a0 : this.hashColor(userId)
    this.avatar.clear()
    this.avatar.circle(0, 0, AVATAR_R)
    this.avatar.fill({ color: hue })
    this.avatar.stroke({ color: isActive ? 0xf0c040 : 0x1a2535, width: isActive ? 2 : 1 })
  }

  private hashColor(str: string): number {
    let hash = 0
    for (let i = 0; i < str.length; i++) {
      hash = (hash * 31 + str.charCodeAt(i)) & 0xffffff
    }
    return 0x406080 | (hash & 0x3f3f3f)
  }
}
