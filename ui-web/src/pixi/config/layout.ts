// ── 画布 ─────────────────────────────────────────────
/** 画布总宽度（16:9，适配 PC / macOS 浏览器） */
export const CANVAS_W = 1600
/** 画布总高度 */
export const CANVAS_H = 900

// ── 牌桌椭圆 ─────────────────────────────────────────
/** 椭圆中心 X（画布水平居中） */
export const TABLE_CX = 800
/** 椭圆中心 Y（偏上，给底部本地玩家留空间） */
export const TABLE_CY = 385
/** 椭圆 X 半轴（毛毡外边缘） */
export const TABLE_RX = 555
/** 椭圆 Y 半轴（毛毡外边缘） */
export const TABLE_RY = 220
/** 木质边框厚度（变得更细莹） */
export const RAIL_W = 17

// ── 座位 ─────────────────────────────────────────────
/** 本地玩家头像半径（主角尺寸，88px 直径） */
export const AVATAR_R_LOCAL = 44
/** 远端玩家头像半径（小头像，60px 直径） */
export const AVATAR_R_REMOTE = 30

// ── 牌桌纹理 (NineSlice) ───────────────────────────────
/**
 * 九宫格牌桌纹理生成的参数配置
 */
export const TABLE_TEX_CONFIG = {
    /** 纹理基础矩形的大小（增大基础画布以支持更大的圆角半径） */
    SIZE: 600,
    /** 牌桌圆角半径，加大此值可让牌桌两端显得更圆（280已经能达到标准牌桌高度的近乎正半圆形） */
    CORNER_RADIUS: 280,
} as const

// ── 座位轨道（极坐标椭圆）────────────────────────────────
/**
 * 座位中心分布在一条围绕牌桌的虚拟椭圆轨道上，坐标公式：
 *   x = TABLE_CX + TRACK_A * cos(θ)
 *   y = TABLE_CY + TRACK_B * sin(θ)
 *
 * 角度约定（PixiJS y 轴向下）：
 *   θ =  90° → 正下方
 *   θ = 270° → 正上方
 *   θ =   0° → 正右方
 *   θ = 180° → 正左方
 */

/** 轨道水平半轴（略大于牌桌外沿 572，使头像压住桌边） */
const TRACK_A = 580
/** 轨道垂直半轴（略大于牌桌外沿 237） */
const TRACK_B = 262

function seatPos(deg: number, nudgeY = 0) {
    const r = (deg * Math.PI) / 180
    return {
        x: Math.round(TABLE_CX + TRACK_A * Math.cos(r)),
        y: Math.round(TABLE_CY + TRACK_B * Math.sin(r)) + nudgeY,
    }
}

/**
 * 9 个座位中心坐标（极坐标轨道生成）
 *
 *          [5]   [4]   [3]
 *      [6]                 [2]
 *      [7]                 [1]
 *          [8]       [0←你]
 */
export const SEAT_POSITIONS: { x: number; y: number }[] = [
    seatPos(65),       // 0  底部右（本地玩家）θ=65°
    seatPos(25),       // 1  右侧下            θ=25°
    seatPos(335),      // 2  右侧上            θ=335°
    seatPos(300),      // 3  顶部右            θ=300°
    seatPos(270, 35),  // 4  顶部中            θ=270°
    seatPos(240),      // 5  顶部左            θ=240°
    seatPos(205),      // 6  左侧上            θ=205°
    seatPos(155),      // 7  左侧下            θ=155°
    seatPos(115),      // 8  底部左            θ=115°
]
