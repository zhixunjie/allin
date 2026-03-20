import { useEffect, useState } from 'react'
import { useGameStore } from '../store/game'

/** 返回当前行动计时器的剩余秒数（过期或未激活时返回 0）。 */
export function useActionTimer(): number {
  const deadlineTs = useGameStore((s) => s.deadlineTs)
  const [remaining, setRemaining] = useState(0)

  useEffect(() => {
    if (!deadlineTs) {
      setRemaining(0)
      return
    }

    const update = () => {
      const secs = Math.max(0, Math.ceil((deadlineTs - Date.now()) / 1000))
      setRemaining(secs)
    }

    update()
    const id = setInterval(update, 250)
    return () => clearInterval(id)
  }, [deadlineTs])

  return remaining
}
