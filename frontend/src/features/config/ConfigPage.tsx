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
      <header className="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between mb-6 md:mb-8">
        <div>
          <p className="text-xs uppercase tracking-[0.24em] text-base-content/50">Runtime</p>
          <h2 className="page-title text-3xl mt-1">Configuration</h2>
          <p className="mt-2 max-w-3xl text-sm text-base-content/65">
            Edit the generated `settings.js` through sectioned forms, previews, snapshots, and import tools.
          </p>
        </div>
        <div className="hidden md:inline-flex rounded-full border border-base-300/60 bg-base-100/55 px-4 py-2 text-xs font-semibold uppercase tracking-[0.18em] text-base-content/60">
          Full settings editor
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
