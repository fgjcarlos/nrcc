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

#### Scenario: Delete button disabled for last admin in UI

- GIVEN exactly one user with role `admin` exists
- WHEN the admin views the user list
- THEN the delete action for that user is disabled (button disabled or hidden)

#### Scenario: Delete non-last admin succeeds

- GIVEN two or more users with role `admin` exist
- WHEN the admin confirms deletion of one admin user
- THEN the user is deleted and removed from the list

---

### Requirement: UI Pre-flight Check Before Demote Submit

The frontend MUST disable the role selector or submit button when the target user is the last admin and the new role would be `viewer`.

#### Scenario: Submit blocked in UI for last-admin demote

- GIVEN the modal is open in `edit_full` mode for the last admin
- WHEN the user selects role `viewer`
- THEN the submit button is disabled and an explanatory message is shown
