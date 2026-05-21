# Exploration: Issue #112 — Complete Admin User Management Page

## Summary
Issue #112 requires completing the admin user management page with role editing, improved UX (ConfirmationDialog), safer deletion, responsive tables, inline loading states, and comprehensive tests.

---

## Current State

### Frontend (React 19 + Vite 8 + TS 6)

#### UsersView.tsx (231 lines)
- **Current capabilities:**
  - Displays list of users in a table (username, role, created date)
  - "Add User" modal for creating new users (username, password, role)
  - "Change Password" button on each row (opens same modal, asks only for password)
  - Delete button with `useConfirmationDialog` hook (correctly uses ConfirmationDialog)
  
- **Problems identified:**
  1. **Modal closes immediately after mutation** (lines 49, 53): `closeModal()` called BEFORE mutation completes
     - `closeModal()` is called synchronously after `mutate()`
     - Should wait for mutation success before closing
  2. **No role editing after user creation:** Modal only allows editing password, not role
  3. **No inline loading/disabled states:** Buttons don't show loading while mutations are in progress
  4. **No empty/error states:** If user list is empty, no message shown
  5. **Table not responsive:** Fixed-width columns, no mobile handling
  6. **No tests:** Zero test coverage for UsersView

#### authService.ts (83 lines)
- `getUsers()` ✅
- `createUser(username, password, role)` ✅ 
- `deleteUser(id)` ✅
- `changePassword(id, password)` ✅
- **Missing:** `updateRole(id, role)` or generic `updateUser(id, updates)` method

#### useUsersActions.ts (50 lines)
- `createMutation` ✅
- `deleteMutation` ✅
- `changePasswordMutation` ✅
- **Missing:** `updateRoleMutation` (no service method to call)

#### ConfirmationDialog.tsx (162 lines)
- ✅ Properly implemented with variant support (danger/warning/default)
- ✅ isPending prop for showing loading state
- ✅ Escape key handling
- ✅ Optional confirmation text confirmation
- ✅ Used correctly in UsersView for delete confirmation

#### useConfirmationDialog.ts (28 lines)
- ✅ Encapsulates dialog state and handlers correctly
- ✅ Generic `<T>` for holding pending item

#### uiCopy.ts (70 lines)
- ✅ Already has `deleteUser`, `deleteUserDescription`, `deleteUserConfirm` copy
- **Missing:** Additional strings for role editing UI if needed

---

### Backend (Go 1.22 + Chi v5.2.5)

#### handler/auth.go (420 lines)
- `Setup()` — POST /api/auth/setup ✅
- `Login()` — POST /api/auth/login ✅
- `GetStatus()` — GET /api/auth/status ✅
- `GetMe()` — GET /api/auth/me ✅
- `Logout()` — POST /api/auth/logout ✅
- `GetUsers()` — GET /api/auth/users ✅
- `CreateUser()` — POST /api/auth/users ✅
- `DeleteUser()` — DELETE /api/auth/users/:id ✅ (with guards: no self-delete, no last-admin-delete)
- `ChangePassword()` — PATCH /api/auth/users/:id/password ✅
- **Missing:** PATCH /api/auth/users/:id for role editing

#### service/auth.go (191 lines)
- `UpdateUser(user *model.CCUser)` ✅ — General update method exists
- `DeleteUser(id)` ✅
- `HashPassword(password)` ✅

#### server/server.go (lines 74-89)
```go
r.Route("/api/auth", func(r chi.Router) {
    // ...public routes...
    r.Group(func(r chi.Router) {
        r.Use(middleware.Auth(authSvc))
        r.Get("/me", authHandler.GetMe)
        r.Post("/logout", authHandler.Logout)
        r.Get("/users", authHandler.GetUsers)
        r.Post("/users", authHandler.CreateUser)
        r.Delete("/users/{id}", authHandler.DeleteUser)           // ✅
        r.Patch("/users/{id}/password", authHandler.ChangePassword) // ✅
        // MISSING: r.Patch("/users/{id}", UpdateRole) or UpdateUser
    })
})
```

---

## Affected Areas

### Frontend
- `frontend/src/features/auth/components/UsersView.tsx` — Main component needs major refactor
- `frontend/src/features/auth/services/authService.ts` — Add `updateRole()` method
- `frontend/src/features/auth/hooks/useUsersActions.ts` — Add `updateRoleMutation`
- `frontend/src/features/auth/components/UsersView.test.tsx` — NEW test file (doesn't exist)
- `frontend/src/shared/constants/uiCopy.ts` — May need additional strings

### Backend
- `internal/handler/auth.go` — Add `UpdateRole()` handler (NEW endpoint)
- `internal/server/server.go` — Register new route `r.Patch("/users/{id}", authHandler.UpdateRole)`

---

## Approaches

### Approach 1: Inline Role Edit (No Modal Redesign)
**Concept:** Role column becomes a clickable dropdown, save inline without a modal.

**Pros:**
- Minimal modal refactor
- Faster UX (no modal context switch)
- Less code

**Cons:**
- No confirmation dialog for role changes (role is less risky than delete, but still admin action)
- Tight table layout becomes even tighter
- Harder to handle loading state in inline dropdown
- Not consistent with "Change Password" UX

**Effort:** Low

---

### Approach 2: Extend Existing Modal (RECOMMENDED)
**Concept:** Modal works for BOTH create AND edit. Edit mode shows username (disabled), password (optional), and role (editable).

**Pros:**
- Consistent UX (one modal for all user operations)
- Reuses modal pattern already in place
- Easier loading state management
- Matches design system (uses existing modal-overlay, surface-panel)
- Single form for create + edit + password-change

**Cons:**
- Modal needs more complex state (create vs edit vs password-only modes)
- Form field visibility logic becomes more complex
- Need to track which fields were actually changed (to avoid unnecessary API calls)

**Effort:** Medium

---

### Approach 3: Separate Modal for Role + Confirmation Dialog
**Concept:** Click "Edit Role" → opens role-only modal → same ConfirmationDialog pattern as delete.

**Pros:**
- Very safe (confirmation before role change)
- Clear separation of concerns
- Mirrors delete confirmation flow

**Cons:**
- Too many modals (create, password change, role edit, delete)
- Feels heavy for a simple role change
- User has to click through multiple layers

**Effort:** Medium-High

---

## Recommendation

**Use Approach 2** — extend the existing modal to support edit mode.

### Rationale:
1. **Modal is already the pattern** — UsersView already uses it for create + password change
2. **Consistent UX** — Admin does all user ops from one modal
3. **Loading states handled correctly** — Button can show `isPending` state
4. **Minimal disruption** — Just add fields and state conditions
5. **Responsive friendly** — Modal scales better on narrow screens than inline dropdowns
6. **Follows design system** — Uses established modal-overlay and surface-panel patterns

### Key implementation decisions:
- Modal modes: `create | edit_password_only | edit_full` (create shows username + role + password; edit_password shows password only; edit_full shows username disabled + role editable + password optional)
- Backend endpoint: `PATCH /api/auth/users/:id` with `{ role?: 'admin' | 'viewer', password?: string }` (both optional)
- Frontend mutation: Generic `updateUser()` that can handle both role and password changes
- Loading states: All action buttons (Add, Delete, Change Password, Save) disabled while mutation in progress
- Empty state: "No users yet" message when list is empty
- Responsive table: Use Tailwind breakpoints to stack table columns on mobile, or switch to card layout on small screens
- Tests: Full coverage of UsersView with Vitest + Testing Library, mocking mutations and testing all modal modes

---

## Risks

1. **Modal complexity increases** — More state conditions means more edge cases. Need thorough testing.
2. **Race conditions if user clicks Save multiple times** — Solution: Disable submit button while `isPending`
3. **Cannot delete last admin and cannot edit self to non-admin** — Backend guards needed (DELETE already has this; PATCH needs it)
4. **Password optional on edit but required on create** — Form validation logic needs to check mode
5. **Users expect inline role editing** — May need UX testing. Current approach is modal-focused.
6. **Table responsiveness trade-off** — Switching to card layout on mobile uses more space. Consider drawer pattern instead.

---

## Backend Gaps

1. **No PATCH /api/auth/users/:id endpoint** — Must be added
2. **Handler `UpdateRole()` missing** — Must validate:
   - Admin-only access
   - Can't edit self to non-admin (if only admin left)
   - Can't make last admin non-admin
   - User exists
   - Role is valid ('admin' or 'viewer')
3. **No frontend method `authService.updateRole()` or generic `updateUser()`** — Add to service

---

## Scope Estimate

### Frontend Changes
- `UsersView.tsx`: ~100 LOC added (modal mode logic, edit state, new mutation handler)
- `authService.ts`: ~10 LOC added (new `updateRole()` method)
- `useUsersActions.ts`: ~20 LOC added (new mutation)
- `UsersView.test.tsx`: ~300 LOC (new test file covering all modes and mutations)
- **Subtotal: ~430 LOC**

### Backend Changes
- `auth.go`: ~60 LOC added (new `UpdateRole()` handler with validation)
- `server.go`: ~2 LOC added (register new route)
- **Subtotal: ~62 LOC**

### Total: ~492 LOC

**Above 400-line threshold** — Consider splitting into phases:
- **Phase 1:** Backend PATCH endpoint + Frontend service method + basic modal refactor (create + edit)
- **Phase 2:** Tests + responsive table + empty states

Or keep as single change if can be implemented as one cohesive unit.

---

## Decomposition (if needed)

### Phase 1: Backend + Basic Frontend (≈250 LOC)
- Add `PATCH /api/auth/users/:id` handler in backend
- Add `updateRole()` to frontend service
- Refactor UsersView modal to support edit mode
- Make modal buttons show loading state

### Phase 2: Tests + Refinements (≈250 LOC)
- Full UsersView test coverage
- Responsive table improvements
- Empty/error states
- uiCopy refinements

---

## Ready for Proposal?

**YES** — Exploration complete. Recommend proceeding with:
- **Approach:** Modal-based edit with full role + password support
- **Backend gap:** Add PATCH /api/auth/users/:id endpoint
- **Scope:** Single change at ~492 LOC (or split into 2 phases if preferred)
- **Next steps:** Create proposal document with exact requirements and API contract
