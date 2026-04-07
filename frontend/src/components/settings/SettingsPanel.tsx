import { useEffect, useState, useMemo } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useSearchParams } from 'react-router-dom'
import { api, APIRequestError } from '../../api'
import { FullAppConfig, FieldError, ExtendedConfigValidationResult, defaultFullAppConfig } from '../../types/config'
import { TabBar } from './TabBar'
import { ServerSection } from './sections/ServerSection'
import { SecuritySection } from './sections/SecuritySection'
import { EditorThemeSection } from './sections/EditorThemeSection'
import { FlowsSection } from './sections/FlowsSection'
import { ContextStorageSection } from './sections/ContextStorageSection'
import { LoggingSection } from './sections/LoggingSection'
import { RuntimeSection } from './sections/RuntimeSection'
import { HTTPSSection } from './sections/HTTPSSection'
import { NodeReconnectSection } from './sections/NodeReconnectSection'
import { PaletteSection } from './sections/PaletteSection'

type SettingsPanelProps = {
  config?: FullAppConfig
  loading: boolean
  onSaved: (restartRequired: boolean) => Promise<void>
  onError: (message: string) => void
}

export function SettingsPanel({ config, loading, onSaved, onError }: SettingsPanelProps) {
  const queryClient = useQueryClient()
  const [searchParams, setSearchParams] = useSearchParams()
  const activeSection = searchParams.get('section') ?? 'server'

  // Local edit state
  const [localConfig, setLocalConfig] = useState<FullAppConfig | null>(null)
  const [dirtySections, setDirtySections] = useState<Set<string>>(new Set())
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({})
  const [errorSections, setErrorSections] = useState<Set<string>>(new Set())
  const [saving, setSaving] = useState(false)

  // Initialize from server config
  useEffect(() => {
    if (config && !localConfig) {
      setLocalConfig(config)
      setDirtySections(new Set())
      setFieldErrors({})
      setErrorSections(new Set())
    }
  }, [config, localConfig])

  // Track dirty sections
  const updateSection = (sectionKey: keyof FullAppConfig, value: any) => {
    if (!localConfig) return
    const updated = { ...localConfig, [sectionKey]: value }
    setLocalConfig(updated)
    setDirtySections((prev) => new Set(prev).add(sectionKey))
  }

  // Save section
  const saveMutation = useMutation({
    mutationFn: async (configToSave: FullAppConfig) => {
      setSaving(true)
      try {
        const result = await api.applyFullConfig(configToSave)
        return result
      } finally {
        setSaving(false)
      }
    },
    onSuccess: async (result) => {
      if (result.valid) {
        setDirtySections(new Set())
        setFieldErrors({})
        setErrorSections(new Set())
        await queryClient.invalidateQueries({ queryKey: ['config'] })
        await onSaved(result.restartRequired)
      } else {
        // Show validation errors
        const errors: Record<string, string> = {}
        const errSections = new Set<string>()
        for (const error of result.errors) {
          errors[error.field] = error.message
          const section = error.field.split('.')[0]
          errSections.add(section)
        }
        setFieldErrors(errors)
        setErrorSections(errSections)
        onError('Configuration has validation errors. Please review and try again.')
      }
    },
    onError: (error) => {
      const message = error instanceof APIRequestError ? error.message : 'Save failed'
      onError(message)
      setSaving(false)
    },
  })

  const handleSave = () => {
    if (!localConfig) return
    saveMutation.mutate(localConfig)
  }

  if (loading || !localConfig) {
    return (
      <section className="settings-panel">
        <p className="muted">Loading configuration...</p>
      </section>
    )
  }

  return (
    <section className="settings-panel">
      <TabBar
        activeSection={activeSection}
        onChange={(section) => setSearchParams({ section })}
        dirtyTabs={dirtySections}
        errorTabs={errorSections}
      />

      <div className="settings-content">
        {activeSection === 'server' && (
          <ServerSection
            value={localConfig.server}
            onChange={(val) => updateSection('server', val)}
            errors={fieldErrors}
          />
        )}
        {activeSection === 'security' && (
          <SecuritySection
            value={localConfig.security}
            onChange={(val) => updateSection('security', val)}
            errors={fieldErrors}
          />
        )}
        {activeSection === 'editorTheme' && (
          <EditorThemeSection
            value={localConfig.editorTheme}
            onChange={(val) => updateSection('editorTheme', val)}
            errors={fieldErrors}
          />
        )}
        {activeSection === 'flows' && (
          <FlowsSection
            value={localConfig.flows}
            onChange={(val) => updateSection('flows', val)}
            errors={fieldErrors}
          />
        )}
        {activeSection === 'contextStorage' && (
          <ContextStorageSection
            value={localConfig.contextStorage}
            onChange={(val) => updateSection('contextStorage', val)}
            errors={fieldErrors}
          />
        )}
        {activeSection === 'logging' && (
          <LoggingSection
            value={localConfig.logging}
            onChange={(val) => updateSection('logging', val)}
            errors={fieldErrors}
          />
        )}
        {activeSection === 'runtime' && (
          <RuntimeSection
            value={localConfig.runtime}
            onChange={(val) => updateSection('runtime', val)}
            errors={fieldErrors}
          />
        )}
        {activeSection === 'https' && (
          <HTTPSSection
            value={localConfig.https}
            onChange={(val) => updateSection('https', val)}
            errors={fieldErrors}
          />
        )}
        {activeSection === 'nodeReconnect' && (
          <NodeReconnectSection
            value={localConfig.nodeReconnect}
            onChange={(val) => updateSection('nodeReconnect', val)}
            errors={fieldErrors}
          />
        )}
        {activeSection === 'palette' && (
          <PaletteSection
            value={localConfig.palette}
            onChange={(val) => updateSection('palette', val)}
            errors={fieldErrors}
          />
        )}
      </div>

      <div className="settings-actions">
        <button
          className="primary-button"
          onClick={handleSave}
          disabled={saveMutation.isPending || dirtySections.size === 0}
        >
          {saveMutation.isPending ? 'Saving...' : 'Save changes'}
        </button>
      </div>

      {saveMutation.data?.restartRequired && (
        <section className="inline-notice warn">
          <strong>Restart required</strong>
          <p>Changes have been saved. Restart Node-RED to apply them.</p>
        </section>
      )}
    </section>
  )
}
