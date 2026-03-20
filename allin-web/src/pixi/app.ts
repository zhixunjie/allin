import { Application } from 'pixi.js'
import { TABLE_H, TABLE_W } from './assets'
import { TableScene } from './scenes/TableScene'

/** 当前活跃的牌桌场景实例（全局单例） */
let scene: TableScene | null = null

/**
 * 初始化 PixiJS 应用并将 canvas 挂载到 `container` 中。
 * 返回一个销毁应用的清理函数，供 React useEffect 清理时调用。
 *
 * 渲染流程：
 * 1. 创建 PixiJS Application（WebGL 渲染器）
 * 2. 配置 canvas 样式（响应式 + 保持宽高比）
 * 3. 构建 TableScene（牌桌、座位、公共牌等所有游戏元素）
 * 4. TableScene 订阅 Zustand gameStore，数据变化时自动更新画面
 */
export async function initPixiApp(container: HTMLElement): Promise<() => void> {
  const app = new Application()

  await app.init({
    width: TABLE_W,             // 设计稿宽度 1200px
    height: TABLE_H,            // 设计稿高度 700px
    backgroundColor: 0x060c14,  // 深空背景色
    antialias: true,            // 抗锯齿
    resolution: window.devicePixelRatio || 1,  // 适配 Retina 屏幕
    autoDensity: true,          // 自动根据 resolution 调整 CSS 尺寸
  })

  // canvas 响应式布局 — 通过 CSS aspectRatio 保持宽高比，避免拉伸模糊
  const canvas = app.canvas as HTMLCanvasElement
  canvas.style.width = '100%'
  canvas.style.maxWidth = `${TABLE_W}px`
  canvas.style.aspectRatio = `${TABLE_W} / ${TABLE_H}`
  canvas.style.display = 'block'
  canvas.style.margin = '0 auto'
  container.appendChild(canvas)

  // 创建牌桌场景并初始化（构建所有子元素 + 订阅状态）
  scene = new TableScene(app)
  scene.init()

  // 返回清理函数：销毁场景 → 销毁 PixiJS 应用（含 canvas）
  return () => {
    scene?.destroy()
    scene = null
    app.destroy(true, { children: true })
  }
}
