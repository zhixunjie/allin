import { useState } from 'react'
import styles from './RoomInfo.module.css'

interface Props {
  code: string
}

export function RoomInfo({ code }: Props) {
  const [copied, setCopied] = useState(false)
  const inviteUrl = `${window.location.origin}/join/${code}`

  function copy() {
    navigator.clipboard.writeText(inviteUrl).then(() => {
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    })
  }

  return (
    <div className={styles.root}>
      <span className={styles.code}>#{code}</span>
      <button className={styles.copyBtn} onClick={copy}>
        {copied ? '✓ 已复制' : '邀请'}
      </button>
    </div>
  )
}
