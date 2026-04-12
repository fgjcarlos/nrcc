import { useEffect, useState } from 'react'
import { useMutation } from '@tanstack/react-query'
import { api, type ManagedEnvState, type ManagedEnvVar } from '../../api'
import { formatErrorMessage } from '../../common/utils/format'
import { InlineNotice } from '../../common/components/InlineNotice'

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
      <article className="card bg-base-200 shadow-elevation-2 rounded-lg">
        <div className="card-body">
          <h3 className="card-title text-lg font-semibold text-base-content">Managed runtime variables</h3>
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
    <article className="card bg-base-200 shadow-elevation-2 rounded-lg">
      <div className="card-body">
        <h3 className="card-title text-lg font-semibold text-base-content">Managed runtime variables</h3>
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
              <div key={`${index}-${variable.name}`} className="flex flex-col sm:flex-row gap-4 items-end">
                <div className="form-control flex-1">
                  <label className="label">
                    <span className="label-text font-medium">Name</span>
                  </label>
                  <input
                    type="text"
                    className="input input-bordered bg-base-100"
                    value={variable.name}
                    onChange={(event) => update(index, { name: event.target.value })}
                    placeholder="API_TOKEN"
                  />
                </div>
                <div className="form-control flex-1">
                  <label className="label">
                    <span className="label-text font-medium">Value</span>
                  </label>
                  <input
                    type="text"
                    className="input input-bordered bg-base-100"
                    value={variable.value}
                    onChange={(event) => update(index, { value: event.target.value })}
                    placeholder="secret-value"
                  />
                </div>
                <button className="btn btn-ghost btn-sm" type="button" onClick={() => removeRow(index)}>
                  Remove
                </button>
              </div>
            ))}
          </div>

          <div className="flex gap-3">
            <button className="btn btn-ghost btn-sm" type="button" onClick={addRow} disabled={applyMutation.isPending}>
              + Add variable
            </button>
            <button className="btn btn-primary btn-sm" type="submit" disabled={applyMutation.isPending}>
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
