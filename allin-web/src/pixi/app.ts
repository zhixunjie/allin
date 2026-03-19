import { Application } from 'pixi.js'
import { TABLE_H, TABLE_W } from './assets'
import { TableScene } from './scenes/TableScene'

let scene: TableScene | null = null

/**
 * Initialise the PixiJS application and mount the canvas into `container`.
 * Returns a cleanup function that destroys the app.
 */
export async function initPixiApp(container: HTMLElement): Promise<() => void> {
  const app = new Application()

  await app.init({
    width: TABLE_W,
    height: TABLE_H,
    backgroundColor: 0x0a1520,
    antialias: true,
    resolution: window.devicePixelRatio || 1,
    autoDensity: true,
  })

  // Make canvas responsive — maintain aspect ratio to avoid stretching blur
  const canvas = app.canvas as HTMLCanvasElement
  canvas.style.width = '100%'
  canvas.style.maxWidth = `${TABLE_W}px`
  canvas.style.aspectRatio = `${TABLE_W} / ${TABLE_H}`
  canvas.style.display = 'block'
  canvas.style.margin = '0 auto'
  container.appendChild(canvas)

  scene = new TableScene(app)
  scene.init()

  return () => {
    scene?.destroy()
    scene = null
    app.destroy(true, { children: true })
  }
}
