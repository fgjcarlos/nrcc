import { i18n } from '@/i18n';
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router-dom'
import { Toaster } from 'sonner'
import { UsersView } from './UsersView'
import { authService, type User } from '../services/authService'
import * as useAuthModule from '../hooks/useAuth'
import * as useUsersDataModule from '../hooks/useUsersData'
import { buildAuthMock, buildUserMock } from '../__test-utils__/authMock'
// Mock the auth service
vi.mock('../services/authService', () => ({
  authService: {
    createUser: vi.fn(),
    deleteUser: vi.fn(),
    changePassword: vi.fn(),
    updateUserRole: vi.fn(),
    getUsers: vi.fn(),
  },
}))

// Mock the useAuth hook
vi.mock('../hooks/useAuth', () => ({
  useAuth: vi.fn(),
}))

// Mock the useUsersData hook with default return value
vi.mock('../hooks/useUsersData', () => {
  const defaultReturn = {
    users: [],
    isLoading: false,
    isError: false,
    error: null,
    refetch: vi.fn(),
  }
  return {
    useUsersData: vi.fn(() => defaultReturn),
  }
})

// Setup mock data
const mockAdminUser: User = {
  id: 'admin-1',
  username: 'admin',
  role: 'admin',
  createdAt: '2026-01-15T00:00:00Z',
}

const mockViewerUser: User = {
  id: 'viewer-1',
  username: 'viewer',
  role: 'viewer',
  createdAt: '2026-01-20T00:00:00Z',
}

const mockAnotherAdminUser: User = {
  id: 'admin-2',
  username: 'admin2',
  role: 'admin',
  createdAt: '2026-02-01T00:00:00Z',
}

const createQueryClient = () =>
  new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  })

const renderWithProviders = (component: React.ReactElement) => {
  const queryClient = createQueryClient()
  return render(
    <MemoryRouter>
      <QueryClientProvider client={queryClient}>
        {component}
        <Toaster />
      </QueryClientProvider>
    </MemoryRouter>
  )
}

describe('UsersView', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    localStorage.clear()
    // Setup default admin user
    vi.mocked(useAuthModule.useAuth).mockReturnValue(
      buildAuthMock({
        isAuthenticated: true,
        isInitialized: true,
        user: buildUserMock({ id: 'admin-1', username: 'admin', role: 'admin' }),
      })
    )
  })

  describe('access control', () => {
    it('shows access denied message for non-admin user', () => {
      vi.mocked(useAuthModule.useAuth).mockReturnValue(
        buildAuthMock({
          isAuthenticated: true,
          isInitialized: true,
          user: buildUserMock({ id: 'viewer-1', username: 'viewer', role: 'viewer' }),
        })
      )

      renderWithProviders(<UsersView />)

      expect(screen.getByText(i18n.t('auth:accessDenied'))).toBeInTheDocument()
      expect(screen.getByText(i18n.t('auth:accessDeniedDescription'))).toBeInTheDocument()
    })

    it('shows user management header for admin user', () => {
      vi.mocked(useUsersDataModule.useUsersData).mockReturnValue({
        users: [mockAdminUser, mockViewerUser],
        isLoading: false,
        isError: false,
        error: null,
        refetch: vi.fn(),
      })

      renderWithProviders(<UsersView />)

      expect(screen.getByText(i18n.t('auth:userManagement'))).toBeInTheDocument()
    })
  })

  describe('create flow', () => {
    beforeEach(() => {
      vi.mocked(useUsersDataModule.useUsersData).mockReturnValue({
        users: [mockAdminUser],
        isLoading: false,
        isError: false,
        error: null,
        refetch: vi.fn(),
      })
    })

    it('opens create modal when clicking Add User button', async () => {
      const user = userEvent.setup()
      renderWithProviders(<UsersView />)

      const addButton = screen.getByRole('button', { name: i18n.t('auth:addUser') })
      await user.click(addButton)

      // Look for the modal heading, not the submit button
      expect(screen.getByRole('heading', { name: i18n.t('auth:createUser') })).toBeInTheDocument()
    })

    it('has editable username, password, and role fields in create mode', async () => {
      const user = userEvent.setup()
      renderWithProviders(<UsersView />)

      const addButton = screen.getByRole('button', { name: i18n.t('auth:addUser') })
      await user.click(addButton)

      // Username input exists and is enabled
      const usernameInputs = screen.getAllByDisplayValue('')
      const usernameInput = usernameInputs.find(
        (el) => el instanceof HTMLInputElement && el.type === 'text'
      ) as HTMLInputElement | undefined
      expect(usernameInput).toBeTruthy()
      expect(usernameInput).toBeEnabled()

      // Role select exists and is enabled
      const roleSelect = screen.getByDisplayValue('Viewer') as HTMLSelectElement
      expect(roleSelect).toBeEnabled()
    })

    it('validates form fields before submission', async () => {
      const user = userEvent.setup()
      renderWithProviders(<UsersView />)

      const addButton = screen.getByRole('button', { name: i18n.t('auth:addUser') })
      await user.click(addButton)

      const submitButton = screen.getByRole('button', { name: i18n.t('auth:createUser') }) as HTMLButtonElement
      // Submit should be disabled when form is empty
      expect(submitButton).toBeDisabled()
    })

    it('enables submit when form is valid', async () => {
      const user = userEvent.setup()
      renderWithProviders(<UsersView />)

      const addButton = screen.getByRole('button', { name: i18n.t('auth:addUser') })
      await user.click(addButton)

      const usernameInputs = screen.getAllByDisplayValue('')
      const usernameInput = usernameInputs.find(
        (el) => el instanceof HTMLInputElement && el.type === 'text'
      ) as HTMLInputElement | undefined

      const passwordInputs = screen.getAllByDisplayValue('')
      const passwordInput = passwordInputs.find(
        (el) => el instanceof HTMLInputElement && el.type === 'password'
      ) as HTMLInputElement | undefined

      if (usernameInput) {
        await user.type(usernameInput, 'newuser')
      }

      if (passwordInput) {
        await user.type(passwordInput, 'securepass123')
      }

      const submitButton = screen.getByRole('button', { name: i18n.t('auth:createUser') }) as HTMLButtonElement
      expect(submitButton).toBeEnabled()
    })
  })

  describe('edit role flow', () => {
    beforeEach(() => {
      vi.mocked(useUsersDataModule.useUsersData).mockReturnValue({
        users: [mockAdminUser, mockViewerUser],
        isLoading: false,
        isError: false,
        error: null,
        refetch: vi.fn(),
      })
    })

    it('opens edit modal with user data pre-filled', async () => {
      const user = userEvent.setup()
      renderWithProviders(<UsersView />)

      // Find the edit button for the viewer user (second row)
      const editButtons = screen.getAllByRole('button', { name: i18n.t('auth:editUser') })
      await user.click(editButtons[1])

      expect(screen.getByRole('heading', { name: i18n.t('auth:editUser') })).toBeInTheDocument()
      expect(screen.getByDisplayValue(mockViewerUser.username)).toBeInTheDocument()
    })

    it('disables username field in edit_full mode', async () => {
      const user = userEvent.setup()
      renderWithProviders(<UsersView />)

      const editButtons = screen.getAllByRole('button', { name: i18n.t('auth:editUser') })
      await user.click(editButtons[1])

      const usernameInput = screen.getByDisplayValue(mockViewerUser.username) as HTMLInputElement
      expect(usernameInput).toBeDisabled()
    })

    it('disables role selector for sole admin user', async () => {
      vi.mocked(useUsersDataModule.useUsersData).mockReturnValue({
        users: [mockAdminUser],
        isLoading: false,
        isError: false,
        error: null,
        refetch: vi.fn(),
      })

      const user = userEvent.setup()
      renderWithProviders(<UsersView />)

      // With only 1 user, getAllByRole will return at least 2 (desktop + mobile)
      const editButtons = screen.getAllByRole('button', { name: i18n.t('auth:editUser') })
      await user.click(editButtons[0])

      const roleSelect = screen.getByDisplayValue('Admin') as HTMLSelectElement
      expect(roleSelect).toBeDisabled()
    })

    it('shows last-admin warning message for sole admin', async () => {
      vi.mocked(useUsersDataModule.useUsersData).mockReturnValue({
        users: [mockAdminUser],
        isLoading: false,
        isError: false,
        error: null,
        refetch: vi.fn(),
      })

      const user = userEvent.setup()
      renderWithProviders(<UsersView />)

      const editButtons = screen.getAllByRole('button', { name: i18n.t('auth:editUser') })
      await user.click(editButtons[0])

      expect(screen.getByText(i18n.t('auth:cannotDemoteLastAdmin'))).toBeInTheDocument()
    })

    it('disables submit button when trying to demote sole admin', async () => {
      vi.mocked(useUsersDataModule.useUsersData).mockReturnValue({
        users: [mockAdminUser],
        isLoading: false,
        isError: false,
        error: null,
        refetch: vi.fn(),
      })

      const user = userEvent.setup()
      renderWithProviders(<UsersView />)

      const editButtons = screen.getAllByRole('button', { name: i18n.t('auth:editUser') })
      await user.click(editButtons[0])

      const confirmButton = screen.getByRole('button', { name: i18n.t('common:confirm') }) as HTMLButtonElement
      expect(confirmButton).toBeDisabled()
    })
  })

  describe('change password flow', () => {
    beforeEach(() => {
      vi.mocked(useUsersDataModule.useUsersData).mockReturnValue({
        users: [mockAdminUser, mockViewerUser],
        isLoading: false,
        isError: false,
        error: null,
        refetch: vi.fn(),
      })
    })

    it('sends the signed-in admin to the profile page to change their own password', () => {
      renderWithProviders(<UsersView />)

      const links = screen.getAllByRole('link', { name: i18n.t('auth:changePassword') })
      expect(links).toHaveLength(1)
      expect(links[0]).toHaveAttribute('href', '/profile')
      expect(screen.getAllByRole('button', { name: i18n.t('auth:changePassword') })).toHaveLength(1)
    })

    it('opens modal in edit_password mode', async () => {
      const user = userEvent.setup()
      renderWithProviders(<UsersView />)

      const changePasswordButtons = screen.getAllByRole('button', { name: i18n.t('auth:changePassword') })
      await user.click(changePasswordButtons[0])

      expect(screen.getByRole('heading', { name: i18n.t('auth:changePassword') })).toBeInTheDocument()
    })

    it('shows only password field in edit_password mode', async () => {
      const user = userEvent.setup()
      renderWithProviders(<UsersView />)

      const changePasswordButtons = screen.getAllByRole('button', { name: i18n.t('auth:changePassword') })
      await user.click(changePasswordButtons[0])

      // Password label should be visible
      const passwordLabels = screen.queryAllByText(i18n.t('auth:newPasswordLabel'))
      expect(passwordLabels.length).toBeGreaterThan(0)
    })

    it('submit button is disabled when password is empty', async () => {
      const user = userEvent.setup()
      renderWithProviders(<UsersView />)

      const changePasswordButtons = screen.getAllByRole('button', { name: i18n.t('auth:changePassword') })
      await user.click(changePasswordButtons[0])

      const confirmButton = screen.getByRole('button', { name: i18n.t('common:confirm') }) as HTMLButtonElement
      expect(confirmButton).toBeDisabled()
    })

    it('submit button is enabled when password is entered', async () => {
      const user = userEvent.setup()
      renderWithProviders(<UsersView />)

      const changePasswordButtons = screen.getAllByRole('button', { name: i18n.t('auth:changePassword') })
      await user.click(changePasswordButtons[0])

      const passwordInputs = screen.getAllByDisplayValue('')
      const passwordInput = passwordInputs.find(
        (el) => el instanceof HTMLInputElement && el.type === 'password'
      ) as HTMLInputElement | undefined

      if (passwordInput) {
        await user.type(passwordInput, 'newpassword123')
      }

      const confirmButton = screen.getByRole('button', { name: i18n.t('common:confirm') }) as HTMLButtonElement
      expect(confirmButton).toBeEnabled()
    })
  })

  describe('delete flow', () => {
    beforeEach(() => {
      vi.mocked(useUsersDataModule.useUsersData).mockReturnValue({
        users: [mockAdminUser, mockViewerUser, mockAnotherAdminUser],
        isLoading: false,
        isError: false,
        error: null,
        refetch: vi.fn(),
      })
    })

    it('shows confirmation dialog when clicking Delete', async () => {
      const user = userEvent.setup()
      renderWithProviders(<UsersView />)

      const deleteButtons = screen.getAllByRole('button', { name: i18n.t('common:delete') })
      await user.click(deleteButtons[1])

      expect(screen.getByRole('heading', { name: i18n.t('auth:deleteUserTitle', { username: mockViewerUser.username }) })).toBeInTheDocument()
    })

    it('disables delete button for sole admin user', () => {
      vi.mocked(useUsersDataModule.useUsersData).mockReturnValue({
        users: [mockAdminUser],
        isLoading: false,
        isError: false,
        error: null,
        refetch: vi.fn(),
      })

      renderWithProviders(<UsersView />)

      const deleteButtons = screen.getAllByRole('button', { name: i18n.t('common:delete') })
      // First delete button (desktop) should be disabled
      expect(deleteButtons[0]).toBeDisabled()
    })

    it('delete button is enabled for non-last-admin users', () => {
      renderWithProviders(<UsersView />)

      // Admin user (not sole admin) should have delete button enabled
      const deleteButtons = screen.getAllByRole('button', { name: i18n.t('common:delete') })
      expect(deleteButtons[1]).toBeEnabled()
    })
  })

  describe('role change submit (regression: canDemote gate)', () => {
    it('enables submit button when a non-last-admin changes their role', async () => {
      vi.mocked(useUsersDataModule.useUsersData).mockReturnValue({
        users: [mockAdminUser, mockAnotherAdminUser],
        isLoading: false,
        isError: false,
        error: null,
        refetch: vi.fn(),
      })
      vi.mocked(authService.updateUserRole).mockResolvedValue({
        ...mockAnotherAdminUser,
        role: 'viewer',
      })

      const user = userEvent.setup()
      renderWithProviders(<UsersView />)

      // Open edit modal for the second admin (not the last admin — adminCount = 2)
      const editButtons = screen.getAllByRole('button', { name: i18n.t('auth:editUser') })
      await user.click(editButtons[1])

      // Confirm modal opened and role select is ENABLED (not last admin)
      expect(screen.getByRole('heading', { name: i18n.t('auth:editUser') })).toBeInTheDocument()
      const roleSelect = screen.getByDisplayValue('Admin') as HTMLSelectElement
      expect(roleSelect).toBeEnabled()

      // Change the role to viewer
      await user.selectOptions(roleSelect, 'viewer')
      expect(roleSelect).toHaveValue('viewer')

      // Submit button MUST be enabled after a role change
      const confirmButton = screen.getByRole('button', { name: i18n.t('common:confirm') }) as HTMLButtonElement
      expect(confirmButton).toBeEnabled()

      // Click submit and assert the mutation is called with the new role
      await user.click(confirmButton)

      await waitFor(() => {
        expect(authService.updateUserRole).toHaveBeenCalledWith(mockAnotherAdminUser.id, 'viewer')
      })
    })
  })

  describe('last-admin protection', () => {
    it('protects the last admin from demotion via UI', async () => {
      vi.mocked(useUsersDataModule.useUsersData).mockReturnValue({
        users: [mockAdminUser],
        isLoading: false,
        isError: false,
        error: null,
        refetch: vi.fn(),
      })

      const user = userEvent.setup()
      renderWithProviders(<UsersView />)

      const editButtons = screen.getAllByRole('button', { name: i18n.t('auth:editUser') })
      await user.click(editButtons[0])

      // Verify role selector is disabled
      const roleSelect = screen.getByDisplayValue('Admin') as HTMLSelectElement
      expect(roleSelect).toBeDisabled()

      // Verify warning is shown
      expect(screen.getByText(i18n.t('auth:cannotDemoteLastAdmin'))).toBeInTheDocument()
    })
  })

  describe('state renders', () => {
    it('shows loading state when data is loading', () => {
      vi.mocked(useUsersDataModule.useUsersData).mockReturnValue({
        users: [],
        isLoading: true,
        isError: false,
        error: null,
        refetch: vi.fn(),
      })

      renderWithProviders(<UsersView />)

      // Header should be visible even in loading state
      expect(screen.getByText(i18n.t('auth:userManagement'))).toBeInTheDocument()
    })

    it('shows empty state when user list is empty', () => {
      vi.mocked(useUsersDataModule.useUsersData).mockReturnValue({
        users: [],
        isLoading: false,
        isError: false,
        error: null,
        refetch: vi.fn(),
      })

      renderWithProviders(<UsersView />)

      expect(screen.getByText(i18n.t('auth:noUsersYet'))).toBeInTheDocument()
      expect(screen.getByText(i18n.t('auth:addFirstUser'))).toBeInTheDocument()
    })

    it('shows error state when query fails', () => {
      vi.mocked(useUsersDataModule.useUsersData).mockReturnValue({
        users: [],
        isLoading: false,
        isError: true,
        error: new Error('Failed to fetch users'),
        refetch: vi.fn(),
      })

      renderWithProviders(<UsersView />)

      expect(screen.getByText(i18n.t('auth:userManagement'))).toBeInTheDocument()
    })

    it('hides table when loading', () => {
      vi.mocked(useUsersDataModule.useUsersData).mockReturnValue({
        users: [],
        isLoading: true,
        isError: false,
        error: null,
        refetch: vi.fn(),
      })

      renderWithProviders(<UsersView />)

      expect(screen.queryByText(mockAdminUser.username)).not.toBeInTheDocument()
    })

    it('hides table when empty', () => {
      vi.mocked(useUsersDataModule.useUsersData).mockReturnValue({
        users: [],
        isLoading: false,
        isError: false,
        error: null,
        refetch: vi.fn(),
      })

      renderWithProviders(<UsersView />)

      expect(screen.queryByText(mockAdminUser.username)).not.toBeInTheDocument()
    })

    it('shows table when data is present', () => {
      vi.mocked(useUsersDataModule.useUsersData).mockReturnValue({
        users: [mockAdminUser, mockViewerUser],
        isLoading: false,
        isError: false,
        error: null,
        refetch: vi.fn(),
      })

      renderWithProviders(<UsersView />)

      const adminUsernames = screen.getAllByText(mockAdminUser.username)
      expect(adminUsernames.length).toBeGreaterThan(0)

      const viewerUsernames = screen.getAllByText(mockViewerUser.username)
      expect(viewerUsernames.length).toBeGreaterThan(0)
    })
  })

  describe('table renders', () => {
    beforeEach(() => {
      vi.mocked(useUsersDataModule.useUsersData).mockReturnValue({
        users: [mockAdminUser, mockViewerUser],
        isLoading: false,
        isError: false,
        error: null,
        refetch: vi.fn(),
      })
    })

    it('renders all users with their usernames', () => {
      renderWithProviders(<UsersView />)

      // Users appear in both desktop table and mobile cards
      const adminUsernames = screen.getAllByText(mockAdminUser.username)
      expect(adminUsernames.length).toBeGreaterThan(0)

      const viewerUsernames = screen.getAllByText(mockViewerUser.username)
      expect(viewerUsernames.length).toBeGreaterThan(0)
    })

    it('renders role badges correctly', () => {
      renderWithProviders(<UsersView />)

      const roleBadges = screen.getAllByText(i18n.t('auth:roles.admin'))
      expect(roleBadges.length).toBeGreaterThan(0)

      const viewerBadges = screen.getAllByText(i18n.t('auth:roles.viewer'))
      expect(viewerBadges.length).toBeGreaterThan(0)
    })

    it('renders action buttons for each user', () => {
      renderWithProviders(<UsersView />)

      // With 2 users, we expect 2 Edit buttons (one per user)
      // Note: screen.getAllByRole finds buttons across both desktop and mobile views
      // So for 2 users we get 2 desktop + 2 mobile = 4 total
      const editButtons = screen.getAllByRole('button', { name: i18n.t('auth:editUser') })
      expect(editButtons.length).toBeGreaterThanOrEqual(2)

      // The signed-in admin's own row links to the profile page instead.
      const changePasswordButtons = screen.getAllByRole('button', { name: i18n.t('auth:changePassword') })
      expect(changePasswordButtons.length).toBeGreaterThanOrEqual(1)
      expect(screen.getAllByRole('link', { name: i18n.t('auth:changePassword') }).length).toBeGreaterThanOrEqual(1)

      const deleteButtons = screen.getAllByRole('button', { name: i18n.t('common:delete') })
      expect(deleteButtons.length).toBeGreaterThanOrEqual(2)
    })
  })

  describe('add user button state', () => {
    it('renders add user button when data is loaded', () => {
      vi.mocked(useUsersDataModule.useUsersData).mockReturnValue({
        users: [mockAdminUser],
        isLoading: false,
        isError: false,
        error: null,
        refetch: vi.fn(),
      })

      renderWithProviders(<UsersView />)

      const addButton = screen.getByRole('button', { name: i18n.t('auth:addUser') })
      expect(addButton).toBeInTheDocument()
    })
  })

  describe('mutation flows', () => {
    it('creates a user from the create modal', async () => {
      vi.mocked(useUsersDataModule.useUsersData).mockReturnValue({
        users: [mockAdminUser],
        isLoading: false,
        isError: false,
        error: null,
        refetch: vi.fn(),
      })
      vi.mocked(authService.createUser).mockResolvedValue({
        id: 'new-user-1',
        username: 'newuser',
        role: 'viewer',
        createdAt: '2026-05-22T00:00:00Z',
      })

      const user = userEvent.setup()
      renderWithProviders(<UsersView />)

      await user.click(screen.getByRole('button', { name: i18n.t('auth:addUser') }))
      await user.type(screen.getByRole('textbox'), 'newuser')

      const passwordInputs = screen.getAllByDisplayValue('')
      const passwordInput = passwordInputs.find(
        (el) => el instanceof HTMLInputElement && el.type === 'password'
      ) as HTMLInputElement | undefined

      expect(passwordInput).toBeTruthy()
      if (passwordInput) {
        await user.type(passwordInput, 'securepass123')
      }

      await user.click(screen.getByRole('button', { name: i18n.t('auth:createUser') }))

      await waitFor(() => {
        expect(authService.createUser).toHaveBeenCalledWith('newuser', 'securepass123', 'viewer')
      })
    })

    it('shows an error toast and keeps the create modal open when create user fails', async () => {
      vi.mocked(useUsersDataModule.useUsersData).mockReturnValue({
        users: [mockAdminUser],
        isLoading: false,
        isError: false,
        error: null,
        refetch: vi.fn(),
      })
      vi.mocked(authService.createUser).mockRejectedValue({
        response: { data: { error: { message: 'Username already exists' } } },
      })

      const user = userEvent.setup()
      renderWithProviders(<UsersView />)

      await user.click(screen.getByRole('button', { name: i18n.t('auth:addUser') }))
      await user.type(screen.getByRole('textbox'), 'existinguser')

      const passwordInputs = screen.getAllByDisplayValue('')
      const passwordInput = passwordInputs.find(
        (el) => el instanceof HTMLInputElement && el.type === 'password'
      ) as HTMLInputElement | undefined

      expect(passwordInput).toBeTruthy()
      if (passwordInput) {
        await user.type(passwordInput, 'securepass123')
      }

      await user.click(screen.getByRole('button', { name: i18n.t('auth:createUser') }))

      await waitFor(() => {
        expect(authService.createUser).toHaveBeenCalledWith('existinguser', 'securepass123', 'viewer')
      })
      expect(await screen.findByText('Username already exists')).toBeInTheDocument()
      expect(screen.getByRole('heading', { name: i18n.t('auth:createUser') })).toBeInTheDocument()
    })

    it('changes a user password from the password modal', async () => {
      vi.mocked(useUsersDataModule.useUsersData).mockReturnValue({
        users: [mockAdminUser, mockViewerUser],
        isLoading: false,
        isError: false,
        error: null,
        refetch: vi.fn(),
      })
      vi.mocked(authService.changePassword).mockResolvedValue(undefined)

      const user = userEvent.setup()
      renderWithProviders(<UsersView />)

      const changePasswordButtons = screen.getAllByRole('button', { name: i18n.t('auth:changePassword') })
      await user.click(changePasswordButtons[0])

      const passwordInputs = screen.getAllByDisplayValue('')
      const passwordInput = passwordInputs.find(
        (el) => el instanceof HTMLInputElement && el.type === 'password'
      ) as HTMLInputElement | undefined

      expect(passwordInput).toBeTruthy()
      if (passwordInput) {
        await user.type(passwordInput, 'newpassword123')
      }

      await user.click(screen.getByRole('button', { name: i18n.t('common:confirm') }))

      await waitFor(() => {
        expect(authService.changePassword).toHaveBeenCalledWith(mockViewerUser.id, 'newpassword123')
      })
    })

    it('shows an error toast and keeps the password modal open when password reset fails', async () => {
      vi.mocked(useUsersDataModule.useUsersData).mockReturnValue({
        users: [mockAdminUser, mockViewerUser],
        isLoading: false,
        isError: false,
        error: null,
        refetch: vi.fn(),
      })
      vi.mocked(authService.changePassword).mockRejectedValue({
        response: { data: { error: { message: 'Password reset failed' } } },
      })

      const user = userEvent.setup()
      renderWithProviders(<UsersView />)

      const changePasswordButtons = screen.getAllByRole('button', { name: i18n.t('auth:changePassword') })
      await user.click(changePasswordButtons[0])

      const passwordInputs = screen.getAllByDisplayValue('')
      const passwordInput = passwordInputs.find(
        (el) => el instanceof HTMLInputElement && el.type === 'password'
      ) as HTMLInputElement | undefined

      expect(passwordInput).toBeTruthy()
      if (passwordInput) {
        await user.type(passwordInput, 'newpassword123')
      }

      await user.click(screen.getByRole('button', { name: i18n.t('common:confirm') }))

      await waitFor(() => {
        expect(authService.changePassword).toHaveBeenCalledWith(mockViewerUser.id, 'newpassword123')
      })
      expect(await screen.findByText('Password reset failed')).toBeInTheDocument()
      expect(screen.getByRole('heading', { name: i18n.t('auth:changePassword') })).toBeInTheDocument()
    })

    it('deletes a user after confirmation', async () => {
      vi.mocked(useUsersDataModule.useUsersData).mockReturnValue({
        users: [mockAdminUser, mockViewerUser, mockAnotherAdminUser],
        isLoading: false,
        isError: false,
        error: null,
        refetch: vi.fn(),
      })
      vi.mocked(authService.deleteUser).mockResolvedValue(undefined)

      const user = userEvent.setup()
      renderWithProviders(<UsersView />)

      const deleteButtons = screen.getAllByRole('button', { name: i18n.t('common:delete') })
      await user.click(deleteButtons[1])
      await user.click(screen.getByRole('button', { name: i18n.t('common:confirm') }))

      await waitFor(() => {
        expect(authService.deleteUser).toHaveBeenCalled()
      })

      expect(vi.mocked(authService.deleteUser).mock.calls[0]?.[0]).toBe(mockViewerUser.id)
    })
  })

  describe('modal closure', () => {
    it('closes modal when clicking close button', async () => {
      vi.mocked(useUsersDataModule.useUsersData).mockReturnValue({
        users: [mockAdminUser],
        isLoading: false,
        isError: false,
        error: null,
        refetch: vi.fn(),
      })

      const user = userEvent.setup()
      renderWithProviders(<UsersView />)

      const addButton = screen.getByRole('button', { name: i18n.t('auth:addUser') })
      await user.click(addButton)

      expect(screen.getByRole('heading', { name: i18n.t('auth:createUser') })).toBeInTheDocument()

      const cancelButton = screen.getByRole('button', { name: i18n.t('common:cancel') })
      await user.click(cancelButton)

      expect(screen.queryByRole('heading', { name: i18n.t('auth:createUser') })).not.toBeInTheDocument()
    })
  })
})

describe('UsersView presentation (T11)', () => {
  const mockUsers = (users: User[]) =>
    vi.mocked(useUsersDataModule.useUsersData).mockReturnValue({
      users,
      isLoading: false,
      isError: false,
      error: null,
      refetch: vi.fn(),
    })

  beforeEach(() => {
    vi.clearAllMocks()
    vi.mocked(useAuthModule.useAuth).mockReturnValue(
      buildAuthMock({
        isAuthenticated: true,
        isInitialized: true,
        user: buildUserMock({ id: 'admin-1', username: 'admin', role: 'admin' }),
      })
    )
  })

  it('labels the add button "Add user", once', () => {
    mockUsers([mockAdminUser])
    renderWithProviders(<UsersView />)

    expect(screen.getByRole('button', { name: 'Add user' })).toHaveTextContent(/^Add user$/)
  })

  it('shows each role as a catalog-labelled chip, not the raw role value', () => {
    mockUsers([mockAdminUser, mockViewerUser])
    renderWithProviders(<UsersView />)

    const [, adminRow, viewerRow] = screen.getAllByRole('row')
    expect(within(adminRow).getByRole('status')).toHaveTextContent(/^Admin$/)
    expect(within(viewerRow).getByRole('status')).toHaveTextContent(/^Viewer$/)
    expect(i18n.t('auth:roles.admin')).toBe('Admin')
    expect(i18n.t('auth:roles.viewer')).toBe('Viewer')
  })

  it('keeps Delete disabled on your own account and says why', () => {
    mockUsers([mockAdminUser, mockAnotherAdminUser])
    renderWithProviders(<UsersView />)

    const [, ownRow, otherRow] = screen.getAllByRole('row')
    const ownDelete = within(ownRow).getByRole('button', { name: 'Delete' })
    expect(ownDelete).toBeDisabled()
    expect(ownDelete).toHaveAccessibleDescription("You can't delete your own account.")
    expect(within(ownRow).getByText("You can't delete your own account.")).toBeVisible()

    const otherDelete = within(otherRow).getByRole('button', { name: 'Delete' })
    expect(otherDelete).toBeEnabled()
    expect(otherDelete).not.toHaveAccessibleDescription()
  })

  it('names the user in the danger delete confirmation', async () => {
    mockUsers([mockAdminUser, mockViewerUser])
    const user = userEvent.setup()
    renderWithProviders(<UsersView />)

    const [, , viewerRow] = screen.getAllByRole('row')
    await user.click(within(viewerRow).getByRole('button', { name: 'Delete' }))

    const dialog = screen.getByRole('dialog')
    expect(within(dialog).getByRole('heading', { name: 'Delete user viewer?' })).toBeInTheDocument()
    expect(dialog).toHaveTextContent('viewer is removed and can no longer sign in to NRCC. This cannot be undone.')
    expect(authService.deleteUser).not.toHaveBeenCalled()
  })
})
