import { afterEach, describe, expect, it, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';

vi.mock('@/features/auth/hooks/useAuth', () => ({
  useAuth: () => ({ isAuthenticated: false, isLoading: false }),
}));

vi.mock('@/features/styleguide/StyleguideView', () => ({
  StyleguideView: () => <div>Styleguide sentinel</div>,
}));

// App reads import.meta.env.DEV at module load, so each case re-imports it.
async function renderAppAt(path: string, dev: boolean) {
  vi.stubEnv('DEV', dev);
  vi.resetModules();
  const { default: App } = await import('./App');
  window.history.pushState({}, '', path);
  render(
    <QueryClientProvider client={new QueryClient()}>
      <App />
    </QueryClientProvider>,
  );
}

describe('/styleguide route (S5: dev-only)', () => {
  afterEach(() => {
    vi.unstubAllEnvs();
  });

  it('renders the styleguide without login in dev', async () => {
    await renderAppAt('/styleguide', true);
    expect(await screen.findByText('Styleguide sentinel')).toBeInTheDocument();
  });

  it('is not registered when DEV is false', async () => {
    await renderAppAt('/styleguide', false);
    // Let any lazy route settle before asserting absence.
    await new Promise((resolve) => setTimeout(resolve, 50));
    expect(screen.queryByText('Styleguide sentinel')).not.toBeInTheDocument();
  });
});
