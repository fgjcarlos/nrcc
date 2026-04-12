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
import { LivePreviewPanel } from './LivePreviewPanel'
import { AdvancedJSONEditor } from './AdvancedJSONEditor'
import { SnapshotPanel } from './SnapshotPanel'
import { ImportDialog } from './ImportDialog'
import { validateFullConfig } from './validation'

type SettingsPanelProps = {
  config?: FullAppConfig
  loading: boolean
  onSaved: (restartRequired: boolean) => Promise<void>
  onError: (message: string) => void
  onToast?: (message: string, type: 'success' | 'error' | 'info') => void
}

export function SettingsPanel({ config, loading, onSaved, onError, onToast }: SettingsPanelProps) {
  const queryClient = useQueryClient()
  const [searchParams, setSearchParams] = useSearchParams()
  const activeSection = searchParams.get('section') ?? 'server'

  // Local edit state
  const [localConfig, setLocalConfig] = useState<FullAppConfig | null>(null)
  const [originalConfig, setOriginalConfig] = useState<FullAppConfig | null>(null)
  const [dirtySections, setDirtySections] = useState<Set<string>>(new Set())
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({})
  const [errorSections, setErrorSections] = useState<Set<string>>(new Set())
  const [saving, setSaving] = useState(false)

  // Panel open/close states
  const [showPreview, setShowPreview] = useState(false)
  const [showJSONEditor, setShowJSONEditor] = useState(false)
  const [showSnapshots, setShowSnapshots] = useState(false)
  const [showImport, setShowImport] = useState(false)

  // Initialize from server config
  useEffect(() => {
    if (config && !localConfig) {
      setLocalConfig(config)
      setOriginalConfig(config)
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
    
    // Clear field error for this section when user modifies it
    const fieldsToKeep = Object.keys(fieldErrors).filter(
      (key) => !key.startsWith(sectionKey + '.')
    )
    setFieldErrors(Object.fromEntries(
      Object.entries(fieldErrors).filter(([key]) => fieldsToKeep.includes(key))
    ))
  }

  // Client-side validation
  const validateBeforeSave = (cfg: FullAppConfig): FieldError[] => {
    return validateFullConfig(cfg)
  }

  // Save section
  const saveMutation = useMutation({
    mutationFn: async (configToSave: FullAppConfig) => {
      // Client-side validation first
      const clientErrors = validateBeforeSave(configToSave)
      if (clientErrors.length > 0) {
        const errorMap: Record<string, string> = {}
        const errorSects = new Set<string>()
        for (const err of clientErrors) {
          errorMap[err.field] = err.message
          const section = err.field.split('.')[0]
          errorSects.add(section)
        }
        setFieldErrors(errorMap)
        setErrorSections(errorSects)
        throw new Error('Validation failed')
      }

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
        setOriginalConfig(localConfig)
        await queryClient.invalidateQueries({ queryKey: ['config'] })
        
        const successMsg = result.restartRequired
          ? 'Configuration saved successfully. Restart Node-RED to apply changes.'
          : 'Configuration saved successfully'
        onToast?.(successMsg, 'success')
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
        onToast?.('Configuration has validation errors', 'error')
      }
    },
    onError: (error) => {
      const message = error instanceof APIRequestError ? error.message : 'Save failed'
      if (message !== 'Validation failed') {
        onError(message)
        onToast?.(message, 'error')
      }
      setSaving(false)
    },
  })

  const handleSave = () => {
    if (!localConfig) return
    saveMutation.mutate(localConfig)
  }

  const handleJSONApply = (cfg: FullAppConfig) => {
    setLocalConfig(cfg)
    setDirtySections(new Set(Object.keys(cfg) as any[]))
    onToast?.('Configuration loaded from JSON', 'info')
  }

  const handleSnapshotRestored = () => {
    // Reload config from server
    queryClient.invalidateQueries({ queryKey: ['config'] })
    setDirtySections(new Set())
    setFieldErrors({})
    setErrorSections(new Set())
    onToast?.('Configuration restored from snapshot', 'success')
  }

  const handleImported = (cfg: FullAppConfig) => {
    setLocalConfig(cfg)
    setDirtySections(new Set(Object.keys(cfg) as any[]))
    onToast?.('Configuration imported from settings.js', 'info')
  }

  if (loading || !localConfig) {
    return (
      <section className="space-y-6">
        <p className="text-sm text-base-content/60">Loading configuration...</p>
      </section>
    )
  }

  return (
    <section className="space-y-6">
      <TabBar
        activeSection={activeSection}
        onChange={(section) => setSearchParams({ section })}
        dirtyTabs={dirtySections}
        errorTabs={errorSections}
      />

      <div className="space-y-6">
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

      <div className="flex flex-wrap gap-3">
        <button
          className="btn btn-primary"
          onClick={handleSave}
          disabled={saveMutation.isPending || dirtySections.size === 0}
        >
          {saveMutation.isPending ? 'Saving...' : 'Save changes'}
        </button>
        <button
          className="btn btn-ghost"
          onClick={() => setShowPreview(!showPreview)}
          title="Preview the rendered settings.js"
        >
          Preview settings.js
        </button>
        <button
          className="btn btn-ghost"
          onClick={() => setShowJSONEditor(true)}
          title="Edit configuration as raw JSON"
        >
          Raw JSON
        </button>
        <button
          className="btn btn-ghost"
          onClick={() => setShowSnapshots(true)}
          title="Manage configuration snapshots"
        >
          Backups
        </button>
        <button
          className="btn btn-ghost"
          onClick={() => setShowImport(true)}
          title="Import from a settings.js file"
        >
          Import settings.js
        </button>
      </div>

      <LivePreviewPanel
        config={localConfig}
        isOpen={showPreview}
        onToggle={() => setShowPreview(!showPreview)}
      />

      {showJSONEditor && (
        <AdvancedJSONEditor
          config={localConfig}
          onApply={handleJSONApply}
          onClose={() => setShowJSONEditor(false)}
        />
      )}

      <SnapshotPanel
        isOpen={showSnapshots}
        onClose={() => setShowSnapshots(false)}
        onRestored={handleSnapshotRestored}
      />

      <ImportDialog
        isOpen={showImport}
        onClose={() => setShowImport(false)}
        onImported={handleImported}
      />

      {saveMutation.data?.restartRequired && (
        <section className="alert alert-warning">
          <strong>Restart required</strong>
          <p>Changes have been saved. Restart Node-RED to apply them.</p>
        </section>
      )}
    </section>
  )
}
