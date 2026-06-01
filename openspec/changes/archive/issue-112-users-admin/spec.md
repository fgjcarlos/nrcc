# Spec: Issue #112 — Complete Admin User Management

## Change

`issue-112-users-admin`

## Domains

| Domain | Type | Spec Path |
|--------|------|-----------|
| `user-update-api` | New | `specs/user-update-api/spec.md` |
| `user-role-editing` | New | `specs/user-role-editing/spec.md` |
| `user-management-ui` | Delta | `specs/user-management-ui/spec.md` |

## Summary

### Phase 1 — Backend

- **user-update-api**: `PATCH /api/auth/users/{id}` accepts `{ role?, password? }` (at least one required). Admin-only (403 otherwise). Returns updated user `{ id, username, role, createdAt }`. Errors: 400 (invalid/empty body, invalid role), 403 (not admin), 404 (user not found).
- **user-role-editing**: Last-admin guard — backend MUST return 403 when demoting or deleting the sole admin. Guard is independently enforced; not coupled to request type.

### Phase 2 — Frontend

- **user-management-ui (delta)**: Modal gains three modes (`create`, `edit_full`, `edit_password`). Submit locks modal while `isPending`. List states (loading, empty, error) via `StateContainer`. Table is responsive (card on mobile, table on md+). All copy from `uiCopy.ts`. Access-control guard for non-admin users.

### Phase 3 — Tests

Covered under `user-management-ui` spec: 15 test scenarios spanning all modal modes, mutations (success + error), last-admin protection (UI + backend response), and state variants.

## Requirements Count

| Domain | Added | Modified | Removed | Scenarios |
|--------|-------|----------|---------|-----------|
| user-update-api | 4 | 0 | 0 | 8 |
| user-role-editing | 3 | 0 | 0 | 6 |
| user-management-ui | 9 | 0 | 0 | 20 |
| **Total** | **16** | **0** | **0** | **34** |
