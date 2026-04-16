import { PaletteConfig } from '../../../types/config'
import { FormField } from '../../../components/forms'

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
    <article className="surface-card border border-base-300/60 p-6 md:p-7 space-y-6">
      <div className="config-section-head">
        <p className="config-section-kicker">Editor catalog</p>
        <h3 className="config-section-title">Palette</h3>
        <p className="config-section-copy">
          Reorder palette categories so the editor matches the workflow and terminology you want operators to see first.
        </p>
      </div>

      <label className="label">
        <span className="label-text font-medium">Categories (ordered)</span>
      </label>

        <div className="space-y-3">
          {value.categories.map((cat, idx) => (
            <div key={idx} className="config-subsection space-y-4">
              <div>
                <p className="config-subsection-title">Category {idx + 1}</p>
                <p className="config-subsection-copy">Move entries up or down to change the order shown in the editor palette.</p>
              </div>
              <FormField
                id={`palette-category-${idx}`}
                label={`Category ${idx + 1}`}
               type="text"
               placeholder="Category name"
                value={cat}
                onChange={(val) => updateCategory(idx, val)}
              />
              <div className="flex flex-wrap gap-2">
                <button
                  type="button"
                  onClick={() => moveCategory(idx, 'up')}
                  disabled={idx === 0}
                  className="action-btn-ghost"
                  title="Move up"
                >
                  ↑
               </button>
               <button
                  type="button"
                  onClick={() => moveCategory(idx, 'down')}
                  disabled={idx === value.categories.length - 1}
                  className="action-btn-ghost"
                  title="Move down"
                >
                  ↓
               </button>
                <button
                  type="button"
                  onClick={() => removeCategory(idx)}
                  className="action-btn-danger"
                >
                  Remove
                </button>
             </div>
           </div>
         ))}
       </div>

      <button
        type="button"
        onClick={addCategory}
        className="action-btn-ghost"
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
