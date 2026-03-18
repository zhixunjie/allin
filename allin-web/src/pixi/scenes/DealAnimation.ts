import type { Application } from 'pixi.js'
import { CardSprite } from '../components/CardSprite'
import { CARD_H, CARD_W, TABLE_CX, TABLE_CY } from '../assets'

interface FlyCard {
  sprite: CardSprite
  vx: number
  vy: number
  tx: number
  ty: number
  progress: number
  onDone: () => void
}

/** Animates cards flying from the deck position to seat positions. */
export class DealAnimation {
  private app: Application
  private flying: FlyCard[] = []

  constructor(app: Application) {
    this.app = app
  }

  /**
   * Fly a face-down card from the deck (table center) to targetX/Y.
   * Calls onDone when the card reaches its destination (caller should hide/remove it).
   */
  flyCard(targetX: number, targetY: number, onDone: () => void) {
    const sprite = new CardSprite()
    sprite.setFaceDown()
    sprite.position.set(TABLE_CX - CARD_W / 2, TABLE_CY - CARD_H / 2)
    this.app.stage.addChild(sprite)

    const dx = targetX - sprite.x
    const dy = targetY - sprite.y
    const dist = Math.sqrt(dx * dx + dy * dy)
    const speed = Math.max(300, dist) // pixels per second

    this.flying.push({
      sprite,
      vx: (dx / dist) * speed,
      vy: (dy / dist) * speed,
      tx: targetX,
      ty: targetY,
      progress: 0,
      onDone,
    })
  }

  tick(deltaMS: number) {
    const dt = deltaMS / 1000
    this.flying = this.flying.filter((f) => {
      f.progress += dt

      const dx = f.tx - f.sprite.x
      const dy = f.ty - f.sprite.y
      const dist = Math.sqrt(dx * dx + dy * dy)

      if (dist < 4) {
        this.app.stage.removeChild(f.sprite)
        f.onDone()
        return false
      }

      const step = Math.min(dist, Math.sqrt(f.vx * f.vx + f.vy * f.vy) * dt)
      f.sprite.x += (dx / dist) * step
      f.sprite.y += (dy / dist) * step
      return true
    })
  }

  get isRunning() {
    return this.flying.length > 0
  }
}
