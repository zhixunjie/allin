/**
 * 自由区：在 PixiJS 画布上直接执行用户输入的代码。
 *
 * 注入到代码执行上下文的变量：
 *   app         — PixiJS Application 实例
 *   stage       — app.stage（根容器）
 *   layer       — 自由区专用 Container（建议往这里加东西，clear 时会统一清理）
 *   Graphics    — pixi.js Graphics
 *   Container   — pixi.js Container
 *   Text        — pixi.js Text
 *   Sprite      — pixi.js Sprite
 *   Texture     — pixi.js Texture
 *   Assets      — pixi.js Assets
 *   C           — 项目色板常量（见 assets.ts）
 *   W / H       — 画布宽高（1600 / 900）
 */
import { useState } from 'react'
import * as PIXI from 'pixi.js'
import { getApp, getFreeLayer } from '../../../pixi/app'
import { C, CANVAS_W, CANVAS_H } from '../../../pixi/assets'

const DEFAULT_CODE = `\
// 示例：画一个金色圆形
const g = new Graphics()
g.circle(W / 2, H / 2, 80)
g.fill({ color: C.GOLD, alpha: 0.8 })
g.stroke({ color: C.GOLD_LIGHT, width: 3 })
layer.addChild(g)
`

export function FreeZone() {
  const [code, setCode] = useState(DEFAULT_CODE)
  const [log, setLog] = useState('')
  const [isError, setIsError] = useState(false)

  function run() {
    const app = getApp()
    const layer = getFreeLayer()
    if (!app || !layer) {
      setLog('PixiJS 尚未初始化')
      setIsError(true)
      return
    }

    try {
      // eslint-disable-next-line no-new-func
      const fn = new Function(
        'app', 'stage', 'layer',
        'Graphics', 'Container', 'Text', 'Sprite', 'Texture', 'Assets',
        'C', 'W', 'H',
        code,
      )
      fn(
        app, app.stage, layer,
        PIXI.Graphics, PIXI.Container, PIXI.Text, PIXI.Sprite, PIXI.Texture, PIXI.Assets,
        C, CANVAS_W, CANVAS_H,
      )
      setLog('✓ 执行成功')
      setIsError(false)
    } catch (e) {
      setLog(e instanceof Error ? e.message : String(e))
      setIsError(true)
    }
  }

  function clear() {
    const layer = getFreeLayer()
    if (!layer) return
    layer.removeChildren()
    setLog('已清空')
    setIsError(false)
  }

  return (
    <div className="flex flex-col gap-2">

      {/* 代码编辑区 */}
      <textarea
        value={code}
        onChange={(e) => setCode(e.target.value)}
        spellCheck={false}
        className={[
          'w-full rounded text-[11px] leading-relaxed font-mono p-2 resize-none',
          'bg-black/60 border border-white/10 text-amber-100/90',
          'focus:outline-none focus:border-amber-500/50',
        ].join(' ')}
        rows={12}
      />

      {/* 操作按钮 */}
      <div className="flex gap-2">
        <button
          onClick={run}
          className="flex-1 text-xs py-1.5 rounded bg-amber-500/20 hover:bg-amber-500/30 text-amber-300 border border-amber-500/40 transition-colors"
        >
          ▶ 执行
        </button>
        <button
          onClick={clear}
          className="text-xs px-3 py-1.5 rounded bg-white/5 hover:bg-white/10 text-white/50 hover:text-white/80 transition-colors"
        >
          清空
        </button>
      </div>

      {/* 执行结果 */}
      {log && (
        <p className={[
          'text-[10px] px-2 py-1 rounded font-mono break-all',
          isError ? 'bg-red-900/30 text-red-300' : 'bg-green-900/20 text-green-400',
        ].join(' ')}>
          {log}
        </p>
      )}

      {/* 可用变量速查 */}
      <details className="text-[10px] text-white/30 cursor-pointer">
        <summary className="hover:text-white/50">可用变量</summary>
        <pre className="mt-1 leading-relaxed text-white/40">{
`app  stage  layer
Graphics  Container
Text  Sprite  Texture  Assets
C  W  H`
        }</pre>
      </details>

    </div>
  )
}
