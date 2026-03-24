import {Container, Graphics, Text} from 'pixi.js'
import {C, FONT_HEADLINE} from '../assets'
import {soundManager} from '../sound'

const TWO_PI = Math.PI * 2
const HALF_PI = Math.PI / 2

/** 覆盖在当前行动玩家头像上的倒计时圆环，调用 setRadius() 匹配头像大小。 */
export class TimerArc extends Container {
    private bgRing = new Graphics()    // 灰色底环（始终可见）
    private ring = new Graphics()      // 彩色进度弧（随剩余时间收缩）
    private badge = new Container()    // 中央秒数徽章容器
    private badgeText: Text            // 徽章上的剩余秒数文字
    private badgeBg = new Graphics()   // 徽章背景圆

    private radius = 54                // 圆环半径，由外部 setRadius() 动态调整以匹配头像尺寸
    private totalSecs = 30             // 本次倒计时总秒数，用于计算弧度比例
    private deadlineTs: number | null = null  // 行动截止时间（Unix 毫秒）；null 表示未激活
    private lastTickSec = -1           // 上次播放 tick 音效对应的整秒值，防止每帧重复触发

    constructor() {
        super()

        this.badgeText = new Text({
            text: '',
            style: {fontFamily: FONT_HEADLINE, fontSize: 10, fontWeight: '900', fill: C.GOLD},
        })
        this.badgeText.anchor.set(0.5)
        this.badge.addChild(this.badgeBg, this.badgeText)

        this.addChild(this.bgRing, this.ring, this.badge)
        this.visible = false
    }

    /** 匹配头像半径，环绘制在头像外沿 +4px 处 */
    setRadius(r: number) {
        this.radius = r + 4
    }

    start(deadlineTs: number, totalSecs: number) {
        this.deadlineTs = deadlineTs
        this.totalSecs = totalSecs
        this.lastTickSec = -1
        this.visible = true
    }

    stop() {
        this.deadlineTs = null
        this.lastTickSec = -1
        this.visible = false
    }

    tick() {
        if (!this.deadlineTs) return

        const remaining = Math.max(0, (this.deadlineTs - Date.now()) / 1000)
        const fraction = remaining / this.totalSecs
        const R = this.radius

        // 充裕(>50%) 银白 → 警示(>15%) 青色 → 紧迫(≤15%) 红
        const color = fraction > 0.5 ? C.TEXT_PRIMARY
            : fraction > 0.15 ? C.WARN
                : C.ERROR

        // 背景底环
        this.bgRing.clear()
        this.bgRing.circle(0, 0, R).stroke({color: C.GOLD_DIM, width: 3, alpha: 0.15})

        // 进度弧（外发光 + 实线双层）
        this.ring.clear()
        if (fraction > 0) {
            const start = -HALF_PI
            const end = start + TWO_PI * fraction
            this.ring.arc(0, 0, R, start, end).stroke({color, width: 7, alpha: 0.15})
            this.ring.arc(0, 0, R, start, end).stroke({color, width: 3, alpha: 0.90})
        }

        // 右上角秒数徽章
        this.badgeBg.clear()
        this.badgeBg.roundRect(-14, -10, 28, 20, 10)
            .fill({color: C.GLASS, alpha: 0.95})
            .stroke({color, width: 1, alpha: 0.7})
        this.badge.position.set(R * 0.72, -R * 0.72)
        this.badgeText.text = `${Math.ceil(remaining)}s`
        this.badgeText.style.fill = color

        // 最后 5 秒：每整秒播放一次 tick 音效
        if (remaining > 0 && remaining <= 10) {
            const sec = Math.ceil(remaining)
            if (sec !== this.lastTickSec) {
                this.lastTickSec = sec
                soundManager.playTick(sec)
            }
        }

        if (remaining <= 0) this.stop()
    }
}
