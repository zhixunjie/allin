import { useEffect, useState } from 'react'
import { useGameStore } from '../store/game'

/** Returns seconds remaining in the current action timer (0 when expired or not active). */
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
