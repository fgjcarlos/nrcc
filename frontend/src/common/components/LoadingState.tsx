import { LoadingSpinner } from './LoadingSpinner'

interface LoadingStateProps {
  message?: string
  size?: 'sm' | 'md' | 'lg'
  className?: string
}

/**
 * LoadingState displays a centered spinner with an optional message.
 * Wraps the existing LoadingSpinner for consistent loading UX across pages.
 */
export function LoadingState({ message, size = 'md', className = '' }: LoadingStateProps) {
  return (
    <div className={`flex flex-col items-center justify-center py-10 px-6 ${className}`}>
      <LoadingSpinner size={size} className="text-base-content/40" />
      {message ? (
        <p className="mt-3 text-sm text-base-content/60">{message}</p>
      ) : null}
    </div>
  )
}
