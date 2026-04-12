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
    <div className="flex border-b border-base-300 overflow-x-auto gap-2 mb-6">
      {SECTIONS.map((section) => {
        const isDirty = dirtyTabs.has(section.id)
        const hasError = errorTabs.has(section.id)
        const isActive = activeSection === section.id

        return (
          <button
            key={section.id}
            className={`
              px-4 py-3 text-sm font-medium whitespace-nowrap transition-colors
              border-b-2
              ${isActive 
                ? 'border-primary text-primary' 
                : 'border-transparent text-base-content/60 hover:text-base-content'
              }
              ${hasError ? 'text-error' : ''}
            `}
            onClick={() => onChange(section.id)}
            type="button"
            title={hasError ? 'Validation errors' : isDirty ? 'Unsaved changes' : undefined}
          >
            <span className="flex items-center gap-2">
              {section.label}
              {isDirty && <span className="badge badge-warning badge-sm" title="Unsaved changes">●</span>}
              {hasError && <span className="badge badge-error badge-sm" title="Validation errors">●</span>}
            </span>
          </button>
        )
      })}
    </div>
  )
}
