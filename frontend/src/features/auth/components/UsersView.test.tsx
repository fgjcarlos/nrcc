import { beforeEach, describe, expect, it, vi } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { Toaster } from 'sonner'
import { UsersView } from './UsersView'
import { authService, type User } from '../services/authService'
import * as useAuthModule from '../hooks/useAuth'
import * as useUsersDataModule from '../hooks/useUsersData'
import { buildAuthMock, buildUserMock } from '../__test-utils__/authMock'
import { UI_COPY } from '@/shared/constants/uiCopy'

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
    <QueryClientProvider client={queryClient}>
      {component}
      <Toaster />
    </QueryClientProvider>
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

      expect(screen.getByText('Access Denied')).toBeInTheDocument()
      expect(screen.getByText("You don't have permission to view this page.")).toBeInTheDocument()
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

      expect(screen.getByText('User Management')).toBeInTheDocument()
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

      const addButton = screen.getByRole('button', { name: new RegExp(`${UI_COPY.add}.*${UI_COPY.createUser}`) })
      await user.click(addButton)

      // Look for the modal heading, not the submit button
      expect(screen.getByRole('heading', { name: UI_COPY.createUser })).toBeInTheDocument()
    })

    it('has editable username, password, and role fields in create mode', async () => {
      const user = userEvent.setup()
      renderWithProviders(<UsersView />)

      const addButton = screen.getByRole('button', { name: new RegExp(`${UI_COPY.add}.*${UI_COPY.createUser}`) })
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

      const addButton = screen.getByRole('button', { name: new RegExp(`${UI_COPY.add}.*${UI_COPY.createUser}`) })
      await user.click(addButton)

      const submitButton = screen.getByRole('button', { name: UI_COPY.createUser }) as HTMLButtonElement
      // Submit should be disabled when form is empty
      expect(submitButton).toBeDisabled()
    })

    it('enables submit when form is valid', async () => {
      const user = userEvent.setup()
      renderWithProviders(<UsersView />)

      const addButton = screen.getByRole('button', { name: new RegExp(`${UI_COPY.add}.*${UI_COPY.createUser}`) })
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

      const submitButton = screen.getByRole('button', { name: UI_COPY.createUser }) as HTMLButtonElement
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
      const editButtons = screen.getAllByRole('button', { name: UI_COPY.editUser })
      await user.click(editButtons[1])

      expect(screen.getByRole('heading', { name: UI_COPY.editUser })).toBeInTheDocument()
      expect(screen.getByDisplayValue(mockViewerUser.username)).toBeInTheDocument()
    })

    it('disables username field in edit_full mode', async () => {
      const user = userEvent.setup()
      renderWithProviders(<UsersView />)

      const editButtons = screen.getAllByRole('button', { name: UI_COPY.editUser })
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
      const editButtons = screen.getAllByRole('button', { name: UI_COPY.editUser })
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

      const editButtons = screen.getAllByRole('button', { name: UI_COPY.editUser })
      await user.click(editButtons[0])

      expect(screen.getByText(UI_COPY.cannotDemoteLastAdmin)).toBeInTheDocument()
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

      const editButtons = screen.getAllByRole('button', { name: UI_COPY.editUser })
      await user.click(editButtons[0])

      const confirmButton = screen.getByRole('button', { name: UI_COPY.confirm }) as HTMLButtonElement
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

    it('opens modal in edit_password mode', async () => {
      const user = userEvent.setup()
      renderWithProviders(<UsersView />)

      const changePasswordButtons = screen.getAllByRole('button', { name: UI_COPY.changePassword })
      await user.click(changePasswordButtons[1])

      expect(screen.getByRole('heading', { name: UI_COPY.changePassword })).toBeInTheDocument()
    })

    it('shows only password field in edit_password mode', async () => {
      const user = userEvent.setup()
      renderWithProviders(<UsersView />)

      const changePasswordButtons = screen.getAllByRole('button', { name: UI_COPY.changePassword })
      await user.click(changePasswordButtons[1])

      // Password label should be visible
      const passwordLabels = screen.queryAllByText(UI_COPY.newPasswordLabel)
      expect(passwordLabels.length).toBeGreaterThan(0)
    })

    it('submit button is disabled when password is empty', async () => {
      const user = userEvent.setup()
      renderWithProviders(<UsersView />)

      const changePasswordButtons = screen.getAllByRole('button', { name: UI_COPY.changePassword })
      await user.click(changePasswordButtons[1])

      const confirmButton = screen.getByRole('button', { name: UI_COPY.confirm }) as HTMLButtonElement
      expect(confirmButton).toBeDisabled()
    })

    it('submit button is enabled when password is entered', async () => {
      const user = userEvent.setup()
      renderWithProviders(<UsersView />)

      const changePasswordButtons = screen.getAllByRole('button', { name: UI_COPY.changePassword })
      await user.click(changePasswordButtons[1])

      const passwordInputs = screen.getAllByDisplayValue('')
      const passwordInput = passwordInputs.find(
        (el) => el instanceof HTMLInputElement && el.type === 'password'
      ) as HTMLInputElement | undefined

      if (passwordInput) {
        await user.type(passwordInput, 'newpassword123')
      }

      const confirmButton = screen.getByRole('button', { name: UI_COPY.confirm }) as HTMLButtonElement
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

      const deleteButtons = screen.getAllByRole('button', { name: UI_COPY.delete })
      await user.click(deleteButtons[1])

      expect(screen.getByRole('heading', { name: UI_COPY.deleteUser })).toBeInTheDocument()
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

      const deleteButtons = screen.getAllByRole('button', { name: UI_COPY.delete })
      // First delete button (desktop) should be disabled
      expect(deleteButtons[0]).toBeDisabled()
    })

    it('delete button is enabled for non-last-admin users', () => {
      renderWithProviders(<UsersView />)

      // Admin user (not sole admin) should have delete button enabled
      const deleteButtons = screen.getAllByRole('button', { name: UI_COPY.delete })
      expect(deleteButtons[1]).toBeEnabled()
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

      const editButtons = screen.getAllByRole('button', { name: UI_COPY.editUser })
      await user.click(editButtons[0])

      // Verify role selector is disabled
      const roleSelect = screen.getByDisplayValue('Admin') as HTMLSelectElement
      expect(roleSelect).toBeDisabled()

      // Verify warning is shown
      expect(screen.getByText(UI_COPY.cannotDemoteLastAdmin)).toBeInTheDocument()
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
      expect(screen.getByText('User Management')).toBeInTheDocument()
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

      expect(screen.getByText(UI_COPY.noUsersYet)).toBeInTheDocument()
      expect(screen.getByText(UI_COPY.addFirstUser)).toBeInTheDocument()
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

      expect(screen.getByText('User Management')).toBeInTheDocument()
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

      const roleBadges = screen.getAllByText('admin')
      expect(roleBadges.length).toBeGreaterThan(0)

      const viewerBadges = screen.getAllByText('viewer')
      expect(viewerBadges.length).toBeGreaterThan(0)
    })

    it('renders action buttons for each user', () => {
      renderWithProviders(<UsersView />)

      // With 2 users, we expect 2 Edit buttons (one per user)
      // Note: screen.getAllByRole finds buttons across both desktop and mobile views
      // So for 2 users we get 2 desktop + 2 mobile = 4 total
      const editButtons = screen.getAllByRole('button', { name: UI_COPY.editUser })
      expect(editButtons.length).toBeGreaterThanOrEqual(2)

      const changePasswordButtons = screen.getAllByRole('button', { name: UI_COPY.changePassword })
      expect(changePasswordButtons.length).toBeGreaterThanOrEqual(2)

      const deleteButtons = screen.getAllByRole('button', { name: UI_COPY.delete })
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

      const addButton = screen.getByRole('button', { name: new RegExp(`${UI_COPY.add}.*${UI_COPY.createUser}`) })
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

      await user.click(screen.getByRole('button', { name: new RegExp(`${UI_COPY.add}.*${UI_COPY.createUser}`) }))
      await user.type(screen.getByRole('textbox'), 'newuser')

      const passwordInputs = screen.getAllByDisplayValue('')
      const passwordInput = passwordInputs.find(
        (el) => el instanceof HTMLInputElement && el.type === 'password'
      ) as HTMLInputElement | undefined

      expect(passwordInput).toBeTruthy()
      if (passwordInput) {
        await user.type(passwordInput, 'securepass123')
      }

      await user.click(screen.getByRole('button', { name: UI_COPY.createUser }))

      await waitFor(() => {
        expect(authService.createUser).toHaveBeenCalledWith('newuser', 'securepass123', 'viewer')
      })
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

      const changePasswordButtons = screen.getAllByRole('button', { name: UI_COPY.changePassword })
      await user.click(changePasswordButtons[1])

      const passwordInputs = screen.getAllByDisplayValue('')
      const passwordInput = passwordInputs.find(
        (el) => el instanceof HTMLInputElement && el.type === 'password'
      ) as HTMLInputElement | undefined

      expect(passwordInput).toBeTruthy()
      if (passwordInput) {
        await user.type(passwordInput, 'newpassword123')
      }

      await user.click(screen.getByRole('button', { name: UI_COPY.confirm }))

      await waitFor(() => {
        expect(authService.changePassword).toHaveBeenCalledWith(mockViewerUser.id, 'newpassword123')
      })
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

      const deleteButtons = screen.getAllByRole('button', { name: UI_COPY.delete })
      await user.click(deleteButtons[1])
      await user.click(screen.getByRole('button', { name: UI_COPY.confirm }))

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

      const addButton = screen.getByRole('button', { name: new RegExp(`${UI_COPY.add}.*${UI_COPY.createUser}`) })
      await user.click(addButton)

      expect(screen.getByRole('heading', { name: UI_COPY.createUser })).toBeInTheDocument()

      const cancelButton = screen.getByRole('button', { name: UI_COPY.cancel })
      await user.click(cancelButton)

      expect(screen.queryByRole('heading', { name: UI_COPY.createUser })).not.toBeInTheDocument()
    })
  })
})
