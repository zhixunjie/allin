import { Container, Graphics, Text } from 'pixi.js'
import {
  C,
  CARD_H, CARD_RADIUS, CARD_W,
  HOLE_CARD_H, HOLE_CARD_RADIUS, HOLE_CARD_W,
  FONT_HEADLINE,
  RANK_DISPLAY,
  SUIT_COLORS, SUIT_SYMBOLS,
} from '../assets'

/**
 * 扑克牌精灵 — 可渲染正面（牌值+花色）或背面（菱形纹理）。
 *
 * 视觉结构（从底到顶）：
 *   bg         — Graphics 层：卡牌阴影 + 底色 + 边框
 *   rankText   — 左上角牌面值文字（如 "A", "10", "K"）
 *   suitSmall  — 牌面值下方的小号花色符号
 *   suitLarge  — 卡牌中央的大号花色符号
 *   diamondText — 牌背面中央的菱形装饰（♦），仅背面可见
 *
 * 使用方式：
 *   公共牌 → new CardSprite()         使用标准尺寸 80×112
 *   手牌   → new CardSprite(true)     使用缩小尺寸 64×90
 *
 * 卡牌字符串格式："Ah"（黑桃A）、"Td"（方块10）、"2c"（梅花2）
 */
export class CardSprite extends Container {
  private bg: Graphics          // 卡牌背景层（阴影 + 底色 + 边框 + 纹理）
  private rankText: Text        // 左上角牌面值（A/K/Q/J/10/9…）
  private suitSmall: Text       // 左上角小花色（♠♥♦♣）
  private suitLarge: Text       // 中央大花色
  private diamondText: Text     // 背面菱形装饰
  private faceDown = false      // 当前是否背面朝上
  private w: number             // 卡牌宽度（px）
  private h: number             // 卡牌高度（px）
  private r: number             // 圆角半径（px）

  /**
   * @param isHoleCard 是否为手牌（true 使用缩小尺寸，false 使用公共牌标准尺寸）
   */
  constructor(isHoleCard = false) {
    super()

    // 根据类型选择尺寸：公共牌 80×112，手牌 64×90
    this.w = isHoleCard ? HOLE_CARD_W : CARD_W
    this.h = isHoleCard ? HOLE_CARD_H : CARD_H
    this.r = isHoleCard ? HOLE_CARD_RADIUS : CARD_RADIUS

    // 背景 Graphics（最底层，承载阴影/底色/边框/纹理）
    this.bg = new Graphics()
    this.addChild(this.bg)

    // 字号根据卡牌大小缩放
    const fontSize = isHoleCard ? 15 : 20
    const suitSmallSize = isHoleCard ? 12 : 14
    const suitLargeSize = isHoleCard ? 26 : 36

    // 左上角牌面值（如 "A", "10"）
    this.rankText = new Text({
      text: '',
      style: {
        fontFamily: FONT_HEADLINE,
        fontSize,
        fontWeight: '900',
        fill: C.BLACK_SUIT,
      },
    })
    this.rankText.position.set(7, 5)
    this.addChild(this.rankText)

    // 牌面值正下方的小花色符号
    this.suitSmall = new Text({
      text: '',
      style: { fontSize: suitSmallSize, fill: C.BLACK_SUIT },
    })
    this.suitSmall.position.set(8, 5 + fontSize + 1)
    this.addChild(this.suitSmall)

    // 卡牌中央的大号花色符号
    this.suitLarge = new Text({
      text: '',
      style: {
        fontFamily: FONT_HEADLINE,
        fontSize: suitLargeSize,
        fontWeight: '900',
        fill: C.BLACK_SUIT,
      },
    })
    this.suitLarge.anchor.set(0.5)
    this.suitLarge.position.set(this.w / 2, this.h / 2 + 6)
    this.addChild(this.suitLarge)

    // 背面菱形装饰（♦） — 常驻子节点，通过 visible 切换显隐
    this.diamondText = new Text({
      text: '♦',
      style: { fontSize: this.w * 0.4, fill: C.CARD_BACK_ACCENT, fontWeight: 'bold' },
    })
    this.diamondText.anchor.set(0.5)
    this.diamondText.alpha = 0.3
    this.diamondText.position.set(this.w / 2, this.h / 2)
    this.diamondText.visible = false
    this.addChild(this.diamondText)

    // 初始绘制背面
    this.drawBack()
  }

  /**
   * 设置为正面并显示指定牌值。
   * @param card 牌字符串，如 "Ah"（黑桃A）、"Td"（方块10）
   *   格式：[牌面值][花色字母]，其中 T=10，花色 c/d/h/s
   */
  setCard(card: string) {
    // 解析牌面值：长度为 3 时取前两位（"10h"），否则取首字符
    const rankChar = card.length === 3 ? card.substring(0, 2) : card[0]
    // 解析花色：始终是最后一个字符
    const suitChar = card[card.length - 1] as keyof typeof SUIT_SYMBOLS
    const color = SUIT_COLORS[suitChar] ?? C.BLACK_SUIT    // 红色或黑色
    const symbol = SUIT_SYMBOLS[suitChar] ?? '?'           // ♠♥♦♣
    const displayRank = RANK_DISPLAY[rankChar] ?? rankChar  // T → "10"

    this.faceDown = false
    this.drawFace()

    // 更新三个文字元素的内容和颜色
    this.rankText.text = displayRank
    this.rankText.style.fill = color
    this.suitSmall.text = symbol
    this.suitSmall.style.fill = color
    this.suitLarge.text = symbol
    this.suitLarge.style.fill = color

    // 正面时：显示牌值/花色，隐藏菱形装饰
    this.rankText.visible = true
    this.suitSmall.visible = true
    this.suitLarge.visible = true
    this.diamondText.visible = false
  }

  /** 设置为背面朝上（隐藏牌值，显示菱形纹理） */
  setFaceDown() {
    this.faceDown = true
    this.rankText.visible = false
    this.suitSmall.visible = false
    this.suitLarge.visible = false
    this.diamondText.visible = true
    this.drawBack()
  }

  /** 绘制正面背景：偏移阴影 → 奶白色底 → 金色描边 */
  private drawFace() {
    const { w, h, r } = this
    this.bg.clear()

    // 右下偏移的半透明阴影，营造立体感
    this.bg.roundRect(2, 3, w, h, r)
    this.bg.fill({ color: 0x000000, alpha: 0.35 })

    // 卡牌主体 — 奶白色 (0xfaf8f0)
    this.bg.roundRect(0, 0, w, h, r)
    this.bg.fill({ color: C.CARD_FACE })

    // 金色边框描边
    this.bg.roundRect(0, 0, w, h, r)
    this.bg.stroke({ color: C.GOLD, width: 2 })
  }

  /**
   * 绘制背面背景：阴影 → 深绿底色 → 中心光晕 → 菱形交叉线纹理 → 金色边框。
   * 纹理通过 ±45° 两组斜线交叉形成菱形网格，模拟赌场扑克牌背面图案。
   */
  private drawBack() {
    const { w, h, r } = this
    this.bg.clear()

    // 右下偏移阴影
    this.bg.roundRect(2, 3, w, h, r)
    this.bg.fill({ color: 0x000000, alpha: 0.35 })

    // 深绿色底色
    this.bg.roundRect(0, 0, w, h, r)
    this.bg.fill({ color: C.CARD_BACK })

    // 中心径向光晕 — 微弱的金色圆形，增加层次感
    this.bg.circle(w / 2, h / 2, w * 0.4)
    this.bg.fill({ color: C.CARD_BACK_ACCENT, alpha: 0.08 })

    // 菱形交叉线图案：两组 ±45° 斜线交叉形成菱形网格
    const m = 6       // 距卡牌边缘的边距
    const spacing = 14 // 斜线间距
    for (let offset = -h; offset < w + h; offset += spacing) {
      // +45° 斜线（左上 → 右下方向）
      const x1 = Math.max(m, offset)
      const y1 = Math.max(m, m + (x1 - offset))
      const x2end = offset + h - 2 * m
      const x2 = Math.min(w - m, x2end)
      const y2 = Math.min(h - m, m + (x2 - offset))
      if (x1 < w - m && y1 < h - m) {
        this.bg.moveTo(x1, y1)
        this.bg.lineTo(x2, y2)
        this.bg.stroke({ color: C.CARD_BACK_ACCENT, width: 0.5, alpha: 0.06 })
      }
      // -45° 斜线（右上 → 左下方向）
      const ax1 = Math.min(w - m, offset)
      const ay1 = Math.max(m, m + (offset - ax1))
      const ax2end = offset - h + 2 * m
      const ax2 = Math.max(m, ax2end)
      const ay2 = Math.min(h - m, m + (offset - ax2))
      if (ax1 > m && ay1 < h - m) {
        this.bg.moveTo(ax1, ay1)
        this.bg.lineTo(ax2, ay2)
        this.bg.stroke({ color: C.CARD_BACK_ACCENT, width: 0.5, alpha: 0.06 })
      }
    }

    // 金色边框描边
    this.bg.roundRect(0, 0, w, h, r)
    this.bg.stroke({ color: C.GOLD, width: 2 })
  }

  /** 当前是否为背面朝上 */
  get isFaceDown() {
    return this.faceDown
  }
}
