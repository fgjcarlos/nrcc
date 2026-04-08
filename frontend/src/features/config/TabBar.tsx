type TabBarProps = {
  activeSection: string
  onChange: (section: string) => void
  dirtyTabs: Set<string>       // tabs with unsaved changes
  errorTabs: Set<string>       // tabs with validation errors
}

const SECTIONS = [
  { id: 'server', label: 'Server' },
  { id: 'security', label: 'Security' },
  { id: 'editorTheme', label: 'Editor Theme' },
  { id: 'flows', label: 'Flows' },
  { id: 'contextStorage', label: 'Context Storage' },
  { id: 'logging', label: 'Logging' },
  { id: 'runtime', label: 'Runtime' },
  { id: 'https', label: 'HTTPS' },
  { id: 'nodeReconnect', label: 'Node Reconnect' },
  { id: 'palette', label: 'Palette' },
]

export function TabBar({ activeSection, onChange, dirtyTabs, errorTabs }: TabBarProps) {
  return (
    <div className="config-tabs">
      <div className="tabs-scroll">
        {SECTIONS.map((section) => {
          const isDirty = dirtyTabs.has(section.id)
          const hasError = errorTabs.has(section.id)
          const isActive = activeSection === section.id

          return (
            <button
              key={section.id}
              className={`tab ${isActive ? 'active' : ''} ${hasError ? 'error' : ''} ${isDirty ? 'dirty' : ''}`}
              onClick={() => onChange(section.id)}
              type="button"
            >
              {section.label}
              {isDirty && <span className="indicator dirty" title="Unsaved changes">●</span>}
              {hasError && <span className="indicator error" title="Validation errors">●</span>}
            </button>
          )
        })}
      </div>
    </div>
  )
}
