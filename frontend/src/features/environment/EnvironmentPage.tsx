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
          <p className="text-sm font-semibold text-base-content/70 uppercase tracking-wide">Runtime</p>
          <h2 className="text-3xl font-bold text-base-content mt-1">Environment</h2>
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
