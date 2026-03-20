/**
 * PixiJS 组件调试实验室
 *
 * 路由：/lab（无需登录）
 * 原理：直接操作 useGameStore 注入模拟状态，TableScene 的 subscribe 回调
 *       会自动响应并重绘所有元素，无需修改任何游戏逻辑代码。
 */
import { useEffect, useRef, useState } from 'react'
import { initPixiApp } from '../../../pixi/app'
import { SCENES } from './scenes'
import { InjectControls } from './InjectControls'
import { StateMonitor } from './StateMonitor'
import { FreeZone } from './FreeZone'

export default function LabPage() {
  const canvasRef = useRef<HTMLDivElement>(null)
  const [activeScene, setActiveScene] = useState<string>('')

  useEffect(() => {
    if (!canvasRef.current) return
    let cleanup: (() => void) | null = null
    initPixiApp(canvasRef.current).then((fn) => {
      cleanup = fn
      const first = Object.keys(SCENES)[0]
      SCENES[first]()
      setActiveScene(first)
    })
    return () => { cleanup?.() }
  }, [])

  function applyScene(name: string) {
    SCENES[name]()
    setActiveScene(name)
  }

  return (
    <div className="flex flex-col h-screen bg-[#060c14] text-white overflow-hidden">

      <div className="flex items-center gap-4 px-5 py-2 bg-black/60 border-b border-amber-900/30 shrink-0">
        <span className="text-amber-400 font-bold tracking-widest text-sm uppercase">PixiJS Lab</span>
        <span className="text-white/20 text-xs">牌桌组件调试实验室</span>
      </div>

      <div className="flex flex-1 overflow-hidden">

        {/* 左侧控制面板 */}
        <aside className="w-56 shrink-0 bg-black/40 border-r border-white/5 flex flex-col overflow-y-auto">
          <Section title="预设场景">
            {Object.keys(SCENES).map((name) => (
              <SceneBtn
                key={name}
                label={name}
                active={activeScene === name}
                onClick={() => applyScene(name)}
              />
            ))}
          </Section>
          <Section title="快速注入">
            <InjectControls />
          </Section>
          <Section title="自由区">
            <FreeZone />
          </Section>
        </aside>

        {/* 中央画布区 */}
        <main className="flex-1 flex items-center justify-center overflow-hidden">
          <div ref={canvasRef} className="w-full h-full flex items-center justify-center" />
        </main>

        {/* 右侧状态监视器 */}
        <aside className="w-52 shrink-0 bg-black/40 border-l border-white/5 overflow-y-auto">
          <StateMonitor />
        </aside>

      </div>
    </div>
  )
}

// ── 布局专用小型组件（仅在此页面内使用，不单独提取） ────────────────────────

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div className="p-3 border-b border-white/5">
      <p className="text-[10px] font-bold text-amber-500/70 uppercase tracking-widest mb-2">{title}</p>
      <div className="flex flex-col gap-1">{children}</div>
    </div>
  )
}

function SceneBtn({ label, active, onClick }: { label: string; active: boolean; onClick: () => void }) {
  return (
    <button
      onClick={onClick}
      className={[
        'text-left text-xs px-3 py-1.5 rounded transition-colors',
        active
          ? 'bg-amber-500/20 text-amber-300 border border-amber-500/40'
          : 'text-white/60 hover:text-white hover:bg-white/5',
      ].join(' ')}
    >
      {label}
    </button>
  )
}
