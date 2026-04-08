import { useState } from 'react'
import { EditorThemeConfig } from '../../../types/config'

type SectionProps<T> = {
  value: T
  onChange: (next: T) => void
  errors: Record<string, string>
}

export function EditorThemeSection({ value, onChange, errors }: SectionProps<EditorThemeConfig>) {
  const [showPage, setShowPage] = useState(!!value.page)
  const [showHeader, setShowHeader] = useState(!!value.header)
  const [showDeployButton, setShowDeployButton] = useState(!!value.deployButton)
  const [showCodeEditor, setShowCodeEditor] = useState(!!value.codeEditor)

  const updateField = <K extends keyof EditorThemeConfig>(key: K, val: EditorThemeConfig[K]) => {
    onChange({ ...value, [key]: val })
  }

  return (
    <article className="settings-section">
      <h3>Editor Theme</h3>

      <label className="form-field">
        <span>Theme</span>
        <input
          type="text"
          value={value.theme}
          onChange={(e) => updateField('theme', e.target.value)}
          placeholder="Theme name"
        />
      </label>

      <label className="form-toggle">
        <input
          type="checkbox"
          checked={value.tours}
          onChange={(e) => updateField('tours', e.target.checked)}
        />
        <span>Enable Tours</span>
      </label>

      <label className="form-toggle">
        <input
          type="checkbox"
          checked={value.userMenu}
          onChange={(e) => updateField('userMenu', e.target.checked)}
        />
        <span>Enable User Menu</span>
      </label>

      <label className="form-toggle">
        <input
          type="checkbox"
          checked={value.projects.enabled}
          onChange={(e) =>
            updateField('projects', { ...value.projects, enabled: e.target.checked })
          }
        />
        <span>Enable Projects</span>
      </label>

      {/* Code Editor */}
      <div className="collapsible-section">
        <label className="form-toggle">
          <input
            type="checkbox"
            checked={showCodeEditor}
            onChange={(e) => {
              setShowCodeEditor(e.target.checked)
              if (!e.target.checked) {
                updateField('codeEditor', undefined)
              } else {
                updateField('codeEditor', { lib: 'ace', options: {} })
              }
            }}
          />
          <span>Code Editor Settings</span>
        </label>

        {showCodeEditor && value.codeEditor && (
          <div className="collapsible-content">
            <label className="form-field">
              <span>Library</span>
              <div className="radio-group">
                <label>
                  <input
                    type="radio"
                    value="ace"
                    checked={value.codeEditor.lib === 'ace'}
                    onChange={(e) =>
                      updateField('codeEditor', {
                        ...value.codeEditor!,
                        lib: e.target.value as 'ace' | 'monaco',
                      })
                    }
                  />
                  ACE
                </label>
                <label>
                  <input
                    type="radio"
                    value="monaco"
                    checked={value.codeEditor.lib === 'monaco'}
                    onChange={(e) =>
                      updateField('codeEditor', {
                        ...value.codeEditor!,
                        lib: e.target.value as 'ace' | 'monaco',
                      })
                    }
                  />
                  Monaco
                </label>
              </div>
            </label>
          </div>
        )}
      </div>

      {/* Page Settings */}
      <div className="collapsible-section">
        <label className="form-toggle">
          <input
            type="checkbox"
            checked={showPage}
            onChange={(e) => {
              setShowPage(e.target.checked)
              if (!e.target.checked) {
                updateField('page', undefined)
              } else {
                updateField('page', { title: 'Node-RED', favicon: '', css: '' })
              }
            }}
          />
          <span>Page Settings</span>
        </label>

        {showPage && value.page && (
          <div className="collapsible-content">
            <label className="form-field">
              <span>Page Title</span>
              <input
                type="text"
                value={value.page.title}
                onChange={(e) =>
                  updateField('page', { ...value.page!, title: e.target.value })
                }
              />
            </label>

            <label className="form-field">
              <span>Favicon URL</span>
              <input
                type="text"
                value={value.page.favicon}
                onChange={(e) =>
                  updateField('page', { ...value.page!, favicon: e.target.value })
                }
              />
            </label>

            <label className="form-field">
              <span>Custom CSS</span>
              <textarea
                value={value.page.css}
                onChange={(e) =>
                  updateField('page', { ...value.page!, css: e.target.value })
                }
                rows={4}
                placeholder="CSS content"
              />
            </label>
          </div>
        )}
      </div>

      {/* Header Settings */}
      <div className="collapsible-section">
        <label className="form-toggle">
          <input
            type="checkbox"
            checked={showHeader}
            onChange={(e) => {
              setShowHeader(e.target.checked)
              if (!e.target.checked) {
                updateField('header', undefined)
              } else {
                updateField('header', { title: 'Node-RED', image: '', url: '' })
              }
            }}
          />
          <span>Header Settings</span>
        </label>

        {showHeader && value.header && (
          <div className="collapsible-content">
            <label className="form-field">
              <span>Header Title</span>
              <input
                type="text"
                value={value.header.title}
                onChange={(e) =>
                  updateField('header', { ...value.header!, title: e.target.value })
                }
              />
            </label>

            <label className="form-field">
              <span>Header Image</span>
              <input
                type="text"
                value={value.header.image}
                onChange={(e) =>
                  updateField('header', { ...value.header!, image: e.target.value })
                }
              />
            </label>

            <label className="form-field">
              <span>Header URL</span>
              <input
                type="text"
                value={value.header.url}
                onChange={(e) =>
                  updateField('header', { ...value.header!, url: e.target.value })
                }
              />
            </label>
          </div>
        )}
      </div>

      {/* Deploy Button Settings */}
      <div className="collapsible-section">
        <label className="form-toggle">
          <input
            type="checkbox"
            checked={showDeployButton}
            onChange={(e) => {
              setShowDeployButton(e.target.checked)
              if (!e.target.checked) {
                updateField('deployButton', undefined)
              } else {
                updateField('deployButton', { type: 'simple', label: 'Deploy' })
              }
            }}
          />
          <span>Deploy Button Settings</span>
        </label>

        {showDeployButton && value.deployButton && (
          <div className="collapsible-content">
            <label className="form-field">
              <span>Type</span>
              <select
                value={value.deployButton.type}
                onChange={(e) =>
                  updateField('deployButton', {
                    ...value.deployButton!,
                    type: e.target.value as 'simple' | 'confirm',
                  })
                }
              >
                <option value="simple">Simple</option>
                <option value="confirm">Confirm</option>
              </select>
            </label>

            <label className="form-field">
              <span>Label</span>
              <input
                type="text"
                value={value.deployButton.label}
                onChange={(e) =>
                  updateField('deployButton', {
                    ...value.deployButton!,
                    label: e.target.value,
                  })
                }
              />
            </label>
          </div>
        )}
      </div>
    </article>
  )
}
