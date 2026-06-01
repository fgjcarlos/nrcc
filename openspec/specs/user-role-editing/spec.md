# User Role Editing Specification

## Purpose

Defines the last-admin protection rules for role changes and deletions. Applies at backend and frontend layers.

## Requirements

### Requirement: Prevent Demoting Last Admin

The system MUST NOT allow changing the role of the last remaining admin user to `viewer`. This guard MUST be enforced on the backend.

#### Scenario: Demote last admin blocked at backend

- GIVEN exactly one user with role `admin` exists
- WHEN an admin sends `{ "role": "viewer" }` for that user
- THEN the server returns 403 Forbidden with message indicating last-admin protection

#### Scenario: Demote non-last admin succeeds

- GIVEN two or more users with role `admin` exist
- WHEN an admin sends `{ "role": "viewer" }` for one of them
- THEN the server returns 200 with the updated user

---

### Requirement: Prevent Deleting Last Admin via UI

The frontend MUST disable the delete action for the last remaining admin user. The backend MUST also enforce this guard on the DELETE endpoint.

The backend delete guard returns **403 Forbidden** with code `CANNOT_DELETE_LAST_ADMIN`, matching the demote-last-admin guard because both enforce the same authorization/business-policy invariant.

#### Scenario: Delete button disabled for last admin in UI

- GIVEN exactly one user with role `admin` exists
- WHEN the admin views the user list
- THEN the delete action for that user is disabled (button disabled or hidden)

#### Scenario: Delete non-last admin succeeds

- GIVEN two or more users with role `admin` exist
- WHEN the admin confirms deletion of one admin user
- THEN the user is deleted and removed from the list

#### Scenario: Delete last admin rejected at backend

- GIVEN exactly one user with role `admin` exists
- WHEN a DELETE request is sent for that user bypassing the UI
- THEN the server returns 403 Forbidden with code `CANNOT_DELETE_LAST_ADMIN`

---

### Requirement: UI Pre-flight Check Before Demote Submit

The frontend MUST block submission in `edit_full` mode when the target user is the last admin (`isLastAdmin` is true). The submit gate is `(mode !== 'edit_full' || !isLastAdmin)`.

A non-last-admin user's role change in `edit_full` mode MUST be submittable. Only the last admin is blocked from submitting in `edit_full` — their role selector is disabled, and password changes for them go through the dedicated change-password modal.

(Previously: described as blocking submit whenever role was changed in edit_full; this was a bug in the old `canDemote` logic that was fixed in PR #242.)

#### Scenario: Submit blocked in UI for last-admin in edit_full

- GIVEN the modal is open in `edit_full` mode for the last admin
- WHEN the form is rendered
- THEN the submit button is disabled (regardless of role selection) and the role selector is disabled

#### Scenario: Submit enabled for non-last-admin role change

- GIVEN the modal is open in `edit_full` mode for a user who is NOT the last admin
- WHEN the admin changes the role field to a different value
- THEN the submit button is enabled and the role change can be submitted
