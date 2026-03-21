import type { Application } from 'pixi.js'
import { CasinoChip, chipStyleForAmount } from '../components/CasinoChip'
import { TABLE_CX, TABLE_CY } from '../assets'

const CHIP_RADIUS = 16

/** 飞行中的筹码状态 */
interface FlyChip {
  chip: CasinoChip  // 筹码精灵（CasinoChip，挂载在 app.stage 上，到达目标后自动移除）
  tx: number        // 目标 X 坐标（底池中心或桌面中途落点）
  ty: number        // 目标 Y 坐标
  speed: number     // 飞行速度（px/s），由出发距离决定，最低 200
}

/**
 * 筹码飞行动画 — 轮次切换时，各座位的下注筹码飞向底池中心。
 * 筹码外观由金额自动匹配面额颜色（$1/$5/$25/$100/$500/$1K）。
 */
export class ChipAnimation {
  private app: Application
  private flying: FlyChip[] = []

  constructor(app: Application) {
    this.app = app
  }

  /** 筹码从座位飞向底池中心（轮次切换时调用） */
  flyChip(fromX: number, fromY: number, amount: number) {
    this._fly(fromX, fromY, TABLE_CX, TABLE_CY, amount)
  }

  private _fly(fromX: number, fromY: number, tx: number, ty: number, amount: number) {
    const style = chipStyleForAmount(amount)
    const chip = new CasinoChip(CHIP_RADIUS, style)
    chip.position.set(fromX, fromY)
    this.app.stage.addChild(chip)

    const dx = tx - fromX
    const dy = ty - fromY
    const dist = Math.sqrt(dx * dx + dy * dy)
    this.flying.push({ chip, tx, ty, speed: Math.max(200, dist) })
  }

  /** 每帧更新：移动筹码，到达目标后移除 */
  tick(deltaMS: number) {
    const dt = deltaMS / 1000
    this.flying = this.flying.filter((f) => {
      const dx = f.tx - f.chip.x
      const dy = f.ty - f.chip.y
      const dist = Math.sqrt(dx * dx + dy * dy)
      if (dist < 4) {
        this.app.stage.removeChild(f.chip)
        return false
      }
      const step = Math.min(dist, f.speed * dt)
      f.chip.x += (dx / dist) * step
      f.chip.y += (dy / dist) * step
      return true
    })
  }

  /** 是否有飞行中的筹码 */
  get isRunning() {
    return this.flying.length > 0
  }
}
