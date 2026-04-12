import { useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from '../../api'
import { FullAppConfig } from '../../types/config'
import { SettingsPanel } from './SettingsPanel'

type ConfigPageCallbacks = {
  onSaved?: (restartRequired: boolean) => void
  onError?: (message: string) => void
  onToast?: (message: string, type: 'success' | 'error' | 'info') => void
}

export function ConfigPage({ onSaved, onError, onToast }: ConfigPageCallbacks = {}) {
  const queryClient = useQueryClient()

  const configQuery = useQuery({
    queryKey: ['full-config'],
    queryFn: api.fullConfig,
  })

  const handleSaved = async (restartRequired: boolean) => {
    await queryClient.invalidateQueries({ queryKey: ['full-config'] })
    onSaved?.(restartRequired)
  }

  const handleError = (message: string) => {
    onError?.(message)
  }

  return (
    <>
      <header className="flex flex-col sm:flex-row sm:items-center gap-6 mb-8">
        <div>
          <p className="text-sm font-semibold text-base-content/70 uppercase tracking-wide">Runtime</p>
          <h2 className="text-3xl font-bold text-base-content mt-1">Configuration</h2>
        </div>
      </header>

      {configQuery.error ? (
        <section className="alert alert-error mb-6">
          <strong>Configuration unavailable</strong>
          <p>
            {configQuery.error instanceof Error
              ? configQuery.error.message
              : 'Full configuration could not be loaded.'}
          </p>
        </section>
      ) : null}

      <SettingsPanel
        config={configQuery.data}
        loading={configQuery.isLoading}
        onSaved={handleSaved}
        onError={handleError}
        onToast={onToast}
      />
    </>
  )
}
