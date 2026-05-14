import { describe, it, expect, beforeEach, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import React from 'react'
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
})
