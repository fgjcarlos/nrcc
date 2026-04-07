import { useState } from 'react'
import { SecurityConfig, AdminAuthUser } from '../../../types/config'

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
    <article className="settings-section">
      <h3>Security</h3>

      <label className="form-field">
        <span>Credential Secret</span>
        <div className="input-group">
          <input
            type={showSecret ? 'text' : 'password'}
            value={value.credentialSecret}
            onChange={(e) => updateField('credentialSecret', e.target.value)}
            placeholder="Leave empty for default"
          />
          <button
            type="button"
            onClick={() => setShowSecret(!showSecret)}
            className="ghost-button small"
          >
            {showSecret ? 'Hide' : 'Show'}
          </button>
          <button
            type="button"
            onClick={() => updateField('credentialSecret', generateRandomSecret())}
            className="ghost-button small"
          >
            Generate
          </button>
        </div>
        {errors['security.credentialSecret'] && (
          <p className="field-error">{errors['security.credentialSecret']}</p>
        )}
      </label>

      <label className="form-field">
        <span>Session Expiry Time</span>
        <div className="input-group">
          <input
            type="number"
            value={value.sessionExpiryTime}
            onChange={(e) => updateField('sessionExpiryTime', parseInt(e.target.value) || 86400)}
            min={300}
            max={2592000}
          />
          <span className="unit">seconds</span>
        </div>
        <div className="preset-buttons">
          {presets.map((p) => (
            <button
              key={p.value}
              type="button"
              onClick={() => updateField('sessionExpiryTime', p.value)}
              className={`preset-button ${value.sessionExpiryTime === p.value ? 'active' : ''}`}
            >
              {p.label}
            </button>
          ))}
        </div>
        {errors['security.sessionExpiryTime'] && (
          <p className="field-error">{errors['security.sessionExpiryTime']}</p>
        )}
      </label>

      {/* Admin Authentication */}
      <div className="collapsible-section">
        <label className="form-toggle">
          <input
            type="checkbox"
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
          <span>Admin Authentication</span>
        </label>

        {showAdminAuth && value.adminAuth && (
          <div className="collapsible-content">
            <label className="form-field">
              <span>Type</span>
              <select
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
            </label>

            {value.adminAuth.type === 'credentials' && (
              <div>
                <label className="form-field">
                  <span>Users</span>
                </label>
                {value.adminAuth.users.map((user, idx) => (
                  <div key={idx} className="admin-user-row">
                    <input
                      type="text"
                      placeholder="Username"
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
                      value={user.password}
                      onChange={(e) => {
                        const users = [...value.adminAuth!.users]
                        users[idx] = { ...user, password: e.target.value }
                        updateField('adminAuth', { ...value.adminAuth!, users })
                      }}
                    />
                    <select
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
                      className="ghost-button"
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
                  className="ghost-button"
                >
                  Add User
                </button>
              </div>
            )}
          </div>
        )}
      </div>

      {/* HTTP Node Auth */}
      <div className="collapsible-section">
        <label className="form-toggle">
          <input
            type="checkbox"
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
          <span>HTTP Node Auth</span>
        </label>

        {showHttpNodeAuth && value.httpNodeAuth && (
          <div className="collapsible-content">
            <label className="form-field">
              <span>Username</span>
              <input
                type="text"
                value={value.httpNodeAuth.user}
                onChange={(e) =>
                  updateField('httpNodeAuth', {
                    ...value.httpNodeAuth!,
                    user: e.target.value,
                  })
                }
              />
              {errors['security.httpNodeAuth.user'] && (
                <p className="field-error">{errors['security.httpNodeAuth.user']}</p>
              )}
            </label>

            <label className="form-field">
              <span>Password (MD5 hash)</span>
              <input
                type="password"
                value={value.httpNodeAuth.pass}
                onChange={(e) =>
                  updateField('httpNodeAuth', {
                    ...value.httpNodeAuth!,
                    pass: e.target.value,
                  })
                }
              />
              {errors['security.httpNodeAuth.pass'] && (
                <p className="field-error">{errors['security.httpNodeAuth.pass']}</p>
              )}
            </label>
          </div>
        )}
      </div>
    </article>
  )
}
