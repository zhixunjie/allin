// ═══════════════════════════════════════════════════════════════
// Galactic Casino — 设计常量
// ═══════════════════════════════════════════════════════════════

// ---- 颜色 ----
export const C = {
  // 表面
  VOID:           0x060c14,  // 深空背景
  SURFACE_DIM:    0x10141a,  // 主暗色表面
  SURFACE:        0x1c2026,  // 容器背景
  SURFACE_HIGH:   0x262a31,  // 抬升表面
  SURFACE_BRIGHT: 0x353940,  // 提示框、活跃元素
  GLASS:          0x040810,  // 玻璃面板基色（需设置透明度）

  // 牌桌
  WOOD_FRAME:     0x2c1a08,  // 深色红木边框
  FELT_CENTER:    0x1e7a40,  // 最亮绿色（中心）
  FELT_MID:       0x1a6b3a,
  FELT_OUTER:     0x145a32,
  FELT_EDGE:      0x0c3320,  // 最暗绿色（边缘）

  // 金色色板
  GOLD:           0xd4af37,  // 主金色
  GOLD_LIGHT:     0xf2ca50,  // 超新星金（高光）
  GOLD_DIM:       0xb8952c,  // 暗金色（阴影）
  GOLD_CONTAINER: 0xe9c349,  // 表面色调

  // 文字
  TEXT_PRIMARY:   0xdfe2eb,  // 背景上文字
  TEXT_SECONDARY: 0xd0c5af,  // 表面变体上文字
  TEXT_DIM:       0x99907c,  // 轮廓 / 弱化

  // 强调色
  GREEN:          0xafcdbd,  // 次要绿色（标签）
  GREEN_DARK:     0x0c3320,  // 次要容器
  GREEN_BET:      0x145a32,  // 下注徽章背景
  RED_SUIT:       0xc0392b,  // 扑克牌红色花色
  BLACK_SUIT:     0x1a1a2e,  // 扑克牌黑色花色

  // 扑克牌
  CARD_FACE:      0xfaf8f0,  // 奶白色
  CARD_BACK:      0x0f3d24,  // 深绿色背面
  CARD_BACK_ACCENT: 0xc9a84c, // 菱形图案强调色

  // 状态
  ERROR:          0xff5252,
  SUCCESS:        0x4caf50,
} as const

// ---- 字体 ----
export const FONT_HEADLINE = "'Space Grotesk', sans-serif"
export const FONT_BODY = "'Manrope', sans-serif"

// ---- 扑克牌尺寸 ----
export const CARD_W = 80
export const CARD_H = 112
export const CARD_RADIUS = 10

export const HOLE_CARD_W = 64
export const HOLE_CARD_H = 90
export const HOLE_CARD_RADIUS = 8

// ---- 花色渲染 ----
export const SUIT_SYMBOLS: Record<string, string> = {
  c: '♣',
  d: '♦',
  h: '♥',
  s: '♠',
}

export const SUIT_COLORS: Record<string, number> = {
  c: C.BLACK_SUIT,
  d: C.RED_SUIT,
  h: C.RED_SUIT,
  s: C.BLACK_SUIT,
}

// ---- 牌面显示 ----
export const RANK_DISPLAY: Record<string, string> = {
  A: 'A', K: 'K', Q: 'Q', J: 'J', T: '10',
  '9': '9', '8': '8', '7': '7', '6': '6',
  '5': '5', '4': '4', '3': '3', '2': '2',
}

// ---- 画布尺寸（16:9，适配 PC macOS 浏览器）----
export const CANVAS_W = 1600       // 画布总宽度
export const CANVAS_H = 900        // 画布总高度

// PixiJS 导出兼容别名（内部用）
export const TABLE_W = CANVAS_W
export const TABLE_H = CANVAS_H

// ---- 牌桌椭圆 ----
export const TABLE_CX = 800        // 椭圆中心 X（画布水平居中）
export const TABLE_CY = 385        // 椭圆中心 Y（偏上，给底部本地玩家留空间）
export const TABLE_RX = 630        // 椭圆 X 半轴（毛毡外边缘）
export const TABLE_RY = 270        // 椭圆 Y 半轴（毛毡外边缘）
export const RAIL_W   = 18         // 木质边框厚度（含金色描边）

// ---- 座位布局 ----
// 头像半径
export const AVATAR_R_LOCAL = 52
export const AVATAR_R_REMOTE = 38

// 9 个座位中心坐标（显示索引 0 = 本地玩家在底部，顺时针排列）
// 边距说明：
//   顶部座位 y ≥ 110（留出头像+位置标签+摊牌手牌的空间）
//   底部本地 y = 745（筹码面板底边约 y+130 = 875，距画布底 25px）
//   左右侧   x 距边缘 ≥ 160（留出头像+下注徽章的空间）
export const SEAT_POSITIONS: { x: number; y: number }[] = [
  { x: 800,  y: 745 },  // 0  底部中央（本地玩家，大头像）
  { x: 1255, y: 645 },  // 1  右下
  { x: 1445, y: 395 },  // 2  右侧
  { x: 1320, y: 165 },  // 3  右上
  { x: 1055, y: 80  },  // 4  上方偏右
  { x: 800,  y: 60  },  // 5  顶部中央
  { x: 545,  y: 80  },  // 6  上方偏左
  { x: 280,  y: 165 },  // 7  左上
  { x: 155,  y: 395 },  // 8  左侧
]

// ---- 位置名称 ----
const POS_NAMES_BY_COUNT: Record<number, string[]> = {
  2: ['BTN', 'BB'],
  3: ['BTN', 'SB', 'BB'],
  4: ['BTN', 'SB', 'BB', 'UTG'],
  5: ['BTN', 'SB', 'BB', 'UTG', 'CO'],
  6: ['BTN', 'SB', 'BB', 'UTG', 'HJ', 'CO'],
  7: ['BTN', 'SB', 'BB', 'UTG', 'UTG1', 'HJ', 'CO'],
  8: ['BTN', 'SB', 'BB', 'UTG', 'UTG1', 'MP', 'HJ', 'CO'],
  9: ['BTN', 'SB', 'BB', 'UTG', 'UTG1', 'MP', 'MP1', 'HJ', 'CO'],
}

/**
 * 根据庄家座位和已占用的服务器座位索引列表，获取指定座位的位置名称。
 */
export function getPositionName(
  serverSeatIdx: number,
  dealerSeat: number,
  occupiedServerSeats: number[],
): string {
  const n = occupiedServerSeats.length
  if (n < 2 || dealerSeat < 0) return ''

  // 从庄家开始，在已占用座位中按顺时针排序
  const ordered: number[] = []
  for (let i = 0; i < 9; i++) {
    const idx = (dealerSeat + i) % 9
    if (occupiedServerSeats.includes(idx)) ordered.push(idx)
  }

  const posIdx = ordered.indexOf(serverSeatIdx)
  if (posIdx < 0) return ''

  const names = POS_NAMES_BY_COUNT[n] ?? POS_NAMES_BY_COUNT[9]!
  return names[posIdx] ?? ''
}
