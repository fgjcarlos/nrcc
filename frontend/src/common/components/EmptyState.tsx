import type { ReactNode } from 'react'

interface EmptyStateProps {
  icon?: ReactNode
  title: string
  description?: string
  action?: {
    label: string
    onClick: () => void
    disabled?: boolean
  }
}

/**
 * EmptyState displays a centered placeholder when a list or section has no data.
 * Follows the project's muted, card-based visual language.
 */
export function EmptyState({ icon, title, description, action }: EmptyStateProps) {
  return (
    <div className="flex flex-col items-center justify-center py-12 px-6 text-center">
      {icon ? (
        <div className="mb-4 text-base-content/30">{icon}</div>
      ) : (
        <svg
          className="mb-4 w-10 h-10 text-base-content/30"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={1.5}
            d="M20 13V6a2 2 0 00-2-2H6a2 2 0 00-2 2v7m16 0v5a2 2 0 01-2 2H6a2 2 0 01-2-2v-5m16 0h-2.586a1 1 0 00-.707.293l-2.414 2.414a1 1 0 01-.707.293h-3.172a1 1 0 01-.707-.293l-2.414-2.414A1 1 0 006.586 13H4"
          />
        </svg>
      )}
      <h3 className="text-sm font-semibold text-base-content/70">{title}</h3>
      {description ? (
        <p className="mt-1 max-w-sm text-xs text-base-content/50">{description}</p>
      ) : null}
      {action ? (
        <button
          className="action-btn-primary mt-4"
          type="button"
          onClick={action.onClick}
          disabled={action.disabled}
        >
          {action.label}
        </button>
      ) : null}
    </div>
  )
}
