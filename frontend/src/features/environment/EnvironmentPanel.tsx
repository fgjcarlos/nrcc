import { useEffect, useState } from 'react'
import { useMutation } from '@tanstack/react-query'
import { api, type ManagedEnvState, type ManagedEnvVar } from '../../api'
import { formatErrorMessage } from '../../common/utils/format'
import { InlineNotice } from '../../common/components/InlineNotice'
import { FormField } from '../../components/forms'

export function EnvironmentPanel({
  state,
  loading,
  onSaved,
  onError,
}: {
  state?: ManagedEnvState
  loading: boolean
  onSaved: () => Promise<void>
  onError: (message: string) => void
}) {
  const [variables, setVariables] = useState<ManagedEnvVar[]>([])
  const [message, setMessage] = useState('')

  useEffect(() => {
    if (state) {
      setVariables(state.variables.length > 0 ? state.variables : [{ name: '', value: '' }])
      setMessage('')
    }
  }, [state])

  const applyMutation = useMutation({
    mutationFn: (payload: ManagedEnvVar[]) => api.applyEnvironment(payload),
    onSuccess: async () => {
      setMessage('Managed environment saved. Restart Node-RED to apply the changes.')
      await onSaved()
    },
    onError: (error) => {
      const next = formatErrorMessage(error, 'Save failed')
      setMessage(next)
      onError(next)
    },
  })

  if (loading || !state) {
    return (
      <article className="surface-card border border-base-300/60 p-6 md:p-7">
        <div>
          <h3 className="text-lg font-semibold text-base-content">Managed runtime variables</h3>
          <p className="text-base-content/60 text-sm">Loading managed environment...</p>
        </div>
      </article>
    )
  }

  function update(index: number, patch: Partial<ManagedEnvVar>) {
    setVariables((current) =>
      current.map((variable, currentIndex) =>
        currentIndex === index ? { ...variable, ...patch } : variable,
      ),
    )
  }

  function addRow() {
    setVariables((current) => [...current, { name: '', value: '' }])
  }

  function removeRow(index: number) {
    setVariables((current) => {
      const next = current.filter((_, currentIndex) => currentIndex !== index)
      return next.length > 0 ? next : [{ name: '', value: '' }]
    })
  }

  return (
    <article className="surface-card border border-base-300/60 p-6 md:p-7">
      <div>
        <h3 className="text-lg font-semibold text-base-content">Managed runtime variables</h3>
        <p className="text-base-content/60 text-sm mb-6">
          These variables are injected into the Node-RED runtime from `.env.managed`. Names prefixed with `NRCC_` and `PORT` are reserved.
        </p>

        <form
          className="space-y-6"
          onSubmit={(event) => {
            event.preventDefault()
            applyMutation.mutate(variables)
          }}
        >
           <div className="space-y-4">
              {variables.map((variable, index) => (
                <div key={`${index}-${variable.name}`} className="surface-panel border border-base-300/60 p-4 md:p-5 rounded-2xl space-y-4">
                  <div className="flex flex-col gap-3 md:flex-row md:items-start md:justify-between">
                    <div>
                      <p className="text-xs uppercase tracking-[0.22em] text-base-content/50">Variable {index + 1}</p>
                      <p className="mt-1 text-sm text-base-content/60">Values are written to `.env.managed` in order.</p>
                    </div>
                    <button
                      className="action-btn-ghost"
                      type="button"
                      onClick={() => removeRow(index)}
                      aria-label={`Remove variable ${index + 1}`}
                    >
                     Remove
                    </button>
                  </div>

                  <div className="grid gap-4 md:grid-cols-2">
                    <FormField
                      id={`env-var-${index}-name`}
                      label="Name"
                      type="text"
                      placeholder="API_TOKEN"
                      value={variable.name}
                      onChange={(val) => update(index, { name: val })}
                    />
                    <FormField
                      id={`env-var-${index}-value`}
                      label="Value"
                      type="text"
                      placeholder="secret-value"
                      value={variable.value}
                      onChange={(val) => update(index, { value: val })}
                    />
                  </div>
                </div>
              ))}
            </div>

          <div className="flex flex-col gap-3 sm:flex-row">
            <button className="action-btn-ghost" type="button" onClick={addRow} disabled={applyMutation.isPending}>
              + Add variable
            </button>
            <button className="action-btn-primary" type="submit" disabled={applyMutation.isPending}>
              {applyMutation.isPending ? 'Saving...' : 'Save environment'}
            </button>
          </div>
        </form>

        {message ? (
          <InlineNotice
            tone={message.includes('failed') ? 'error' : 'warn'}
            title={message.includes('failed') ? 'Save failed' : 'Saved'}
            detail={message}
          />
        ) : null}
      </div>
    </article>
  )
}
