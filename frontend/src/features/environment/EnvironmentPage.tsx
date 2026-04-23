import { useQueryClient } from '@tanstack/react-query'
import { InlineNotice } from '../../common/components'
import { formatErrorMessage } from '../../common/utils/format'
import { useToasts } from '../shell/useToasts'
import { EnvironmentPanel } from './EnvironmentPanel'
import { useEnvironmentData } from './useEnvironmentData'

export function EnvironmentPage() {
  const { state, loading, error } = useEnvironmentData()
  const queryClient = useQueryClient()
  const { pushToast } = useToasts()

  const handleSaved = async () => {
    await queryClient.invalidateQueries({ queryKey: ['environment'] })
    pushToast({
      tone: 'success',
      title: 'Environment saved',
      detail: 'Managed runtime variables were updated. Restart Node-RED to apply them.',
    })
  }

  const handleError = (message: string) => {
    pushToast({
      tone: 'error',
      title: 'Environment update failed',
      detail: message,
    })
  }
  return (
    <>
      <header className="flex flex-col sm:flex-row sm:items-center gap-6 mb-8">
        <div>
          <p className="text-xs uppercase tracking-[0.24em] text-base-content/50">Runtime</p>
          <h2 className="page-title text-3xl mt-1">Environment</h2>
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

      <EnvironmentPanel state={state} loading={loading} onSaved={handleSaved} onError={handleError} />
    </>
  )
}
