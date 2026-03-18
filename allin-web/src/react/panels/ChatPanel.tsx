import { useRef, useState, useEffect } from 'react'
import { wsClient } from '../../api/ws'
import { useChatStore } from '../../store/chat'
import styles from './ChatPanel.module.css'

export function ChatPanel() {
  const messages = useChatStore((s) => s.messages)
  const [text, setText] = useState('')
  const [open, setOpen] = useState(false)
  const bottomRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (open) bottomRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages, open])

  function send() {
    const t = text.trim()
    if (!t) return
    wsClient.send('chat', { text: t })
    setText('')
  }

  function onKey(e: React.KeyboardEvent) {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      send()
    }
  }

  return (
    <div className={`${styles.root} ${open ? styles.expanded : ''}`}>
      <button className={styles.toggle} onClick={() => setOpen((v) => !v)}>
        💬 聊天 {!open && messages.length > 0 && <span className={styles.badge}>{messages.length}</span>}
      </button>

      {open && (
        <>
          <div className={styles.messages}>
            {messages.map((m) => (
              <div key={m.id} className={styles.msg}>
                <span className={styles.name}>{m.displayName}</span>
                <span className={styles.text}>{m.text}</span>
              </div>
            ))}
            <div ref={bottomRef} />
          </div>
          <div className={styles.inputRow}>
            <input
              className={styles.input}
              placeholder="说点什么…"
              value={text}
              onChange={(e) => setText(e.target.value)}
              onKeyDown={onKey}
              maxLength={200}
            />
            <button className={styles.sendBtn} onClick={send}>发送</button>
          </div>
        </>
      )}
    </div>
  )
}
