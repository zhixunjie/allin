import { useState, FormEvent } from 'react'
import { useNavigate } from 'react-router-dom'
import { authAPI } from '../../api/http'
import { useAuthStore } from '../../store/auth'
import { AuthMode } from '../../types/enums'
import styles from './LoginPage.module.css'

export default function LoginPage() {
  const navigate = useNavigate()
  const setAuth = useAuthStore((s) => s.setAuth)

  const [mode, setMode] = useState<AuthMode>(AuthMode.Login)
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [displayName, setDisplayName] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    setError('')
    setLoading(true)
    try {
      let resp
      if (mode === AuthMode.Login) {
        resp = await authAPI.login(username, password)
      } else {
        resp = await authAPI.register(username, password, displayName || username)
      }
      setAuth(resp.token, resp.user)
      navigate('/lobby')
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : '请求失败')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className={styles.container}>
      <div className={styles.card}>
        <h1 className={styles.title}>AllIn</h1>
        <p className={styles.subtitle}>熟人德州扑克</p>

        <div className={styles.tabs}>
          <button
            className={mode === AuthMode.Login ? styles.tabActive : styles.tab}
            onClick={() => setMode(AuthMode.Login)}
          >
            登录
          </button>
          <button
            className={mode === AuthMode.Register ? styles.tabActive : styles.tab}
            onClick={() => setMode(AuthMode.Register)}
          >
            注册
          </button>
        </div>

        <form onSubmit={handleSubmit} className={styles.form}>
          <input
            className={styles.input}
            type="text"
            placeholder="用户名"
            value={username}
            onChange={(e) => setUsername(e.target.value)}
            required
            autoComplete="username"
          />
          {mode === AuthMode.Register && (
            <input
              className={styles.input}
              type="text"
              placeholder="昵称（可选）"
              value={displayName}
              onChange={(e) => setDisplayName(e.target.value)}
            />
          )}
          <input
            className={styles.input}
            type="password"
            placeholder="密码"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            required
            autoComplete={mode === AuthMode.Login ? 'current-password' : 'new-password'}
          />

          {error && <p className={styles.error}>{error}</p>}

          <button className={styles.submit} type="submit" disabled={loading}>
            {loading ? '请稍候...' : mode === AuthMode.Login ? '登录' : '注册'}
          </button>
        </form>
      </div>
    </div>
  )
}
