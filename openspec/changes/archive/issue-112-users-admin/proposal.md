# Proposal: Issue #112 — Complete Admin User Management

## Intent

Complete the admin user management page to enable role editing (create, read, update, delete with validation). Current state allows creating and deleting users, but lacks role modification after creation. Backend PATCH endpoint is missing; frontend modal needs refactor to support edit mode with proper loading states. Last-admin protection must be enforced at both backend and UI layers.

## Scope

### In Scope
- Backend PATCH `/api/auth/users/{id}` endpoint with role + password update capability
- Last-admin protection: prevent deletion and demotion of final admin (backend guard + UI)
- Modal refactor to unified create/edit/password-change interface
- Loading states on all action buttons during mutations
- Empty and error states in user table
- Responsive layout (stack table on mobile, preserve on desktop)
- Full test coverage for UsersView (all modal modes, mutations, guards)

### Out of Scope
- User search/filtering (defer to issue-113)
- Batch user operations (defer)
- Password strength validation UI (backend already validates)
- Audit logging (future)

## Capabilities

### New Capabilities
- `user-role-editing`: Modify user role (admin/viewer) via PATCH endpoint; backend guards prevent demoting/deleting last admin
- `user-update-api`: Generic backend PATCH handler for role and password updates

### Modified Capabilities
- `user-management-ui`: Modal extended to support create + edit (full) + edit (password-only) modes; loading states added to action buttons; empty state UI added

## Approach

**Extend existing modal pattern** (already used for create + password change) to support role editing:
1. Modal gains **three modes**: `create` (show username + password + role), `edit_full` (show username disabled + role + optional password), `edit_password` (show password only)
2. **Backend PATCH `/api/auth/users/{id}`**: accepts optional `role` and `password` fields; validates role, checks last-admin guard, returns updated user
3. **Frontend service**: add `updateUser(id, { role?, password? })` method; mutations handle all three modes
4. **UI states**: action buttons disabled during mutation; empty state "No users yet" when list is empty; error messages from failed mutations shown in toast/inline
5. **Responsive**: use Tailwind grid + breakpoints to collapse columns on mobile or switch to card layout
6. **Tests**: 300+ LOC covering modal modes, loading states, mutations, last-admin guard, and empty state

## API Contract

### PATCH `/api/auth/users/{id}`

**Request body:**
```json
{
  "role": "admin" | "viewer",  // optional
  "password": "string"          // optional; at least one field required
}
```

**Guards (backend):**
- Admin-only access (check auth middleware)
- Cannot modify own admin status if only admin exists
- Cannot delete role (PATCH is not DELETE)
- Role must be valid enum (admin | viewer)
- User must exist

**Response (200 OK):**
```json
{
  "id": "uuid",
  "username": "string",
  "role": "admin" | "viewer",
  "createdAt": "2025-05-21T..."
}
```

**Error responses:**
- `400 Bad Request`: invalid role or missing body
- `403 Forbidden`: unauthorized (not admin) or trying to demote last admin
- `404 Not Found`: user does not exist
- `409 Conflict`: user deleted or other race condition

## Implementation Phases

Split into **3 PRs** to keep review load ≤ 400 lines per PR:

### Phase 1: Backend PATCH Endpoint (~60 LOC)
- Add `UpdateUser()` handler in `internal/handler/auth.go`
- Register route in `internal/server/server.go` (2 LOC)
- Implement last-admin guard logic
- Add `updateUser()` service method if not present

**PR target:** feature/issue-112-users-admin
**Tests included:** unit tests for last-admin guard in auth handler

### Phase 2: Frontend Modal Refactor + Service (~130 LOC)
- Add `updateUser()` to `authService.ts`
- Refactor `UsersView.tsx` modal state (create vs edit modes)
- Add mutation handler for role updates
- Add loading/disabled states to buttons
- Add empty state UI
- Make table responsive with Tailwind

**PR target:** feature/issue-112-users-admin (child of Phase 1)
**Tests included:** none (defer to Phase 3)

### Phase 3: Tests + Polish (~300+ LOC)
- Create `UsersView.test.tsx` (new file)
- Test coverage: modal modes, mutations, loading states, empty state, error handling
- Polish UI copy via `uiCopy.ts` if needed
- End-to-end scenarios

**PR target:** feature/issue-112-users-admin (child of Phase 2)
**Tests included:** Vitest + Testing Library full coverage

## Affected Areas

| Path | Impact | Description |
|------|--------|-------------|
| `internal/handler/auth.go` | New | Add `UpdateUser()` handler with last-admin guard |
| `internal/server/server.go` | Modified | Register PATCH `/api/auth/users/{id}` route |
| `frontend/src/features/auth/services/authService.ts` | Modified | Add `updateUser(id, updates)` method |
| `frontend/src/features/auth/hooks/useUsersActions.ts` | Modified | Add `updateUserMutation` hook |
| `frontend/src/features/auth/components/UsersView.tsx` | Modified | Modal refactor (modes), loading states, empty state, responsiveness |
| `frontend/src/features/auth/components/UsersView.test.tsx` | New | Full test coverage (~300+ LOC) |
| `frontend/src/shared/constants/uiCopy.ts` | Minor | Add UI strings for role edit if needed |

## Risks

| Risk | Likelihood | Mitigation |
|---|---|---|
| Modal complexity increases; edge cases in mode switching | Medium | Thorough unit + integration tests; keep mode logic in `useState` reducer if complex |
| Race condition: user clicks Save multiple times | Low | Disable submit button while `isPending` (already pattern from issue-116) |
| Last-admin protection inconsistency between backend + UI | Medium | Test both layers; UI guards show error toast; backend enforces with 403 |
| Table responsiveness breaks on narrow viewports | Low | Test on mobile in Vitest snapshot + manual QA; use Tailwind breakpoints carefully |
| Password optional on edit but required on create | Low | Form validation checks mode; clear test coverage |

## Rollback Plan

1. **Phase 1 revert**: Delete `UpdateUser()` handler; remove PATCH route from server.go → API returns 404
2. **Phase 2 revert**: Remove modal mode logic; revert to create-only modal; users can only create/delete/password-change
3. **Phase 3 revert**: Delete test file; non-blocking

**Full rollback**: Revert all three commits; feature branch deleted. Users revert to current create/delete/password-change-only workflow until feature is re-attempted.

## Dependencies

- **Issue #116 (merged)**: Establishes `uiCopy.ts`, `StateContainer`, `useConfirmationDialog` patterns used throughout this change
- **ConfirmationDialog** already available and tested; reuse for delete confirmation
- **Vitest + Testing Library** already configured; no new test tooling needed

## Success Criteria

- [ ] Admin can create user with role (existing, still works)
- [ ] Admin can edit user role (new) via modal or inline selector
- [ ] Admin cannot delete or demote last admin (guarded at backend + UI error shown)
- [ ] Modal shows loading state while mutation in progress; buttons disabled
- [ ] Empty state "No users yet" shown when user list is empty
- [ ] Table responsive on mobile (columns collapse or card layout)
- [ ] All flows covered in tests: create, edit (role), edit (password), delete, last-admin guard, empty state
- [ ] No console errors; mutations retry sensibly on transient errors

## Notes

- **PR split strategy**: Each PR is atomic; Phase 2 depends on Phase 1 (backend), Phase 3 depends on Phase 2 (UI)
- **Delivery**: Chained/stacked PRs to feature/issue-112-users-admin, then single PR to main after all phases pass review
- **Review focus**: Phase 1 (logic + guards), Phase 2 (modal state), Phase 3 (test coverage + edge cases)
