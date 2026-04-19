import { useState, useRef, useEffect, useCallback } from 'react'
import { EditorThemeConfig } from '../../../types/config'
import { FormField } from '../../../components/forms'
import { api, AssetInfo } from '../../../api'

type SectionProps<T> = {
  value: T
  onChange: (next: T) => void
  errors: Record<string, string>
}

type BrandingSlotProps = {
  id: string
  label: string
  hint: string
  category: string
  currentUrl: string
  onUrlChange: (url: string) => void
}

function BrandingSlot({ id, label, hint, category, currentUrl, onUrlChange }: BrandingSlotProps) {
  const [assets, setAssets] = useState<AssetInfo[]>([])
  const [uploading, setUploading] = useState(false)
  const [error, setError] = useState('')
  const [showUrlInput, setShowUrlInput] = useState(false)
  const fileInputRef = useRef<HTMLInputElement>(null)

  const loadAssets = useCallback(async () => {
    try {
      const list = await api.listAssets(category)
      setAssets(list.items)
    } catch {
      // ignore — category may not have any assets yet
    }
  }, [category])

  useEffect(() => {
    loadAssets()
  }, [loadAssets])

  const handleUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return
    setError('')
    setUploading(true)
    try {
      const asset = await api.uploadAsset(category, file)
      onUrlChange(asset.url)
      await loadAssets()
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Upload failed')
    } finally {
      setUploading(false)
      if (fileInputRef.current) fileInputRef.current.value = ''
    }
  }

  const handleDelete = async (assetId: string, assetUrl: string) => {
    try {
      await api.deleteAsset(category, assetId)
      if (currentUrl === assetUrl) {
        onUrlChange('')
      }
      await loadAssets()
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Delete failed')
    }
  }

  const handleSelect = (url: string) => {
    onUrlChange(url)
  }

  const isLocalAsset = currentUrl.startsWith('/assets/')

  return (
    <div className="space-y-3">
      <div>
        <label className="label">
          <span className="label-text font-medium">{label}</span>
        </label>
        <p className="text-xs text-base-content/60 mb-2">{hint}</p>
      </div>

      {/* Current image preview */}
      {currentUrl && (
        <div className="flex items-center gap-3 p-3 bg-base-200/50 rounded-lg">
          <img
            src={currentUrl}
            alt={label}
            className="h-10 w-10 object-contain rounded border border-base-300"
            onError={(e) => { (e.target as HTMLImageElement).style.display = 'none' }}
          />
          <div className="flex-1 min-w-0">
            <p className="text-sm truncate">{currentUrl}</p>
          </div>
          <button
            type="button"
            className="btn btn-ghost btn-xs"
            onClick={() => onUrlChange('')}
          >
            Clear
          </button>
        </div>
      )}

      {/* Upload button */}
      <div className="flex flex-wrap gap-2">
        <input
          ref={fileInputRef}
          type="file"
          accept="image/png,image/jpeg,image/gif,image/svg+xml,image/x-icon,.ico"
          className="hidden"
          id={`upload-${id}`}
          onChange={handleUpload}
        />
        <button
          type="button"
          className="btn btn-sm btn-outline"
          onClick={() => fileInputRef.current?.click()}
          disabled={uploading}
        >
          {uploading ? 'Uploading...' : 'Upload Image'}
        </button>
        <button
          type="button"
          className="btn btn-sm btn-ghost"
          onClick={() => setShowUrlInput(!showUrlInput)}
        >
          {showUrlInput ? 'Hide URL Input' : 'Use URL Instead'}
        </button>
      </div>

      {/* Manual URL input */}
      {showUrlInput && (
        <FormField
          id={`${id}-url`}
          label="Image URL"
          type="text"
          value={isLocalAsset ? '' : currentUrl}
          onChange={onUrlChange}
          placeholder="https://example.com/image.png"
        />
      )}

      {/* Uploaded assets gallery */}
      {assets.length > 0 && (
        <div className="space-y-1">
          <p className="text-xs font-medium text-base-content/70">Uploaded files</p>
          <div className="flex flex-wrap gap-2">
            {assets.map((asset) => (
              <div
                key={asset.id}
                className={`relative group flex items-center gap-2 p-2 rounded border cursor-pointer transition-colors ${
                  currentUrl === asset.url
                    ? 'border-primary bg-primary/10'
                    : 'border-base-300 hover:border-primary/50'
                }`}
                onClick={() => handleSelect(asset.url)}
              >
                <img
                  src={asset.url}
                  alt={asset.original}
                  className="h-8 w-8 object-contain"
                  onError={(e) => { (e.target as HTMLImageElement).style.display = 'none' }}
                />
                <span className="text-xs truncate max-w-[100px]">{asset.original}</span>
                <button
                  type="button"
                  className="btn btn-ghost btn-xs opacity-0 group-hover:opacity-100 transition-opacity"
                  onClick={(e) => {
                    e.stopPropagation()
                    handleDelete(asset.id, asset.url)
                  }}
                  title="Delete"
                >
                  &times;
                </button>
              </div>
            ))}
          </div>
        </div>
      )}

      {error && <p className="text-xs text-error">{error}</p>}
    </div>
  )
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
    <article className="surface-card border border-base-300/60 p-6 md:p-7 space-y-6">
      <div className="config-section-head">
        <p className="config-section-kicker">Presentation</p>
        <h3 className="config-section-title">Editor Theme</h3>
        <p className="config-section-copy">
          Configure headline branding, optional editor modules, and page-level appearance knobs exposed by Node-RED.
        </p>
      </div>

      <FormField
        id="editor-theme-theme"
        label="Theme"
        type="text"
        value={value.theme}
        onChange={(v) => updateField('theme', v)}
        placeholder="Theme name"
      />

      <label className="config-toggle-row cursor-pointer">
          <input
            type="checkbox"
            className="checkbox"
            checked={value.tours}
            onChange={(e) => updateField('tours', e.target.checked)}
          />
          <span className="config-toggle-copy">
            <span className="config-toggle-title">Enable Tours</span>
            <span className="config-toggle-hint">Keep the built-in guided tours available inside the editor.</span>
          </span>
      </label>

      <label className="config-toggle-row cursor-pointer">
          <input
            type="checkbox"
            className="checkbox"
            checked={value.userMenu}
            onChange={(e) => updateField('userMenu', e.target.checked)}
          />
          <span className="config-toggle-copy">
            <span className="config-toggle-title">Enable User Menu</span>
            <span className="config-toggle-hint">Show the built-in user menu in the editor chrome.</span>
          </span>
      </label>

      <label className="config-toggle-row cursor-pointer">
          <input
            type="checkbox"
            className="checkbox"
            checked={value.projects.enabled}
            onChange={(e) =>
              updateField('projects', { ...value.projects, enabled: e.target.checked })
            }
          />
          <span className="config-toggle-copy">
            <span className="config-toggle-title">Enable Projects</span>
            <span className="config-toggle-hint">Allow git-backed project workflows from inside the Node-RED editor.</span>
          </span>
      </label>

      {/* Code Editor */}
      <div className="divider"></div>
      <label className="config-toggle-row cursor-pointer">
          <input
            type="checkbox"
            className="checkbox"
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
          <span className="config-toggle-copy">
            <span className="config-toggle-title">Code Editor Settings</span>
            <span className="config-toggle-hint">Choose the embedded editor engine used by code-oriented panels.</span>
          </span>
      </label>

      {showCodeEditor && value.codeEditor && (
        <div className="config-subsection space-y-4">
          <div>
            <p className="config-subsection-title">Code editor engine</p>
            <p className="config-subsection-copy">Switch between the default ACE editor and Monaco where supported.</p>
          </div>
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
      <label className="config-toggle-row cursor-pointer">
          <input
            type="checkbox"
            className="checkbox"
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
          <span className="config-toggle-copy">
            <span className="config-toggle-title">Page Settings</span>
            <span className="config-toggle-hint">Override the page title, favicon, and injected CSS used by the editor shell.</span>
          </span>
      </label>

      {showPage && value.page && (
        <div className="config-subsection space-y-4">
          <FormField
            id="editor-page-title"
            label="Page Title"
             type="text"
             value={value.page.title}
             onChange={(v) =>
               updateField('page', { ...value.page!, title: v })
             }
           />

           <BrandingSlot
             id="editor-page-favicon"
             label="Favicon"
             hint="Upload or link a favicon image (.ico, .png, .svg)."
             category="favicon"
             currentUrl={value.page.favicon}
             onUrlChange={(v) =>
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
      <label className="config-toggle-row cursor-pointer">
          <input
            type="checkbox"
            className="checkbox"
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
          <span className="config-toggle-copy">
            <span className="config-toggle-title">Header Settings</span>
            <span className="config-toggle-hint">Control the title, link target, and optional image rendered in the editor header.</span>
          </span>
      </label>

      {showHeader && value.header && (
        <div className="config-subsection space-y-4">
           <FormField
              id="editor-header-title"
              label="Header Title"
             type="text"
             value={value.header.title}
             onChange={(v) =>
               updateField('header', { ...value.header!, title: v })
             }
           />

           <BrandingSlot
             id="editor-header-image"
             label="Header Image"
             hint="Upload or link a header logo image."
             category="header"
             currentUrl={value.header.image}
             onUrlChange={(v) =>
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
      <label className="config-toggle-row cursor-pointer">
          <input
            type="checkbox"
            className="checkbox"
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
          <span className="config-toggle-copy">
            <span className="config-toggle-title">Deploy Button Settings</span>
            <span className="config-toggle-hint">Adjust the deploy button mode and wording shown in the editor toolbar.</span>
          </span>
      </label>

      {showDeployButton && value.deployButton && (
        <div className="config-subsection space-y-4">
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
