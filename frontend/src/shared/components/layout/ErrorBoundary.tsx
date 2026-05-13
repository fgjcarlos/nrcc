import { Component, type ReactNode } from 'react';

interface Props {
  children: ReactNode;
  fallback?: ReactNode;
}

interface State {
  hasError: boolean;
  error: Error | null;
}

export class ErrorBoundary extends Component<Props, State> {
  constructor(props: Props) {
    super(props);
    this.state = { hasError: false, error: null };
  }

  static getDerivedStateFromError(error: Error): State {
    return { hasError: true, error };
  }

  componentDidCatch(error: Error, errorInfo: React.ErrorInfo) {
    // Capture error details via lifecycle hook
    this.setState({ error });
  }

  render() {
    if (this.state.hasError) {
      // If custom fallback provided, render it
      if (this.props.fallback) {
        return this.props.fallback;
      }

      // Default fallback: show error message
      return (
        <div className="flex flex-col items-center justify-center min-h-screen p-8 text-center">
          <h1 className="text-2xl font-bold text-base-content mb-4">
            Something went wrong
          </h1>
          <p className="text-base-content/70 mb-4">
            {this.state.error?.message || 'An error occurred. Please reload the page.'}
          </p>
          <button
            onClick={() => window.location.reload()}
            className="btn btn-primary"
          >
            Reload Page
          </button>
        </div>
      );
    }

    return this.props.children;
  }
}
