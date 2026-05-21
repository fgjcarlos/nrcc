# Tasks: Issue #112 — Complete Admin User Management

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | 500–620 LOC |
| 400-line budget risk | High |
| Chained PRs recommended | Yes |
| Suggested split | Phase 1 → Phase 2 → Phase 3 (stacked-to-main) |
| Delivery strategy | ask-on-risk (requires decision on chaining before apply) |

**Decision needed before apply**: Yes
**Chained PRs recommended**: Yes
**Chain strategy**: stacked-to-main
**400-line budget risk**: High

### Breakdown by Phase

| Phase | Tasks | Estimated LOC | Focus |
|-------|-------|--------------|-------|
| Phase 1 | 4 | ~60 LOC | Backend: UpdateUser handler, last-admin guard, route registration |
| Phase 2 | 8 | ~130 LOC | Frontend: authService, modal modes, UI components, uiCopy, hooks |
| Phase 3 | 9 | ~300 LOC | Tests: integration flows, last-admin protection, state renders |
| **Total** | **21** | **~490 LOC** | |

### Implementation Order

**Phase 1 (Backend)** provides the HTTP contract that Phase 2 (Frontend) depends on. Phase 2 adds UI layer and wires the integration. Phase 3 verifies all three modes and edge cases. This order allows:
- Phase 1 → Phase 2: Unblock frontend once backend tests pass
- Phase 2 → Phase 3: Tests validate integration against live backend

---

## Phase 1: Backend — User Update Handler (~60 LOC)

**Target**: `main` branch
**Rationale**: New API endpoint; all other changes depend on this.

- [x] 1.1 Create request/response structs in `internal/handler/auth.go`
  - `UpdateUserRequest { Role *string }` (pointer = optional field in JSON)
  - Ensure `json:"role,omitempty"` tag for optional serialization
  - Validate: at least one field present (error at handler entry if empty)

- [x] 1.2 Implement `UpdateUser` handler method in `internal/handler/auth.go` (~40 LOC)
  - Signature: `func (h *AuthHandler) UpdateUser(w http.ResponseWriter, r *http.Request)`
  - Extract ID from `chi.URLParam(r, "id")`
  - Bind JSON to `UpdateUserRequest`, validate not empty
  - Check admin authorization (same pattern as existing `DeleteUser`)
  - Validate role if present (must be `admin` or `viewer`)
  - Call handler's last-admin guard before updating (see 1.3)
  - Call `authSvc.UpdateUser(user)` to persist
  - Return 200 with user object (same serialization as GET /users/{id})
  - Error codes: 400 (bad body/invalid role), 403 (not admin / last admin), 404 (user not found)

- [x] 1.3 Implement last-admin guard in `UpdateUser` handler
  - Before updating: if `UpdateUserRequest.Role` is `"viewer"` and user is currently `admin`
  - Count admins in database (via `authSvc.GetAllUsers()` or dedicated query)
  - If count == 1, return 403 with message `"cannot demote last admin"` (or similar)
  - Matches existing `DeleteUser` pattern for consistency

- [x] 1.4 Register PATCH route in `internal/server/server.go`
  - In protected auth routes group: `r.Patch("/users/{id}", authHandler.UpdateUser)`
  - Ensure route comes after auth middleware check (already in place)
  - Verify via test: calling without auth token returns 401

---

## Phase 2: Frontend — Modal Refactor & Service Integration (~130 LOC)

**Target**: Phase 1 branch (or `feature/issue-112-backend` if auto-chaining)
**Rationale**: Depends on Phase 1 API; unblocks Phase 3 tests.

- [x] 2.1 Add `updateUserRole()` method to `authService.ts` (~8 LOC)
   - Signature: `updateUserRole(id: string, role: 'admin' | 'viewer'): Promise<User>`
   - Calls `PATCH /auth/users/{id}` with body `{ role }`
   - Returns typed `User` object from response `data.data`

- [x] 2.2 Extend `useUsersActions` hook with update mutation (~15 LOC)
   - Add `updateRoleMutation = useMutation({ mutationFn: (args: { id, role }) => authService.updateUserRole(...) })`
   - Wire `onSuccess`: invalidate `['users']` cache (via `queryClient.invalidateQueries`)
   - Wire `onError`: call `toast.error(error.message)` for both validation and last-admin 403
   - Export mutation so `UserModal` can access `isPending` and `mutate`

- [x] 2.3 Add UI copy keys to `uiCopy.ts` (~12 LOC)
   - `editUser: 'Edit User'`
   - `changeRole: 'Change Role'`
   - `changePassword: 'Change Password'`
   - `createUser: 'Create User'`
   - `roleLabel: 'Role'`
   - `usernameLabel: 'Username'`
   - `passwordLabel: 'Password'`
   - `newPasswordLabel: 'New Password'`
   - `cannotDemoteLastAdmin: 'Cannot demote the last admin user'`
   - `noUsersYet: 'No users yet'`
   - `addFirstUser: 'Add the first user to get started'`

- [x] 2.4 Create `UserTable.tsx` presentational component (~35 LOC)
   - Receives props: `users: User[]`, `onEdit: (user) => void`, `onDelete: (user) => void`, `onChangePassword: (user) => void`
   - Renders responsive layout:
     - **md+**: standard HTML table with thead/tbody, columns: Username, Role, Actions
     - **mobile (< md)**: stacked card per user (use `div` with card styling, DaisyUI classes)
   - Action buttons: "Edit", "Change Password", "Delete"
     - Disable "Delete" if user is sole admin (pass `adminCount` prop or derive locally)
     - All copy from `uiCopy`

- [x] 2.5 Create `UserModal.tsx` presentational component (~50 LOC)
   - Receives props: `mode: 'create' | 'edit_full' | 'edit_password'`, `editingUser: User | null`, `isPending: boolean`, `onSubmit: (data) => void`, `onClose: () => void`
   - Render form based on mode:
     - **create**: username (editable, required), password (editable, required), role (select, required)
     - **edit_full**: username (disabled), role (select, editable), password (text, optional for password reset)
     - **edit_password**: password field only (required)
   - Last-admin guard in `edit_full` mode:
     - If editing sole admin, disable role selector and show warning: `uiCopy.cannotDemoteLastAdmin`
     - Disable submit if role change to `viewer` for sole admin
   - Submit button: disabled when `isPending`, show spinner via `<Spinner />` (or existing loading indicator)
   - Close button: always enabled
   - Modal closes on successful mutation (handled by parent `UsersView`; this component does not auto-close)

- [x] 2.6 Refactor `UsersView.tsx` to use new components (~25 LOC net delta)
   - Create state: `modalState: null | { mode: ModalMode; editingUser: User | null }`
   - Replace binary create/password logic with three-mode discriminated union
   - Delegate table render to `<UserTable users={users} onEdit={...} onDelete={...} onChangePassword={...} />`
   - Delegate modal render to `<UserModal mode={...} editingUser={...} isPending={...} onSubmit={...} onClose={...} />`
   - Handler functions: `openCreateModal`, `openEditModal`, `openPasswordModal`, `closeModal`
   - Wire mutations:
     - Create: `createUserMutation.mutate(formData)`
     - Edit role: `updateRoleMutation.mutate({ id, role })`
     - Change password: `changePasswordMutation.mutate({ id, password })`
     - Delete: show confirm dialog, call `deleteUserMutation.mutate(id)`
   - Close modal only on mutation success (mutation's `onSuccess` calls `closeModal`)
   - Render with access control: if not admin, show `<AccessDenied />`

- [x] 2.7 Add state container for loading/empty/error (~20 LOC in UsersView)
   - Use existing `StateContainer` or create wrapper logic
   - `isLoading` → show loading skeleton/spinner
   - `data.length === 0` → show empty state with "No users yet" and "Add the first user" copy
   - `isError` → show error state with error message
   - Otherwise: render `<UserTable />`

- [x] 2.8 Verify all action buttons have loading/disabled state
   - "Add User" button: disabled when any mutation is `isPending`
   - "Edit", "Change Password", "Delete" buttons in table: disabled when any mutation is `isPending` (or just their mutation is pending)
   - Submit button in modal: disabled when `isPending`, shows spinner

---

## Phase 3: Tests — Full Integration Suite (~300 LOC)

**Target**: Phase 2 branch
**Rationale**: Validates all three modal modes, mutations, and guards.

- [ ] 3.1 Setup test file: `UsersView.test.tsx` (~15 LOC)
  - Import `describe`, `it`, `expect`, `render`, `screen`, `fireEvent` from vitest/testing-library
  - Mock `authService` via `vi.mock('@/features/auth/services/authService')`
  - Mock `useAuth` hook to return `{ user: { role: 'admin', id: 'admin-1' } }`
  - Mock `useUsersQuery` to return typed user list
  - Setup `QueryClient` for mutation tests

- [ ] 3.2 Access control tests (~20 LOC)
  - Test: Non-admin user (role=`viewer`) sees access-denied message
  - Test: Admin user sees user table and action buttons
  - Assertion: "Access denied" text visible for non-admin; table visible for admin

- [ ] 3.3 Create flow tests (~40 LOC)
  - Test: Click "Add User" → modal opens in `create` mode
  - Test: Render validation: username/password/role fields editable
  - Test: Submit with valid data → mutation called, modal closes on success, table refreshes
  - Test: Submit with empty username → validation error shown, modal stays open
  - Test: Submit → API error (e.g., 500) → error toast shown, modal stays open

- [ ] 3.4 Edit role flow tests (~45 LOC)
  - Test: Click "Edit" on user → modal opens in `edit_full` mode with user data pre-filled
  - Test: Username field is disabled
  - Test: Change role to `viewer` → submit → mutation called with updated role
  - Test: On success → table re-renders with new role
  - Test: Last-admin guard: sole admin user → role selector disabled, submit disabled, warning shown
  - Test: Last-admin guard: API returns 403 → error toast shown, modal stays open

- [ ] 3.5 Change password flow tests (~35 LOC)
  - Test: Click "Change Password" on user → modal opens in `edit_password` mode
  - Test: Only password field visible
  - Test: Submit with new password → mutation called (updateRoleMutation or ChangePassword?)
  - Test: On success → modal closes, toast shown, table refreshes
  - Test: Submit → API error → error toast shown, modal stays open

- [ ] 3.6 Delete flow tests (~40 LOC)
  - Test: Click "Delete" on non-last-admin user → confirmation dialog shown
  - Test: Confirm delete → mutation called, user removed from table
  - Test: Delete button disabled for sole admin user
  - Test: Try to delete last admin → button is disabled (UI guard)
  - Test: Delete → API error (e.g., 500) → error toast shown

- [ ] 3.7 Last-admin protection tests (~50 LOC)
  - Test: UI disables role selector for sole admin in `edit_full` mode
  - Test: Warning message shown: `uiCopy.cannotDemoteLastAdmin`
  - Test: Submit button disabled if changing sole admin role to `viewer`
  - Test: Backend 403 on demotion attempt → caught in `useUsersActions.updateRoleMutation.onError`
  - Test: Backend 403 on delete attempt → caught in `useUsersActions.deleteUserMutation.onError`
  - Assert: error toast displayed with appropriate message

- [ ] 3.8 State renders tests (~30 LOC)
  - Test: `useUsersQuery` returns loading → StateContainer shows loading spinner
  - Test: `useUsersQuery` returns empty array → StateContainer shows empty state with `uiCopy.noUsersYet`
  - Test: `useUsersQuery` returns error → StateContainer shows error state
  - Assert: user table hidden in all non-success states

- [ ] 3.9 Modal pending state tests (~25 LOC)
  - Test: During mutation (`isPending=true`): modal stays open, submit button disabled with spinner
  - Test: On mutation success: modal closes
  - Test: On mutation error: modal stays open, error toast shown
  - Assert: no premature modal close

---

## Notes

### Dependency Order
- **1.x → 2.x**: Phase 2 calls `PATCH /api/auth/users/{id}` (requires Phase 1)
- **2.x → 3.x**: Phase 3 tests frontend components and mutations (requires Phase 2 code)

### File Summary
- **Create**: `UserTable.tsx`, `UserModal.tsx`, `UsersView.test.tsx`
- **Modify**: `auth.go`, `server.go`, `authService.ts`, `useUsersActions.ts`, `UsersView.tsx`, `uiCopy.ts`
- **Delete**: None

### Testing Strategy
- Unit: Component renders (UserTable, UserModal, UsersView access control)
- Integration: Full flows (create, edit, delete, password change) with mocked service
- Guard validation: Last-admin protection at both UI and API layers

### Chaining Rationale
Total estimated LOC: **~490** (60 + 130 + 300). Backend is ~12% of budget, frontend ~33%, tests ~61%. However:
- Phase 1 (backend) is self-contained and low-risk (~60 LOC, single handler)
- Phase 2 (frontend) is higher volume (~130 LOC) and depends on Phase 1
- Phase 3 (tests) is the largest slice (~300 LOC) but independent of reviewability

**Recommendation: Stacked PRs to main**
- PR 1 (Phase 1): `main` ← backend handler + route (~60 LOC, high confidence)
- PR 2 (Phase 2): `main` ← frontend components + integration (~130 LOC, depends on PR 1)
- PR 3 (Phase 3): `main` ← test suite (~300 LOC, validates both phases)

Each PR is independently reviewable, testable, and has clear verification. The stacked approach allows parallel review while maintaining clear dependency boundaries.
