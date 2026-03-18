import { useConnectionStore } from '../../store/connection'
import styles from './ConnectionBanner.module.css'

export function ConnectionBanner() {
  const { status, attempt } = useConnectionStore()
  if (status === 'connected') return null

  return (
    <div className={styles.root}>
      {status === 'reconnecting'
        ? `连接已断开，正在重连… (${attempt}/${10})`
        : '连接已断开，请刷新页面重试'}
    </div>
  )
}
