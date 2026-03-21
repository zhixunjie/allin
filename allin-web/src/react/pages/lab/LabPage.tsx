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
import { ActionDemo } from './ActionDemo'

export default function LabPage() {
  const canvasRef = useRef<HTMLDivElement>(null)
  const [activeScene, setActiveScene] = useState<string>('')
  const [topOpen, setTopOpen]   = useState(true)
  const [leftOpen, setLeftOpen] = useState(true)
  const [rightOpen, setRightOpen] = useState(false)

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
    <div className="relative h-screen bg-[#060c14] text-white overflow-hidden">

      {/* 画布：始终铺满全屏 */}
      <main className="absolute inset-0 flex items-center justify-center">
        <div ref={canvasRef} className="w-full h-full flex items-center justify-center" />
      </main>

      {/* 顶部抽屉 */}
      <div
        className={[
          'absolute top-0 left-0 right-0 z-20',
          'flex items-center gap-4 px-5 py-2',
          'bg-[#0a1018]/95 backdrop-blur-sm border-b border-amber-900/30',
          'transition-transform duration-300 ease-in-out',
          topOpen ? 'translate-y-0' : '-translate-y-full',
        ].join(' ')}
      >
        <span className="text-amber-400 font-bold tracking-widest text-sm uppercase">PixiJS Lab</span>
        <span className="text-white/20 text-xs">牌桌组件调试实验室</span>
        <div className="ml-auto flex items-center gap-2">
          <DrawerToggle label="控制面板" open={leftOpen} onClick={() => setLeftOpen(v => !v)} />
          <DrawerToggle label="状态监视" open={rightOpen} onClick={() => setRightOpen(v => !v)} />
          <button
            onClick={() => setTopOpen(false)}
            className="ml-2 text-white/20 hover:text-white/50 text-xs px-1 transition-colors"
            title="收起"
          >
            ▲
          </button>
        </div>
      </div>

      {/* 顶部收起后的展开 tab */}
      <button
        onClick={() => setTopOpen(true)}
        className={[
          'absolute top-0 left-1/2 -translate-x-1/2 z-20',
          'px-4 py-0.5 rounded-b text-[10px] text-amber-500/60 hover:text-amber-400',
          'bg-[#0a1018]/80 border-x border-b border-amber-900/20',
          'transition-opacity duration-300',
          topOpen ? 'opacity-0 pointer-events-none' : 'opacity-100',
        ].join(' ')}
      >
        ▼ LAB
      </button>

      {/* 左侧抽屉 */}
      <aside
        className={[
          'absolute top-0 left-0 h-full w-60 z-10',
          'bg-[#0a1018]/95 backdrop-blur-sm border-r border-white/8',
          'flex flex-col overflow-y-auto',
          'transition-transform duration-300 ease-in-out',
          leftOpen ? 'translate-x-0' : '-translate-x-full',
        ].join(' ')}
      >
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
        <Section title="行动演示">
          <ActionDemo />
        </Section>
        <Section title="自由区">
          <FreeZone />
        </Section>
      </aside>

      {/* 右侧抽屉 */}
      <aside
        className={[
          'absolute top-0 right-0 h-full w-56 z-10',
          'bg-[#0a1018]/95 backdrop-blur-sm border-l border-white/8',
          'overflow-y-auto',
          'transition-transform duration-300 ease-in-out',
          rightOpen ? 'translate-x-0' : 'translate-x-full',
        ].join(' ')}
      >
        <StateMonitor />
      </aside>

    </div>
  )
}

// ── 布局专用小型组件 ──────────────────────────────────────────────────────────

function DrawerToggle({ label, open, onClick }: { label: string; open: boolean; onClick: () => void }) {
  return (
    <button
      onClick={onClick}
      className={[
        'text-xs px-3 py-1 rounded border transition-colors',
        open
          ? 'bg-amber-500/20 text-amber-300 border-amber-500/40'
          : 'text-white/40 border-white/10 hover:text-white/70 hover:border-white/20',
      ].join(' ')}
    >
      {label}
    </button>
  )
}

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
