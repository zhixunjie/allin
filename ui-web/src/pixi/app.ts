import {Application, Container} from 'pixi.js'
import {initDevtools} from '@pixi/devtools'
import {CANVAS_H, CANVAS_W} from './assets'
import {TableScene} from './scenes/TableScene'

/** 当前活跃的牌桌场景实例（全局单例） */
let scene: TableScene | null = null

/** 当前 PixiJS Application 实例（初始化后可用） */
let _app: Application | null = null

/** Lab 自由区专用容器，叠在所有游戏元素之上 */
let _freeLayer: Container | null = null

/** 获取当前 PixiJS Application，仅供 Lab 调试使用 */
export function getApp(): Application | null {
    return _app
}

/** 获取自由区容器，仅供 Lab 调试使用 */
export function getFreeLayer(): Container | null {
    return _freeLayer
}

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
    _app = app

    await app.init({
        width: CANVAS_W,             // 画布宽度 1600px（16:9）
        height: CANVAS_H,            // 画布高度 900px（16:9）
        backgroundColor: 0x060c14,  // 深空背景色
        antialias: true,            // 抗锯齿
        resolution: window.devicePixelRatio || 1,  // 适配 Retina 屏幕
        autoDensity: true,          // 自动根据 resolution 调整 CSS 尺寸
    })

    // canvas 响应式布局 — 通过 CSS aspectRatio 保持宽高比，避免拉伸模糊
    const canvas = app.canvas as HTMLCanvasElement
    canvas.style.width = '100%'
    canvas.style.maxWidth = `${CANVAS_W}px`
    canvas.style.aspectRatio = `${CANVAS_W} / ${CANVAS_H}`
    canvas.style.display = 'block'
    canvas.style.margin = '0 auto'
    container.appendChild(canvas)

    // 暴露给 PixiJS DevTools Chrome 插件（必须在 app.init() 之后调用）
    // @ts-ignore
    globalThis.__PIXI_APP__ = app
    await initDevtools({app})

    // 创建牌桌场景并初始化（构建所有子元素 + 订阅状态）
    scene = new TableScene(app)
    scene.init()

    // 自由区容器：叠在所有游戏元素之上，供 Lab 直接添加任意 PixiJS 对象
    _freeLayer = new Container()
    app.stage.addChild(_freeLayer)

    // 返回清理函数：销毁场景 → 销毁 PixiJS 应用（含 canvas）
    return () => {
        scene?.destroy()
        scene = null
        _freeLayer = null
        _app = null
        app.destroy(true, {children: true})
    }
}
