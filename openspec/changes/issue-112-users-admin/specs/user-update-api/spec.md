# User Update API Specification

## Purpose

Defines the backend PATCH endpoint for updating a user's role and/or password. This is a new capability with no existing spec.

## Requirements

### Requirement: Accept Partial User Update

The endpoint MUST accept `PATCH /api/auth/users/{id}` with a JSON body containing at least one of `role` or `password`. Requests missing both fields MUST be rejected.

| Field | Type | Required |
|-------|------|----------|
| role | `"admin" \| "viewer"` | optional |
| password | string | optional |

At least one field MUST be present.

#### Scenario: Update role only

- GIVEN an authenticated admin
- WHEN they send `{ "role": "viewer" }` for a non-last-admin user
- THEN the server returns 200 with the updated user object `{ id, username, role, createdAt }`

#### Scenario: Update password only

- GIVEN an authenticated admin
- WHEN they send `{ "password": "newpass" }` for any user
- THEN the server returns 200 with the updated user object

#### Scenario: Empty body rejected

- GIVEN an authenticated admin
- WHEN they send `{}` or omit the body
- THEN the server returns 400 Bad Request

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
