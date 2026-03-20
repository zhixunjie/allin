import { useConnectionStore } from '../../store/connection'
import { ConnectionStatus } from '../../types/enums'
import styles from './ConnectionBanner.module.css'

export function ConnectionBanner() {
  const { status, attempt } = useConnectionStore()
  if (status === ConnectionStatus.Connected) return null

  return (
    <div className={styles.root}>
      {status === ConnectionStatus.Reconnecting
        ? `连接已断开，正在重连… (${attempt}/${10})`
        : '连接已断开，请刷新页面重试'}
    </div>
  )
}
