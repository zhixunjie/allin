/**
 * SoundManager — 基于 Web Audio API 的程序化音效系统。
 * 全部声音通过合成生成，无需外部音频文件。
 *
 * 使用方式：
 *   soundManager.play('deal')
 *   soundManager.setMuted(true)
 */

export type SoundName =
    | 'deal'      // 发牌（洗牌噪声爆发）
    | 'check'     // 过牌（轻拍）
    | 'call'      // 跟注（筹码叮当）
    | 'bet'       // 下注（筹码推出声）
    | 'raise'     // 加注（多颗筹码叮当）
    | 'fold'      // 弃牌（牌张滑落）
    | 'allin'     // 全押（戏剧性重击 + 筹码瀑布）
    | 'myTurn'    // 轮到本人行动（双音铃声提示）
    | 'win'       // 赢得手牌（上行琶音）
    | 'showdown'  // 摊牌（短促紧张击鼓声）

class SoundManager {
    private ctx: AudioContext | null = null
    private masterGain: GainNode | null = null
    private _muted = false

    /** 懒初始化 AudioContext（必须在用户手势后调用） */
    private getCtx(): AudioContext {
        if (!this.ctx) {
            this.ctx = new AudioContext()
            this.masterGain = this.ctx.createGain()
            this.masterGain.gain.value = this._muted ? 0 : 0.55
            this.masterGain.connect(this.ctx.destination)
        }
        // 浏览器策略限制：首次交互前 context 可能处于 suspended 状态
        if (this.ctx.state === 'suspended') {
            void this.ctx.resume()
        }
        return this.ctx
    }

    get muted(): boolean {
        return this._muted
    }

    setMuted(v: boolean): void {
        this._muted = v
        if (this.masterGain) {
            this.masterGain.gain.value = v ? 0 : 0.55
        }
    }

    play(name: SoundName): void {
        try {
            switch (name) {
                case 'deal':     this.playDeal();     break
                case 'check':    this.playCheck();    break
                case 'call':     this.playCall();     break
                case 'bet':      this.playBet();      break
                case 'raise':    this.playRaise();    break
                case 'fold':     this.playFold();     break
                case 'allin':    this.playAllIn();    break
                case 'myTurn':   this.playMyTurn();   break
                case 'win':      this.playWin();      break
                case 'showdown': this.playShowdown(); break
            }
        } catch {
            // AudioContext 不可用时静默失败
        }
    }

    // ── 合成工具 ──────────────────────────────────────────────────

    /** 生成白噪声 AudioBufferSourceNode */
    private noise(durationSec: number): AudioBufferSourceNode {
        const ctx = this.getCtx()
        const len = Math.ceil(ctx.sampleRate * durationSec)
        const buf = ctx.createBuffer(1, len, ctx.sampleRate)
        const data = buf.getChannelData(0)
        for (let i = 0; i < len; i++) data[i] = Math.random() * 2 - 1
        const src = ctx.createBufferSource()
        src.buffer = buf
        return src
    }

    /** 创建带 attack/decay 包络的 GainNode */
    private envelope(attackSec: number, decaySec: number, peak = 1.0): GainNode {
        const ctx = this.getCtx()
        const g = ctx.createGain()
        const t = ctx.currentTime
        g.gain.setValueAtTime(0, t)
        g.gain.linearRampToValueAtTime(peak, t + attackSec)
        g.gain.linearRampToValueAtTime(0, t + attackSec + decaySec)
        return g
    }

    /** 播放单个振荡器音符（自动连接到 masterGain） */
    private tone(
        freq: number,
        type: OscillatorType,
        attackSec: number,
        decaySec: number,
        peak = 0.6,
        delayMs = 0,
    ): void {
        const play = () => {
            const ctx = this.getCtx()
            const osc = ctx.createOscillator()
            osc.type = type
            osc.frequency.value = freq
            const env = this.envelope(attackSec, decaySec, peak)
            osc.connect(env)
            env.connect(this.masterGain!)
            const t = ctx.currentTime
            osc.start(t)
            osc.stop(t + attackSec + decaySec + 0.01)
        }
        if (delayMs > 0) setTimeout(play, delayMs)
        else play()
    }

    // ── 具体音效 ─────────────────────────────────────────────────

    /** 发牌：白噪声经带通滤波器，模拟纸牌滑过桌面的沙沙声 */
    private playDeal(): void {
        const ctx = this.getCtx()
        const src = this.noise(0.12)
        const filter = ctx.createBiquadFilter()
        filter.type = 'bandpass'
        filter.frequency.value = 2200
        filter.Q.value = 0.6
        const env = this.envelope(0.005, 0.09, 0.5)
        src.connect(filter)
        filter.connect(env)
        env.connect(this.masterGain!)
        src.start()
        src.stop(ctx.currentTime + 0.12)
    }

    /** 过牌：低频短促正弦轻拍 */
    private playCheck(): void {
        this.tone(300, 'sine', 0.005, 0.12, 0.45)
    }

    /** 跟注：中频三角波筹码叮当，带小延迟余韵 */
    private playCall(): void {
        this.tone(720, 'triangle', 0.003, 0.18, 0.55)
        this.tone(640, 'triangle', 0.003, 0.13, 0.28, 65)
    }

    /** 下注：低频重音 + 筹码叮当组合 */
    private playBet(): void {
        this.tone(180, 'sine', 0.006, 0.22, 0.65)
        this.tone(620, 'triangle', 0.003, 0.12, 0.38)
    }

    /** 加注：三颗筹码快速连叮，频率略有差异 */
    private playRaise(): void {
        const freqs = [760, 690, 830]
        freqs.forEach((f, i) => this.tone(f, 'triangle', 0.003, 0.16, 0.48, i * 65))
    }

    /** 弃牌：锯齿波频率下扫，模拟牌张扔出的滑落感 */
    private playFold(): void {
        const ctx = this.getCtx()
        const osc = ctx.createOscillator()
        osc.type = 'sawtooth'
        const t = ctx.currentTime
        osc.frequency.setValueAtTime(380, t)
        osc.frequency.exponentialRampToValueAtTime(70, t + 0.22)
        const env = ctx.createGain()
        env.gain.setValueAtTime(0.32, t)
        env.gain.linearRampToValueAtTime(0, t + 0.22)
        osc.connect(env)
        env.connect(this.masterGain!)
        osc.start(t)
        osc.stop(t + 0.25)
    }

    /** 全押：低沉重击 + 五颗筹码瀑布式叮当 */
    private playAllIn(): void {
        this.tone(140, 'sine', 0.008, 0.45, 0.75)
        for (let i = 0; i < 5; i++) {
            this.tone(580 + i * 90, 'triangle', 0.003, 0.16, 0.38, i * 55)
        }
    }

    /** 轮到本人行动：双音阶悦耳铃声，第二音稍高并延迟 */
    private playMyTurn(): void {
        const ctx = this.getCtx()
        const pairs: Array<[number, number]> = [[900, 0], [1120, 110]]
        pairs.forEach(([freq, delayMs]) => {
            const play = () => {
                const osc = ctx.createOscillator()
                osc.type = 'sine'
                osc.frequency.value = freq
                const env = ctx.createGain()
                const t = ctx.currentTime
                env.gain.setValueAtTime(0, t)
                env.gain.linearRampToValueAtTime(0.52, t + 0.01)
                env.gain.exponentialRampToValueAtTime(0.001, t + 0.65)
                osc.connect(env)
                env.connect(this.masterGain!)
                osc.start(t)
                osc.stop(t + 0.68)
            }
            if (delayMs > 0) setTimeout(play, delayMs)
            else play()
        })
    }

    /** 赢得手牌：C5→E5→G5→C6 上行琶音 */
    private playWin(): void {
        const notes = [523, 659, 784, 1047]
        notes.forEach((freq, i) => this.tone(freq, 'sine', 0.01, 0.32, 0.48, i * 105))
    }

    /** 摊牌：白噪声鼓声渐强 + 收尾低沉一击 */
    private playShowdown(): void {
        const ctx = this.getCtx()
        const src = this.noise(0.32)
        const filter = ctx.createBiquadFilter()
        filter.type = 'lowpass'
        filter.frequency.value = 900
        const env = ctx.createGain()
        const t = ctx.currentTime
        env.gain.setValueAtTime(0.08, t)
        env.gain.linearRampToValueAtTime(0.45, t + 0.22)
        env.gain.linearRampToValueAtTime(0, t + 0.32)
        src.connect(filter)
        filter.connect(env)
        env.connect(this.masterGain!)
        src.start()
        src.stop(t + 0.35)
        // 收尾一击
        this.tone(190, 'sine', 0.005, 0.28, 0.62, 310)
    }
}

export const soundManager = new SoundManager()
