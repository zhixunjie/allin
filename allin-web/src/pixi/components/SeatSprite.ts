import {Assets, Container, Graphics, Sprite, Text, Texture} from 'pixi.js'
import type {SeatSnapshot} from '../../store/game'
import {
    AVATAR_R_LOCAL,
    AVATAR_R_REMOTE,
    C,
    FONT_BODY,
    FONT_HEADLINE,
    HOLE_CARD_W,
    SEAT_POSITIONS,
    TABLE_CX,
    TABLE_CY,
} from '../assets'
import {CardSprite} from './CardSprite'
import {CasinoChip, chipStyleForAmount} from './CasinoChip'

/**
 * 座位精灵 — 表示牌桌上一个座位的完整视觉元素。
 *
 * 视觉结构（从底到顶）：
 *   avatarBg       — 头像圆形背景（玻璃面板 + 描边/活跃光环）
 *   avatarIcon     — 头像 emoji（玩家基于 userId 哈希分配，bot 固定 🤖）
 *   posTag         — 位置标签（BTN/SB/BB/UTG 等），悬浮于头像左上角（225° 方向）
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
    private allInGlow!: Graphics     // ALL-IN 脉冲光晕层（独立图层，仅 tick 时更新）
    private avatarImg!: Sprite       // 头像图片
    private nameText!: Text          // 玩家昵称
    private stackText!: Text         // 筹码金额
    private stackLabel!: Text        // 本地玩家的 "筹码余额" 副标签
    private betBadge!: Container     // 下注徽章容器（背景 + 筹码图标 + 文字）
    private betText!: Text           // 下注金额文字
    private betChip: CasinoChip | null = null  // 下注金额旁的小筹码图标
    private lastBetAmount = -1                  // 上次渲染的金额（避免重建）
    private statusBadge!: Container  // 状态徽章容器（"已弃牌" / "ALL-IN"）
    private statusBadgeBg!: Graphics // 状态徽章背景
    private statusBadgeText!: Text   // 状态徽章文字
    private posTag!: Container       // 位置标签容器（BTN/SB/BB 等）
    private posTagText!: Text        // 位置标签文字
    private card0!: CardSprite       // 手牌 1
    private card1!: CardSprite       // 手牌 2
    private nameBadge!: Graphics     // 本地玩家金色名称底板
    private stackPanel!: Graphics    // 本地玩家玻璃筹码面板

    private isLocal: boolean        // 是否为本地玩家（底部大头像），false 表示远端玩家
    private avatarR: number         // 头像半径
    private displayIdx = 0          // 显示索引（0=本地，用于计算朝向）
    private isAllIn = false       // 当前是否处于 ALL-IN 状态（驱动脉冲动画）
    private allInTime = 0           // 累计时间（ms），用于 sin 脉冲计算

    /**
     * @param isLocal 是否为本地玩家座位（displayIdx=0，使用大头像 + 特殊布局）
     */
    constructor(isLocal = false) {
        super()
        this.isLocal = isLocal
        this.avatarR = isLocal ? AVATAR_R_LOCAL : AVATAR_R_REMOTE

        this.buildAvatar()        // 1. 基础头像构建（底板光环 + 裁切遮罩 + 实际图象）
        this.buildPosTag()        // 2. 位置身份标签（BTN/SB/BB 悬浮位于正上方）
        this.buildNameText()      // 3. 昵称文本区（本地大牌采用金底黑字，远程直接渲染文本）
        this.buildStackElements() // 4. 自身筹码额（本地附带宽体玻璃质感托盘底板）
        this.buildBetBadge()      // 5. 本街有效下注额展现徽章（绿底红心金币标识）
        this.buildStatusBadge()   // 6. 特殊极化行为标识徽章（全押爆点，或弃牌高斯灰）
        this.buildCards()         // 7. 座位的两张手牌物理容器（默认遮盖底牌）

        // 初始状态：绘制空座位
        this.drawEmpty()
    }

    // ─── 构造分解 ──────────────────────────────────────────────

    /** 初始化玩家头像层：含有一层白色裁切遮罩和用于装载网络图或本地后备图的 Sprite */
    private buildAvatar() {
        this.avatarBg = new Graphics()
        this.addChild(this.avatarBg)

        // ALL-IN 脉冲光晕层：位于 avatarBg 之上、avatarImg 之下，仅 tick 时更新
        this.allInGlow = new Graphics()
        this.addChild(this.allInGlow)

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

    /**
     * 初始化桌位身份标签（BTN / SB / BB / UTG 等）。
     *
     * 结构：posTag（Container）
     *   ├─ posTagBg（Graphics）  — 圆角药丸背景，颜色在 layoutPosTag() 中每帧动态绘制
     *   └─ posTagText（Text）    — 标签文字，锚点居中，内容在 updatePosTag() 中赋值
     *
     * 默认隐藏（visible = false），仅当该座位被分配到身份时才显示。
     * 位置由 layoutPosTag() 计算，固定悬浮在头像左上角（225° 方向）。
     */
    private buildPosTag() {
        this.posTag = new Container()
        this.posTag.visible = false

        // 背景：此处只创建节点，实际形状/颜色在 layoutPosTag() 里动态 clear+重绘
        const posTagBg = new Graphics()
        this.posTag.addChild(posTagBg)

        // 文字：银白色标题字体，锚点 (0.5, 0.5) 使文字在背景中居中
        this.posTagText = new Text({
            text: '',
            style: {
                fontFamily: FONT_HEADLINE,
                fontSize: 10,
                fontWeight: '800',
                fill: C.GREEN,
                letterSpacing: 1,
            },
        })
        this.posTagText.anchor.set(0.5)
        this.posTag.addChild(this.posTagText)

        // 挂载到座位根容器
        this.addChild(this.posTag)
    }

    /** 初始化玩家昵称展示：本地玩家配有金色底标，远程玩家为底部极简文字 */
    private buildNameText() {
        this.nameBadge = new Graphics()

        if (this.isLocal) {
            // 本地玩家：头像底部的金色药丸形名称徽章
            this.addChild(this.nameBadge)
            this.nameText = new Text({
                text: '',
                style: {
                    fontFamily: FONT_BODY,
                    fontSize: 10,
                    fontWeight: '900',
                    fill: C.VOID,
                    letterSpacing: 3,
                },
            })
            this.nameText.anchor.set(0.5)
        } else {
            // 远程玩家：头像下方普通文字
            this.nameText = new Text({
                text: '',
                style: {
                    fontFamily: FONT_BODY,
                    fontSize: 11,
                    fontWeight: '700',
                    fill: C.TEXT_PRIMARY,
                    letterSpacing: 0.5,
                },
            })
            this.nameText.anchor.set(0.5, 0)
        }
        this.addChild(this.nameText)
    }

    /** 初始化筹码面板组件：包括大基数的总筹码额和本地玩家独占的"玻璃托盘" */
    private buildStackElements() {
        // 本地玩家有独立的玻璃面板底板
        this.stackPanel = new Graphics()
        if (this.isLocal) this.addChild(this.stackPanel)

        this.stackText = new Text({
            text: '',
            style: {
                fontFamily: FONT_HEADLINE,
                fontSize: this.isLocal ? 16 : 11,   // 本地略大，远程小号
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
                fontSize: 10,
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

    /** 初始化下注徽章：🪙 + 金额，朝桌心方向放置，模拟筹码推入桌面效果 */
    private buildBetBadge() {
        this.betBadge = new Container()
        this.betBadge.visible = false

        this.betText = new Text({
            text: '',
            style: {
                fontFamily: FONT_HEADLINE,
                fontSize: 11,
                fontWeight: '700',
                fill: C.GOLD,
            },
        })
        this.betText.anchor.set(0, 0.5)
        this.betBadge.addChild(new Graphics(), this.betText)
        this.addChild(this.betBadge)
    }

    /** 初始化状态徽章：用于高亮显示特殊行为导致的状态突变（如 ALL-IN、已弃牌） */
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

    /** 初始化当前座位的左/右手牌对象（默认隐藏待发牌） */
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

        this.updateBaseLayer(seat, isActive)   // 头像框光晕 / 弃牌灰度
        this.updatePlayerInfo(seat)             // 昵称 + 筹码余额
        this.updatePosTag(posName, seat.all_in && !seat.folded)  // ALL-IN 时标签变红
        this.updateBetBadge(seat)               // 本街下注徽章
        this.updateStatusBadge(seat)            // 已弃牌 / ALL-IN 状态徽章
        this.updateHandCards(seat, myHole)      // 手牌（本地明牌，远程背面）
    }

    // ─── 更新分解 ──────────────────────────────────────────────

    /**
     * 核心渲染底层容器的流态：
     * 包括轮光（Active/倒计时）、头像图像和玻璃衬底的弃牌透明衰减，
     * 以及决定是否去网络爬取自定义 URL 图片或直接启用本地渲染生成后备图。
     */
    private updateBaseLayer(seat: SeatSnapshot, isActive: boolean) {
        // 绘制有玩家的座位背景（头像圆 + 活跃光环/普通描边）
        this.drawFilled(seat.user_id, isActive, seat.is_bot ?? false, seat.folded, seat.all_in ?? false)

        const setTex = (tex: Texture) => {
            if (this.avatarImg && !this.avatarImg.destroyed) {
                this.avatarImg.texture = tex
                this.avatarImg.width = this.avatarR * 2
                this.avatarImg.height = this.avatarR * 2
            }
        }

        const fallback = () => setTex(makeDefaultAvatarTexture(seat.user_id, seat.display_name))
        if (seat.avatar) {
            Assets.load(seat.avatar).then(setTex).catch(fallback)
        } else {
            loadDiceBearTexture(seat.user_id).then(setTex).catch(fallback)
        }

        this.avatarImg.alpha = seat.folded ? 0.4 : 1  // 弃牌时半透明
    }

    /** 更新基本数据字段：截断昵称以及筹码数显 */
    private updatePlayerInfo(seat: SeatSnapshot) {
        // 本地玩家标签固定为 "YOU"，远端玩家显示昵称
        this.nameText.text = this.isLocal ? 'YOU' : seat.display_name.slice(0, 10)

        // 筹码金额
        this.stackText.text = `$${seat.stack.toLocaleString()}`

        // 本地玩家筹码位置已在 drawAvatar 中设置，无需重新布局
    }

    /** 更新座次头衔（BTN/SB等）：如无身份则隐形；ALL-IN 时标签变红 */
    private updatePosTag(posName?: string, isAllIn = false) {
        if (posName) {
            this.posTag.visible = true
            this.posTagText.text = posName
            this.layoutPosTag(posName, isAllIn)
        } else {
            this.posTag.visible = false
        }
    }

    /** 重绘本街有效下注额的动态呈现，如果已盖牌或未下注则隐现 */
    private updateBetBadge(seat: SeatSnapshot) {
        if (seat.bet > 0 && !seat.folded) {
            this.betBadge.visible = true
            this.betText.text = `$${seat.bet.toLocaleString()}`
            // 金额变化时重建筹码图标
            if (seat.bet !== this.lastBetAmount) {
                this.lastBetAmount = seat.bet
                if (this.betChip) this.betBadge.removeChild(this.betChip)
                this.betChip = new CasinoChip(7, {...chipStyleForAmount(seat.bet), label: ''})
                this.betBadge.addChild(this.betChip)
            }
            this.layoutBetBadge()
        } else {
            this.betBadge.visible = false
        }
    }

    /** 切变行为标记：ALL-IN 红底白字居中，已弃牌灰色作废章居中 */
    private updateStatusBadge(seat: SeatSnapshot) {
        if (seat.folded) {
            this.statusBadge.visible = true
            this.statusBadgeText.text = '已弃牌'
            this.statusBadgeText.style.fill = 0xaaaaaa
            this.layoutStatusBadge(false)
        } else if (seat.all_in) {
            this.statusBadge.visible = false
        } else {
            this.statusBadge.visible = false
        }
    }

    /** 根据客户端身份和 Showdown 公开协议判断并渲染当事两张底牌的倾角分列位 */
    private updateHandCards(seat: SeatSnapshot, myHole?: string[]) {
        // 手牌显示逻辑：本地玩家用 myHole，远程使用 seat.hole（仅 show down 可见）
        const hole = this.isLocal ? myHole : seat.hole

        if (hole && hole.length === 2 && !seat.folded) {
            // '?' 占位符 → 背面（已发牌但不可见）；否则显示正面牌值
            hole[0] === '?' ? this.card0.setFaceDown() : this.card0.setCard(hole[0])
            hole[1] === '?' ? this.card1.setFaceDown() : this.card1.setCard(hole[1])
            this.card0.visible = true
            this.card1.visible = true

            // 左侧玩家手牌放左边，右边其余玩家放右边
            // 左侧：卡牌锚点在左上角，需额外偏移 HOLE_CARD_W 使右边缘贴住头像
            const R = this.avatarR
            const onLeft = this.displayIdx === 6 || this.displayIdx === 7
            if (onLeft) {
                this.card0.position.set(-(R + 10 + HOLE_CARD_W), -34)
                this.card0.rotation = 0.08
                this.card1.position.set(-(R + 40 + HOLE_CARD_W), -34)
                this.card1.rotation = -0.08
            } else {
                this.card0.position.set(R + 10, -34)
                this.card0.rotation = -0.08
                this.card1.position.set(R + 40, -34)
                this.card1.rotation = 0.08
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
     * 绘制座位头像框，本地玩家与远端玩家使用不同的框架风格。
     *
     * 本地玩家（YOU）：
     *   普通  — 内金边 + 外环境散晕，彰显主角地位
     *   行动中 — 多层金色光晕向外扩散，营造紧迫感
     *
     * 远端玩家：
     *   普通  — 银蓝双环
     *   行动中 — 金色多层光晕
     *
     * 弃牌：整体暗淡，去掉所有装饰
     */
    private drawFilled(_userId: string, isActive: boolean, _isBot: boolean, isFolded: boolean, isAllIn: boolean) {
        const R = this.avatarR
        this.avatarBg.clear()

        // 更新 ALL-IN 动画状态（非 ALL-IN 时清空脉冲层，重置计时）
        this.isAllIn = isAllIn && !isFolded
        if (!this.isAllIn) {
            this.allInGlow.clear()
            this.allInTime = 0
        }

        // ── 底层玻璃填充 ──────────────────────────────────────
        this.avatarBg.circle(0, 0, R)
        this.avatarBg.fill({color: C.GLASS, alpha: isFolded ? 0.3 : 0.88})

        if (isFolded) {
            // 弃牌：灰色细环 + 整体大幅压暗
            this.avatarBg.circle(0, 0, R).stroke({color: 0x555555, width: 1.5, alpha: 0.4})
            this.alpha = 0.4
        } else if (isAllIn) {
            // ALL-IN 静态底框（动画部分由 tick 驱动 allInGlow 覆盖）
            this.drawAllInFrame(R)
            this.alpha = 1
        } else if (this.isLocal) {
            this.drawLocalFrame(R, isActive)
            this.alpha = 1
        } else {
            this.drawRemoteFrame(R)
            this.alpha = 1
        }


        if (this.isLocal) {
            // 本地玩家：头像底部挂 "YOU" 金色药丸标签（昵称显示在筹码面板里）
            const nameY = R + 18  // 避开 TimerArc 环底边（R+4+stroke/2 ≈ R+7.5）
            this.nameBadge.clear()
            this.nameBadge.roundRect(-22, nameY - 10, 44, 20, 10)
            this.nameBadge.fill({color: C.GOLD})
            this.nameBadge.roundRect(-22, nameY - 10, 44, 20, 10)
            this.nameBadge.stroke({color: C.GOLD_LIGHT, width: 1, alpha: 0.5})
            // 把 nameText 临时借用来显示 "YOU"（固定文字，不展示昵称）
            this.nameText.text = 'YOU'
            this.nameText.position.set(0, nameY)
            this.nameText.style.fill = C.VOID

            // 筹码直接放在 YOU 标签下方，无边框面板
            this.stackPanel.clear()
            this.stackText.position.set(0, nameY + 20)
            this.stackText.alpha = 1
        } else {
            // 远程玩家：名称在头像正下方
            this.nameText.position.set(0, R + 10)
            this.nameText.style.fill = isFolded ? C.TEXT_DIM : C.TEXT_PRIMARY

            // 筹码金额在名称下方
            this.stackText.position.set(0, R + 26)
            this.stackText.alpha = isFolded ? 0.4 : 0.8
        }
    }

    /**
     * 本地玩家（YOU）头像框：内金边 + 外环境散晕。
     * 普通状态体现主角地位；行动时多层光晕向外扩散。
     */
    private drawLocalFrame(R: number, isActive: boolean) {
        if (isActive) {
            // 行动中：多层金色向外大范围扩散光晕
            this.avatarBg.circle(0, 0, R + 40).stroke({color: C.GOLD, width: 1, alpha: 0.03})
            this.avatarBg.circle(0, 0, R + 30).stroke({color: C.GOLD, width: 1.5, alpha: 0.08})
            this.avatarBg.circle(0, 0, R + 20).stroke({color: C.GOLD, width: 2, alpha: 0.16})
            this.avatarBg.circle(0, 0, R + 10).stroke({color: C.GOLD, width: 3, alpha: 0.32})
            this.avatarBg.circle(0, 0, R + 3).stroke({color: C.GOLD, width: 4, alpha: 0.65})
            this.avatarBg.circle(0, 0, R + 1).stroke({color: C.GOLD, width: 5, alpha: 0.95})
            this.avatarBg.circle(0, 0, R - 3).stroke({color: C.GOLD, width: 1.5, alpha: 0.40})
        } else {
            // 普通状态：同款多层扩散，alpha 整体调低，主角常驻光感
            this.avatarBg.circle(0, 0, R + 40).stroke({color: C.GOLD, width: 1, alpha: 0.015})
            this.avatarBg.circle(0, 0, R + 30).stroke({color: C.GOLD, width: 1.5, alpha: 0.04})
            this.avatarBg.circle(0, 0, R + 20).stroke({color: C.GOLD, width: 2, alpha: 0.09})
            this.avatarBg.circle(0, 0, R + 10).stroke({color: C.GOLD, width: 3, alpha: 0.18})
            this.avatarBg.circle(0, 0, R + 3).stroke({color: C.GOLD, width: 4, alpha: 0.38})
            this.avatarBg.circle(0, 0, R + 1).stroke({color: C.GOLD, width: 5, alpha: 0.65})
            this.avatarBg.circle(0, 0, R - 3).stroke({color: C.GOLD, width: 1.5, alpha: 0.22})
        }
    }

    /**
     * 远端玩家头像框：银蓝双环
     * 行动时替换为金色多层光晕
     */
    private drawRemoteFrame(R: number) {
        // if (isActive) {
        //     // 行动中：金色多层大范围光晕
        //     this.avatarBg.circle(0, 0, R + 32).stroke({color: C.GOLD, width: 1, alpha: 0.04})
        //     this.avatarBg.circle(0, 0, R + 22).stroke({color: C.GOLD, width: 1.5, alpha: 0.10})
        //     this.avatarBg.circle(0, 0, R + 13).stroke({color: C.GOLD, width: 2, alpha: 0.22})
        //     this.avatarBg.circle(0, 0, R + 5).stroke({color: C.GOLD, width: 3, alpha: 0.45})
        //     this.avatarBg.circle(0, 0, R + 1).stroke({color: C.GOLD, width: 4.5, alpha: 0.90})
        // } else {
        // 普通状态：多层银白向外大范围扩散光晕
        this.avatarBg.circle(0, 0, R + 32).stroke({color: C.TEXT_PRIMARY, width: 1, alpha: 0.015})
        this.avatarBg.circle(0, 0, R + 22).stroke({color: C.TEXT_PRIMARY, width: 1.5, alpha: 0.05})
        this.avatarBg.circle(0, 0, R + 13).stroke({color: C.TEXT_PRIMARY, width: 2, alpha: 0.11})
        this.avatarBg.circle(0, 0, R + 5).stroke({color: C.TEXT_PRIMARY, width: 3, alpha: 0.24})
        this.avatarBg.circle(0, 0, R + 1).stroke({color: C.TEXT_PRIMARY, width: 4.5, alpha: 0.52})
        this.avatarBg.circle(0, 0, R - 3).stroke({color: C.TEXT_PRIMARY, width: 1, alpha: 0.16})
        // }
    }


    /** ALL-IN 静态底框：红色多层扩散光晕，由 drawFilled 绘制一次 */
    private drawAllInFrame(R: number) {
        this.avatarBg.circle(0, 0, R + 30).stroke({color: C.ERROR, width: 1, alpha: 0.04})
        this.avatarBg.circle(0, 0, R + 20).stroke({color: C.ERROR, width: 1.5, alpha: 0.10})
        this.avatarBg.circle(0, 0, R + 12).stroke({color: C.ERROR, width: 2, alpha: 0.22})
        this.avatarBg.circle(0, 0, R + 5).stroke({color: C.ERROR, width: 3, alpha: 0.48})
        this.avatarBg.circle(0, 0, R + 1).stroke({color: C.ERROR, width: 5, alpha: 0.90})
        this.avatarBg.circle(0, 0, R - 3).stroke({color: C.ERROR, width: 1.5, alpha: 0.35})
    }

    /**
     * 每帧驱动动画，由 TableScene.onTick 调用。
     * 目前仅处理 ALL-IN 脉冲光晕：以 1.5s 为周期在红金之间呼吸，
     * 外层光晕半径和 alpha 随 sin 波浮动，制造"能量燃烧"感。
     */
    tick(deltaMS: number) {
        if (!this.isAllIn) return

        this.allInTime += deltaMS
        const R = this.avatarR

        // sin 波：0→1→0，周期 1500ms
        const pulse = (Math.sin((this.allInTime / 1500) * Math.PI * 2 - Math.PI / 2) + 1) / 2

        // 浅粉红(0xff8a80) 与 金(C.GOLD) 之间线性插值，随 pulse 平滑过渡
        const r1 = 0xff, g1 = 0x8a, b1 = 0x80  // 浅粉红
        const r2 = (C.GOLD >> 16) & 0xff, g2 = (C.GOLD >> 8) & 0xff, b2 = C.GOLD & 0xff
        const r = Math.round(r1 + (r2 - r1) * pulse)
        const g = Math.round(g1 + (g2 - g1) * pulse)
        const b = Math.round(b1 + (b2 - b1) * pulse)
        const color = (r << 16) | (g << 8) | b

        // 外晕半径随脉冲扩张 (R+28 ~ R+38)
        const outerR = R + 28 + pulse * 10

        this.allInGlow.clear()
        this.allInGlow.circle(0, 0, outerR).stroke({color, width: 1, alpha: 0.04 + pulse * 0.06})
        this.allInGlow.circle(0, 0, R + 18).stroke({color, width: 1.5, alpha: 0.08 + pulse * 0.12})
        this.allInGlow.circle(0, 0, R + 8).stroke({color, width: 2.5, alpha: 0.18 + pulse * 0.22})
        this.allInGlow.circle(0, 0, R + 2).stroke({color, width: 5, alpha: 0.55 + pulse * 0.35})
    }

    /** 布局位置标签（BTN/SB/BB 等）— 悬浮于头像正上方；ALL-IN 时边框变红，文字保持不变 */
    private layoutPosTag(name: string, isAllIn = false) {
        const R = this.avatarR
        this.posTag.position.set(0, -R - 16)

        const tagColor = isAllIn ? C.ERROR : C.GREEN
        this.posTagText.style.fill = isAllIn ? C.ERROR : C.GREEN

        const bg = this.posTag.getChildAt(0) as Graphics
        bg.clear()

        const tw = Math.max(28, name.length * 9 + 14)
        bg.roundRect(-tw / 2, -10, tw, 20, 10)
        bg.fill({color: C.GLASS, alpha: 0.95})
        bg.roundRect(-tw / 2, -10, tw, 20, 10)
        bg.stroke({color: tagColor, width: 1, alpha: isAllIn ? 0.8 : 0.5})
    }

    /**
     * 布局下注徽章 — 通过向量计算从座位朝牌桌中心方向放置。
     * 计算座位到桌心的单位向量，由于头像目前大幅度压入牌桌，我们缩短偏移量至 R+25 避免下注徽章深陷牌桌中央。
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

        // 朝桌心方向偏移，模拟玩家把筹码推到桌面的效果
        const dist = R + 50
        this.betBadge.position.set(dirX * dist, dirY * dist)

        // 宽度紧贴内容：筹码直径(14) + 间距(4) + 文字宽度 + 左右内边距(8+6)
        const badgeW = 14 + 4 + this.betText.text.length * 7 + 14
        const bg = this.betBadge.getChildAt(0) as Graphics
        bg.clear()
        bg.roundRect(-badgeW / 2, -11, badgeW, 22, 11)
        bg.fill({color: C.GLASS, alpha: 0.85})
        bg.stroke({color: C.GOLD, width: 1, alpha: 0.35})

        // 筹码居左紧贴内边距，文字紧随其右
        if (this.betChip) this.betChip.position.set(-badgeW / 2 + 11, 0)
        this.betText.position.set(-badgeW / 2 + 26, 0)
    }

    /**
     * 布局状态徽章 — 所有玩家统一居中覆盖在头像上，形成"盖章"效果。
     * @param isAllIn true = ALL-IN 红底；false = 已弃牌灰底
     */
    private layoutStatusBadge(isAllIn: boolean) {
        // 统一居中，不再朝桌心偏移
        this.statusBadge.position.set(0, 0)

        const text = this.statusBadgeText.text
        const badgeW = Math.max(60, text.length * 9 + 24)

        this.statusBadgeBg.clear()
        if (isAllIn) {
            // ALL-IN：红色实心药丸，亮边描框
            this.statusBadgeBg.roundRect(-badgeW / 2, -13, badgeW, 26, 13)
            this.statusBadgeBg.fill({color: C.ERROR, alpha: 0.90})
            this.statusBadgeBg.roundRect(-badgeW / 2, -13, badgeW, 26, 13)
            this.statusBadgeBg.stroke({color: 0xff9999, width: 1.5, alpha: 0.65})
        } else {
            // 已弃牌：深灰半透明，仿"作废章"
            this.statusBadgeBg.roundRect(-badgeW / 2, -13, badgeW, 26, 13)
            this.statusBadgeBg.fill({color: 0x1a1a1a, alpha: 0.80})
            this.statusBadgeBg.roundRect(-badgeW / 2, -13, badgeW, 26, 13)
            this.statusBadgeBg.stroke({color: 0x888888, width: 1.5, alpha: 0.45})
        }
    }
}

// ── 默认头像生成 ─────────────────────────────────────────────────────────────

// ── DiceBear 卡通头像加载 ──────────────────────────────────────────────────────

/** 缓存：相同 userId 只请求一次 */
const _diceBearCache = new Map<string, Promise<Texture>>()

/**
 * 从 DiceBear API 加载 fun-emoji 风格 SVG，渲染到离屏 Canvas 后转为 PixiJS Texture。
 * PixiJS 不支持直接渲染 SVG，需要中间经过 Canvas 步骤。
 * 同一 userId 的请求会被 Promise 缓存，不会重复发起网络请求。
 */
function loadDiceBearTexture(userId: string): Promise<Texture> {
    const cached = _diceBearCache.get(userId)
    if (cached) return cached

    const url = `https://api.dicebear.com/9.x/toon-head/svg?seed=${encodeURIComponent(userId)}`
    const promise = fetch(url)
        .then(r => r.text())
        .then(svgText => new Promise<Texture>((resolve, reject) => {
            const SIZE = 128
            const canvas = document.createElement('canvas')
            canvas.width = SIZE
            canvas.height = SIZE
            const ctx = canvas.getContext('2d')!
            const img = new Image()
            img.onload = () => {
                ctx.drawImage(img, 0, 0, SIZE, SIZE)
                resolve(Texture.from(canvas))
            }
            img.onerror = reject
            img.src = 'data:image/svg+xml;charset=utf-8,' + encodeURIComponent(svgText)
        }))

    _diceBearCache.set(userId, promise)
    return promise
}

/** 缓存：相同 userId 只生成一次 */
const _defaultAvatarCache = new Map<string, Texture>()

/**
 * 使用纯 Canvas API 本地降级生成备选头像材质：
 * 背景色由其 userId 进行 Hash 并映射至一套稳定的色相光域中，中心附带昵称的单字符简显。
 * 生成出来的图像结果会永久全局缓存于 _defaultAvatarCache 中，多席同位均无性消耗。
 *
 * @param userId 确定性散列凭证
 * @param displayName 获取首字母印花来源
 */
function makeDefaultAvatarTexture(userId: string, displayName: string): Texture {
    const cached = _defaultAvatarCache.get(userId)
    if (cached) return cached

    const SIZE = 128
    const canvas = document.createElement('canvas')
    canvas.width = SIZE
    canvas.height = SIZE
    const ctx = canvas.getContext('2d')!

    // 精选色板：避免 HSL 全域随机产生的泥黄/黄绿等丑色
    const PALETTE: [string, string][] = [
        ['#1e4d7a', '#0d2440'],  // 深海蓝
        ['#1a5c45', '#0d2e22'],  // 翡翠绿
        ['#5c1a4a', '#2e0d24'],  // 深玫红
        ['#4a1e6e', '#250d38'],  // 暗紫
        ['#1e5c5c', '#0d2e2e'],  // 青碧
        ['#6e3a1e', '#381d0d'],  // 琥珀棕
        ['#1e3d6e', '#0d1e38'],  // 宝石蓝
        ['#5c3a1a', '#2e1d0d'],  // 赤铜
    ]
    const idx = Math.abs(userId.split('').reduce((acc, c) => acc * 31 + c.charCodeAt(0), 0)) % PALETTE.length
    const [light, dark] = PALETTE[idx]

    const grad = ctx.createRadialGradient(SIZE / 2, SIZE / 2, 0, SIZE / 2, SIZE / 2, SIZE / 2)
    grad.addColorStop(0, light)
    grad.addColorStop(1, dark)
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
