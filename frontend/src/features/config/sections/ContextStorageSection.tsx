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
    <article className="space-y-6">
      <h3 className="text-xl font-semibold text-base-content">Context Storage</h3>

       <div className="form-control">
         <label className="label">
           <span className="label-text font-medium">Default Store</span>
         </label>
         <select
           className={`select select-bordered bg-base-100${errors['contextStorage.default'] ? ' select-error' : ''}`}
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
           <span className="form-field-error-msg">
             <svg className="w-4 h-4 flex-shrink-0" fill="currentColor" viewBox="0 0 20 20" xmlns="http://www.w3.org/2000/svg">
               <path fillRule="evenodd" d="M18.101 12.93a1 1 0 00-1.414-1.414L11 14.586l-2.687-2.687a1 1 0 00-1.414 1.414l4.1 4.1a1 1 0 001.414 0l8.101-8.101z" clipRule="evenodd" />
               <path fillRule="evenodd" d="M10 2a8 8 0 100 16 8 8 0 000-16zm0 14a6 6 0 110-12 6 6 0 010 12z" clipRule="evenodd" />
             </svg>
             <span>{errors['contextStorage.default']}</span>
           </span>
         )}
       </div>

      <div>
        <label className="label">
          <span className="label-text font-medium">Stores</span>
        </label>

        <div className="space-y-3">
          {storeNames.map((name) => {
            const store = value.stores[name]
            return (
              <div key={name} className="flex gap-2 items-end">
                <input
                  type="text"
                  className="input input-bordered bg-base-100 flex-1"
                  placeholder="Store name"
                  value={name}
                  readOnly
                  disabled
                />
                <select
                  className="select select-bordered bg-base-100"
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
                    className="btn btn-ghost btn-sm"
                  >
                    Remove
                  </button>
                )}
              </div>
            )
          })}
        </div>

        <button
          type="button"
          onClick={addStore}
          className="btn btn-ghost btn-sm mt-3"
        >
          + Add Store
        </button>
      </div>
    </article>
  )
}
