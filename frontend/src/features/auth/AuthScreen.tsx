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
    <main className="flex flex-col items-center justify-center min-h-screen bg-base-100 px-6">
      <div className="card bg-base-200 shadow-xl w-full max-w-md p-8">
        <div className="card-body">
          <p className="text-xs font-semibold text-base-content opacity-60 uppercase tracking-wide">Node-RED Control Center</p>
          <h1 className="text-3xl font-bold text-base-content">{title}</h1>
          <p className="text-base-content opacity-70 mt-4">
            {mode === 'register'
              ? 'This machine has not been initialized yet. Create the first local administrator account.'
              : 'Use your local administrator account to access runtime controls and diagnostics.'}
          </p>

          <form className="form-control space-y-4 mt-6" onSubmit={handleSubmit}>
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

            <button className="btn btn-primary w-full mt-6" type="submit" disabled={busy}>
              {busy ? 'Working…' : mode === 'register' ? 'Create account' : 'Sign in'}
            </button>
          </form>

          <div className="flex gap-2 mt-6 justify-center">
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
