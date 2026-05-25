import './index.css';
import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { Toaster } from 'sonner';
import App from './App';
import { queryClientConfig } from '@/shared/lib/queryKeys';

// Apply theme synchronously before React renders to prevent flash of wrong colors
(function applyInitialTheme() {
  const stored = localStorage.getItem('cc-theme');
  const theme = stored === 'light' || stored === 'dark' || stored === 'system' ? stored : 'system';
  const resolved =
    theme === 'system'
      ? window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
      : theme;
  document.documentElement.setAttribute(
    'data-theme',
    resolved === 'dark' ? 'corporateDark' : 'corporateLight'
  );
})();

const queryClient = new QueryClient(queryClientConfig);

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <QueryClientProvider client={queryClient}>
      <App />
      <Toaster position="top-right" />
    </QueryClientProvider>
  </StrictMode>
);
