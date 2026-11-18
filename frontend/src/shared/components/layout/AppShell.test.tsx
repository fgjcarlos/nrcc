import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { http, HttpResponse } from 'msw'
import { server } from '@/test/msw/server'
import { editableHostStatus } from '@/test/msw/fixtures'
import { i18n } from '@/i18n'
import { Layout } from './Layout'

// T6 (S1): the shell shared by every authenticated page, driven through
// the public render with the MSW fixture API.

const auth = vi.hoisted(() => ({ role: 'admin' as 'admin' | 'viewer' }))

vi.mock('@/features/auth/hooks/useAuth', () => ({
  useAuth: () => ({
    user: { id: 'u1', username: 'operator', role: auth.role, createdAt: '2024-01-01T00:00:00Z' },
    logout: vi.fn(),
  }),
}))

vi.mock('@/features/updates/components/UpdateNotificationChip', () => ({
  UpdateNotificationChip: () => null,
}))

const ok = (data: unknown) =>
  HttpResponse.json({ success: true, data, timestamp: new Date(0).toISOString() })

function renderShell(path = '/overview') {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <MemoryRouter initialEntries={[path]}>
      <QueryClientProvider client={queryClient}>
        <Routes>
          <Route element={<Layout />}>
            <Route path="*" element={<div>Page body</div>} />
          </Route>
        </Routes>
      </QueryClientProvider>
    </MemoryRouter>,
  )
}

describe('App shell (T6)', () => {
  beforeEach(async () => {
    auth.role = 'admin'
    localStorage.clear()
    window.innerWidth = 1440
    window.matchMedia = vi.fn().mockImplementation((query: string) => ({
      matches: false,
      media: query,
      onchange: null,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      addListener: vi.fn(),
      removeListener: vi.fn(),
      dispatchEvent: vi.fn(),
    }))
    await i18n.changeLanguage('en')
  })

  afterEach(async () => {
    await i18n.changeLanguage('en')
  })

  it('labels every nav link with its page title from the catalog (es)', async () => {
    await i18n.changeLanguage('es')
    renderShell()

    const nav = screen.getByRole('navigation', { name: 'Navegación principal' })
    for (const name of [
      'Resumen',
      'Configuración',
      'Seguridad',
      'Variables de entorno',
      'Respaldos',
      'Host y entorno',
      'Usuarios',
      'Actualizaciones',
      'Bibliotecas',
    ]) {
      expect(within(nav).getByRole('link', { name })).toBeInTheDocument()
    }
    expect(screen.getByRole('group', { name: 'Administración' })).toBeInTheDocument()
  })

  it('names nav links like the page headings in en and drops the old jargon', () => {
    renderShell()

    const nav = screen.getByRole('navigation', { name: 'Main navigation' })
    expect(within(nav).getByRole('link', { name: 'Backups' })).toHaveAttribute('href', '/backups')
    expect(within(nav).getByRole('link', { name: 'Host & environment' })).toHaveAttribute('href', '/bootstrap')
    expect(within(nav).getByRole('link', { name: 'Updates' })).toHaveAttribute('href', '/maintenance/updates')
    expect(within(nav).getByRole('link', { name: 'Libraries' })).toHaveAttribute('href', '/maintenance/libraries')
    for (const legacy of [/Recovery/, /Bootstrap/, /Maintenance/, /Orchestrator/]) {
      expect(screen.queryByText(legacy)).not.toBeInTheDocument()
    }
  })

  it('renders no Spanish accessible names in en', () => {
    renderShell()

    const spanish = /navegación|abrir|contraer|expandir|tema|idioma|activo|cambiar a/i
    const named = Array.from(document.querySelectorAll('[aria-label], [title]'))
    for (const el of named) {
      expect(el.getAttribute('aria-label') ?? '').not.toMatch(spanish)
      expect(el.getAttribute('title') ?? '').not.toMatch(spanish)
    }
    expect(screen.getByRole('button', { name: 'Collapse navigation' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /^Theme: / })).toBeInTheDocument()
  })

  it('groups the admin links under Administration for admins only', () => {
    const { unmount } = renderShell()
    const group = screen.getByRole('group', { name: 'Administration' })
    for (const name of ['Users', 'Updates', 'Libraries']) {
      expect(within(group).getByRole('link', { name })).toBeInTheDocument()
    }
    unmount()

    auth.role = 'viewer'
    renderShell()
    expect(screen.queryByRole('group', { name: 'Administration' })).not.toBeInTheDocument()
    for (const name of ['Users', 'Updates', 'Libraries']) {
      expect(screen.queryByRole('link', { name })).not.toBeInTheDocument()
    }
  })

  it('shows one runtime status from the host status the pages use', async () => {
    renderShell()

    // Loading first, never a guessed value.
    expect(screen.getByTestId('runtime-status')).toHaveTextContent('Checking runtime')

    // Fixture: Node-RED 4.0.0, read-only.
    const status = await screen.findByRole('link', { name: /Node-RED 4\.0\.0 · Read-only/ })
    expect(status).toHaveAttribute('href', '/bootstrap')
    expect(screen.getAllByTestId('runtime-status')).toHaveLength(1)
    for (const legacy of ['node-red-version-chip', 'compat-mode-chip', 'edge-mode-chip', 'api-base-chip']) {
      expect(screen.queryByTestId(legacy)).not.toBeInTheDocument()
    }
  })

  it('reports an editable runtime with the same version Configuration shows', async () => {
    server.use(http.get('/api/bootstrap/status', () => ok(editableHostStatus)))
    renderShell()

    expect(await screen.findByRole('link', { name: /Node-RED 5\.0\.6 · Editable/ })).toBeInTheDocument()
  })

  it('says the runtime status is unknown when the host status fails', async () => {
    server.use(http.get('/api/bootstrap/status', () => HttpResponse.json({ success: false }, { status: 500 })))
    renderShell()

    expect(
      await screen.findByRole('link', { name: /Node-RED · Status unknown/ }, { timeout: 6000 }),
    ).toBeInTheDocument()
  }, 10_000)

  it('opens the navigation drawer from a 44px menu button on mobile and closes it on Escape', async () => {
    window.innerWidth = 390
    const user = userEvent.setup()
    renderShell()

    expect(screen.queryByRole('navigation', { name: 'Main navigation' })).not.toBeInTheDocument()
    const menu = screen.getByRole('button', { name: 'Open main navigation' })
    expect(menu).toHaveClass('h-11', 'w-11')
    expect(menu).toHaveAttribute('aria-expanded', 'false')
    // The runtime status and the command palette stay reachable on mobile.
    expect(screen.getByTestId('runtime-status')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /Command palette/ })).toBeInTheDocument()

    await user.click(menu)
    const drawer = screen.getByRole('dialog', { name: 'Main navigation' })
    expect(within(drawer).getByRole('navigation', { name: 'Main navigation' })).toBeInTheDocument()
    await waitFor(() => expect(drawer.contains(document.activeElement)).toBe(true))

    await user.keyboard('{Escape}')
    expect(screen.queryByRole('dialog', { name: 'Main navigation' })).not.toBeInTheDocument()
    expect(menu).toHaveFocus()
  })

  it('closes the drawer when a nav link is followed', async () => {
    window.innerWidth = 390
    const user = userEvent.setup()
    renderShell()

    await user.click(screen.getByRole('button', { name: 'Open main navigation' }))
    await user.click(within(screen.getByRole('dialog')).getByRole('link', { name: 'Configuration' }))
    expect(screen.queryByRole('dialog', { name: 'Main navigation' })).not.toBeInTheDocument()
  })
})
