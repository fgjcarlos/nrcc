import { FormEvent, useState } from 'react'
import type { AuthMode } from '../../common/types'
import { FormField } from '../../components/forms'

export function AuthScreen({
  mode,
  message,
  busy,
  onSubmit,
}: {
  mode: AuthMode
  message: string
  busy: boolean
  onSubmit: (username: string, password: string) => void
}) {
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')

  function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    onSubmit(username, password)
  }

  const title =
    mode === 'register' ? 'Create Administrator Account' : 'Sign In'

  return (
    <main id="auth-main" tabIndex={-1} className="auth-shell flex min-h-screen items-center justify-center px-6 py-12">
      <div className="surface-card no-hover w-full max-w-md border border-base-300 p-8 sm:p-10">
        <div>
          {mode === 'register' ? (
            <>
              <span className="text-xs font-semibold uppercase tracking-[0.22em] text-primary/70">
                Initial Setup
              </span>
              <h1 className="mt-3 text-3xl font-bold tracking-tight text-base-content">{title}</h1>
            </>
          ) : (
            <>
              <p className="text-xs uppercase tracking-[0.32em] text-base-content/55">Node-RED Control Center</p>
              <h1 className="mt-4 text-3xl font-bold tracking-tight text-base-content">{title}</h1>
            </>
          )}

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
              {busy ? 'Working…' : mode === 'register' ? 'Create Administrator' : 'Sign In'}
            </button>
          </form>
        </div>
      </div>
    </main>
  )
}
