/** 按人数对应的位置名称顺序（从庄家开始顺时针） */
const POS_NAMES_BY_COUNT: Record<number, string[]> = {
  2: ['BTN', 'BB'],
  3: ['BTN/UTG', 'SB', 'BB'],
  4: ['BTN', 'SB', 'BB', 'UTG'],
  5: ['BTN', 'SB', 'BB', 'UTG', 'CO'],
  6: ['BTN', 'SB', 'BB', 'UTG', 'HJ', 'CO'],
  7: ['BTN', 'SB', 'BB', 'UTG', 'UTG1', 'HJ', 'CO'],
  8: ['BTN', 'SB', 'BB', 'UTG', 'UTG1', 'MP', 'HJ', 'CO'],
  9: ['BTN', 'SB', 'BB', 'UTG', 'UTG1', 'MP', 'MP1', 'HJ', 'CO'],
}

/**
 * 根据庄家座位和已占用的服务端座位索引列表，返回指定座位的位置名称。
 *
 * @param serverSeatIdx      目标座位的服务端索引
 * @param dealerSeat         庄家座位的服务端索引
 * @param occupiedServerSeats 当前所有已占用座位的服务端索引列表
 * @returns 位置名称（如 "BTN"、"SB"、"BB"），无法计算时返回空字符串
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
