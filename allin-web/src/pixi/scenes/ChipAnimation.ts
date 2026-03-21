import type { Application } from 'pixi.js'
import { Graphics, Text } from 'pixi.js'
import { C, FONT_BODY, TABLE_CX, TABLE_CY } from '../assets'

/** 飞行中的筹码状态 */
interface FlyChip {
  g: Graphics  // 筹码圆形（金色圆 + 金额文字子节点）
  tx: number   // 目标 X
  ty: number   // 目标 Y
  speed: number
}

/**
 * 筹码飞行动画 — 轮次切换时，各座位的下注筹码飞向底池中心。
 * 每个筹码显示为金色圆形 + 下注金额，到达中心后自动消失。
 */
export class ChipAnimation {
  private app: Application
  private flying: FlyChip[] = []

  constructor(app: Application) {
    this.app = app
  }

  /** 创建一个筹码代币，从座位位置飞向牌桌中心 */
  flyChip(fromX: number, fromY: number, amount: number) {
    const g = new Graphics()
    g.circle(0, 0, 14)
    g.fill({ color: C.GOLD })
    g.stroke({ color: C.GOLD_DIM, width: 2 })
    g.position.set(fromX, fromY)

    const label = new Text({
      text: amount > 0 ? amount.toLocaleString() : '',
      style: { fontFamily: FONT_BODY, fontSize: 9, fill: C.VOID, fontWeight: 'bold' },
    })
    label.anchor.set(0.5)
    g.addChild(label)

    this.app.stage.addChild(g)

    const dx = TABLE_CX - fromX
    const dy = TABLE_CY - fromY
    const dist = Math.sqrt(dx * dx + dy * dy)
    this.flying.push({ g, tx: TABLE_CX, ty: TABLE_CY, speed: Math.max(200, dist) })
  }

  /** 每帧更新：移动筹码，到达中心后移除 */
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

  /** 是否有飞行中的筹码 */
  get isRunning() {
    return this.flying.length > 0
  }
}
