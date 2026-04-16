import { useState } from 'react'
import { SecurityConfig } from '../../../types/config'
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
    <article className="surface-card border border-base-300/60 p-6 md:p-7 space-y-6">
      <div className="config-section-head">
        <p className="config-section-kicker">Identity</p>
        <h3 className="config-section-title">Security</h3>
        <p className="config-section-copy">
          Manage credential encryption, session expiry, and the runtime authentication blocks written into `settings.js`.
        </p>
      </div>

      <div className="config-section-card space-y-3">
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
                className="action-btn-ghost"
              >
                {showSecret ? 'Hide' : 'Show'}
              </button>
              <button
                type="button"
                onClick={() => updateField('credentialSecret', generateRandomSecret())}
                className="action-btn-ghost"
              >
                Generate
              </button>
           </div>
         </div>
         {errors['security.credentialSecret'] && (
           <span id="security-credentialSecret-error" className="form-field-error-msg">
             <svg className="w-4 h-4 flex-shrink-0" fill="currentColor" viewBox="0 0 20 20" xmlns="http://www.w3.org/2000/svg">
               <path fillRule="evenodd" d="M18.101 12.93a1 1 0 00-1.414-1.414L11 14.586l-2.687-2.687a1 1 0 00-1.414 1.414l4.1 4.1a1 1 0 001.414 0l8.101-8.101z" clipRule="evenodd" />
               <path fillRule="evenodd" d="M10 2a8 8 0 100 16 8 8 0 000-16zm0 14a6 6 0 110-12 6 6 0 010 12z" clipRule="evenodd" />
             </svg>
             <span>{errors['security.credentialSecret']}</span>
           </span>
         )}
        </div>

      <div className="config-section-card space-y-3">
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
         </div>
          <div className="config-chip-row">
            {presets.map((p) => (
              <button
                key={p.value}
                type="button"
                onClick={() => updateField('sessionExpiryTime', p.value)}
                className={`config-chip-btn ${value.sessionExpiryTime === p.value ? 'config-chip-btn-active' : ''}`}
              >
                {p.label}
              </button>
           ))}
         </div>
         {errors['security.sessionExpiryTime'] && (
           <span id="security-sessionExpiryTime-error" className="form-field-error-msg">
             <svg className="w-4 h-4 flex-shrink-0" fill="currentColor" viewBox="0 0 20 20" xmlns="http://www.w3.org/2000/svg">
               <path fillRule="evenodd" d="M18.101 12.93a1 1 0 00-1.414-1.414L11 14.586l-2.687-2.687a1 1 0 00-1.414 1.414l4.1 4.1a1 1 0 001.414 0l8.101-8.101z" clipRule="evenodd" />
               <path fillRule="evenodd" d="M10 2a8 8 0 100 16 8 8 0 000-16zm0 14a6 6 0 110-12 6 6 0 010 12z" clipRule="evenodd" />
             </svg>
             <span>{errors['security.sessionExpiryTime']}</span>
           </span>
         )}
       </div>

      {/* Admin Authentication */}
      <div className="divider"></div>
      <label className="config-toggle-row cursor-pointer">
          <input
            type="checkbox"
            className="checkbox"
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
          <span className="config-toggle-copy">
            <span className="config-toggle-title">Admin Authentication</span>
            <span className="config-toggle-hint">Protect the editor and admin API with user credentials or an external strategy.</span>
          </span>
      </label>

        {showAdminAuth && value.adminAuth && (
         <div className="config-subsection space-y-4">
          <div>
            <p className="config-subsection-title">Admin authentication block</p>
            <p className="config-subsection-copy">Users and permissions here map directly to the `adminAuth` section in `settings.js`.</p>
          </div>
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
                  <div key={idx} className="config-subsection space-y-3">
                    <FormField
                      id={`admin-auth-user-${idx}-username`}
                      label="Username"
                     type="text"
                     placeholder="Username"
                     value={user.username}
                     onChange={(e) => {
                       const users = [...value.adminAuth!.users]
                       users[idx] = { ...user, username: e }
                       updateField('adminAuth', { ...value.adminAuth!, users })
                     }}
                   />
                   <FormField
                     id={`admin-auth-user-${idx}-password`}
                     label="Password"
                     type="password"
                     placeholder="Password"
                     value={user.password}
                     onChange={(e) => {
                       const users = [...value.adminAuth!.users]
                       users[idx] = { ...user, password: e }
                       updateField('adminAuth', { ...value.adminAuth!, users })
                     }}
                   />
                   <div className="form-control">
                     <label className="label">
                       <span className="label-text font-medium">Permissions</span>
                     </label>
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
                   </div>
                    <button
                      type="button"
                      onClick={() => {
                        const users = value.adminAuth!.users.filter((_, i) => i !== idx)
                        updateField('adminAuth', { ...value.adminAuth!, users })
                      }}
                      className="action-btn-danger"
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
                  className="action-btn-ghost"
                >
                  + Add User
                </button>
             </div>
           )}
        </div>
      )}

      {/* HTTP Node Auth */}
      <div className="divider"></div>
      <label className="config-toggle-row cursor-pointer">
          <input
            type="checkbox"
            className="checkbox"
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
          <span className="config-toggle-copy">
            <span className="config-toggle-title">HTTP Node Auth</span>
            <span className="config-toggle-hint">Require credentials for HTTP In nodes and other exposed runtime endpoints.</span>
          </span>
      </label>

        {showHttpNodeAuth && value.httpNodeAuth && (
         <div className="config-subsection space-y-4">
          <div>
            <p className="config-subsection-title">HTTP node credentials</p>
            <p className="config-subsection-copy">Passwords are stored as MD5 hashes, matching Node-RED's expected format.</p>
          </div>
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
