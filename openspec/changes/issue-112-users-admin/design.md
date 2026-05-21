# Design: Issue #112 — Complete Admin User Management

## Technical Approach

Extend the existing auth handler and modal pattern with minimal surface area:
- **Backend**: Single new `UpdateUser` handler for PATCH `/api/auth/users/{id}` — role-only updates (password already handled by `ChangePassword`). Last-admin guard mirrors the existing `DeleteUser` guard pattern.
- **Frontend**: Refactor `UsersView` modal from binary (create vs. password-change) to three modes (`create | edit_full | edit_password`) via a discriminated `ModalMode` type. Extract `UserTable` and `UserModal` as presentational components. Add `updateUserRole` mutation to `useUsersActions`.

## Architecture Decisions

| Decision | Choice | Rejected | Rationale |
|---|---|---|---|
| PATCH scope | Role-only (`/users/{id}`) | Unified role+password PATCH | Password already has a dedicated endpoint (`/users/{id}/password`). Merging adds complexity for no gain; two endpoints are cleaner and separately testable. |
| Modal state model | `ModalMode` discriminated union + `editingUser` ref | Full reducer / useReducer | State is shallow (3 modes × isPending × error); `useState` is sufficient. useReducer would be over-engineering. |
| Component split | `UsersView` (container) + `UserTable` + `UserModal` | Keep monolithic `UsersView` | Current file is 231 LOC before the change; adding edit mode would push it past 400. Split aligns with container/presentational pattern already used in the project. |
| Last-admin guard — backend | Count admins in handler, return 403 | Service-layer guard | Consistent with existing `DeleteUser` pattern; keeps guard logic in handler where other auth guards live. |
| Last-admin guard — frontend | UI disables role selector for sole admin; error toast on 403 | Block submit in form | Backend is authoritative; frontend guard is UX-only. 403 from backend triggers `onError` → `toast.error` as already established in `useUsersActions`. |

## Data Flow

### PATCH Role Update

```
UserModal (edit_full) ──→ useUsersActions.updateRoleMutation
        │                         │
        │                   authService.updateUserRole(id, role)
        │                         │
        │                   PATCH /api/auth/users/{id}   { role }
        │                         │
        │                   UpdateUser handler
        │                         │
        │                   last-admin guard → 403 if sole admin
        │                         │
        │                   authSvc.UpdateUser(user)
        │                         │
        └── onSuccess: invalidate ['users'] → refetch → table re-renders
```

### Modal Mode State Machine

```
null ──openCreate──→ { mode: 'create', editingUser: null }
                              │
                         closeModal
                              │
User row ──openEdit──→ { mode: 'edit_full', editingUser: User }
                              │
                    "Change Password" action
                              │
                     { mode: 'edit_password', editingUser: User }
```

## File Changes

| File | Action | Description |
|---|---|---|
| `internal/handler/auth.go` | Modify | Add `UpdateUserRequest` struct + `UpdateUser` handler (~55 LOC) |
| `internal/server/server.go` | Modify | Register `r.Patch("/users/{id}", authHandler.UpdateUser)` in protected auth group |
| `frontend/src/features/auth/services/authService.ts` | Modify | Add `updateUserRole(id, role)` method — PATCH `/auth/users/{id}` |
| `frontend/src/features/auth/hooks/useUsersActions.ts` | Modify | Add `updateRoleMutation` using `authService.updateUserRole` |
| `frontend/src/features/auth/components/UsersView.tsx` | Modify | Replace modal binary logic with `ModalMode`; delegate render to `UserModal`; use `UserTable` |
| `frontend/src/features/auth/components/UserTable.tsx` | Create | Presentational: renders rows, role badges, action buttons; receives `onEdit`, `onDelete`, `onChangePassword` callbacks |
| `frontend/src/features/auth/components/UserModal.tsx` | Create | Presentational: 3-mode form (create / edit_full / edit_password); receives `mode`, `editingUser`, `isPending`, `onSubmit`, `onClose` |
| `frontend/src/shared/constants/uiCopy.ts` | Modify | Add keys listed below |
| `frontend/src/features/auth/components/UsersView.test.tsx` | Create | Full test suite (~300 LOC) |

## Interfaces / Contracts

### Backend — new struct + handler signature

```go
// UpdateUserRequest — PATCH /api/auth/users/{id}
type UpdateUserRequest struct {
    Role *model.UserRole `json:"role,omitempty"` // pointer: nil means "not provided"
}

func (h *AuthHandler) UpdateUser(w http.ResponseWriter, r *http.Request)
// Guards: admin-only, user exists, role valid, not demoting sole admin
// Returns: 200 CCUserPublic | 400 | 403 | 404
```

### Frontend — ModalMode type

```ts
type ModalMode = 'create' | 'edit_full' | 'edit_password';

interface ModalState {
  mode: ModalMode;
  editingUser: User | null;
}
// Closed state: null (not { mode, editingUser })
```

### authService addition

```ts
updateUserRole: async (id: string, role: 'admin' | 'viewer'): Promise<User> => {
  const response = await api.patch<{ data: User }>(`/auth/users/${id}`, { role });
  return response.data.data;
},
```

### uiCopy.ts keys to add

```ts
// Users section
noUsersYet: 'No users yet',
addFirstUser: 'Add the first user to get started',
editUser: 'Edit User',
changeRole: 'Change Role',
changePassword: 'Change Password',
createUser: 'Create User',
roleLabel: 'Role',
usernameLabel: 'Username',
passwordLabel: 'Password',
newPasswordLabel: 'New Password',
cannotDemoteLastAdmin: 'Cannot demote the last admin user',
```

## Testing Strategy

| Layer | What | Approach |
|---|---|---|
| Unit — handler | `UpdateUser`: valid role, invalid role, non-existent user, sole-admin demotion | Go table-driven tests in `internal/handler/auth_test.go` |
| Unit — frontend | `UserModal` renders correct fields per mode; submit calls correct callback | Vitest + RTL, `vi.mock('@/features/auth/services/authService')` |
| Integration — frontend | Full flow: open edit modal → change role → mutation fires → table updates; empty state; last-admin guard shows toast | `UsersView.test.tsx` with mocked `authService` and `useAuth` |
| Test file | `frontend/src/features/auth/components/UsersView.test.tsx` | Groups: `describe('create flow')`, `describe('edit role flow')`, `describe('change password flow')`, `describe('delete flow')`, `describe('last-admin guard')`, `describe('empty state')` |

Mock strategy: `vi.mock('@/features/auth/services/authService')` returns typed mocks; `vi.mock('@/features/auth/hooks/useAuth')` returns `{ user: { role: 'admin', id: 'u1' } }`.

## Migration / Rollout

No migration required. PATCH endpoint is additive; existing routes unaffected.

## Open Questions

- [ ] Should `edit_full` mode allow username rename? (Proposal implies disabled username field — confirm before implementing)
- [ ] Does the role badge in `UserTable` need an inline selector (click-to-change) or is modal-only sufficient? (Proposal says "modal or inline selector" — clarify scope boundary)
