import {Assets, Container, Graphics, Sprite, Text, Texture} from 'pixi.js'
import type {SeatSnapshot} from '../../store/game'
import {
    AVATAR_R_LOCAL,
    AVATAR_R_REMOTE,
    C,
    FONT_BODY,
    FONT_HEADLINE,
    SEAT_POSITIONS,
    TABLE_CX,
    TABLE_CY,
} from '../assets'
import {CardSprite} from './CardSprite'

/**
 * 座位精灵 — 表示牌桌上一个座位的完整视觉元素。
 *
 * 视觉结构（从底到顶）：
 *   avatarBg       — 头像圆形背景（玻璃面板 + 描边/活跃光环）
 *   avatarIcon     — 头像 emoji（玩家基于 userId 哈希分配，bot 固定 🤖）
 *   posTag         — 位置标签（BTN/SB/BB/UTG 等），浮动于头像上方
 *   nameBadge      — 本地玩家的金色名称底板（仅本地玩家）
 *   nameText       — 玩家昵称
 *   stackPanel     — 本地玩家的筹码面板（玻璃底板 + 金色边框）
 *   stackText      — 筹码金额文字
 *   stackLabel     — "筹码余额" 副标签（仅本地玩家）
 *   betBadge       — 下注徽章（🪙 + 金额），朝牌桌中心浮动
 *   statusBadge    — 状态徽章（"已弃牌" / "ALL-IN"）
 *   card0, card1   — 两张手牌 CardSprite
 *
 * 两种模式：
 *   本地玩家（displayIdx=0） — 大头像(R=52)，底部金色名称徽章，下方玻璃筹码面板，手牌扇形展开在右侧
 *   远程玩家（displayIdx>0） — 小头像(R=38)，名称在头像下方，筹码在名称下方，手牌显示在头像上方
 */
export class SeatSprite extends Container {
    private avatarBg!: Graphics      // 头像圆形背景 + 描边
    private avatarImg!: Sprite       // 头像图片
    private nameText!: Text          // 玩家昵称
    private stackText!: Text         // 筹码金额
    private stackLabel!: Text        // 本地玩家的 "筹码余额" 副标签
    private betBadge!: Container     // 下注徽章容器（背景 + 图标 + 文字）
    private betIcon!: Text           // 下注图标 🪙
    private betText!: Text           // 下注金额文字
    private statusBadge!: Container  // 状态徽章容器（"已弃牌" / "ALL-IN"）
    private statusBadgeBg!: Graphics // 状态徽章背景
    private statusBadgeText!: Text   // 状态徽章文字
    private posTag!: Container       // 位置标签容器（BTN/SB/BB 等）
    private posTagText!: Text        // 位置标签文字
    private card0!: CardSprite       // 手牌 1
    private card1!: CardSprite       // 手牌 2
    private nameBadge!: Graphics     // 本地玩家金色名称底板
    private stackPanel!: Graphics    // 本地玩家玻璃筹码面板

    private isLocal: boolean        // 是否为本地玩家（底部大头像）
    private avatarR: number         // 头像半径（本地 52，远程 38）
    private displayIdx = 0          // 显示索引（0=本地，用于计算朝向）

    /**
     * @param isLocal 是否为本地玩家座位（displayIdx=0，使用大头像 + 特殊布局）
     */
    constructor(isLocal = false) {
        super()
        this.isLocal = isLocal
        this.avatarR = isLocal ? AVATAR_R_LOCAL : AVATAR_R_REMOTE

        this.buildAvatar()
        this.buildPosTag()
        this.buildNameText()
        this.buildStackElements()
        this.buildBetBadge()
        this.buildStatusBadge()
        this.buildCards()

        // 初始状态：绘制空座位
        this.drawEmpty()
    }

    // ─── 构造分解 ──────────────────────────────────────────────

    private buildAvatar() {
        this.avatarBg = new Graphics()
        this.addChild(this.avatarBg)

        // 头像图片 Sprite
        this.avatarImg = new Sprite()
        this.avatarImg.anchor.set(0.5)

        // 裁切遮罩：把方形图片裁成正圆
        const mask = new Graphics()
        mask.circle(0, 0, this.avatarR)
        mask.fill({color: 0xffffff})
        this.avatarImg.mask = mask

        this.addChild(this.avatarImg)
        this.addChild(mask)
    }

    private buildPosTag() {
        this.posTag = new Container()
        this.posTag.visible = false

        const posTagBg = new Graphics()  // 标签背景（玻璃面板 + 金色边框）
        this.posTag.addChild(posTagBg)

        this.posTagText = new Text({
            text: '',
            style: {
                fontFamily: FONT_HEADLINE,
                fontSize: 10,
                fontWeight: '800',
                fill: C.GOLD,
                letterSpacing: 1,
            },
        })
        this.posTagText.anchor.set(0.5)
        this.posTag.addChild(this.posTagText)
        this.addChild(this.posTag)
    }

    private buildNameText() {
        this.nameBadge = new Graphics()
        
        if (this.isLocal) {
            // 本地玩家：头像底部的金色药丸形名称徽章
            this.addChild(this.nameBadge)
            this.nameText = new Text({
                text: '',
                style: {
                    fontFamily: FONT_BODY,
                    fontSize: 11,
                    fontWeight: '800',
                    fill: C.VOID,        // 深色文字（金底反色）
                    letterSpacing: 2,
                },
            })
            this.nameText.anchor.set(0.5)
        } else {
            // 远程玩家：头像下方普通文字
            this.nameText = new Text({
                text: '',
                style: {
                    fontFamily: FONT_BODY,
                    fontSize: 12,
                    fontWeight: '700',
                    fill: C.TEXT_PRIMARY,
                    letterSpacing: 0.5,
                },
            })
            this.nameText.anchor.set(0.5, 0)
        }
        this.addChild(this.nameText)
    }

    private buildStackElements() {
        // 本地玩家有独立的玻璃面板底板
        this.stackPanel = new Graphics()
        if (this.isLocal) this.addChild(this.stackPanel)

        this.stackText = new Text({
            text: '',
            style: {
                fontFamily: FONT_HEADLINE,
                fontSize: this.isLocal ? 22 : 11,   // 本地大号，远程小号
                fontWeight: '900',
                fill: C.GOLD,
                letterSpacing: this.isLocal ? 1.5 : 0,
            },
        })
        this.stackText.anchor.set(0.5, this.isLocal ? 0.5 : 0)
        this.addChild(this.stackText)

        // 本地玩家筹码金额下方的 "筹码余额" 说明文字
        this.stackLabel = new Text({
            text: '筹码余额',
            style: {
                fontFamily: FONT_BODY,
                fontSize: 8,
                fontWeight: '700',
                fill: C.GREEN,
                letterSpacing: 2,
            },
        })
        this.stackLabel.anchor.set(0.5, 0)
        this.stackLabel.alpha = 0.7
        this.stackLabel.visible = false
        if (this.isLocal) this.addChild(this.stackLabel)
    }

    private buildBetBadge() {
        this.betBadge = new Container()
        this.betBadge.visible = false

        const betBg = new Graphics()  // 徽章背景（绿色药丸 + 金色边框）
        this.betBadge.addChild(betBg)

        this.betIcon = new Text({ text: '🪙', style: {fontSize: 12} })
        this.betIcon.anchor.set(0.5)
        this.betBadge.addChild(this.betIcon)

        this.betText = new Text({
            text: '',
            style: {
                fontFamily: FONT_BODY,
                fontSize: 11,
                fontWeight: '700',
                fill: C.GREEN,
            },
        })
        this.betText.anchor.set(0, 0.5)
        this.betBadge.addChild(this.betText)
        this.addChild(this.betBadge)
    }

    private buildStatusBadge() {
        this.statusBadge = new Container()
        this.statusBadge.visible = false

        this.statusBadgeBg = new Graphics()
        this.statusBadge.addChild(this.statusBadgeBg)

        this.statusBadgeText = new Text({
            text: '',
            style: {
                fontFamily: FONT_BODY,
                fontSize: 10,
                fontWeight: '700',
                fill: C.TEXT_SECONDARY,
            },
        })
        this.statusBadgeText.anchor.set(0.5)
        this.statusBadge.addChild(this.statusBadgeText)
        this.addChild(this.statusBadge)
    }

    private buildCards() {
        this.card0 = new CardSprite(true)  // isHoleCard=true
        this.card1 = new CardSprite(true)
        this.card0.visible = false
        this.card1.visible = false
        this.addChild(this.card0, this.card1)
    }

    /**
     * 从 GameStore 状态更新座位显示。由 TableScene.updateFromState() 每帧调用。
     */
    update(
        seat: SeatSnapshot | null,
        isActive: boolean,
        myHole?: string[],
        posName?: string,
        displayIdx?: number,
    ) {
        this.displayIdx = displayIdx ?? 0

        if (!seat) {
            this.drawEmpty()
            return
        }

        this.updateBaseLayer(seat, isActive)
        this.updatePlayerInfo(seat)
        this.updatePosTag(posName)
        this.updateBetBadge(seat)
        this.updateStatusBadge(seat)
        this.updateHandCards(seat, myHole)
    }

    // ─── 更新分解 ──────────────────────────────────────────────

    private updateBaseLayer(seat: SeatSnapshot, isActive: boolean) {
        // 绘制有玩家的座位背景（头像圆 + 活跃光环/普通描边）
        this.drawFilled(seat.user_id, isActive, seat.is_bot ?? false, seat.folded)

        const setTex = (tex: Texture) => {
            if (this.avatarImg && !this.avatarImg.destroyed) {
                this.avatarImg.texture = tex
                this.avatarImg.width = this.avatarR * 2
                this.avatarImg.height = this.avatarR * 2
            }
        }

        if (seat.avatar) {
            // 有自定义头像：远程加载，失败则降级为默认头像
            Assets.load(seat.avatar).then(setTex).catch(() => {
                setTex(makeDefaultAvatarTexture(seat.user_id, seat.display_name))
            })
        } else {
            // 无头像：本地直接生成
            setTex(makeDefaultAvatarTexture(seat.user_id, seat.display_name))
        }

        this.avatarImg.alpha = seat.folded ? 0.4 : 1  // 弃牌时半透明
    }

    private updatePlayerInfo(seat: SeatSnapshot) {
        // 玩家昵称（截断到 10 字符）
        this.nameText.text = seat.display_name.slice(0, 10)

        // 筹码金额
        this.stackText.text = `$${seat.stack.toLocaleString()}`

        // 本地玩家需要额外布局玻璃筹码面板
        if (this.isLocal) {
            this.layoutLocalStack()
        }
    }

    private updatePosTag(posName?: string) {
        if (posName) {
            this.posTag.visible = true
            this.posTagText.text = posName
            this.layoutPosTag(posName)
        } else {
            this.posTag.visible = false
        }
    }

    private updateBetBadge(seat: SeatSnapshot) {
        if (seat.bet > 0 && !seat.folded) {
            this.betBadge.visible = true
            this.betText.text = `下注 $${seat.bet.toLocaleString()}`
            this.layoutBetBadge()
        } else {
            this.betBadge.visible = false
        }
    }

    private updateStatusBadge(seat: SeatSnapshot) {
        if (seat.folded) {
            this.statusBadge.visible = true
            this.statusBadgeText.text = '已弃牌'
            this.statusBadgeText.style.fill = C.TEXT_SECONDARY
            this.layoutStatusBadge()
        } else if (seat.all_in) {
            this.statusBadge.visible = true
            this.statusBadgeText.text = 'ALL-IN'
            this.statusBadgeText.style.fill = C.GOLD
            this.layoutStatusBadge()
        } else {
            this.statusBadge.visible = false
        }
    }

    private updateHandCards(seat: SeatSnapshot, myHole?: string[]) {
        // 手牌显示逻辑：本地玩家用 myHole，远程使用 seat.hole（仅 show down 可见）
        const hole = this.isLocal ? myHole : seat.hole
        
        if (hole && hole.length === 2 && !seat.folded) {
            this.card0.setCard(hole[0])
            this.card1.setCard(hole[1])
            this.card0.visible = true
            this.card1.visible = true

            if (this.isLocal) {
                // 本地玩家：手牌在头像右侧扇形展开，微微旋转
                this.card0.position.set(this.avatarR + 24, -50)
                this.card0.rotation = -0.08
                this.card1.position.set(this.avatarR + 60, -50)
                this.card1.rotation = 0.08
            } else {
                // 远程玩家：摊牌时手牌显示在头像上方
                this.card0.position.set(-36, -this.avatarR - 70)
                this.card1.position.set(4, -this.avatarR - 70)
            }
        } else {
            this.card0.visible = false
            this.card1.visible = false
        }
    }

    /** 绘制空座位状态：带有“入座”暗示的样式，清空所有信息元素 */
    private drawEmpty() {
        const R = this.avatarR

        this.avatarBg.clear()
        
        // 1. 底板填充（微亮）
        this.avatarBg.circle(0, 0, R)
        this.avatarBg.fill({color: C.SURFACE_HIGH, alpha: 0.15})
        
        // 2. 淡虚线或纯细线边框以暗示空位
        this.avatarBg.circle(0, 0, R)
        this.avatarBg.stroke({color: C.TEXT_DIM, width: 1.5, alpha: 0.2})

        // 3. 正中心增加一个低调的 "+" 符号
        const cross = this.isLocal ? 16 : 10
        this.avatarBg.moveTo(-cross, 0).lineTo(cross, 0)
                     .stroke({color: C.TEXT_DIM, width: 3, alpha: 0.3, cap: 'round'})
        this.avatarBg.moveTo(0, -cross).lineTo(0, cross)
                     .stroke({color: C.TEXT_DIM, width: 3, alpha: 0.3, cap: 'round'})

        // 移除图像元素
        this.avatarImg.texture = Texture.EMPTY

        this.nameText.text = ''
        this.stackText.text = ''
        this.stackLabel.visible = false
        this.betBadge.visible = false
        this.statusBadge.visible = false
        this.posTag.visible = false
        this.card0.visible = false
        this.card1.visible = false
        if (this.isLocal) {
            this.stackPanel.clear()
            this.nameBadge.clear()
        }
    }

    /**
     * 绘制有玩家的座位背景。
     * 活跃状态（当前行动者）显示金色光环；弃牌状态整体变暗。
     */
    private drawFilled(_userId: string, isActive: boolean, _isBot: boolean, isFolded: boolean) {
        const R = this.avatarR

        this.avatarBg.clear()

        // 头像圆形背景 — 深色玻璃面板
        this.avatarBg.circle(0, 0, R)
        this.avatarBg.fill({color: C.GLASS, alpha: 0.8})

        if (isActive) {
            // 当前行动者：外扩金色光环（R+3 半径）
            this.avatarBg.circle(0, 0, R + 3)
            this.avatarBg.stroke({color: C.GOLD, width: 3, alpha: 0.7})
        } else {
            // 非行动者：微弱幽灵边框（弃牌更暗）
            this.avatarBg.circle(0, 0, R)
            this.avatarBg.stroke({color: isFolded ? 0x333333 : 0xffffff, width: 2, alpha: isFolded ? 0.08 : 0.05})
        }

        // 弃牌时整个座位半透明
        this.alpha = isFolded ? 0.5 : 1

        if (this.isLocal) {
            // 本地玩家：头像底部的金色药丸形名称徽章
            const nameY = R + 2
            this.nameBadge.clear()
            this.nameBadge.roundRect(-36, nameY - 10, 72, 20, 8)
            this.nameBadge.fill({color: C.GOLD})
            this.nameText.position.set(0, nameY)
            this.nameText.style.fill = C.VOID

            this.layoutLocalStack()
        } else {
            // 远程玩家：名称在头像正下方
            this.nameText.position.set(0, R + 10)
            this.nameText.style.fill = isFolded ? C.TEXT_DIM : C.TEXT_PRIMARY

            // 筹码金额在名称下方
            this.stackText.position.set(0, R + 26)
            this.stackText.alpha = isFolded ? 0.4 : 0.8
        }
    }

    /** 布局本地玩家的筹码面板：头像下方的玻璃药丸底板 + 金额 + "筹码余额" 标签 */
    private layoutLocalStack() {
        const R = this.avatarR
        const panelY = R + 24
        const panelW = 150
        const panelH = 54

        this.stackPanel.clear()
        this.stackPanel.roundRect(-panelW / 2, panelY, panelW, panelH, 12)
        this.stackPanel.fill({color: C.GLASS, alpha: 0.95})
        this.stackPanel.roundRect(-panelW / 2, panelY, panelW, panelH, 12)
        this.stackPanel.stroke({color: C.GOLD, width: 1, alpha: 0.3})

        this.stackText.position.set(0, panelY + panelH / 2 - 4)

        // 金额下方的 "筹码余额" 标签
        this.stackLabel.visible = true
        this.stackLabel.position.set(0, panelY + panelH / 2 + 12)
    }

    /** 布局位置标签（BTN/SB/BB 等）— 放置在头像正上方，动态宽度的玻璃药丸背景 */
    private layoutPosTag(name: string) {
        const R = this.avatarR
        // 头像上方
        this.posTag.position.set(0, -R - 16)

        const bg = this.posTag.getChildAt(0) as Graphics
        bg.clear()

        const tw = Math.max(28, name.length * 9 + 14)
        bg.roundRect(-tw / 2, -10, tw, 20, 10)
        bg.fill({color: C.GLASS, alpha: 0.95})
        bg.roundRect(-tw / 2, -10, tw, 20, 10)
        bg.stroke({color: C.GOLD, width: 1, alpha: 0.6})
    }

    /**
     * 布局下注徽章 — 通过向量计算从座位朝牌桌中心方向放置。
     * 计算座位到桌心的单位向量，沿该方向偏移 R+40 距离放置徽章。
     */
    private layoutBetBadge() {
        const R = this.avatarR
        const pos = SEAT_POSITIONS[this.displayIdx]
        // 计算座位 → 桌心的单位方向向量
        const dx = TABLE_CX - pos.x
        const dy = TABLE_CY - pos.y
        const len = Math.sqrt(dx * dx + dy * dy) || 1
        const dirX = dx / len
        const dirY = dy / len

        // 沿方向向量偏移放置徽章
        const dist = R + 40
        this.betBadge.position.set(dirX * dist, dirY * dist)

        // 动态宽度药丸背景：[🪙图标] [下注金额文字]
        const textStr = this.betText.text
        const badgeW = Math.max(90, textStr.length * 7 + 30)

        const bg = this.betBadge.getChildAt(0) as Graphics
        bg.clear()
        bg.roundRect(-badgeW / 2, -13, badgeW, 26, 8)
        bg.fill({color: C.GREEN_BET, alpha: 0.95})
        bg.roundRect(-badgeW / 2, -13, badgeW, 26, 8)
        bg.stroke({color: C.GOLD, width: 1, alpha: 0.2})

        this.betIcon.position.set(-badgeW / 2 + 14, 0)
        this.betText.position.set(-badgeW / 2 + 26, 0)
    }

    /**
     * 布局状态徽章（"已弃牌" / "ALL-IN"）。
     * 本地玩家放在头像中心；远程玩家朝桌心方向偏移放置。
     */
    private layoutStatusBadge() {
        const R = this.avatarR
        const pos = SEAT_POSITIONS[this.displayIdx]

        // 计算朝桌心方向
        const dx = TABLE_CX - pos.x
        const dy = TABLE_CY - pos.y
        const len = Math.sqrt(dx * dx + dy * dy) || 1

        if (this.isLocal) {
            // 本地玩家：徽章覆盖在头像中心
            this.statusBadge.position.set(0, 0)
        } else {
            // 远程玩家：朝桌心偏移（比下注徽章更近）
            this.statusBadge.position.set((dx / len) * (R + 20), (dy / len) * (R + 10))
        }

        const text = this.statusBadgeText.text
        const badgeW = Math.max(50, text.length * 8 + 16)

        this.statusBadgeBg.clear()
        this.statusBadgeBg.roundRect(-badgeW / 2, -11, badgeW, 22, 8)
        this.statusBadgeBg.fill({color: C.SURFACE_HIGH, alpha: 0.9})
        this.statusBadgeBg.roundRect(-badgeW / 2, -11, badgeW, 22, 8)
        this.statusBadgeBg.stroke({color: 0xffffff, width: 0.5, alpha: 0.05})
    }
}

// ── 默认头像生成 ─────────────────────────────────────────────────────────────

/** 缓存：相同 userId 只生成一次 */
const _defaultAvatarCache = new Map<string, Texture>()

/**
 * 用 Canvas 本地生成默认头像纹理。
 * 背景色由 userId 哈希决定（相同玩家颜色稳定），中央显示昵称首字。
 * 结果被缓存，多次调用同一 userId 不会重复绘制。
 */
function makeDefaultAvatarTexture(userId: string, displayName: string): Texture {
    const cached = _defaultAvatarCache.get(userId)
    if (cached) return cached

    const SIZE = 128
    const canvas = document.createElement('canvas')
    canvas.width = SIZE
    canvas.height = SIZE
    const ctx = canvas.getContext('2d')!

    // 背景色：基于 userId 的稳定哈希值
    const hue = Math.abs(userId.split('').reduce((acc, c) => acc * 31 + c.charCodeAt(0), 0)) % 360
    const grad = ctx.createRadialGradient(SIZE / 2, SIZE / 2, 0, SIZE / 2, SIZE / 2, SIZE / 2)
    grad.addColorStop(0, `hsl(${hue}, 55%, 40%)`)
    grad.addColorStop(1, `hsl(${hue}, 45%, 22%)`)
    ctx.fillStyle = grad
    ctx.beginPath()
    ctx.arc(SIZE / 2, SIZE / 2, SIZE / 2, 0, Math.PI * 2)
    ctx.fill()

    // 首字母
    const letter = (displayName || '?')[0].toUpperCase()
    ctx.fillStyle = 'rgba(255,255,255,0.90)'
    ctx.font = `bold ${SIZE * 0.42}px sans-serif`
    ctx.textAlign = 'center'
    ctx.textBaseline = 'middle'
    ctx.fillText(letter, SIZE / 2, SIZE / 2 + 2)

    const tex = Texture.from(canvas)
    _defaultAvatarCache.set(userId, tex)
    return tex
}
