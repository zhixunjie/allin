// Programmatic rendering — no external image assets required for MVP.
// This module is a placeholder for future asset loading (card sprites, chips, etc.)

export const CARD_W = 60
export const CARD_H = 84
export const CARD_RADIUS = 6

export const SUIT_SYMBOLS: Record<string, string> = {
  c: '♣',
  d: '♦',
  h: '♥',
  s: '♠',
}

export const SUIT_COLORS: Record<string, number> = {
  c: 0x111111,
  d: 0xdd2222,
  h: 0xdd2222,
  s: 0x111111,
}

// Table layout constants
export const TABLE_W = 1200
export const TABLE_H = 700
export const TABLE_CX = 600
export const TABLE_CY = 320

// 9 seat positions arranged around an oval (seat 0 = local player, bottom center)
export const SEAT_POSITIONS: { x: number; y: number }[] = [
  { x: 600, y: 590 }, // 0  bottom center  (local player)
  { x: 870, y: 545 }, // 1  bottom right
  { x: 1065, y: 400 }, // 2  right
  { x: 1010, y: 185 }, // 3  top right
  { x: 810, y: 85 },  // 4  top center-right
  { x: 600, y: 62 },  // 5  top center
  { x: 390, y: 85 },  // 6  top center-left
  { x: 185, y: 185 }, // 7  left
  { x: 135, y: 400 }, // 8  bottom left
]
