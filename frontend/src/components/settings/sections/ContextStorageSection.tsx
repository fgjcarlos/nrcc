import { ContextStorageConfig, ContextStoreEntry } from '../../../types/config'

type SectionProps<T> = {
  value: T
  onChange: (next: T) => void
  errors: Record<string, string>
}

export function ContextStorageSection({ value, onChange, errors }: SectionProps<ContextStorageConfig>) {
  const updateField = <K extends keyof ContextStorageConfig>(
    key: K,
    val: ContextStorageConfig[K]
  ) => {
    onChange({ ...value, [key]: val })
  }

  const storeNames = Object.keys(value.stores)

  const addStore = () => {
    const newName = `store-${Date.now()}`
    const newStores = {
      ...value.stores,
      [newName]: { module: 'memory' as const },
    }
    updateField('stores', newStores)
  }

  const removeStore = (name: string) => {
    const newStores = { ...value.stores }
    delete newStores[name]
    updateField('stores', newStores)
  }

  const updateStore = (name: string, store: ContextStoreEntry) => {
    const newStores = { ...value.stores, [name]: store }
    updateField('stores', newStores)
  }

  return (
    <article className="settings-section">
      <h3>Context Storage</h3>

      <label className="form-field">
        <span>Default Store</span>
        <select
          value={value.default}
          onChange={(e) => updateField('default', e.target.value)}
        >
          {storeNames.map((name) => (
            <option key={name} value={name}>
              {name}
            </option>
          ))}
        </select>
        {errors['contextStorage.default'] && (
          <p className="field-error">{errors['contextStorage.default']}</p>
        )}
      </label>

      <div className="form-field">
        <label>
          <span>Stores</span>
        </label>

        {storeNames.map((name) => {
          const store = value.stores[name]
          return (
            <div key={name} className="store-entry">
              <input
                type="text"
                placeholder="Store name"
                value={name}
                readOnly
                disabled
              />
              <select
                value={store.module}
                onChange={(e) =>
                  updateStore(name, {
                    ...store,
                    module: e.target.value as 'memory' | 'localfilesystem',
                  })
                }
              >
                <option value="memory">Memory</option>
                <option value="localfilesystem">Local Filesystem</option>
              </select>
              {name !== 'default' && (
                <button
                  type="button"
                  onClick={() => removeStore(name)}
                  className="ghost-button"
                >
                  Remove
                </button>
              )}
            </div>
          )
        })}

        <button
          type="button"
          onClick={addStore}
          className="ghost-button"
        >
          Add Store
        </button>
      </div>
    </article>
  )
}
