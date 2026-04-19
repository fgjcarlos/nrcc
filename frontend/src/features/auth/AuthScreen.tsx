import { FormEvent, useState } from 'react'
import type { AuthMode } from '../../common/types'
import { FormField } from '../../components/forms'

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
    <main id="auth-main" tabIndex={-1} className="auth-shell flex min-h-screen items-center justify-center px-6 py-12">
      <div className="surface-card w-full max-w-md border border-base-300 p-8 sm:p-10">
        <div>
          <p className="text-xs uppercase tracking-[0.32em] text-base-content/55">Node-RED Control Center</p>
          <h1 className="mt-4 text-3xl font-bold tracking-tight text-base-content">{title}</h1>
          <p className="mt-4 text-base-content/70">
            {mode === 'register'
              ? 'This machine has not been initialized yet. Create the first local administrator account.'
              : 'Use your local administrator account to access runtime controls and diagnostics.'}
          </p>
          <p className="mt-2 text-sm text-base-content/60">
            Stable `.localhost` access is optional. If `portless` is installed during setup, NRCC will try to publish a named local URL automatically.
          </p>

          <form className="form-control mt-8 space-y-4" onSubmit={handleSubmit}>
            <FormField
              id="username"
              label="Username"
              type="text"
              value={username}
              onChange={setUsername}
              required
            />

            <FormField
              id="password"
              label="Password"
              type="password"
              value={password}
              onChange={setPassword}
              required
            />

            {message ? <div className="alert alert-error text-sm">{message}</div> : null}

            <button className="btn btn-primary mt-6 w-full" type="submit" disabled={busy}>
              {busy ? 'Working…' : mode === 'register' ? 'Create account' : 'Sign in'}
            </button>
          </form>

          <div className="mt-6 flex justify-center gap-2">
            <button
              className={`btn btn-sm ${mode === 'login' ? 'btn-primary' : 'btn-ghost'}`}
              type="button"
              onClick={() => onModeChange('login')}
            >
              Login
            </button>
            <button
              className={`btn btn-sm ${mode === 'register' ? 'btn-primary' : 'btn-ghost'}`}
              type="button"
              onClick={() => onModeChange('register')}
            >
              Bootstrap
            </button>
          </div>
        </div>
      </div>
    </main>
  )
}
