import { useState } from 'react'
import { EditorThemeConfig } from '../../../types/config'
import { FormField } from '../../../components/forms'

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
    <article className="space-y-6">
      <h3 className="text-xl font-semibold text-base-content">Editor Theme</h3>

      <FormField
        id="editor-theme-theme"
        label="Theme"
        type="text"
        value={value.theme}
        onChange={(v) => updateField('theme', v)}
        placeholder="Theme name"
      />

      <div className="form-control">
        <label className="label cursor-pointer gap-3">
          <input
            type="checkbox"
            className="checkbox checkbox-sm"
            checked={value.tours}
            onChange={(e) => updateField('tours', e.target.checked)}
          />
          <span className="label-text font-medium">Enable Tours</span>
        </label>
      </div>

      <div className="form-control">
        <label className="label cursor-pointer gap-3">
          <input
            type="checkbox"
            className="checkbox checkbox-sm"
            checked={value.userMenu}
            onChange={(e) => updateField('userMenu', e.target.checked)}
          />
          <span className="label-text font-medium">Enable User Menu</span>
        </label>
      </div>

      <div className="form-control">
        <label className="label cursor-pointer gap-3">
          <input
            type="checkbox"
            className="checkbox checkbox-sm"
            checked={value.projects.enabled}
            onChange={(e) =>
              updateField('projects', { ...value.projects, enabled: e.target.checked })
            }
          />
          <span className="label-text font-medium">Enable Projects</span>
        </label>
      </div>

      {/* Code Editor */}
      <div className="divider"></div>
      <div className="form-control">
        <label className="label cursor-pointer gap-3">
          <input
            type="checkbox"
            className="checkbox checkbox-sm"
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
          <span className="label-text font-medium">Code Editor Settings</span>
        </label>
      </div>

      {showCodeEditor && value.codeEditor && (
        <div className="pl-4 border-l-2 border-[color:var(--border-indent)] space-y-4">
          <label className="label">
            <span className="label-text font-medium">Library</span>
          </label>
          <div className="space-y-2">
            <label className="label cursor-pointer gap-2">
              <input
                type="radio"
                value="ace"
                className="radio radio-sm"
                checked={value.codeEditor.lib === 'ace'}
                onChange={(e) =>
                  updateField('codeEditor', {
                    ...value.codeEditor!,
                    lib: e.target.value as 'ace' | 'monaco',
                  })
                }
              />
              <span className="label-text text-sm">ACE</span>
            </label>
            <label className="label cursor-pointer gap-2">
              <input
                type="radio"
                value="monaco"
                className="radio radio-sm"
                checked={value.codeEditor.lib === 'monaco'}
                onChange={(e) =>
                  updateField('codeEditor', {
                    ...value.codeEditor!,
                    lib: e.target.value as 'ace' | 'monaco',
                  })
                }
              />
              <span className="label-text text-sm">Monaco</span>
            </label>
          </div>
        </div>
      )}

      {/* Page Settings */}
      <div className="divider"></div>
      <div className="form-control">
        <label className="label cursor-pointer gap-3">
          <input
            type="checkbox"
            className="checkbox checkbox-sm"
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
          <span className="label-text font-medium">Page Settings</span>
        </label>
      </div>

      {showPage && value.page && (
        <div className="pl-4 border-l-2 border-[color:var(--border-indent)] space-y-4">
           <FormField
             id="editor-page-title"
             label="Page Title"
             type="text"
             value={value.page.title}
             onChange={(v) =>
               updateField('page', { ...value.page!, title: v })
             }
           />

           <FormField
             id="editor-page-favicon"
             label="Favicon URL"
             type="text"
             value={value.page.favicon}
             onChange={(v) =>
               updateField('page', { ...value.page!, favicon: v })
             }
           />

          <div className="form-control">
            <label className="label">
              <span className="label-text font-medium">Custom CSS</span>
            </label>
            <textarea
              className="textarea textarea-bordered bg-base-100"
              value={value.page.css}
              onChange={(e) =>
                updateField('page', { ...value.page!, css: e.target.value })
              }
              rows={4}
              placeholder="CSS content"
            />
          </div>
        </div>
      )}

      {/* Header Settings */}
      <div className="divider"></div>
      <div className="form-control">
        <label className="label cursor-pointer gap-3">
          <input
            type="checkbox"
            className="checkbox checkbox-sm"
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
          <span className="label-text font-medium">Header Settings</span>
        </label>
      </div>

      {showHeader && value.header && (
        <div className="pl-4 border-l-2 border-[color:var(--border-indent)] space-y-4">
           <FormField
             id="editor-header-title"
             label="Header Title"
             type="text"
             value={value.header.title}
             onChange={(v) =>
               updateField('header', { ...value.header!, title: v })
             }
           />

           <FormField
             id="editor-header-image"
             label="Header Image"
             type="text"
             value={value.header.image}
             onChange={(v) =>
               updateField('header', { ...value.header!, image: v })
             }
           />

           <FormField
             id="editor-header-url"
             label="Header URL"
             type="text"
             value={value.header.url}
             onChange={(v) =>
               updateField('header', { ...value.header!, url: v })
             }
           />
        </div>
      )}

      {/* Deploy Button Settings */}
      <div className="divider"></div>
      <div className="form-control">
        <label className="label cursor-pointer gap-3">
          <input
            type="checkbox"
            className="checkbox checkbox-sm"
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
          <span className="label-text font-medium">Deploy Button Settings</span>
        </label>
      </div>

      {showDeployButton && value.deployButton && (
        <div className="pl-4 border-l-2 border-[color:var(--border-indent)] space-y-4">
          <div className="form-control">
            <label className="label">
              <span className="label-text font-medium">Type</span>
            </label>
            <select
              className="select select-bordered bg-base-100"
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
          </div>

          <FormField
            id="editor-deploy-button-label"
            label="Label"
            type="text"
            value={value.deployButton.label}
            onChange={(v) =>
              updateField('deployButton', {
                ...value.deployButton!,
                label: v,
              })
            }
          />
        </div>
      )}
    </article>
  )
}
