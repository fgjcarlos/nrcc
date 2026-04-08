import { useEffect, useState } from 'react'
import { useMutation } from '@tanstack/react-query'
import { api, type ManagedEnvState, type ManagedEnvVar } from '../../api'
import { formatErrorMessage } from '../../common/utils/format'

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
      <article className="panel">
        <div className="panel-header">
          <h3>Managed runtime variables</h3>
        </div>
        <p className="muted">Loading managed environment...</p>
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
    <article className="panel">
      <div className="panel-header">
        <h3>Managed runtime variables</h3>
      </div>
      <p className="muted">
        These variables are injected into the Node-RED runtime from `.env.managed`. Names prefixed with `NRCC_` and `PORT` are reserved.
      </p>

      <form
        className="env-form"
        onSubmit={(event) => {
          event.preventDefault()
          applyMutation.mutate(variables)
        }}
      >
        <div className="env-rows">
          {variables.map((variable, index) => (
            <div className="env-row" key={`${index}-${variable.name}`}>
              <label>
                <span>Name</span>
                <input
                  value={variable.name}
                  onChange={(event) => update(index, { name: event.target.value })}
                  placeholder="API_TOKEN"
                />
              </label>
              <label>
                <span>Value</span>
                <input
                  value={variable.value}
                  onChange={(event) => update(index, { value: event.target.value })}
                  placeholder="secret-value"
                />
              </label>
              <button className="ghost-button env-remove" type="button" onClick={() => removeRow(index)}>
                Remove
              </button>
            </div>
          ))}
        </div>

        <div className="config-actions">
          <button className="ghost-button" type="button" onClick={addRow} disabled={applyMutation.isPending}>
            Add variable
          </button>
          <button className="primary-button" type="submit" disabled={applyMutation.isPending}>
            {applyMutation.isPending ? 'Saving...' : 'Save environment'}
          </button>
        </div>
      </form>

      {message ? <p className="config-message">{message}</p> : null}
    </article>
  )
}
