import type { Application } from 'pixi.js'
import { CardSprite } from '../components/CardSprite'
import { CARD_H, CARD_W, TABLE_CX, TABLE_CY } from '../assets'

/** 飞行中的卡牌状态 */
interface FlyCard {
  sprite: CardSprite
  tx: number          // 目标 X
  ty: number          // 目标 Y
  speed: number       // 飞行速度（px/s）
  onDone: () => void
}

/**
 * 发牌飞行动画 — 卡牌背面朝上从牌桌中心飞向目标座位。
 * 到达目标后自动移除精灵并触发回调。
 */
export class DealAnimation {
  private app: Application
  private flying: FlyCard[] = []

  constructor(app: Application) {
    this.app = app
  }

  /** 创建一张背面朝上的卡牌，从牌桌中心飞向目标位置 */
  flyCard(targetX: number, targetY: number, onDone: () => void) {
    const sprite = new CardSprite()
    sprite.setFaceDown()
    sprite.position.set(TABLE_CX - CARD_W / 2, TABLE_CY - CARD_H / 2)
    this.app.stage.addChild(sprite)

    const dx = targetX - sprite.x
    const dy = targetY - sprite.y
    const dist = Math.sqrt(dx * dx + dy * dy)
    this.flying.push({ sprite, tx: targetX, ty: targetY, speed: Math.max(300, dist), onDone })
  }

  /** 每帧更新：移动飞行中的卡牌，到达目标后移除 */
  tick(deltaMS: number) {
    const dt = deltaMS / 1000
    this.flying = this.flying.filter((f) => {
      const dx = f.tx - f.sprite.x
      const dy = f.ty - f.sprite.y
      const dist = Math.sqrt(dx * dx + dy * dy)
      if (dist < 4) {
        this.app.stage.removeChild(f.sprite)
        f.onDone()
        return false
      }
      const step = Math.min(dist, f.speed * dt)
      f.sprite.x += (dx / dist) * step
      f.sprite.y += (dy / dist) * step
      return true
    })
  }

  /** 是否有飞行中的卡牌 */
  get isRunning() {
    return this.flying.length > 0
  }
}
