/** 色板常量 — Galactic Casino 设计系统 */
export const C = {
    // ── 表面 ──────────────────────────────────────────
    VOID: 0x060c14,  // 深空背景
    SURFACE_DIM: 0x10141a,  // 主暗色表面
    SURFACE: 0x1c2026,  // 容器背景
    SURFACE_HIGH: 0x262a31,  // 抬升表面
    SURFACE_BRIGHT: 0x353940,  // 提示框、活跃元素
    GLASS: 0x040810,  // 玻璃面板基色（需设置透明度）

    // ── 牌桌 ──────────────────────────────────────────
    WOOD_FRAME: 0x2c1a08,  // 深色红木边框
    FELT_CENTER: 0x1e7a40,  // 最亮绿色（中心）
    FELT_EDGE: 0x0c3320,  // 最暗绿色（边缘）

    // ── 金色 ──────────────────────────────────────────
    GOLD: 0xd4af37,  // 主金色
    GOLD_LIGHT: 0xf2ca50,  // 超新星金（高光）
    GOLD_DIM: 0xb8952c,  // 暗金色（阴影）
    GOLD_CONTAINER: 0xe9c349,  // 表面色调

    // ── 文字 ──────────────────────────────────────────
    TEXT_PRIMARY: 0xdfe2eb,  // 背景上文字
    TEXT_SECONDARY: 0xd0c5af,  // 表面变体上文字
    TEXT_DIM: 0x99907c,  // 轮廓 / 弱化

    // ── 强调色 ────────────────────────────────────────
    GREEN: 0xafcdbd,  // 次要绿色（标签）
    GREEN_DARK: 0x0c3320,  // 次要容器
    GREEN_BET: 0x145a32,  // 下注徽章背景
    RED_SUIT: 0xc0392b,  // 扑克牌红色花色
    BLACK_SUIT: 0x1a1a2e,  // 扑克牌黑色花色

    // ── 扑克牌 ────────────────────────────────────────
    CARD_FACE: 0xfaf8f0,  // 奶白色正面
    CARD_BACK: 0x0f3d24,  // 深绿色背面
    CARD_BACK_ACCENT: 0xc9a84c,  // 背面菱形图案强调色

    // ── 状态 ──────────────────────────────────────────
    ERROR: 0xff5252,
    SUCCESS: 0x4caf50,
} as const

/** 标题字体（数字、标签） */
export const FONT_HEADLINE = "'Space Grotesk', sans-serif"

/** 正文字体（名字、说明） */
export const FONT_BODY = "'Manrope', sans-serif"
