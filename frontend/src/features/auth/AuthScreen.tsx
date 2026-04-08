import { FormEvent, useState } from 'react'
import type { AuthMode } from '../../common/types'

export function AuthScreen({
  mode,
  message,
  busy,
  onModeChange,
  onSubmit,
}: {
  mode: AuthMode
  message: string
  busy: boolean
  onModeChange: (mode: AuthMode) => void
  onSubmit: (username: string, password: string) => void
}) {
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')

  function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    onSubmit(username, password)
  }

  const title =
    mode === 'register' ? 'Create the first administrator' : 'Sign in to the local control center'

  return (
    <main className="auth-shell">
      <section className="auth-panel">
        <p className="eyebrow">Node-RED Control Center</p>
        <h1>{title}</h1>
        <p className="auth-copy">
          {mode === 'register'
            ? 'This machine has not been initialized yet. Create the first local administrator account.'
            : 'Use your local administrator account to access runtime controls and diagnostics.'}
        </p>

        <form className="auth-form" onSubmit={handleSubmit}>
          <label>
            <span>Username</span>
            <input value={username} onChange={(event) => setUsername(event.target.value)} required />
          </label>
          <label>
            <span>Password</span>
            <input
              type="password"
              value={password}
              onChange={(event) => setPassword(event.target.value)}
              required
            />
          </label>

          {message ? <p className="auth-error">{message}</p> : null}

          <button className="primary-button" type="submit" disabled={busy}>
            {busy ? 'Working...' : mode === 'register' ? 'Create account' : 'Sign in'}
          </button>
        </form>

        <div className="auth-toggle">
          <button
            className={mode === 'login' ? 'ghost-button active' : 'ghost-button'}
            type="button"
            onClick={() => onModeChange('login')}
          >
            Login
          </button>
          <button
            className={mode === 'register' ? 'ghost-button active' : 'ghost-button'}
            type="button"
            onClick={() => onModeChange('register')}
          >
            Bootstrap
          </button>
        </div>
      </section>
    </main>
  )
}
