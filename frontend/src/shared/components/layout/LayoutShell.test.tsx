import { describe, it, expect, beforeEach, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { Layout } from './Layout'

vi.mock('@/features/auth/hooks/useAuth', () => ({
  useAuth: () => ({
    user: { username: 'admin', role: 'admin' },
    logout: vi.fn(),
  }),
}))

vi.mock('@/features/updates/components/UpdateNotificationChip', () => ({
  UpdateNotificationChip: () => null,
}))

const renderShell = () => {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })

  return render(
    <MemoryRouter initialEntries={['/dashboard']}>
      <QueryClientProvider client={queryClient}>
        <Routes>
          <Route element={<Layout />}>
            <Route path="/dashboard" element={<div>Dashboard content</div>} />
          </Route>
        </Routes>
      </QueryClientProvider>
    </MemoryRouter>
  )
}

describe('Layout shell visual hierarchy', () => {
  beforeEach(() => {
    localStorage.clear()
    window.matchMedia = vi.fn().mockImplementation((query) => ({
      matches: false,
      media: query,
      onchange: null,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      addListener: vi.fn(),
      removeListener: vi.fn(),
      dispatchEvent: vi.fn(),
    }))
  })

  it('separates global navigation chrome from the routed page content', () => {
    renderShell()

    const contentShell = screen.getByTestId('page-content-shell')
    expect(contentShell).toHaveClass('surface-panel')
    expect(contentShell).toHaveClass('rounded-2xl')
    expect(contentShell).toHaveClass('border')
    expect(screen.getByText('Dashboard content')).toBeInTheDocument()
  })

  it('uses branded shell surfaces for sidebar, topbar and theme control', () => {
    renderShell()

    expect(screen.getByTestId('app-sidebar')).toHaveClass('app-sidebar-shell')
    expect(screen.getByTestId('app-topbar')).toHaveClass('app-topbar-shell')
    expect(screen.getByRole('button', { name: /tema:/i })).toHaveClass('theme-toggle-shell')
  })

  // Issue #966: the sidebar must switch to drawer mode at the same `lg`
  // (1024px) breakpoint the DaisyUI drawer (Layout.tsx) and the toggle
  // (Header.tsx) use, not at its own narrower constant.
  it('renders drawer-mode navigation at tablet width even when collapsed', () => {
    window.innerWidth = 900
    localStorage.setItem('sidebar-collapsed', 'true')
    renderShell()

    const sidebar = screen.getByTestId('app-sidebar')
    // A collapsed desktop sidebar would render w-16 and hide the labels.
    expect(sidebar).toHaveClass('w-64')
    expect(sidebar).not.toHaveClass('w-16')
    expect(screen.getByRole('link', { name: 'Configuration' })).toHaveTextContent('Configuration')
    expect(screen.queryByRole('button', { name: /navegación/i })).not.toBeInTheDocument()
  })

  it('keeps the collapsible desktop sidebar above the lg breakpoint', () => {
    window.innerWidth = 1280
    localStorage.setItem('sidebar-collapsed', 'true')
    renderShell()

    expect(screen.getByTestId('app-sidebar')).toHaveClass('w-16')
    expect(screen.getByRole('button', { name: /Expandir navegación/ })).toBeInTheDocument()
  })
})
