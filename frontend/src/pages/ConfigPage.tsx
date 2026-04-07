import { useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from '../api'
import { FullAppConfig } from '../types/config'
import { SettingsPanel } from '../components/settings/SettingsPanel'

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
      <header className="topbar">
        <div>
          <p className="eyebrow">Runtime</p>
          <h2>Configuration</h2>
        </div>
      </header>

      {configQuery.error ? (
        <section className="inline-notice error">
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
