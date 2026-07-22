# User Update API Specification

## Purpose

Defines the backend PATCH endpoints for updating a user's role and/or password. Role and password changes use two separate endpoints. This is a new capability with no existing spec.

## Requirements

### Requirement: Update User Role via PATCH /api/auth/users/{id}

The endpoint MUST accept `PATCH /api/auth/users/{id}` with a JSON body containing `role`. The request body MUST NOT include a `password` field on this endpoint — password changes MUST use the separate `PATCH /api/auth/users/{id}/password` endpoint. Requests missing `role` MUST be rejected with 400 Bad Request.

| Field | Type | Required |
|-------|------|----------|
| role | `"admin" \| "viewer"` | required |

Note: The success response returns `{ id, username, role, createdAt, updatedAt }` — `updatedAt` is present in the code as a superset of the originally-specced shape.

#### Scenario: Update role only

- GIVEN an authenticated admin
- WHEN they send `{ "role": "viewer" }` for a non-last-admin user to `PATCH /api/auth/users/{id}`
- THEN the server returns 200 with the updated user object `{ id, username, role, createdAt, updatedAt }`

#### Scenario: Empty body rejected

- GIVEN an authenticated admin
- WHEN they send `{}` or omit the body to `PATCH /api/auth/users/{id}`
- THEN the server returns 400 Bad Request

---

### Requirement: Update User Password via PATCH /api/auth/users/{id}/password

Password changes MUST use a dedicated endpoint `PATCH /api/auth/users/{id}/password` with a body containing `password`. This endpoint is distinct from the role-update endpoint.

| Field | Type | Required |
|-------|------|----------|
| password | string | required |

#### Scenario: Update password only

- GIVEN an authenticated admin
- WHEN they send `{ "password": "newpass" }` to `PATCH /api/auth/users/{id}/password`
- THEN the server returns 200 with the updated user object

---

### Requirement: Validate Role Enum

The `role` field, when present, MUST be one of `"admin"` or `"viewer"`. Any other value MUST be rejected.

#### Scenario: Invalid role value

- GIVEN an authenticated admin
- WHEN they send `{ "role": "superadmin" }`
- THEN the server returns 400 Bad Request

---

### Requirement: Require Admin Authorization

The endpoint MUST be admin-only. Non-admin callers MUST receive 403 Forbidden.

#### Scenario: Non-admin caller

- GIVEN a user with role `viewer`
- WHEN they call `PATCH /api/auth/users/{id}`
- THEN the server returns 403 Forbidden

---

### Requirement: Return 404 for Unknown User

If the target user does not exist, the server MUST return 404 Not Found.

#### Scenario: User not found

- GIVEN an authenticated admin
- WHEN they send a valid body for a non-existent user ID
- THEN the server returns 404 Not Found
