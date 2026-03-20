import { Container, Graphics, Text } from 'pixi.js'
import { C, FONT_HEADLINE } from '../assets'

const TWO_PI = Math.PI * 2

/**
 * 覆盖在当前行动玩家头像上的计时器圆环。
 * 定位前调用 setRadius() 以匹配头像大小。
 */
export class TimerArc extends Container {
  private ring: Graphics
  private bgRing: Graphics
  private badge: Container
  private badgeText: Text
  private radius = 54
  private totalSecs = 30
  private deadlineTs: number | null = null

  constructor() {
    super()

    // 背景圆环（完整圆，暗色）
    this.bgRing = new Graphics()
    this.addChild(this.bgRing)

    // 进度圆环
    this.ring = new Graphics()
    this.addChild(this.ring)

    // 倒计时徽章（右上角）
    this.badge = new Container()
    this.badgeText = new Text({
      text: '',
      style: {
        fontFamily: FONT_HEADLINE,
        fontSize: 10,
        fontWeight: '900',
        fill: C.GOLD,
      },
    })
    this.badgeText.anchor.set(0.5)

    const badgeBg = new Graphics()
    this.badge.addChild(badgeBg)
    this.badge.addChild(this.badgeText)
    this.addChild(this.badge)

    this.visible = false
  }

  setRadius(r: number) {
    this.radius = r + 4
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
    const secs = Math.ceil(remaining)

    const color = fraction > 0.4 ? C.SUCCESS : fraction > 0.2 ? C.GOLD : C.ERROR
    const R = this.radius

    // 背景圆环
    this.bgRing.clear()
    this.bgRing.circle(0, 0, R)
    this.bgRing.stroke({ color: C.GOLD, width: 4, alpha: 0.1 })

    // 进度弧
    this.ring.clear()
    if (fraction > 0) {
      const startAngle = -Math.PI / 2
      const endAngle = startAngle + TWO_PI * fraction

      this.ring.arc(0, 0, R, startAngle, endAngle)
      this.ring.stroke({ color, width: 4, alpha: 0.9 })
    }

    // 徽章位置：圆环右上方
    const badgeX = R * 0.7
    const badgeY = -R * 0.7

    // 绘制徽章背景
    const badgeBg = this.badge.getChildAt(0) as Graphics
    badgeBg.clear()
    badgeBg.roundRect(-14, -10, 28, 20, 10)
    badgeBg.fill({ color: C.GLASS, alpha: 0.95 })
    badgeBg.roundRect(-14, -10, 28, 20, 10)
    badgeBg.stroke({ color: C.GOLD, width: 1 })

    this.badge.position.set(badgeX, badgeY)
    this.badgeText.text = `${secs}s`
    this.badgeText.style.fill = color

    if (remaining <= 0) this.stop()
  }
}
