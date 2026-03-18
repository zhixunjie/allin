import type { Application } from 'pixi.js'
import { Graphics, Text } from 'pixi.js'
import { TABLE_CX, TABLE_CY } from '../assets'

interface FlyChip {
  g: Graphics
  label: Text
  tx: number
  ty: number
  speed: number
}

/** Animates a chip value token flying toward the pot center. */
export class ChipAnimation {
  private app: Application
  private flying: FlyChip[] = []

  constructor(app: Application) {
    this.app = app
  }

  flyChip(fromX: number, fromY: number, amount: number) {
    const g = new Graphics()
    g.circle(0, 0, 14)
    g.fill({ color: 0xf0c040 })
    g.stroke({ color: 0xb8943f, width: 2 })
    g.position.set(fromX, fromY)

    const label = new Text({
      text: amount > 0 ? amount.toLocaleString() : '',
      style: { fontSize: 10, fill: 0x000000, fontWeight: 'bold' },
    })
    label.anchor.set(0.5)
    g.addChild(label)

    this.app.stage.addChild(g)

    const dx = TABLE_CX - fromX
    const dy = TABLE_CY - fromY
    const dist = Math.sqrt(dx * dx + dy * dy)

    this.flying.push({ g, label, tx: TABLE_CX, ty: TABLE_CY, speed: Math.max(200, dist) })
  }

  tick(deltaMS: number) {
    const dt = deltaMS / 1000
    this.flying = this.flying.filter((f) => {
      const dx = f.tx - f.g.x
      const dy = f.ty - f.g.y
      const dist = Math.sqrt(dx * dx + dy * dy)
      if (dist < 4) {
        this.app.stage.removeChild(f.g)
        return false
      }
      const step = Math.min(dist, f.speed * dt)
      f.g.x += (dx / dist) * step
      f.g.y += (dy / dist) * step
      return true
    })
  }

  get isRunning() {
    return this.flying.length > 0
  }
}
