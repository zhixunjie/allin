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

    // 配色：全部使用色板内颜色
    //   充裕(>40%) — TEXT_PRIMARY 银白，平静不打扰视线
    //   警示(>15%) — GOLD 主题金，与全场金色系一致
    //   紧迫(≤15%) — ERROR 红，最后关头的强警示
    const color = fraction > 0.4 ? C.TEXT_PRIMARY
                : fraction > 0.15 ? C.GOLD
                :                   C.ERROR

    const R = this.radius

    // 背景圆环：始终暗金，作为刻度底盘
    this.bgRing.clear()
    this.bgRing.circle(0, 0, R)
    this.bgRing.stroke({ color: C.GOLD_DIM, width: 3, alpha: 0.15 })

    // 进度弧（外发光 + 主线双层）
    this.ring.clear()
    if (fraction > 0) {
      const start = -Math.PI / 2
      const end   = start + TWO_PI * fraction
      this.ring.arc(0, 0, R, start, end)
      this.ring.stroke({ color, width: 7, alpha: 0.15 })
      this.ring.arc(0, 0, R, start, end)
      this.ring.stroke({ color, width: 3, alpha: 0.90 })
    }

    // 徽章：边框跟随弧线颜色，底色固定深色
    const badgeX = R * 0.72
    const badgeY = -R * 0.72
    const badgeBg = this.badge.getChildAt(0) as Graphics
    badgeBg.clear()
    badgeBg.roundRect(-14, -10, 28, 20, 10)
    badgeBg.fill({ color: C.GLASS, alpha: 0.95 })
    badgeBg.roundRect(-14, -10, 28, 20, 10)
    badgeBg.stroke({ color, width: 1, alpha: 0.7 })

    this.badge.position.set(badgeX, badgeY)
    this.badgeText.text = `${secs}s`
    this.badgeText.style.fill = color

    if (remaining <= 0) this.stop()
  }
}
