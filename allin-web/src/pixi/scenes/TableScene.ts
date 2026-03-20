import {Application, Container, Graphics, NineSliceSprite, Text, Texture, Ticker} from 'pixi.js'
import {Street} from '../../types/enums'
import {useGameStore} from '../../store/game'
import {
    AVATAR_R_LOCAL,
    AVATAR_R_REMOTE,
    C,
    CANVAS_H,
    CANVAS_W,
    CARD_W,
    FONT_HEADLINE,
    getPositionName,
    RAIL_W,
    SEAT_POSITIONS,
    TABLE_CX,
    TABLE_CY,
    TABLE_RX,
    TABLE_RY,
    TABLE_TEX_CONFIG,
} from '../assets'
import {CardSprite} from '../components/CardSprite'
import {PotDisplay} from '../components/PotDisplay'
import {SeatSprite} from '../components/SeatSprite'
import {TimerArc} from '../components/TimerArc'
import {DealAnimation} from './DealAnimation'
import {ChipAnimation} from './ChipAnimation'

type GameState = ReturnType<typeof useGameStore.getState>

/**
 * 牌桌主场景 — 管理所有游戏视觉元素的构建和状态更新。
 *
 * 层级结构：
 * ┌─────────────────────────────────────────┐
 * │ root (Container)                        │
 * │  ├─ 背景层  — 深空底色、星空、环境光晕    │
 * │  ├─ 牌桌层  — 木框、毛毡、装饰线、标签    │
 * │  ├─ 座位层  — 9 个 SeatSprite           │
 * │  └─ 公共牌区 — 5 张 CardSprite + 计时弧  │
 * └─────────────────────────────────────────┘
 *
 * 数据流：Zustand gameStore → subscribe → update*() → 更新子元素
 *
 * 座位旋转：本地玩家始终在底部（displayIdx = 0）
 *   server → display : (serverIdx - myServerSeat + 9) % 9
 *   display → server : (displayIdx + myServerSeat) % 9
 */
export class TableScene {
    private app: Application
    private root: Container

    // ── 子元素 ────────────────────────────────────────
    private seatSprites: SeatSprite[] = []
    private communityCards: CardSprite[] = []
    private potDisplay: PotDisplay
    private timerArc: TimerArc
    private dealAnim: DealAnimation
    private chipAnim: ChipAnimation
    private streetLabel: Text
    private dealerChip: Graphics

    // ── 帧间状态（用于检测变化） ──────────────────────
    private myServerSeat = -1
    private prevStreet: Street = Street.Idle
    private prevActionSeat = -1

    private unsubscribe: () => void = () => {
    }

    constructor(app: Application) {
        this.app = app
        this.root = new Container()
        this.app.stage.addChild(this.root)

        this.potDisplay = new PotDisplay()
        this.timerArc = new TimerArc()
        this.dealAnim = new DealAnimation(app)
        this.chipAnim = new ChipAnimation(app)
        this.streetLabel = this.createStreetLabel()
        this.dealerChip = this.createDealerChip()
    }

    // ═══════ 生命周期 ═══════

    /** 构建所有子元素，启动动画循环，订阅状态 */
    init() {
        this.buildBackground()
        this.buildTable()
        this.buildSeats()
        this.buildCommunityArea()

        this.app.ticker.add(this.onTick, this)

        this.unsubscribe = useGameStore.subscribe((state) => this.updateFromState(state))
        this.updateFromState(useGameStore.getState())
    }

    /** 取消状态订阅，移除动画循环 */
    destroy() {
        this.unsubscribe()
        this.app.ticker.remove(this.onTick, this)
    }

    private onTick(ticker: Ticker) {
        this.timerArc.tick()
        this.dealAnim.tick(ticker.deltaMS)
        this.chipAnim.tick(ticker.deltaMS)
    }

    // ═══════ 构建方法 ═══════

    /** 背景层：深空底色 + 确定性星空（固定种子）+ 环境金色光晕 */
    private buildBackground() {
        const bg = new Graphics()
        bg.rect(0, 0, CANVAS_W, CANVAS_H)
        bg.fill({color: C.VOID})
        this.root.addChild(bg)

        const stars = new Graphics()
        const seed = 42
        for (let i = 0; i < 120; i++) {
            const px = this.pseudoRandom(seed + i * 3) * CANVAS_W
            const py = this.pseudoRandom(seed + i * 7 + 1) * CANVAS_H
            const size = this.pseudoRandom(seed + i * 13 + 2)
            const alpha = 0.05 + size * 0.12
            const r = 0.5 + size * 1.2
            stars.circle(px, py, r)
            stars.fill({color: 0xffffff, alpha})
        }
        this.root.addChild(stars)

        const glow = new Graphics()
        glow.ellipse(TABLE_CX, TABLE_CY, TABLE_RX + 80, TABLE_RY + 60)
        glow.fill({color: C.GOLD, alpha: 0.03})
        glow.ellipse(TABLE_CX, TABLE_CY, TABLE_RX + 40, TABLE_RY + 30)
        glow.fill({color: C.GOLD, alpha: 0.02})
        this.root.addChild(glow)
    }

    /**
     * 牌桌层：由内部动态生成的九宫格纹理 (NineSliceSprite) 渲染。
     * 可以完美地自适应尺寸并支持无损的边缘拉伸。
     */
    private buildTable() {
        // 使用生成的纹理创建九宫格精灵
        const tableTex = this.createTableTexture()
        const rad = TABLE_TEX_CONFIG.CORNER_RADIUS
        const tableSprite = new NineSliceSprite({
            texture: tableTex,
            leftWidth: rad,
            rightWidth: rad,
            topHeight: rad,
            bottomHeight: rad,
        })

        // 设定目标的牌桌总宽高
        const targetWidth = (TABLE_RX + RAIL_W) * 2
        const targetHeight = (TABLE_RY + RAIL_W) * 2

        tableSprite.width = targetWidth
        tableSprite.height = targetHeight

        // 九宫格的定位基于左上角，我们用 (X - width/2, Y - height/2) 将其居中
        tableSprite.position.set(TABLE_CX - targetWidth / 2, TABLE_CY - targetHeight / 2)
        this.root.addChild(tableSprite)

        // 背景全局发光已在 buildBackground 完成，此处保留功能元素逻辑
        this.streetLabel.position.set(TABLE_CX, TABLE_CY + TABLE_RY * 0.65)
        this.root.addChild(this.streetLabel)

        this.potDisplay.position.set(TABLE_CX - 110, TABLE_CY + 40)
        this.root.addChild(this.potDisplay)

        this.root.addChild(this.dealerChip)
    }

    /**
     * 动态生成一个支持九宫格缩放的牌桌基础纹理。
     * 利用 Canvas 2D API 绘制完美平滑的真·径向渐变毛毡。
     */
    private createTableTexture() {
        const size = TABLE_TEX_CONFIG.SIZE
        const radius = TABLE_TEX_CONFIG.CORNER_RADIUS

        // 1. 利用 Canvas 2D 创建平滑径向渐变毛毡贴图
        const canvas = document.createElement('canvas')
        canvas.width = size
        canvas.height = size
        const ctx = canvas.getContext('2d')!

        const hexToRgb = (hex: number) => `${hex >> 16}, ${(hex >> 8) & 0xff}, ${hex & 0xff}`

        // 底层实色（边缘基底颜色，Alpha 1.0）
        ctx.fillStyle = `rgb(${hexToRgb(C.FELT_EDGE)})`
        ctx.fillRect(0, 0, size, size)

        // 中心高亮叠加径向光晕（融合到完全透明）
        // 增加更多的扩散阶段，让高光衰减有着像光学漫反射那样富有层次感的效果
        const gradient = ctx.createRadialGradient(size / 2, size / 2, 0, size / 2, size / 2, size / 2)
        gradient.addColorStop(0, `rgba(${hexToRgb(C.FELT_CENTER)}, 0.8)`)
        gradient.addColorStop(0.25, `rgba(${hexToRgb(C.FELT_CENTER)}, 0.6)`)
        gradient.addColorStop(0.5, `rgba(${hexToRgb(C.FELT_CENTER)}, 0.4)`)
        gradient.addColorStop(0.75, `rgba(${hexToRgb(C.FELT_CENTER)}, 0.15)`)
        gradient.addColorStop(1, `rgba(${hexToRgb(C.FELT_CENTER)}, 0)`)

        ctx.fillStyle = gradient
        ctx.fillRect(0, 0, size, size)

        const feltTexture = Texture.from(canvas)

        const g = new Graphics()

        // 2. 深色木质边框
        g.roundRect(0, 0, size, size, radius)
        g.fill({color: C.WOOD_FRAME})

        // 3. 金边
        g.roundRect(2, 2, size - 4, size - 4, radius - 2).stroke({color: C.GOLD, width: 3, alpha: 0.6})
        g.roundRect(5, 5, size - 10, size - 10, radius - 5).stroke({color: C.GOLD, width: 1, alpha: 0.15})

        // 4. 绿毛毡渐变层 (填充刚创建的无缝渐变贴图)
        const feltEdge = RAIL_W
        g.roundRect(feltEdge, feltEdge, size - feltEdge * 2, size - feltEdge * 2, radius - feltEdge)
        g.fill({texture: feltTexture})

        return this.app.renderer.generateTexture(g)
    }

    /** 9 个座位精灵，displayIdx=0 为本地玩家（大头像） */
    private buildSeats() {
        for (let i = 0; i < 9; i++) {
            const sprite = new SeatSprite(i === 0)
            this.root.addChild(sprite)
            this.seatSprites.push(sprite)
        }
    }

    /** 5 张公共牌 + 行动计时器弧 */
    private buildCommunityArea() {
        const spacing = CARD_W + 12
        const startX = TABLE_CX - (5 * spacing - 12) / 2

        for (let i = 0; i < 5; i++) {
            const card = new CardSprite()
            card.visible = false
            card.position.set(startX + i * spacing, TABLE_CY - 70)
            this.root.addChild(card)
            this.communityCards.push(card)
        }

        this.root.addChild(this.timerArc)
    }

    // ═══════ 状态更新 ═══════

    /** 将 gameStore 快照映射到所有子元素（每次状态变化触发） */
    private updateFromState(state: GameState) {
        if (state.myUserId) {
            const mySeat = state.seats.find((s) => s.user_id === state.myUserId)
            if (mySeat) this.myServerSeat = mySeat.seat_index
        }

        this.updateSeats(state)
        this.updateTimer(state)
        this.updateCommunityCards(state)
        this.updateTableLabels(state)
        this.triggerChipAnimation(state)

        this.prevStreet = state.street
        this.prevActionSeat = state.action_seat
    }

    /** 更新 9 个座位精灵的位置和内容 */
    private updateSeats(state: GameState) {
        const myUserId = state.myUserId
        const occupiedServerSeats = state.seats.map((s) => s.seat_index)

        for (let displayIdx = 0; displayIdx < 9; displayIdx++) {
            const serverIdx = this.toServerSeat(displayIdx)
            const seatData = state.seats.find((s) => s.seat_index === serverIdx) ?? null
            const isActive = seatData !== null
                && state.action_seat === serverIdx
                && state.street !== Street.Idle

            const posName = seatData && state.dealer_seat >= 0
                ? getPositionName(serverIdx, state.dealer_seat, occupiedServerSeats)
                : undefined

            const pos = SEAT_POSITIONS[displayIdx]
            this.seatSprites[displayIdx].position.set(pos.x, pos.y)
            this.seatSprites[displayIdx].update(
                seatData,
                isActive,
                seatData?.user_id === myUserId ? state.myHole : undefined,
                posName,
                displayIdx,
            )
        }
    }

    /** 更新行动计时器弧的位置和进度 */
    private updateTimer(state: GameState) {
        if (state.deadlineTs && state.action_seat >= 0) {
            const displayIdx = this.toDisplaySeat(state.action_seat)
            const pos = SEAT_POSITIONS[displayIdx]
            const r = displayIdx === 0 ? AVATAR_R_LOCAL : AVATAR_R_REMOTE

            this.timerArc.setRadius(r)
            this.timerArc.position.set(pos.x, pos.y)
            this.timerArc.start(state.deadlineTs, state.config?.action_time_sec ?? 30)
        } else if (this.prevActionSeat !== state.action_seat) {
            this.timerArc.stop()
        }
    }

    /** 更新 5 张公共牌（已发正面 / 未发背面 / idle 隐藏） */
    private updateCommunityCards(state: GameState) {
        const community = state.community ?? []
        const isActiveHand = state.street !== Street.Idle

        for (let i = 0; i < 5; i++) {
            if (i < community.length) {
                this.communityCards[i].setCard(community[i])
                this.communityCards[i].visible = true
            } else if (isActiveHand) {
                this.communityCards[i].setFaceDown()
                this.communityCards[i].visible = true
            } else {
                this.communityCards[i].visible = false
            }
        }
    }

    /** 更新底池金额、轮次标签 */
    private updateTableLabels(state: GameState) {
        this.potDisplay.setPot(state.pot ?? 0)
        this.streetLabel.text = state.street === Street.Idle ? '' : state.street.toUpperCase()
        this.dealerChip.visible = false // 位置标签已传达庄家信息，D 标记常隐藏
    }

    /** 轮次切换时触发筹码飞向底池动画 */
    private triggerChipAnimation(state: GameState) {
        if (state.street === this.prevStreet || state.street === Street.Idle) return

        for (const seat of state.seats) {
            if (seat.bet > 0) {
                const displayIdx = this.toDisplaySeat(seat.seat_index)
                const pos = SEAT_POSITIONS[displayIdx]
                this.chipAnim.flyChip(pos.x, pos.y, seat.bet)
            }
        }
    }

    // ═══════ 工具方法 ═══════

    /** 显示索引 → 服务器座位索引 */
    private toServerSeat(displayIdx: number): number {
        if (this.myServerSeat < 0) return displayIdx
        return (displayIdx + this.myServerSeat) % 9
    }

    /** 服务器座位索引 → 显示索引 */
    private toDisplaySeat(serverIdx: number): number {
        if (this.myServerSeat < 0) return serverIdx
        return (serverIdx - this.myServerSeat + 9) % 9
    }

    /** 确定性伪随机（sin 哈希），用于星空生成，返回 0~1 */
    private pseudoRandom(seed: number): number {
        const x = Math.sin(seed * 127.1 + 311.7) * 43758.5453
        return x - Math.floor(x)
    }

    // ═══════ 构造辅助 ═══════

    private createStreetLabel(): Text {
        const label = new Text({
            text: '',
            style: {
                fontFamily: FONT_HEADLINE,
                fontSize: 12,
                fontWeight: '500',
                fill: C.TEXT_DIM,
                letterSpacing: 3,
            },
        })
        label.anchor.set(0.5)
        return label
    }

    private createDealerChip(): Graphics {
        const chip = new Graphics()
        chip.circle(0, 0, 12)
        chip.fill({color: C.GOLD})
        chip.stroke({color: C.GOLD_DIM, width: 2})

        const label = new Text({
            text: 'D',
            style: {fontFamily: FONT_HEADLINE, fontSize: 11, fill: C.VOID, fontWeight: '900'},
        })
        label.anchor.set(0.5)
        chip.addChild(label)
        chip.visible = false
        return chip
    }
}
