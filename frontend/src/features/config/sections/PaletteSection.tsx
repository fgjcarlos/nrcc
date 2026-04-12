import { PaletteConfig } from '../../../types/config'

type SectionProps<T> = {
  value: T
  onChange: (next: T) => void
  errors: Record<string, string>
}

export function PaletteSection({ value, onChange, errors }: SectionProps<PaletteConfig>) {
  const updateField = <K extends keyof PaletteConfig>(key: K, val: PaletteConfig[K]) => {
    onChange({ ...value, [key]: val })
  }

  const addCategory = () => {
    const newCategories = [...value.categories, '']
    updateField('categories', newCategories)
  }

  const updateCategory = (idx: number, newVal: string) => {
    const newCategories = [...value.categories]
    newCategories[idx] = newVal
    updateField('categories', newCategories)
  }

  const removeCategory = (idx: number) => {
    const newCategories = value.categories.filter((_, i) => i !== idx)
    updateField('categories', newCategories)
  }

  const moveCategory = (idx: number, direction: 'up' | 'down') => {
    const newCategories = [...value.categories]
    if (direction === 'up' && idx > 0) {
      [newCategories[idx], newCategories[idx - 1]] = [newCategories[idx - 1], newCategories[idx]]
    } else if (direction === 'down' && idx < newCategories.length - 1) {
      [newCategories[idx], newCategories[idx + 1]] = [newCategories[idx + 1], newCategories[idx]]
    }
    updateField('categories', newCategories)
  }

  return (
    <article className="space-y-6">
      <h3 className="text-xl font-semibold text-base-content">Palette</h3>

      <label className="label">
        <span className="label-text font-medium">Categories (ordered)</span>
      </label>

      <div className="space-y-3">
        {value.categories.map((cat, idx) => (
          <div key={idx} className="flex gap-2 items-center">
            <input
              type="text"
              className="input input-bordered bg-base-100 flex-1"
              value={cat}
              onChange={(e) => updateCategory(idx, e.target.value)}
              placeholder="Category name"
            />
            <button
              type="button"
              onClick={() => moveCategory(idx, 'up')}
              disabled={idx === 0}
              className="btn btn-ghost btn-sm"
              title="Move up"
            >
              ↑
            </button>
            <button
              type="button"
              onClick={() => moveCategory(idx, 'down')}
              disabled={idx === value.categories.length - 1}
              className="btn btn-ghost btn-sm"
              title="Move down"
            >
              ↓
            </button>
            <button
              type="button"
              onClick={() => removeCategory(idx)}
              className="btn btn-ghost btn-sm"
            >
              Remove
            </button>
          </div>
        ))}
      </div>

      <button
        type="button"
        onClick={addCategory}
        className="btn btn-ghost btn-sm"
      >
        + Add Category
      </button>

      {errors['palette.categories'] && (
        <span className="form-field-error-msg">
          <svg className="w-4 h-4 flex-shrink-0" fill="currentColor" viewBox="0 0 20 20" xmlns="http://www.w3.org/2000/svg">
            <path fillRule="evenodd" d="M18.101 12.93a1 1 0 00-1.414-1.414L11 14.586l-2.687-2.687a1 1 0 00-1.414 1.414l4.1 4.1a1 1 0 001.414 0l8.101-8.101z" clipRule="evenodd" />
            <path fillRule="evenodd" d="M10 2a8 8 0 100 16 8 8 0 000-16zm0 14a6 6 0 110-12 6 6 0 010 12z" clipRule="evenodd" />
          </svg>
          <span>{errors['palette.categories']}</span>
        </span>
      )}
    </article>
  )
}
