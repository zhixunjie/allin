import styles from './BetSlider.module.css'

interface Props {
  pot: number
  min: number
  max: number
  step: number
  value: number
  onChange: (v: number) => void
}

export function BetSlider({ pot, min, max, step, value, onChange }: Props) {
  const clamp = (v: number) => Math.max(min, Math.min(max, v))

  return (
    <div className={styles.root}>
      <div className={styles.row}>
        <button className={styles.adj} onClick={() => onChange(clamp(value - step))}>−</button>
        <input
          type="range"
          className={styles.slider}
          min={min}
          max={max}
          step={step}
          value={value}
          onChange={(e) => onChange(Number(e.target.value))}
        />
        <button className={styles.adj} onClick={() => onChange(clamp(value + step))}>+</button>
      </div>
      <div className={styles.presets}>
        <button className={styles.preset} onClick={() => onChange(clamp(Math.round(pot * 0.33)))}>1/3</button>
        <button className={styles.preset} onClick={() => onChange(clamp(Math.round(pot * 0.5)))}>1/2</button>
        <button className={styles.preset} onClick={() => onChange(clamp(Math.round(pot * 0.75)))}>3/4</button>
        <button className={styles.preset} onClick={() => onChange(clamp(pot))}>Pot</button>
        <button className={styles.preset} onClick={() => onChange(max)}>All-In</button>
        <span className={styles.amount}>{value.toLocaleString()}</span>
      </div>
    </div>
  )
}
