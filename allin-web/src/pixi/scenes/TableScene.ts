import { Application, Container, Graphics, Text, Ticker } from 'pixi.js'
import { useGameStore } from '../../store/game'
import {
  SEAT_POSITIONS,
  TABLE_CX,
  TABLE_CY,
  TABLE_H,
  TABLE_W,
} from '../assets'
import { CardSprite } from '../components/CardSprite'
import { PotDisplay } from '../components/PotDisplay'
import { SeatSprite } from '../components/SeatSprite'
import { TimerArc } from '../components/TimerArc'
import { DealAnimation } from './DealAnimation'
import { ChipAnimation } from './ChipAnimation'

export class TableScene {
  private app: Application
  private root: Container
  private seatSprites: SeatSprite[] = []
  private communityCards: CardSprite[] = []
  private potDisplay: PotDisplay
  private timerArc: TimerArc
  private dealAnim: DealAnimation
  private chipAnim: ChipAnimation
  private streetLabel: Text
  private dealerChip: Graphics
  private unsubscribe: () => void = () => {}

  private myServerSeat = -1
  private prevStreet = 'idle'
  private prevActionSeat = -1

  constructor(app: Application) {
    this.app = app
    this.root = new Container()
    this.app.stage.addChild(this.root)

    this.potDisplay = new PotDisplay()
    this.timerArc = new TimerArc()
    this.dealAnim = new DealAnimation(app)
    this.chipAnim = new ChipAnimation(app)

    this.streetLabel = new Text({
      text: '',
      style: { fontSize: 14, fill: 0x8a9bb0, letterSpacing: 2 },
    })
    this.streetLabel.anchor.set(0.5)

    this.dealerChip = new Graphics()
    this.dealerChip.circle(0, 0, 12)
    this.dealerChip.fill({ color: 0xffffff })
    this.dealerChip.stroke({ color: 0x333333, width: 2 })
    const dLabel = new Text({ text: 'D', style: { fontSize: 11, fill: 0x111111, fontWeight: 'bold' } })
    dLabel.anchor.set(0.5)
    this.dealerChip.addChild(dLabel)
    this.dealerChip.visible = false
  }

  init() {
    this.buildTable()
    this.buildSeats()
    this.buildCommunityArea()

    // Start ticker
    this.app.ticker.add(this.onTick, this)

    // Subscribe to game store
    this.unsubscribe = useGameStore.subscribe((state) => {
      this.updateFromState(state)
    })
    this.updateFromState(useGameStore.getState())
  }

  destroy() {
    this.unsubscribe()
    this.app.ticker.remove(this.onTick, this)
  }

  private onTick(ticker: Ticker) {
    this.timerArc.tick()
    this.dealAnim.tick(ticker.deltaMS)
    this.chipAnim.tick(ticker.deltaMS)
  }

  // ---- Build ----

  private buildTable() {
    const felt = new Graphics()
    felt.rect(0, 0, TABLE_W, TABLE_H)
    felt.fill({ color: 0x0a1520 })

    const table = new Graphics()
    table.ellipse(TABLE_CX, TABLE_CY, 490, 240)
    table.fill({ color: 0x1a5c2a })
    table.stroke({ color: 0x0d3018, width: 6 })

    const rail = new Graphics()
    rail.ellipse(TABLE_CX, TABLE_CY, 470, 222)
    rail.stroke({ color: 0x2d7a40, width: 3 })

    this.root.addChild(felt, table, rail)

    // Street label (e.g. "FLOP")
    this.streetLabel.position.set(TABLE_CX, TABLE_CY + 200)
    this.root.addChild(this.streetLabel)

    // Pot display
    this.potDisplay.position.set(TABLE_CX - 70, TABLE_CY - 28)
    this.root.addChild(this.potDisplay)

    // Dealer chip
    this.root.addChild(this.dealerChip)
  }

  private buildSeats() {
    for (let i = 0; i < 9; i++) {
      const sprite = new SeatSprite(i === 0)
      // Seats are centered on their position
      sprite.pivot.set(60, 40)
      this.root.addChild(sprite)
      this.seatSprites.push(sprite)
    }
  }

  private buildCommunityArea() {
    // Community cards
    for (let i = 0; i < 5; i++) {
      const card = new CardSprite()
      card.visible = false
      // Center the 5-card row
      card.position.set(TABLE_CX - 5 * 33 + i * 66, TABLE_CY - 42)
      this.root.addChild(card)
      this.communityCards.push(card)
    }

    // Timer arc near center
    this.timerArc.position.set(TABLE_CX + 220, TABLE_CY)
    this.root.addChild(this.timerArc)
  }

  // ---- Update ----

  private updateFromState(state: ReturnType<typeof useGameStore.getState>) {
    const myUserId = state.myUserId

    // Determine my server seat index
    if (myUserId) {
      const mySeat = state.seats.find((s) => s.user_id === myUserId)
      if (mySeat) this.myServerSeat = mySeat.seat_index
    }

    // Update seat sprites
    for (let displayIdx = 0; displayIdx < 9; displayIdx++) {
      const serverIdx = this.toServerSeat(displayIdx)
      const seatData = state.seats.find((s) => s.seat_index === serverIdx) ?? null
      const isActive =
        seatData !== null &&
        state.action_seat === serverIdx &&
        state.street !== 'idle'

      const pos = SEAT_POSITIONS[displayIdx]
      this.seatSprites[displayIdx].position.set(pos.x, pos.y)
      this.seatSprites[displayIdx].update(
        seatData,
        isActive,
        seatData?.user_id === myUserId ? state.myHole : undefined,
      )
    }

    // Update community cards
    const community = state.community ?? []
    for (let i = 0; i < 5; i++) {
      if (i < community.length) {
        this.communityCards[i].setCard(community[i])
        this.communityCards[i].visible = true
      } else {
        this.communityCards[i].visible = false
      }
    }

    // Update pot
    this.potDisplay.setPot(state.pot ?? 0)

    // Street label
    const street = state.street ?? 'idle'
    this.streetLabel.text = street === 'idle' ? '' : street.toUpperCase()

    // Dealer chip
    if (state.dealer_seat >= 0) {
      const displayDealer = this.toDisplaySeat(state.dealer_seat)
      const pos = SEAT_POSITIONS[displayDealer]
      this.dealerChip.position.set(pos.x + 40, pos.y - 40)
      this.dealerChip.visible = true
    } else {
      this.dealerChip.visible = false
    }

    // Action timer
    if (state.deadlineTs && state.action_seat >= 0) {
      const timeSec = state.config?.action_time_sec ?? 30
      this.timerArc.start(state.deadlineTs, timeSec)
    } else if (this.prevActionSeat !== state.action_seat) {
      this.timerArc.stop()
    }

    // Detect new street → chip animation from each seat to pot
    if (state.street !== this.prevStreet && state.street !== 'idle') {
      // Street transition: chips fly to center
      for (const seat of state.seats) {
        if (seat.bet > 0) {
          const displayIdx = this.toDisplaySeat(seat.seat_index)
          const pos = SEAT_POSITIONS[displayIdx]
          this.chipAnim.flyChip(pos.x, pos.y, seat.bet)
        }
      }
    }

    this.prevStreet = state.street
    this.prevActionSeat = state.action_seat
  }

  /** Convert display seat index (0=local player) to server seat index. */
  private toServerSeat(displayIdx: number): number {
    if (this.myServerSeat < 0) return displayIdx
    return (displayIdx + this.myServerSeat) % 9
  }

  /** Convert server seat index to display seat index. */
  private toDisplaySeat(serverIdx: number): number {
    if (this.myServerSeat < 0) return serverIdx
    return (serverIdx - this.myServerSeat + 9) % 9
  }
}
