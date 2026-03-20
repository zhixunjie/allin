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
/** 本地玩家头像半径（大幅提升存在感） */
export const AVATAR_R_LOCAL = 64
/** 远端玩家头像半径（增大为视觉锚点） */
export const AVATAR_R_REMOTE = 46

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

// ── 座位坐标自动引擎 ─────────────────────────────────────

// 推算牌桌真实半长、半宽
const TABLE_HALF_W = TABLE_RX + RAIL_W
const TABLE_HALF_H = TABLE_RY + RAIL_W
// WebGL 圆角在九宫格机制下的物理极限钳制
const MAX_RADIUS = Math.min(TABLE_TEX_CONFIG.CORNER_RADIUS, TABLE_HALF_H)

// 头像陷入牌桌边缘的重叠感象素（侵入式设计）
// 将 OVERLAP 提高到超过 50% 半径，产生一种“手臂压在桌上、筹码堆在眼前”的强烈聚堆感。
const OVERLAP = 28
const OVERLAP_LOCAL = 16 // 本地玩家有底部面板，保持适度且不失焦的重叠

// 两侧完全半圆圆心相距的几何位置
const CX_RIGHT = TABLE_CX + (TABLE_HALF_W - MAX_RADIUS)
const CX_LEFT  = TABLE_CX - (TABLE_HALF_W - MAX_RADIUS)

// 直线平面的横向发散度
const DX_TOP = TABLE_HALF_W * 0.52    // 顶部左右的横向扩张距
const DX_BOTTOM = TABLE_HALF_W * 0.35 // 底部偏拢（为筹码展示栏留出安全区）
// 曲面两侧的纵向发散度
const DY_SIDE = TABLE_HALF_H * 0.48   // 左右边缘玩家上下分离的绝对物理高度

/** 
 * 圆角弧度侧求 X 函数：
 * (x - cx)^2 + (y - cy)^2 = R^2 反推 x
 */
const calcSideX = (cyOffset: number) => {
    const rNormal = MAX_RADIUS + AVATAR_R_REMOTE - OVERLAP
    const cornerCyOffset = Math.max(0, Math.abs(cyOffset) - (TABLE_HALF_H - MAX_RADIUS))
    return Math.sqrt(Math.max(0, rNormal ** 2 - cornerCyOffset ** 2))
}

const SIDE_DX = calcSideX(DY_SIDE)

/**
 * 9 个座位中心坐标结构生成（基于表征常量自动映射闭环）。
 *
 * 分布：底部 2（本地在右）→ 右侧 2 → 顶部 3 → 左侧 2
 */
export const SEAT_POSITIONS: { x: number; y: number }[] = [
  // 0  底部右（本地玩家大头像，靠底部平直平沿）
  { x: TABLE_CX + DX_BOTTOM, y: TABLE_CY + TABLE_HALF_H + AVATAR_R_LOCAL - OVERLAP_LOCAL },
  // 1  右侧下（附着于右半圆曲率弧线上）
  { x: CX_RIGHT + SIDE_DX,   y: TABLE_CY + DY_SIDE },
  // 2  右侧上
  { x: CX_RIGHT + SIDE_DX,   y: TABLE_CY - DY_SIDE },
  // 3  顶部右（附着于绝对顶部直线地带）
  { x: TABLE_CX + DX_TOP,    y: TABLE_CY - TABLE_HALF_H - AVATAR_R_REMOTE + OVERLAP },
  // 4  顶部中
  { x: TABLE_CX,             y: TABLE_CY - TABLE_HALF_H - AVATAR_R_REMOTE + OVERLAP },
  // 5  顶部左
  { x: TABLE_CX - DX_TOP,    y: TABLE_CY - TABLE_HALF_H - AVATAR_R_REMOTE + OVERLAP },
  // 6  左侧上（附着于左半圆曲率弧线上）
  { x: CX_LEFT - SIDE_DX,    y: TABLE_CY - DY_SIDE },
  // 7  左侧下
  { x: CX_LEFT - SIDE_DX,    y: TABLE_CY + DY_SIDE },
  // 8  底部左
  { x: TABLE_CX - DX_BOTTOM, y: TABLE_CY + TABLE_HALF_H + AVATAR_R_REMOTE - OVERLAP },
]
