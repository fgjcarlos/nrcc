# Delta for User Management UI

## ADDED Requirements

### Requirement: Modal Supports Three Modes

The user modal MUST support three distinct modes determined at open time:

| Mode | Fields Shown | Use Case |
|------|-------------|---------|
| `create` | username (editable), password (required), role | New user |
| `edit_full` | username (disabled), role (editable), password (optional) | Edit role or reset password |
| `edit_password` | password only | Password-only reset |

#### Scenario: Create mode shows all fields

- GIVEN the admin clicks "Add User"
- WHEN the modal opens
- THEN username, password, and role fields are all visible and editable

#### Scenario: Edit full mode disables username

- GIVEN the admin clicks "Edit" on an existing user
- WHEN the modal opens in `edit_full` mode
- THEN the username field is visible but disabled; role and optional password are editable

#### Scenario: Edit password mode shows only password

- GIVEN the admin clicks "Change Password" on an existing user
- WHEN the modal opens in `edit_password` mode
- THEN only the password field is shown

---

### Requirement: Modal Pending State

While a mutation is in flight, the modal MUST stay open, the submit button MUST be disabled, and the button MUST display a loading spinner.

#### Scenario: Submit locks modal while pending

- GIVEN the modal is open and the user clicks Submit
- WHEN the mutation is `isPending`
- THEN the modal remains open and the submit button is disabled with a spinner

#### Scenario: Modal closes only on success

- GIVEN a mutation is in flight
- WHEN the mutation resolves successfully
- THEN the modal closes and the user list refreshes

#### Scenario: Modal stays open on error

- GIVEN a mutation is in flight
- WHEN the mutation resolves with an error
- THEN the modal stays open and an error message is shown

---

### Requirement: User List Loading State

While the user list is being fetched, the view MUST render a loading indicator via `StateContainer`.

#### Scenario: Loading state shown during fetch

- GIVEN the user list query is in flight
- WHEN the component renders
- THEN a loading state (spinner or skeleton) is visible via `StateContainer`

---

### Requirement: User List Empty State

When the user list returns successfully but contains zero items, the view MUST render an empty state via `StateContainer`.

#### Scenario: Empty state shown for zero users

- GIVEN the user list fetch succeeds with an empty array
- WHEN the component renders
- THEN the empty state UI is shown with appropriate copy from `uiCopy.ts`

---

### Requirement: User List Error State

When the user list fetch fails, the view MUST render an error state via `StateContainer`.

#### Scenario: Error state shown on fetch failure

- GIVEN the user list fetch returns an error
- WHEN the component renders
- THEN the error state UI is shown with appropriate copy from `uiCopy.ts`

---

### Requirement: Responsive Table Layout

The user table MUST render as a standard table on `md+` breakpoints and as a card/stacked layout on mobile.

#### Scenario: Table layout on desktop

- GIVEN viewport width >= 768px (md breakpoint)
- WHEN the user list renders
- THEN a full table with column headers is shown

#### Scenario: Card layout on mobile

- GIVEN viewport width < 768px
- WHEN the user list renders
- THEN each user is rendered as a stacked card without a table header row

---

### Requirement: All UI Copy from uiCopy.ts

All user-visible strings in `UsersView` MUST be sourced from `uiCopy.ts`. Hard-coded strings in JSX MUST NOT be introduced.

#### Scenario: Copy sourced from uiCopy.ts

- GIVEN any visible label, button text, or status message in UsersView
- WHEN the component renders
- THEN the string is imported from `uiCopy.ts`

---

### Requirement: UsersView Access Control

`UsersView` MUST only render its content for users with role `admin`. Non-admin users MUST see an access-denied state.

#### Scenario: Admin sees user management UI

- GIVEN the current user has role `admin`
- WHEN UsersView renders
- THEN the user table and action buttons are visible

#### Scenario: Non-admin is denied

- GIVEN the current user has role `viewer`
- WHEN UsersView renders
- THEN the user table is not shown and an access-denied message is displayed

---

### Requirement: Test Coverage for UsersView

The test suite MUST cover the following scenarios using Vitest + Testing Library:

| Scenario | Type |
|----------|------|
| Renders correctly for admin | Unit |
| Denies access for non-admin | Unit |
| Create user — success | Integration |
| Create user — validation error | Integration |
| Create user — API error | Integration |
| Edit role — success | Integration |
| Edit role — last-admin protection | Integration |
| Password reset — success | Integration |
| Password reset — API error | Integration |
| Delete — confirmation dialog shown | Integration |
| Delete — success | Integration |
| Delete — last-admin protection | Integration |
| Empty state renders | Unit |
| Error state renders | Unit |
| Loading state renders | Unit |

#### Scenario: Test suite covers all modal modes

- GIVEN `UsersView.test.tsx` exists
- WHEN all tests run
- THEN each modal mode (`create`, `edit_full`, `edit_password`) is exercised in at least one test

#### Scenario: Last-admin guard tested in both layers

- GIVEN test suite runs
- WHEN the last-admin protection scenario executes
- THEN both UI disable behavior and backend 403 response handling are asserted
