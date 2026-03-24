/**
 * SoundManager — 基于 Web Audio API 的程序化音效 + 背景音乐系统。
 * 全部声音通过合成生成，无需外部音频文件。
 *
 * 技术：FM 合成（金属质感筹码）、卷积混响（空间感）、和弦琶音循环 BGM。
 */

export type SoundName =
    | 'deal'      // 发牌
    | 'check'     // 过牌
    | 'call'      // 跟注
    | 'bet'       // 下注
    | 'raise'     // 加注
    | 'fold'      // 弃牌
    | 'allin'     // 全押
    | 'myTurn'    // 轮到本人
    | 'win'       // 赢得手牌
    | 'showdown'  // 摊牌


class SoundManager {
    private ctx: AudioContext | null = null
    private masterGain: GainNode | null = null
    private reverbNode: ConvolverNode | null = null
    private _muted = false

    private getCtx(): AudioContext {
        if (!this.ctx) {
            this.ctx = new AudioContext()
            this.masterGain = this.ctx.createGain()
            this.masterGain.gain.value = this._muted ? 0 : 0.6
            this.masterGain.connect(this.ctx.destination)
            this.reverbNode = this.buildReverb(1.4, 3.2)
            this.reverbNode.connect(this.masterGain)
        }
        if (this.ctx.state === 'suspended') void this.ctx.resume()
        return this.ctx
    }

    setMuted(v: boolean): void {
        this._muted = v
        if (this.masterGain) this.masterGain.gain.value = v ? 0 : 0.6
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
        } catch { /* AudioContext 不可用时静默失败 */ }
    }

    // ── 合成工具 ──────────────────────────────────────────────────

    private buildReverb(durationSec: number, decayFactor: number): ConvolverNode {
        const ctx = this.getCtx()
        const len = Math.floor(ctx.sampleRate * durationSec)
        const buf = ctx.createBuffer(2, len, ctx.sampleRate)
        for (let ch = 0; ch < 2; ch++) {
            const d = buf.getChannelData(ch)
            for (let i = 0; i < len; i++) {
                d[i] = (Math.random() * 2 - 1) * Math.pow(1 - i / len, decayFactor)
            }
        }
        const node = ctx.createConvolver()
        node.buffer = buf
        return node
    }

    private noise(durationSec: number): AudioBufferSourceNode {
        const ctx = this.getCtx()
        const len = Math.ceil(ctx.sampleRate * durationSec)
        const buf = ctx.createBuffer(1, len, ctx.sampleRate)
        const d = buf.getChannelData(0)
        for (let i = 0; i < len; i++) d[i] = Math.random() * 2 - 1
        const src = ctx.createBufferSource()
        src.buffer = buf
        return src
    }

    private fmChip(
        carrier: number,
        modRatio: number,
        modDepth: number,
        decaySec: number,
        peak: number,
        delayMs = 0,
        toReverb = false,
    ): void {
        const play = () => {
            const ctx = this.getCtx()
            const now = ctx.currentTime
            const modOsc = ctx.createOscillator()
            modOsc.type = 'sine'
            modOsc.frequency.value = carrier * modRatio
            const modGain = ctx.createGain()
            modGain.gain.value = carrier * modDepth
            const carOsc = ctx.createOscillator()
            carOsc.type = 'sine'
            carOsc.frequency.value = carrier
            const env = ctx.createGain()
            env.gain.setValueAtTime(peak, now + 0.001)
            env.gain.exponentialRampToValueAtTime(0.001, now + 0.001 + decaySec)
            modOsc.connect(modGain)
            modGain.connect(carOsc.frequency)
            carOsc.connect(env)
            env.connect(toReverb ? this.reverbNode! : this.masterGain!)
            modOsc.start(now); modOsc.stop(now + decaySec + 0.02)
            carOsc.start(now); carOsc.stop(now + decaySec + 0.02)
        }
        if (delayMs > 0) setTimeout(play, delayMs)
        else play()
    }

    private thump(freq: number, decaySec: number, peak: number, delayMs = 0): void {
        const play = () => {
            const ctx = this.getCtx()
            const now = ctx.currentTime
            const osc = ctx.createOscillator()
            osc.type = 'sine'
            osc.frequency.setValueAtTime(freq * 2.2, now)
            osc.frequency.exponentialRampToValueAtTime(freq, now + 0.015)
            const env = ctx.createGain()
            env.gain.setValueAtTime(peak, now + 0.001)
            env.gain.exponentialRampToValueAtTime(0.001, now + decaySec)
            osc.connect(env)
            env.connect(this.masterGain!)
            osc.start(now); osc.stop(now + decaySec + 0.02)
        }
        if (delayMs > 0) setTimeout(play, delayMs)
        else play()
    }

    // ── 具体音效 ─────────────────────────────────────────────────

    private playDeal(): void {
        const ctx = this.getCtx()
        const now = ctx.currentTime
        const src = this.noise(0.09)
        const bp = ctx.createBiquadFilter()
        bp.type = 'bandpass'; bp.frequency.value = 3500; bp.Q.value = 1.2
        const env1 = ctx.createGain()
        env1.gain.setValueAtTime(0.6, now)
        env1.gain.exponentialRampToValueAtTime(0.001, now + 0.07)
        src.connect(bp); bp.connect(env1); env1.connect(this.masterGain!)
        src.start(); src.stop(now + 0.1)
        this.thump(110, 0.07, 0.25, 50)
    }

    private playCheck(): void {
        const ctx = this.getCtx()
        const now = ctx.currentTime
        this.thump(90, 0.11, 0.55)
        this.thump(180, 0.07, 0.28)
        const src = this.noise(0.04)
        const hp = ctx.createBiquadFilter()
        hp.type = 'highpass'; hp.frequency.value = 1500
        const g = ctx.createGain()
        g.gain.setValueAtTime(0.18, now)
        g.gain.exponentialRampToValueAtTime(0.001, now + 0.04)
        src.connect(hp); hp.connect(g); g.connect(this.masterGain!)
        src.start(); src.stop(now + 0.05)
    }

    private playCall(): void {
        this.fmChip(820, 3.1, 2.8, 0.38, 0.62)
        this.fmChip(740, 2.9, 2.5, 0.30, 0.42, 70)
    }

    private playBet(): void {
        this.thump(75, 0.18, 0.6)
        this.fmChip(880, 3.3, 3.0, 0.32, 0.55, 60)
        this.fmChip(810, 3.0, 2.7, 0.28, 0.45, 120)
        this.fmChip(750, 2.8, 2.4, 0.25, 0.35, 180)
    }

    private playRaise(): void {
        this.thump(65, 0.22, 0.7)
        const chips = [
            { freq: 920, delay: 40 },
            { freq: 860, delay: 100 },
            { freq: 800, delay: 160 },
            { freq: 740, delay: 220 },
            { freq: 680, delay: 280 },
        ]
        chips.forEach(({ freq, delay }) =>
            this.fmChip(freq, 3.2, 2.9, 0.35, 0.48, delay)
        )
    }

    private playFold(): void {
        const ctx = this.getCtx()
        const now = ctx.currentTime
        const src1 = this.noise(0.03)
        const hp = ctx.createBiquadFilter()
        hp.type = 'highpass'; hp.frequency.value = 4000
        const g1 = ctx.createGain()
        g1.gain.setValueAtTime(0.55, now)
        g1.gain.exponentialRampToValueAtTime(0.001, now + 0.025)
        src1.connect(hp); hp.connect(g1); g1.connect(this.masterGain!)
        src1.start(); src1.stop(now + 0.03)
        setTimeout(() => {
            const ctx2 = this.getCtx()
            const t = ctx2.currentTime
            const src2 = this.noise(0.18)
            const bp = ctx2.createBiquadFilter()
            bp.type = 'bandpass'; bp.Q.value = 0.8
            bp.frequency.setValueAtTime(2800, t)
            bp.frequency.exponentialRampToValueAtTime(300, t + 0.18)
            const g2 = ctx2.createGain()
            g2.gain.setValueAtTime(0.42, t)
            g2.gain.exponentialRampToValueAtTime(0.001, t + 0.18)
            src2.connect(bp); bp.connect(g2); g2.connect(this.masterGain!)
            src2.start(); src2.stop(t + 0.2)
        }, 25)
    }

    private playAllIn(): void {
        const ctx = this.getCtx()
        const now = ctx.currentTime
        const bass = ctx.createOscillator()
        bass.type = 'sine'
        bass.frequency.setValueAtTime(55, now)
        bass.frequency.exponentialRampToValueAtTime(28, now + 0.5)
        const bassEnv = ctx.createGain()
        bassEnv.gain.setValueAtTime(0.9, now + 0.001)
        bassEnv.gain.exponentialRampToValueAtTime(0.001, now + 0.5)
        bass.connect(bassEnv)
        bassEnv.connect(this.masterGain!)
        bassEnv.connect(this.reverbNode!)
        bass.start(now); bass.stop(now + 0.55)
        const src = this.noise(0.06)
        const lp = ctx.createBiquadFilter()
        lp.type = 'lowpass'; lp.frequency.value = 600
        const ng = ctx.createGain()
        ng.gain.setValueAtTime(0.7, now)
        ng.gain.exponentialRampToValueAtTime(0.001, now + 0.06)
        src.connect(lp); lp.connect(ng); ng.connect(this.masterGain!)
        src.start(); src.stop(now + 0.07)
        const cascade = [
            { freq: 980, delay: 55,  decay: 0.38 },
            { freq: 910, delay: 105, decay: 0.34 },
            { freq: 850, delay: 150, decay: 0.30 },
            { freq: 790, delay: 190, decay: 0.27 },
            { freq: 730, delay: 225, decay: 0.24 },
            { freq: 670, delay: 255, decay: 0.21 },
            { freq: 620, delay: 280, decay: 0.18 },
        ]
        cascade.forEach(({ freq, delay, decay }) =>
            this.fmChip(freq, 3.2, 3.0, decay, 0.5, delay, true)
        )
    }

    private playMyTurn(): void {
        const ctx = this.getCtx()
        const chord = [1047, 1319, 1568]  // C6-E6-G6
        chord.forEach((freq, i) => {
            setTimeout(() => {
                const t = ctx.currentTime
                const osc = ctx.createOscillator()
                osc.type = 'sine'
                osc.frequency.value = freq
                const env = ctx.createGain()
                env.gain.setValueAtTime(0, t)
                env.gain.linearRampToValueAtTime(0.38, t + 0.015)
                env.gain.exponentialRampToValueAtTime(0.001, t + 0.9)
                osc.connect(env)
                env.connect(this.masterGain!)
                env.connect(this.reverbNode!)
                osc.start(t); osc.stop(t + 0.92)
            }, i * 45)
        })
    }

    private playWin(): void {
        const ctx = this.getCtx()
        const scale = [392, 494, 587, 740]
        scale.forEach((freq, i) => {
            setTimeout(() => {
                const t = ctx.currentTime
                const osc = ctx.createOscillator()
                osc.type = 'sawtooth'
                osc.frequency.value = freq
                const lp = ctx.createBiquadFilter()
                lp.type = 'lowpass'; lp.frequency.value = 2200
                const env = ctx.createGain()
                env.gain.setValueAtTime(0, t)
                env.gain.linearRampToValueAtTime(0.32, t + 0.02)
                env.gain.exponentialRampToValueAtTime(0.001, t + 0.28)
                osc.connect(lp); lp.connect(env); env.connect(this.masterGain!)
                osc.start(t); osc.stop(t + 0.32)
            }, i * 95)
        })
        const finalChord = [784, 988, 1175]
        finalChord.forEach((freq) => {
            setTimeout(() => {
                const t = ctx.currentTime
                const osc = ctx.createOscillator()
                osc.type = 'sine'
                osc.frequency.value = freq
                const env = ctx.createGain()
                env.gain.setValueAtTime(0, t)
                env.gain.linearRampToValueAtTime(0.42, t + 0.01)
                env.gain.exponentialRampToValueAtTime(0.001, t + 1.1)
                osc.connect(env)
                env.connect(this.masterGain!)
                env.connect(this.reverbNode!)
                osc.start(t); osc.stop(t + 1.15)
            }, 420)
        })
    }

    private playShowdown(): void {
        const ctx = this.getCtx()
        const now = ctx.currentTime
        const src = this.noise(0.55)
        const bp = ctx.createBiquadFilter()
        bp.type = 'bandpass'; bp.Q.value = 1.5
        bp.frequency.setValueAtTime(300, now)
        bp.frequency.exponentialRampToValueAtTime(2200, now + 0.5)
        const roll = ctx.createGain()
        roll.gain.setValueAtTime(0.05, now)
        roll.gain.linearRampToValueAtTime(0.55, now + 0.48)
        roll.gain.linearRampToValueAtTime(0, now + 0.55)
        src.connect(bp); bp.connect(roll); roll.connect(this.masterGain!)
        src.start(); src.stop(now + 0.58)
        setTimeout(() => {
            const t = ctx.currentTime
            this.thump(70, 0.4, 0.8)
            const src2 = this.noise(0.08)
            const hp = ctx.createBiquadFilter()
            hp.type = 'highpass'; hp.frequency.value = 8000
            const g = ctx.createGain()
            g.gain.setValueAtTime(0.45, t)
            g.gain.exponentialRampToValueAtTime(0.001, t + 0.06)
            src2.connect(hp); hp.connect(g)
            g.connect(this.masterGain!)
            g.connect(this.reverbNode!)
            src2.start(); src2.stop(t + 0.09)
        }, 530)
    }
}

export const soundManager = new SoundManager()
