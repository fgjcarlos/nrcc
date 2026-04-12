import { useState } from 'react'
import { SecurityConfig, AdminAuthUser } from '../../../types/config'
import { FormField } from '../../../components/forms'

type SectionProps<T> = {
  value: T
  onChange: (next: T) => void
  errors: Record<string, string>
}

function generateRandomSecret(length: number = 24): string {
  const chars = 'abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789'
  let result = ''
  for (let i = 0; i < length; i++) {
    result += chars.charAt(Math.floor(Math.random() * chars.length))
  }
  return result
}

function generateMD5Hash(input: string): string {
  // Simple mock — in real app, use crypto-js or call server
  // For now, just return a placeholder
  return `md5(${input})`
}

export function SecuritySection({ value, onChange, errors }: SectionProps<SecurityConfig>) {
  const [showSecret, setShowSecret] = useState(false)
  const [showAdminAuth, setShowAdminAuth] = useState(!!value.adminAuth)
  const [showHttpNodeAuth, setShowHttpNodeAuth] = useState(!!value.httpNodeAuth)

  const updateField = <K extends keyof SecurityConfig>(key: K, val: SecurityConfig[K]) => {
    onChange({ ...value, [key]: val })
  }

  const presets = [
    { label: '1 hour', value: 3600 },
    { label: '8 hours', value: 28800 },
    { label: '24 hours', value: 86400 },
    { label: '7 days', value: 604800 },
  ]

  return (
    <article className="space-y-6">
      <h3 className="text-xl font-semibold text-base-content">Security</h3>

      <div className="form-control">
        <label className="label">
          <span className="label-text font-medium">Credential Secret</span>
        </label>
        <div className="flex gap-2">
          <input
            type={showSecret ? 'text' : 'password'}
            className={`input input-bordered bg-base-100 flex-1${errors['security.credentialSecret'] ? ' input-error' : ''}`}
            value={value.credentialSecret}
            onChange={(e) => updateField('credentialSecret', e.target.value)}
            placeholder="Leave empty for default"
            aria-describedby={errors['security.credentialSecret'] ? 'security-credentialSecret-error' : undefined}
          />
          <button
            type="button"
            onClick={() => setShowSecret(!showSecret)}
            className="btn btn-ghost btn-sm"
          >
            {showSecret ? 'Hide' : 'Show'}
          </button>
          <button
            type="button"
            onClick={() => updateField('credentialSecret', generateRandomSecret())}
            className="btn btn-ghost btn-sm"
          >
            Generate
          </button>
        </div>
        {errors['security.credentialSecret'] && (
          <span id="security-credentialSecret-error" className="form-field-error-msg">
            <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
              <path fillRule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2v-3a1 1 0 00-1-1H9z" clipRule="evenodd" />
            </svg>
            {errors['security.credentialSecret']}
          </span>
        )}
      </div>

      <div className="form-control">
        <label className="label">
          <span className="label-text font-medium">Session Expiry Time</span>
        </label>
        <div className="flex gap-2 items-center">
          <input
            type="number"
            className={`input input-bordered bg-base-100 flex-1${errors['security.sessionExpiryTime'] ? ' input-error' : ''}`}
            value={value.sessionExpiryTime}
            onChange={(e) => updateField('sessionExpiryTime', parseInt(e.target.value) || 86400)}
            min={300}
            max={2592000}
            aria-describedby={errors['security.sessionExpiryTime'] ? 'security-sessionExpiryTime-error' : undefined}
          />
          <span className="text-base-content/60 text-sm min-w-max">seconds</span>
        </div>
        <div className="flex flex-wrap gap-2 mt-3">
          {presets.map((p) => (
            <button
              key={p.value}
              type="button"
              onClick={() => updateField('sessionExpiryTime', p.value)}
              className={`btn btn-sm ${value.sessionExpiryTime === p.value ? 'btn-primary' : 'btn-ghost'}`}
            >
              {p.label}
            </button>
          ))}
        </div>
        {errors['security.sessionExpiryTime'] && (
          <span id="security-sessionExpiryTime-error" className="form-field-error-msg">
            <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
              <path fillRule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2v-3a1 1 0 00-1-1H9z" clipRule="evenodd" />
            </svg>
            {errors['security.sessionExpiryTime']}
          </span>
        )}
      </div>

      {/* Admin Authentication */}
      <div className="divider"></div>
      <div className="form-control">
        <label className="label cursor-pointer gap-3">
          <input
            type="checkbox"
            className="checkbox checkbox-sm"
            checked={showAdminAuth}
            onChange={(e) => {
              setShowAdminAuth(e.target.checked)
              if (!e.target.checked) {
                updateField('adminAuth', undefined)
              } else {
                updateField('adminAuth', {
                  type: 'credentials',
                  users: [],
                })
              }
            }}
          />
          <span className="label-text font-medium">Admin Authentication</span>
        </label>
      </div>

       {showAdminAuth && value.adminAuth && (
         <div className="pl-4 border-l-2 border-[color:var(--border-indent)] space-y-4">
          <div className="form-control">
            <label className="label">
              <span className="label-text font-medium">Type</span>
            </label>
            <select
              className="select select-bordered bg-base-100"
              value={value.adminAuth.type}
              onChange={(e) =>
                updateField('adminAuth', {
                  ...value.adminAuth!,
                  type: e.target.value as 'credentials' | 'strategy',
                })
              }
            >
              <option value="credentials">Credentials</option>
              <option value="strategy">Strategy</option>
            </select>
          </div>

          {value.adminAuth.type === 'credentials' && (
            <div className="space-y-3">
              <p className="label-text font-medium">Users</p>
              {value.adminAuth.users.map((user, idx) => (
                <div key={idx} className="flex gap-2 items-end">
                  <input
                    type="text"
                    placeholder="Username"
                    className="input input-bordered bg-base-100 flex-1"
                    value={user.username}
                    onChange={(e) => {
                      const users = [...value.adminAuth!.users]
                      users[idx] = { ...user, username: e.target.value }
                      updateField('adminAuth', { ...value.adminAuth!, users })
                    }}
                  />
                  <input
                    type="password"
                    placeholder="Password"
                    className="input input-bordered bg-base-100 flex-1"
                    value={user.password}
                    onChange={(e) => {
                      const users = [...value.adminAuth!.users]
                      users[idx] = { ...user, password: e.target.value }
                      updateField('adminAuth', { ...value.adminAuth!, users })
                    }}
                  />
                  <select
                    className="select select-bordered bg-base-100"
                    value={user.permissions}
                    onChange={(e) => {
                      const users = [...value.adminAuth!.users]
                      users[idx] = { ...user, permissions: e.target.value as '*' | 'read' }
                      updateField('adminAuth', { ...value.adminAuth!, users })
                    }}
                  >
                    <option value="*">Admin (*)</option>
                    <option value="read">Read-only</option>
                  </select>
                  <button
                    type="button"
                    onClick={() => {
                      const users = value.adminAuth!.users.filter((_, i) => i !== idx)
                      updateField('adminAuth', { ...value.adminAuth!, users })
                    }}
                    className="btn btn-ghost btn-sm"
                  >
                    Remove
                  </button>
                </div>
              ))}
              <button
                type="button"
                onClick={() => {
                  const users = [
                    ...value.adminAuth!.users,
                    { username: '', password: '', permissions: '*' as const },
                  ]
                  updateField('adminAuth', { ...value.adminAuth!, users })
                }}
                className="btn btn-ghost btn-sm"
              >
                + Add User
              </button>
            </div>
          )}
        </div>
      )}

      {/* HTTP Node Auth */}
      <div className="divider"></div>
      <div className="form-control">
        <label className="label cursor-pointer gap-3">
          <input
            type="checkbox"
            className="checkbox checkbox-sm"
            checked={showHttpNodeAuth}
            onChange={(e) => {
              setShowHttpNodeAuth(e.target.checked)
              if (!e.target.checked) {
                updateField('httpNodeAuth', undefined)
              } else {
                updateField('httpNodeAuth', { user: '', pass: '' })
              }
            }}
          />
          <span className="label-text font-medium">HTTP Node Auth</span>
        </label>
      </div>

       {showHttpNodeAuth && value.httpNodeAuth && (
         <div className="pl-4 border-l-2 border-[color:var(--border-indent)] space-y-4">
          <FormField
            id="security-http-node-auth-user"
            label="Username"
            type="text"
            value={value.httpNodeAuth.user}
            onChange={(v) =>
              updateField('httpNodeAuth', {
                ...value.httpNodeAuth!,
                user: v,
              })
            }
            error={errors['security.httpNodeAuth.user']}
          />

          <FormField
            id="security-http-node-auth-pass"
            label="Password (MD5 hash)"
            type="password"
            value={value.httpNodeAuth.pass}
            onChange={(v) =>
              updateField('httpNodeAuth', {
                ...value.httpNodeAuth!,
                pass: v,
              })
            }
            error={errors['security.httpNodeAuth.pass']}
          />
         </div>
       )}
    </article>
  )
}
