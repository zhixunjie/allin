/**
 * 统一导出入口 — 保持现有 import 路径不变。
 *
 * 内部结构：
 *   config/colors.ts  — 色板、字体
 *   config/card.ts    — 扑克牌尺寸、花色符号、点数映射
 *   config/layout.ts  — 画布尺寸、牌桌椭圆、座位坐标
 *   utils/position.ts — getPositionName 位置名称计算
 */
export * from './config/colors'
export * from './config/card'
export * from './config/layout'
export * from './utils/position'
