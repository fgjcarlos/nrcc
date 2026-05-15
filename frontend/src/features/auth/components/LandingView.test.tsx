import { beforeEach, describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { LandingView } from './LandingView'
import { authService } from '../services/authService'

vi.mock('../services/authService', () => ({
  authService: {
    getStatus: vi.fn(),
    getToken: vi.fn(),
    getMe: vi.fn(),
  },
}))

const mockAuthService = vi.mocked(authService)

const renderLanding = () =>
  render(
    <MemoryRouter initialEntries={['/']}>
      <Routes>
        <Route path="/" element={<LandingView />} />
        <Route path="/dashboard" element={<div>Dashboard route</div>} />
        <Route path="/login" element={<div>Login route</div>} />
        <Route path="/setup" element={<div>Setup route</div>} />
      </Routes>
    </MemoryRouter>
  )

describe('LandingView branded home', () => {
  beforeEach(() => {
    vi.clearAllMocks()
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

  it('renders a branded project introduction before the main flow starts', async () => {
    mockAuthService.getStatus.mockResolvedValue({ initialized: true })
    mockAuthService.getToken.mockReturnValue(null)

    renderLanding()

    expect(await screen.findByRole('heading', { name: /node-red control center/i })).toBeInTheDocument()
    expect(
      screen.getByText(/controla instancias node-red desde una consola operativa/i)
    ).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: /orquestación de flows/i })).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: /observabilidad industrial/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /loguearse/i })).toBeInTheDocument()
    expect(screen.queryByText('Login route')).not.toBeInTheDocument()
  })

  it('lets authenticated users start the process from the primary CTA', async () => {
    const user = userEvent.setup()
    mockAuthService.getStatus.mockResolvedValue({ initialized: true })
    mockAuthService.getToken.mockReturnValue('token')
    mockAuthService.getMe.mockResolvedValue({
      id: '1',
      username: 'admin',
      role: 'admin',
      createdAt: '2026-05-15T00:00:00Z',
    })

    renderLanding()

    const cta = await screen.findByRole('button', { name: /iniciar proceso/i })
    await user.click(cta)

    expect(await screen.findByText('Dashboard route')).toBeInTheDocument()
  })
})
