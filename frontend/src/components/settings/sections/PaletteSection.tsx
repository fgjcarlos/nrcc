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
    <article className="settings-section">
      <h3>Palette</h3>

      <label className="form-field">
        <span>Categories (ordered)</span>
      </label>

      <div className="category-list">
        {value.categories.map((cat, idx) => (
          <div key={idx} className="category-row">
            <input
              type="text"
              value={cat}
              onChange={(e) => updateCategory(idx, e.target.value)}
              placeholder="Category name"
            />
            <button
              type="button"
              onClick={() => moveCategory(idx, 'up')}
              disabled={idx === 0}
              className="ghost-button small"
              title="Move up"
            >
              ↑
            </button>
            <button
              type="button"
              onClick={() => moveCategory(idx, 'down')}
              disabled={idx === value.categories.length - 1}
              className="ghost-button small"
              title="Move down"
            >
              ↓
            </button>
            <button
              type="button"
              onClick={() => removeCategory(idx)}
              className="ghost-button small"
            >
              Remove
            </button>
          </div>
        ))}
      </div>

      <button
        type="button"
        onClick={addCategory}
        className="ghost-button"
      >
        Add Category
      </button>

      {errors['palette.categories'] && (
        <p className="field-error">{errors['palette.categories']}</p>
      )}
    </article>
  )
}
