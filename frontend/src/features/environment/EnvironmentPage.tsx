import type { ManagedEnvState } from '../../api'
import { InlineNotice } from '../../common/components'
import { formatErrorMessage } from '../../common/utils/format'
import { EnvironmentPanel } from './EnvironmentPanel'

export function EnvironmentPage({
  state,
  loading,
  error,
  onSaved,
  onError,
}: {
  state?: ManagedEnvState
  loading: boolean
  error: unknown
  onSaved: () => Promise<void>
  onError: (message: string) => void
}) {
  return (
    <>
      <header className="flex flex-col sm:flex-row sm:items-center gap-6 mb-8">
        <div>
          <p className="text-xs uppercase tracking-[0.24em] text-base-content/50">Runtime</p>
          <h2 className="text-3xl font-bold tracking-tight text-base-content mt-1">Environment</h2>
          <p className="mt-2 max-w-2xl text-sm text-base-content/65">
            Manage `.env.managed` variables with the same dense, card-based workflow used in `frontend_old`.
          </p>
        </div>
      </header>

      {error ? (
        <InlineNotice
          tone="error"
          title="Environment unavailable"
          detail={formatErrorMessage(error, 'Managed runtime variables could not be loaded.')}
        />
      ) : null}

      <EnvironmentPanel state={state} loading={loading} onSaved={onSaved} onError={onError} />
    </>
  )
}
