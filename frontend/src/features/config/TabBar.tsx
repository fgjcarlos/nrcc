import { KeyboardEvent, useRef } from 'react'

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
  const tabRefs = useRef<Record<string, HTMLButtonElement | null>>({})

  function activateSection(sectionId: string) {
    onChange(sectionId)
    tabRefs.current[sectionId]?.focus()
  }

  function handleTabKeyDown(event: KeyboardEvent<HTMLButtonElement>, currentIndex: number) {
    if (event.key === 'ArrowRight' || event.key === 'ArrowDown') {
      event.preventDefault()
      activateSection(SECTIONS[(currentIndex + 1) % SECTIONS.length].id)
    }

    if (event.key === 'ArrowLeft' || event.key === 'ArrowUp') {
      event.preventDefault()
      activateSection(SECTIONS[(currentIndex - 1 + SECTIONS.length) % SECTIONS.length].id)
    }

    if (event.key === 'Home') {
      event.preventDefault()
      activateSection(SECTIONS[0].id)
    }

    if (event.key === 'End') {
      event.preventDefault()
      activateSection(SECTIONS[SECTIONS.length - 1].id)
    }
  }

  return (
    <div className="surface-panel section-tabbar border border-base-300/60 mb-6" role="tablist" aria-label="Configuration sections">
      {SECTIONS.map((section, index) => {
        const isDirty = dirtyTabs.has(section.id)
        const hasError = errorTabs.has(section.id)
        const isActive = activeSection === section.id

        return (
          <button
            key={section.id}
            ref={(node) => {
              tabRefs.current[section.id] = node
            }}
            id={`config-tab-${section.id}`}
            className={`section-tab ${isActive ? 'section-tab-active' : ''} ${hasError ? 'text-error' : ''}`}
            onClick={() => onChange(section.id)}
            onKeyDown={(event) => handleTabKeyDown(event, index)}
            type="button"
            role="tab"
            tabIndex={isActive ? 0 : -1}
            aria-selected={isActive}
            aria-controls={`config-panel-${section.id}`}
            title={hasError ? 'Validation errors' : isDirty ? 'Unsaved changes' : undefined}
          >
            <span className="min-w-0 text-left">
              <span className="flex items-center gap-2">
                {section.label}
                {isDirty && (
                  <>
                    <span className="status-dot status-dot-warning" aria-hidden="true" title="Unsaved changes" />
                    <span className="sr-only">Unsaved changes</span>
                  </>
                )}
                {hasError && (
                  <>
                    <span className="status-dot status-dot-error" aria-hidden="true" title="Validation errors" />
                    <span className="sr-only">Validation errors</span>
                  </>
                )}
              </span>
              <span className="section-tab-copy">{section.copy}</span>
            </span>
          </button>
        )
      })}
    </div>
  )
}
