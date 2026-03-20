import { Application, Container, Graphics, Text, Ticker } from 'pixi.js'
import { Street } from '../../types/enums'
import { useGameStore } from '../../store/game'
import {
  C,
  CARD_W,
  FONT_HEADLINE,
  SEAT_POSITIONS,
  TABLE_CX, TABLE_CY,
  TABLE_H, TABLE_W,
  TABLE_RX, TABLE_RY,
  FRAME_THICKNESS,
  AVATAR_R_LOCAL, AVATAR_R_REMOTE,
  getPositionName,
} from '../assets'
import { CardSprite } from '../components/CardSprite'
import { PotDisplay } from '../components/PotDisplay'
import { SeatSprite } from '../components/SeatSprite'
import { TimerArc } from '../components/TimerArc'
import { DealAnimation } from './DealAnimation'
import { ChipAnimation } from './ChipAnimation'

/**
 * 牌桌主场景 — 管理所有游戏视觉元素的构建和状态更新。
 *
 * 架构概览：
 * ┌─────────────────────────────────────────┐
 * │ root (Container)                        │
 * │  ├─ 背景层 (buildBackground)            │
 * │  │   ├─ 深空背景色                       │
 * │  │   ├─ 确定性星空（120 颗，伪随机分布）   │
 * │  │   └─ 环境金色光晕                     │
 * │  ├─ 牌桌层 (buildTable)                 │
 * │  │   ├─ 木质边框椭圆                     │
 * │  │   ├─ 金色装饰描边                     │
 * │  │   ├─ 绿色毛毡（4 层同心椭圆渐变）      │
 * │  │   ├─ 内侧金色装饰线                   │
 * │  │   ├─ 轮次标签 (PREFLOP/FLOP 等)      │
 * │  │   ├─ 底池显示 (PotDisplay)           │
 * │  │   └─ 庄家筹码 (D 标记)               │
 * │  ├─ 座位层 (buildSeats)                 │
 * │  │   └─ 9 个 SeatSprite                 │
 * │  └─ 公共牌区 (buildCommunityArea)       │
 * │      ├─ 5 张 CardSprite                 │
 * │      └─ TimerArc（行动计时器弧）         │
 * └─────────────────────────────────────────┘
 *
 * 数据流：Zustand gameStore → subscribe → updateFromState() → 更新所有子元素
 *
 * 座位旋转：本地玩家始终显示在底部（displayIdx=0）。
 *   服务器座位 → 显示座位：(serverIdx - myServerSeat + 9) % 9
 *   显示座位 → 服务器座位：(displayIdx + myServerSeat) % 9
 */
export class TableScene {
  private app: Application
  private root: Container              // 根容器，所有元素的父节点
  private seatSprites: SeatSprite[] = [] // 9 个座位精灵（displayIdx 顺序）
  private communityCards: CardSprite[] = [] // 5 张公共牌精灵
  private potDisplay: PotDisplay        // 底池金额显示
  private timerArc: TimerArc            // 行动计时器弧
  private dealAnim: DealAnimation       // 发牌飞行动画
  private chipAnim: ChipAnimation       // 筹码飞向底池动画
  private streetLabel: Text             // 轮次标签（PREFLOP/FLOP/TURN/RIVER）
  private dealerChip: Graphics          // 庄家 D 标记（当前隐藏，由位置标签替代）
  private unsubscribe: () => void = () => {} // Zustand 订阅取消函数

  private myServerSeat = -1             // 本地玩家的服务器座位索引（-1 表示未入座）
  private prevStreet: Street = Street.Idle // 上一帧的轮次（用于检测轮次切换）
  private prevActionSeat = -1            // 上一帧的行动座位（用于检测变化）

  constructor(app: Application) {
    this.app = app
    this.root = new Container()
    this.app.stage.addChild(this.root)

    this.potDisplay = new PotDisplay()
    this.timerArc = new TimerArc()
    this.dealAnim = new DealAnimation(app)
    this.chipAnim = new ChipAnimation(app)

    this.streetLabel = new Text({
      text: '',
      style: {
        fontFamily: FONT_HEADLINE,
        fontSize: 12,
        fontWeight: '500',
        fill: C.TEXT_DIM,
        letterSpacing: 3,
      },
    })
    this.streetLabel.anchor.set(0.5)

    // 庄家筹码 — 金色圆形，带 "D" 标识
    this.dealerChip = new Graphics()
    this.dealerChip.circle(0, 0, 12)
    this.dealerChip.fill({ color: C.GOLD })
    this.dealerChip.stroke({ color: C.GOLD_DIM, width: 2 })
    const dLabel = new Text({
      text: 'D',
      style: { fontFamily: FONT_HEADLINE, fontSize: 11, fill: C.VOID, fontWeight: '900' },
    })
    dLabel.anchor.set(0.5)
    this.dealerChip.addChild(dLabel)
    this.dealerChip.visible = false
  }

  /** 初始化场景：构建所有子元素，启动动画循环，订阅状态变化 */
  init() {
    this.buildBackground()
    this.buildTable()
    this.buildSeats()
    this.buildCommunityArea()

    // 注册每帧回调（驱动计时器弧、发牌动画、筹码飞行动画）
    this.app.ticker.add(this.onTick, this)

    // 订阅 Zustand gameStore — 任何状态变化都会触发画面更新
    this.unsubscribe = useGameStore.subscribe((state) => {
      this.updateFromState(state)
    })
    // 立即用当前状态渲染一次（处理断线重连后的快照恢复）
    this.updateFromState(useGameStore.getState())
  }

  /** 销毁场景：取消状态订阅 + 移除动画循环 */
  destroy() {
    this.unsubscribe()
    this.app.ticker.remove(this.onTick, this)
  }

  /** 每帧回调：驱动计时器弧进度 + 发牌/筹码飞行动画 */
  private onTick(ticker: Ticker) {
    this.timerArc.tick()
    this.dealAnim.tick(ticker.deltaMS)
    this.chipAnim.tick(ticker.deltaMS)
  }

  // ═══════ 构建方法 ═══════

  /** 构建背景层：深空底色 + 确定性星空 + 环境光晕 */
  private buildBackground() {
    // 深空背景色（0x060c14）— 全屏矩形
    const bg = new Graphics()
    bg.rect(0, 0, TABLE_W, TABLE_H)
    bg.fill({ color: C.VOID })
    this.root.addChild(bg)

    // 确定性星空：使用伪随机函数 + 固定种子生成 120 颗星星
    // 确保每次打开游戏星空布局一致
    const stars = new Graphics()
    const seed = 42
    for (let i = 0; i < 120; i++) {
      const px = this.pseudoRandom(seed + i * 3) * TABLE_W
      const py = this.pseudoRandom(seed + i * 7 + 1) * TABLE_H
      const size = this.pseudoRandom(seed + i * 13 + 2)
      const alpha = 0.05 + size * 0.12
      const r = 0.5 + size * 1.2
      stars.circle(px, py, r)
      stars.fill({ color: 0xffffff, alpha })
    }
    this.root.addChild(stars)

    // 牌桌后方的环境金色光晕 — 两层椭圆，从外到内逐渐变亮
    const ambientGlow = new Graphics()
    ambientGlow.ellipse(TABLE_CX, TABLE_CY, TABLE_RX + 80, TABLE_RY + 60)
    ambientGlow.fill({ color: C.GOLD, alpha: 0.03 })
    ambientGlow.ellipse(TABLE_CX, TABLE_CY, TABLE_RX + 40, TABLE_RY + 30)
    ambientGlow.fill({ color: C.GOLD, alpha: 0.02 })
    this.root.addChild(ambientGlow)
  }

  /**
   * 构建牌桌层：木质边框 → 金色装饰 → 绿色毛毡（径向渐变） → 装饰线 → 功能元素。
   * 毛毡渐变通过 4 层同心椭圆实现（PixiJS 不支持原生径向渐变）。
   */
  private buildTable() {
    // ---- 深色木质边框（外边界）----
    const frame = new Graphics()
    frame.ellipse(TABLE_CX, TABLE_CY, TABLE_RX + FRAME_THICKNESS, TABLE_RY + FRAME_THICKNESS)
    frame.fill({ color: C.WOOD_FRAME })
    this.root.addChild(frame)

    // ---- 金色边框装饰 ----
    const goldBorder = new Graphics()
    goldBorder.ellipse(TABLE_CX, TABLE_CY, TABLE_RX + 2, TABLE_RY + 2)
    goldBorder.stroke({ color: C.GOLD, width: 3, alpha: 0.6 })
    goldBorder.ellipse(TABLE_CX, TABLE_CY, TABLE_RX + 5, TABLE_RY + 5)
    goldBorder.stroke({ color: C.GOLD, width: 1, alpha: 0.15 })
    this.root.addChild(goldBorder)

    // ---- 绿色毛毡 — 4 层同心椭圆模拟径向渐变（边缘暗 → 中心亮）----
    const felt = new Graphics()
    felt.ellipse(TABLE_CX, TABLE_CY, TABLE_RX, TABLE_RY)
    felt.fill({ color: C.FELT_EDGE })
    felt.ellipse(TABLE_CX, TABLE_CY, TABLE_RX * 0.88, TABLE_RY * 0.88)
    felt.fill({ color: C.FELT_OUTER })
    felt.ellipse(TABLE_CX, TABLE_CY, TABLE_RX * 0.7, TABLE_RY * 0.7)
    felt.fill({ color: C.FELT_MID })
    felt.ellipse(TABLE_CX, TABLE_CY, TABLE_RX * 0.4, TABLE_RY * 0.4)
    felt.fill({ color: C.FELT_CENTER })

    // 内阴影（使毛毡边缘变暗）
    felt.ellipse(TABLE_CX, TABLE_CY, TABLE_RX, TABLE_RY)
    felt.stroke({ color: 0x000000, width: 50, alpha: 0.3 })

    this.root.addChild(felt)

    // ---- 内侧金色装饰线 ----
    const innerGold = new Graphics()
    innerGold.ellipse(TABLE_CX, TABLE_CY, TABLE_RX * 0.78, TABLE_RY * 0.78)
    innerGold.stroke({ color: C.GOLD, width: 1, alpha: 0.1 })
    this.root.addChild(innerGold)

    // ---- 轮次标签 ----
    this.streetLabel.position.set(TABLE_CX, TABLE_CY + TABLE_RY * 0.65)
    this.root.addChild(this.streetLabel)

    // ---- 底池显示（公共牌下方居中）----
    this.potDisplay.position.set(TABLE_CX - 110, TABLE_CY + 40)
    this.root.addChild(this.potDisplay)

    // ---- 庄家筹码 ----
    this.root.addChild(this.dealerChip)
  }

  /** 构建 9 个座位精灵。displayIdx=0 的是本地玩家（大头像），其余为远程玩家 */
  private buildSeats() {
    for (let i = 0; i < 9; i++) {
      const sprite = new SeatSprite(i === 0)  // i===0 → isLocal=true
      this.root.addChild(sprite)
      this.seatSprites.push(sprite)
    }
  }

  /** 构建公共牌区域（5 张标准尺寸 CardSprite）和行动计时器弧 */
  private buildCommunityArea() {
    // 公共牌 — 5 张牌水平排列，等间距居中
    const cardSpacing = CARD_W + 12  // 牌宽 + 间距
    const totalW = 5 * cardSpacing - 12
    const startX = TABLE_CX - totalW / 2

    for (let i = 0; i < 5; i++) {
      const card = new CardSprite()
      card.visible = false
      card.position.set(startX + i * cardSpacing, TABLE_CY - 70)
      this.root.addChild(card)
      this.communityCards.push(card)
    }

    // 计时器弧（根据当前行动座位重新定位）
    this.root.addChild(this.timerArc)
  }

  // ═══════ 状态更新 ═══════

  /**
   * 核心更新方法 — 将 Zustand gameStore 状态映射到所有 PixiJS 子元素。
   * 由 subscribe 回调触发，每次状态变化都会执行。
   */
  private updateFromState(state: ReturnType<typeof useGameStore.getState>) {
    const myUserId = state.myUserId

    // 记录本地玩家的服务器座位索引（用于座位旋转计算）
    if (myUserId) {
      const mySeat = state.seats.find((s) => s.user_id === myUserId)
      if (mySeat) this.myServerSeat = mySeat.seat_index
    }

    // 收集所有已占用的服务器座位索引（用于位置名称计算）
    const occupiedServerSeats = state.seats.map((s) => s.seat_index)

    // 遍历 9 个显示位，通过旋转映射找到对应的服务器座位数据
    for (let displayIdx = 0; displayIdx < 9; displayIdx++) {
      const serverIdx = this.toServerSeat(displayIdx)
      const seatData = state.seats.find((s) => s.seat_index === serverIdx) ?? null
      const isActive =
        seatData !== null &&
        state.action_seat === serverIdx &&
        state.street !== Street.Idle

      const pos = SEAT_POSITIONS[displayIdx]
      this.seatSprites[displayIdx].position.set(pos.x, pos.y)

      const posName = seatData && state.dealer_seat >= 0
        ? getPositionName(serverIdx, state.dealer_seat, occupiedServerSeats)
        : undefined

      this.seatSprites[displayIdx].update(
        seatData,
        isActive,
        seatData?.user_id === myUserId ? state.myHole : undefined,
        posName,
        displayIdx,
      )
    }

    // 计时器弧 — 定位到当前行动座位的头像处
    if (state.deadlineTs && state.action_seat >= 0) {
      const displayIdx = this.toDisplaySeat(state.action_seat)
      const pos = SEAT_POSITIONS[displayIdx]
      const r = displayIdx === 0 ? AVATAR_R_LOCAL : AVATAR_R_REMOTE
      this.timerArc.setRadius(r)
      this.timerArc.position.set(pos.x, pos.y)

      const timeSec = state.config?.action_time_sec ?? 30
      this.timerArc.start(state.deadlineTs, timeSec)
    } else if (this.prevActionSeat !== state.action_seat) {
      this.timerArc.stop()
    }

    // 公共牌 — 活跃手牌期间未发出的位置显示为背面
    const community = state.community ?? []
    const isActiveHand = state.street !== Street.Idle
    for (let i = 0; i < 5; i++) {
      if (i < community.length) {
        this.communityCards[i].setCard(community[i])
        this.communityCards[i].visible = true
      } else if (isActiveHand) {
        // 未发出的位置显示背面朝上的牌
        this.communityCards[i].setFaceDown()
        this.communityCards[i].visible = true
      } else {
        this.communityCards[i].visible = false
      }
    }

    // 底池
    this.potDisplay.setPot(state.pot ?? 0)

    // 轮次标签
    const street = state.street ?? Street.Idle
    this.streetLabel.text = street === Street.Idle ? '' : street.toUpperCase()

    // 庄家筹码 — 位置标签可见时隐藏（信息已由标签传达）
    this.dealerChip.visible = false

    // 轮次切换时的筹码动画
    if (state.street !== this.prevStreet && state.street !== Street.Idle) {
      for (const seat of state.seats) {
        if (seat.bet > 0) {
          const displayIdx = this.toDisplaySeat(seat.seat_index)
          const pos = SEAT_POSITIONS[displayIdx]
          this.chipAnim.flyChip(pos.x, pos.y, seat.bet)
        }
      }
    }

    this.prevStreet = state.street
    this.prevActionSeat = state.action_seat
  }

  // ═══════ 工具方法 ═══════

  /**
   * 显示索引 → 服务器座位索引。
   * 将画面上的位置（0=底部本地）转换为服务器端的座位编号。
   */
  private toServerSeat(displayIdx: number): number {
    if (this.myServerSeat < 0) return displayIdx
    return (displayIdx + this.myServerSeat) % 9
  }

  /**
   * 服务器座位索引 → 显示索引。
   * 将服务器端座位编号转换为画面上的位置（0=底部本地）。
   */
  private toDisplaySeat(serverIdx: number): number {
    if (this.myServerSeat < 0) return serverIdx
    return (serverIdx - this.myServerSeat + 9) % 9
  }

  /**
   * 确定性伪随机数生成器（基于 sin 哈希）。
   * 用于星空背景的随机位置/大小/透明度，确保每次渲染结果一致。
   * @returns 0~1 之间的浮点数
   */
  private pseudoRandom(seed: number): number {
    const x = Math.sin(seed * 127.1 + 311.7) * 43758.5453
    return x - Math.floor(x)
  }
}
