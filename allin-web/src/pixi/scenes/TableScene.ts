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
        this.chipAnim.tick(ticker.deltaMS)
        // ALL-IN 脉冲光晕动画
        for (const seat of this.seatSprites) {
            seat.tick(ticker.deltaMS)
        }
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
        // 将街道标签 (PREFLOP, FLOP) 移动到公共牌的上方居中位置
        this.streetLabel.position.set(TABLE_CX, TABLE_CY - 110)
        this.root.addChild(this.streetLabel)

        // 底池背景 200宽 -> 居中 x - 100
        this.potDisplay.position.set(TABLE_CX - 100, TABLE_CY + 65)
        this.root.addChild(this.potDisplay)

        this.root.addChild(this.dealerChip)
    }

    /**
     * 动态生成一个支持九宫格缩放的牌桌基础纹理。
     */
    private createTableTexture() {
        const {SIZE: size, CORNER_RADIUS: radius} = TABLE_TEX_CONFIG

        // 1. 生成内部真·径向渐变毛毡材质
        const feltTexture = this.createFeltGradientTexture(size)

        // 2. 依次叠加桌侧组件
        const g = new Graphics()
        this.drawWoodFrame(g, size, radius)
        this.drawGoldBorders(g, size, radius)
        this.drawFeltSurface(g, size, radius, feltTexture)

        return this.app.renderer.generateTexture(g)
    }

    /** 用 Canvas API 离屏绘制高质量绿毛毡渐变贴图 */
    private createFeltGradientTexture(size: number): Texture {
        const canvas = document.createElement('canvas')
        canvas.width = size
        canvas.height = size
        const ctx = canvas.getContext('2d')!

        const hexToRgb = (hex: number) => `${hex >> 16}, ${(hex >> 8) & 0xff}, ${hex & 0xff}`

        ctx.fillStyle = `rgb(${hexToRgb(C.FELT_EDGE)})`
        ctx.fillRect(0, 0, size, size)

        // 增大径向渐变的半径 (size*0.75)，让中间的高光扩散得更自然，避免边缘衰减得太快形成全黑
        const gradient = ctx.createRadialGradient(size / 2, size / 2, 0, size / 2, size / 2, size * 0.75)
        const centerColor = hexToRgb(C.FELT_CENTER)

        gradient.addColorStop(0, `rgba(${centerColor}, 0.9)`)
        gradient.addColorStop(0.3, `rgba(${centerColor}, 0.7)`)
        gradient.addColorStop(0.6, `rgba(${centerColor}, 0.4)`)
        gradient.addColorStop(0.85, `rgba(${centerColor}, 0.15)`)
        gradient.addColorStop(1, `rgba(${centerColor}, 0.05)`)

        ctx.fillStyle = gradient
        ctx.fillRect(0, 0, size, size)

        return Texture.from(canvas)
    }

    /** 绘制主体木质外沿 */
    private drawWoodFrame(g: Graphics, size: number, radius: number) {
        g.roundRect(0, 0, size, size, radius)
        g.fill({color: C.WOOD_FRAME})
    }

    /** 绘制嵌在木框内侧边缘的金线 */
    private drawGoldBorders(g: Graphics, size: number, radius: number) {
        const feltEdge = RAIL_W

        // 靠内的一条较粗的主金线（宽度3，从 feltEdge-3 出方向内侧描边，刚好贴合绒布边缘）
        const offset1 = feltEdge - 3
        g.roundRect(offset1, offset1, size - offset1 * 2, size - offset1 * 2, Math.max(0, radius - offset1))
            .stroke({color: C.GOLD, width: 3, alpha: 0.6, alignment: 1})

        // 靠外的一条极细的副金线（增加包裹的层次质感）
        const offset2 = feltEdge - 6
        g.roundRect(offset2, offset2, size - offset2 * 2, size - offset2 * 2, Math.max(0, radius - offset2))
            .stroke({color: C.GOLD, width: 1, alpha: 0.15, alignment: 1})
    }

    /** 绘制绿色毛毡体本身以及它的内阴影 */
    private drawFeltSurface(g: Graphics, size: number, radius: number, feltTex: Texture) {
        const feltEdge = RAIL_W
        const innerRadius = Math.max(0, radius - feltEdge)
        const innerSize = size - feltEdge * 2

        // 填充绒布纹理
        g.roundRect(feltEdge, feltEdge, innerSize, innerSize, innerRadius)
        g.fill({texture: feltTex})

        // 施加柔和内阴影：利用多层宽度衰减的纯黑内侧描边，代替原有单层 25px 的“粗暴黑眼圈”
        const shadowSteps = 6
        const maxShadowWidth = 16
        for (let i = 1; i <= shadowSteps; i++) {
            // 从最宽(最浅层)画到最窄(最深层)，靠边的位置会叠加6层变得暗沉，内部分布变浅
            g.roundRect(feltEdge, feltEdge, innerSize, innerSize, innerRadius)
                .stroke({color: 0x000000, width: (maxShadowWidth * i) / shadowSteps, alpha: 0.05, alignment: 1})
        }
    }

    /** 始终创建 9 个座位精灵，displayIdx=0 为本地玩家（大头像） */
    private buildSeats() {
        for (let i = 0; i < 9; i++) {
            const sprite = new SeatSprite(i === 0)
            this.root.addChild(sprite)
            this.seatSprites.push(sprite)
        }
    }

    /** 5 张公共牌 + 行动计时器弧 */
    private buildCommunityArea() {
        // 增加公共牌之间的间距，让牌面不至于太过拥挤
        const cardGap = 20
        const spacing = CARD_W + cardGap
        const startX = TABLE_CX - (5 * spacing - cardGap) / 2

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

    /** 更新 9 个座位精灵；超出 maxPlayers 的槽位直接隐藏 */
    private updateSeats(state: GameState) {
        const myUserId = state.myUserId
        const occupiedServerSeats = state.seats.map((s) => s.seat_index)
        const maxPlayers = state.config?.max_players ?? 9

        for (let displayIdx = 0; displayIdx < 9; displayIdx++) {
            const sprite = this.seatSprites[displayIdx]

            // 超出房间座位数的槽位隐藏
            if (displayIdx >= maxPlayers) {
                sprite.visible = false
                continue
            }
            sprite.visible = true

            const serverIdx = this.toServerSeat(displayIdx)
            const seatData = state.seats.find((s) => s.seat_index === serverIdx) ?? null
            const isActive = seatData !== null
                && state.action_seat === serverIdx
                && state.street !== Street.Idle

            const posName = seatData && state.dealer_seat >= 0
                ? getPositionName(serverIdx, state.dealer_seat, occupiedServerSeats)
                : undefined

            const pos = SEAT_POSITIONS[displayIdx]
            sprite.position.set(pos.x, pos.y)
            sprite.update(
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
            // 本地玩家：弧道放在金色光环外侧的银白轨道上（R+50 → setRadius 内部 +4 → 实际 R+54）
            // 远端玩家：贴着头像边缘（R+4）
            const r = displayIdx === 0 ? AVATAR_R_LOCAL + 50 : AVATAR_R_REMOTE

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

    /**
     * 创建游戏阶段（回合 / Street）的中央提示标签。
     * 它可以展示如 "PREFLOP", "FLOP", "TURN", "RIVER" 或 "SHOWDOWN" 等状态。
     * 采用加宽的字距（letterSpacing: 3）和暗色字体（TEXT_DIM），
     * 让其以一种内敛高级、具有质感的形态悬浮在公共牌堆正上方。
     */
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

    /**
     * 创建庄家标识牌（Dealer Button，俗称 D 扣）。
     * 指代当前局里的庄家（Button）位置。
     * 外观是一枚纯金材质、并带有黑色（C.VOID）粗体 "D" 字样的实体圆形按扣图标。
     *
     * （当前版本使用 SeatSprite 的悬浮位置标签来展示身份，因此该 D 扣暂时设为 visible = false，作为彩蛋备用）
     */
    private createDealerChip(): Graphics {
        const chip = new Graphics()
        // 1. D扣质感底盘
        chip.circle(0, 0, 12)
        chip.fill({color: C.GOLD})
        chip.stroke({color: C.GOLD_DIM, width: 2})

        // 2. 正中间的 D 字样
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
