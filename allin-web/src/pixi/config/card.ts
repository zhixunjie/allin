import { C } from './colors'

// ── 公共牌尺寸（桌面展示）────────────────────────────
export const CARD_W = 80
export const CARD_H = 112
export const CARD_RADIUS = 10

// ── 手牌尺寸（座位旁小牌）────────────────────────────
export const HOLE_CARD_W = 64
export const HOLE_CARD_H = 90
export const HOLE_CARD_RADIUS = 8

// ── 花色符号 ─────────────────────────────────────────
export const SUIT_SYMBOLS: Record<string, string> = {
  c: '♣',
  d: '♦',
  h: '♥',
  s: '♠',
}

// ── 花色颜色 ─────────────────────────────────────────
export const SUIT_COLORS: Record<string, number> = {
  c: C.BLACK_SUIT,
  d: C.RED_SUIT,
  h: C.RED_SUIT,
  s: C.BLACK_SUIT,
}

// ── 点数显示（T → 10）────────────────────────────────
export const RANK_DISPLAY: Record<string, string> = {
  A: 'A', K: 'K', Q: 'Q', J: 'J', T: '10',
  '9': '9', '8': '8', '7': '7', '6': '6',
  '5': '5', '4': '4', '3': '3', '2': '2',
}
