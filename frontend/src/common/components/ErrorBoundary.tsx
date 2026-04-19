import { Component, type ErrorInfo, type ReactNode } from 'react'

interface ErrorBoundaryProps {
  children: ReactNode
  fallback?: ReactNode
}

interface ErrorBoundaryState {
  hasError: boolean
  error: Error | null
}

/**
 * ErrorBoundary catches unhandled React errors and displays a recovery screen.
 * Wrap the main app routes with this to prevent full-page crashes.
 */
export class ErrorBoundary extends Component<ErrorBoundaryProps, ErrorBoundaryState> {
  constructor(props: ErrorBoundaryProps) {
    super(props)
    this.state = { hasError: false, error: null }
  }

  static getDerivedStateFromError(error: Error): ErrorBoundaryState {
    return { hasError: true, error }
  }

  componentDidCatch(error: Error, errorInfo: ErrorInfo) {
    console.error('[ErrorBoundary] Uncaught error:', error, errorInfo)
  }

  handleRetry = () => {
    this.setState({ hasError: false, error: null })
  }

  render() {
    if (this.state.hasError) {
      if (this.props.fallback) {
        return this.props.fallback
      }

      return (
        <div className="flex min-h-screen items-center justify-center p-6">
          <div className="surface-card border border-base-300/60 p-8 max-w-md w-full text-center">
            <svg
              className="mx-auto mb-4 w-12 h-12 text-error/60"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={1.5}
                d="M12 9v3.75m-9.303 3.376c-.866 1.5.217 3.374 1.948 3.374h14.71c1.73 0 2.813-1.874 1.948-3.374L13.949 3.378c-.866-1.5-3.032-1.5-3.898 0L2.697 16.126zM12 15.75h.007v.008H12v-.008z"
              />
            </svg>
            <h2 className="text-lg font-semibold text-base-content mb-2">Something went wrong</h2>
            <p className="text-sm text-base-content/60 mb-1">
              An unexpected error occurred in the interface.
            </p>
            {this.state.error ? (
              <p className="text-xs text-base-content/40 mb-6 font-mono break-all">
                {this.state.error.message}
              </p>
            ) : (
              <div className="mb-6" />
            )}
            <button className="action-btn-primary" type="button" onClick={this.handleRetry}>
              Try again
            </button>
          </div>
        </div>
      )
    }

    return this.props.children
  }
}
