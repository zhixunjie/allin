import type { Application } from 'pixi.js'
import { CardSprite } from '../components/CardSprite'
import { CARD_H, CARD_W, TABLE_CX, TABLE_CY } from '../assets'

/** 飞行中的卡牌状态 */
interface FlyCard {
  sprite: CardSprite  // 飞行中的卡牌精灵（背面朝上）
  vx: number          // X 方向速度分量
  vy: number          // Y 方向速度分量
  tx: number          // 目标 X 坐标
  ty: number          // 目标 Y 坐标
  progress: number    // 飞行已用时间（秒）
  onDone: () => void  // 到达目标后的回调
}

/**
 * 发牌飞行动画 — 卡牌背面朝上从牌桌中心飞向目标座位。
 * 使用方向向量 + 匀速移动，到达目标后自动移除精灵并触发回调。
 */
export class DealAnimation {
  private app: Application
  private flying: FlyCard[] = []  // 当前飞行中的卡牌列表

  constructor(app: Application) {
    this.app = app
  }

  /** 创建一张背面朝上的卡牌，从牌桌中心飞向目标位置 */
  flyCard(targetX: number, targetY: number, onDone: () => void) {
    // 创建背面朝上的临时卡牌精灵
    const sprite = new CardSprite()
    sprite.setFaceDown()
    sprite.position.set(TABLE_CX - CARD_W / 2, TABLE_CY - CARD_H / 2)
    this.app.stage.addChild(sprite)

    // 计算飞行方向和速度
    const dx = targetX - sprite.x
    const dy = targetY - sprite.y
    const dist = Math.sqrt(dx * dx + dy * dy)
    const speed = Math.max(300, dist)  // 最低速度 300，确保近距离也不会太慢

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

  /** 每帧更新：移动飞行中的卡牌，到达目标后移除 */
  tick(deltaMS: number) {
    const dt = deltaMS / 1000
    this.flying = this.flying.filter((f) => {
      f.progress += dt

      const dx = f.tx - f.sprite.x
      const dy = f.ty - f.sprite.y
      const dist = Math.sqrt(dx * dx + dy * dy)

      // 距目标小于 4px 视为到达
      if (dist < 4) {
        this.app.stage.removeChild(f.sprite)
        f.onDone()
        return false
      }

      // 沿方向向量匀速移动
      const step = Math.min(dist, Math.sqrt(f.vx * f.vx + f.vy * f.vy) * dt)
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
