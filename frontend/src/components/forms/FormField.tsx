import { ChangeEvent, ReactNode } from 'react'

/**
 * FormField component — reusable form field wrapper with unified styling,
 * label, hint, and error state support.
 *
 * This component is the single source of truth for form input styling across
 * the application (auth + config sections). It encapsulates:
 * - Consistent input styling via DaisyUI input-bordered base classes
 * - Unified error state (input-error modifier + icon + message)
 * - Optional hint text in a muted style
 * - Accessibility: label linked to input via htmlFor
 *
 * Design Decision: Why one FormField for all use-cases?
 * - The audit found only one divergence: AuthScreen used input-primary (a theme modifier).
 * - All other inputs in config sections use input-bordered (the standard).
 * - A single FormField with a `type` prop covers text, password, email, number inputs.
 * - This is simpler than maintaining separate AuthInput + ConfigInput + base class duplication.
 *
 * Why input-error modifier (not inline styles)?
 * - DaisyUI's input-error modifier already exists and respects --color-error (themed).
 * - Inline styles bypass the theme cascade and break dark mode.
 * - The modifier is the cleanest, theme-aware approach.
 */

export interface FormFieldProps {
  /** Unique id for the input element — binds label htmlFor and error aria-describedby */
  id: string
  /** Visible label text displayed above the input */
  label: string
  /** Input type (defaults to 'text') */
  type?: 'text' | 'password' | 'email' | 'number'
  /** Current input value */
  value: string | number
  /** Called when the input value changes */
  onChange: (value: string) => void
  /** Placeholder text shown when input is empty */
  placeholder?: string
  /** When truthy, renders input-error modifier + error icon + message text */
  error?: string
  /** Helper text rendered below the input (muted style) */
  hint?: string
  /** Disabled state */
  disabled?: boolean
  /** HTML required attribute */
  required?: boolean
  /** For type="number": minimum value */
  min?: number
  /** For type="number": maximum value */
  max?: number
  /** Extra CSS classes merged onto the <input> element */
  className?: string
}

/** Error icon — inline SVG for consistency */
function ErrorIcon() {
  return (
    <svg
      className="w-4 h-4 flex-shrink-0"
      fill="currentColor"
      viewBox="0 0 20 20"
      xmlns="http://www.w3.org/2000/svg"
    >
      <path
        fillRule="evenodd"
        d="M18.101 12.93a1 1 0 00-1.414-1.414L11 14.586l-2.687-2.687a1 1 0 00-1.414 1.414l4.1 4.1a1 1 0 001.414 0l8.101-8.101z"
        clipRule="evenodd"
      />
      <path
        fillRule="evenodd"
        d="M10 2a8 8 0 100 16 8 8 0 000-16zm0 14a6 6 0 110-12 6 6 0 010 12z"
        clipRule="evenodd"
      />
    </svg>
  )
}

export function FormField(props: FormFieldProps): JSX.Element {
  const {
    id,
    label,
    type = 'text',
    value,
    onChange,
    placeholder,
    error,
    hint,
    disabled = false,
    required = false,
    min,
    max,
    className,
  } = props

  function handleChange(e: ChangeEvent<HTMLInputElement>) {
    onChange(e.target.value)
  }

  // Build input class list
  const inputClasses = [
    'input',
    'input-bordered',
    'bg-base-100',
    error ? 'input-error' : '',
    className,
  ]
    .filter(Boolean)
    .join(' ')

  return (
    <div className="form-control">
      <label className="label" htmlFor={id}>
        <span className="label-text font-semibold">
          {label}
          {required ? <span className="text-error"> *</span> : null}
        </span>
      </label>

      <input
        id={id}
        type={type}
        className={inputClasses}
        value={value}
        onChange={handleChange}
        placeholder={placeholder}
        disabled={disabled}
        required={required}
        min={min}
        max={max}
        aria-describedby={error ? `${id}-error` : undefined}
      />

      {error && (
        <span id={`${id}-error`} className="form-field-error-msg">
          <ErrorIcon />
          <span>{error}</span>
        </span>
      )}

      {hint && <p className="form-field-hint">{hint}</p>}
    </div>
  )
}
