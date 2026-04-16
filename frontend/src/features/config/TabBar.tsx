type TabBarProps = {
  activeSection: string
  onChange: (section: string) => void
  dirtyTabs: Set<string>       // tabs with unsaved changes
  errorTabs: Set<string>       // tabs with validation errors
}

const SECTIONS = [
  { id: 'server', label: 'Server', copy: 'Ports and roots' },
  { id: 'security', label: 'Security', copy: 'Secrets and auth' },
  { id: 'editorTheme', label: 'Editor Theme', copy: 'Branding and UI' },
  { id: 'flows', label: 'Flows', copy: 'Files and storage' },
  { id: 'contextStorage', label: 'Context Storage', copy: 'Stores and defaults' },
  { id: 'logging', label: 'Logging', copy: 'Console behavior' },
  { id: 'runtime', label: 'Runtime', copy: 'Execution limits' },
  { id: 'https', label: 'HTTPS', copy: 'Certificates' },
  { id: 'nodeReconnect', label: 'Node Reconnect', copy: 'Reconnect timing' },
  { id: 'palette', label: 'Palette', copy: 'Category order' },
]

export function TabBar({ activeSection, onChange, dirtyTabs, errorTabs }: TabBarProps) {
  return (
    <div className="surface-panel section-tabbar border border-base-300/60 mb-6">
      {SECTIONS.map((section) => {
        const isDirty = dirtyTabs.has(section.id)
        const hasError = errorTabs.has(section.id)
        const isActive = activeSection === section.id

        return (
          <button
            key={section.id}
            className={`section-tab ${isActive ? 'section-tab-active' : ''} ${hasError ? 'text-error' : ''}`}
            onClick={() => onChange(section.id)}
            type="button"
            title={hasError ? 'Validation errors' : isDirty ? 'Unsaved changes' : undefined}
          >
            <span>
              <span className="flex items-center gap-2">
                {section.label}
                {isDirty && <span className="status-dot status-dot-warning" title="Unsaved changes" />}
                {hasError && <span className="status-dot status-dot-error" title="Validation errors" />}
              </span>
              <span className="section-tab-copy">{section.copy}</span>
            </span>
          </button>
        )
      })}
    </div>
  )
}
