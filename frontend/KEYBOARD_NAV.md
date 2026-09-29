# Keyboard Navigation Guide

This document describes the keyboard interaction patterns implemented across the Node-RED Control Center frontend. All interactive modals and menus follow consistent patterns for accessibility and user experience.

## Overview

The application supports keyboard navigation for:
- **Modals**: ConfirmationDialog
- **Menus**: UserMenu (header dropdown)
- **Focus Management**: Automatic focus trap and restoration

---

## ConfirmationDialog

The confirmation dialog is used for destructive actions (delete, remove, clear, reset). It traps focus within the modal and supports keyboard-driven completion.

### Keyboard Behavior

| Key | Action | Conditions |
|-----|--------|-----------|
| **Escape** | Close dialog (cancel action) | Only when `isPending = false` |
| **Enter** | Confirm action | Only when `isPending = false` AND `canConfirm() = true` |
| **Tab / Shift+Tab** | Cycle focus within dialog | Always; focus does NOT leave dialog to background content |

### Focus Management

#### Auto-Focus on Open

When the dialog opens:
- The text input receives focus when a `confirmText` gate is present
- Otherwise, the Cancel button receives focus so a destructive action is never the default target
- Closing the dialog restores focus to the control that opened it

#### Focus Trap

Once the dialog is open:
- Tab and Shift+Tab cycle through all focusable elements within the dialog
- Focus wraps: tabbing forward from the last element moves to the first; tabbing backward from the first moves to the last
- Focus CANNOT escape to background content (no tabbing to hidden elements)

#### Top Layer

The shared dialog is portaled directly to `document.body` with a layer above page drawers and
other local stacking contexts. Feature pages must use this shared dialog rather than adding a
feature-specific modal implementation.

#### Focus Restoration

When the dialog closes:
- Focus should return to the element that triggered the dialog (handled by the calling component via `onCancel`)
- This is typically the "Delete" button or action that opened the dialog

### Escape Behavior

**When NOT pending** (`isPending = false`):
- Pressing Escape calls `onCancel()`
- The dialog closes without executing the action
- No data is modified

**When pending** (`isPending = true`):
- Pressing Escape is IGNORED
- This prevents accidental dismissal while the mutation is in-flight
- The user must wait for the mutation to complete before they can close the dialog

### Enter Behavior

**When NOT pending AND `canConfirm()` is true**:
- Pressing Enter calls `onConfirm()`
- The action is executed
- This allows keyboard-only workflow: open dialog → type text (if required) → press Enter to confirm

**When NOT pending AND `canConfirm()` is false** (e.g., type-to-confirm with incorrect text):
- Pressing Enter is ignored
- The Confirm button is disabled, providing visual feedback

**When pending** (`isPending = true`):
- Pressing Enter is ignored
- Prevents double-submission or race conditions

### Type-to-Confirm Pattern

The dialog can require the user to type a confirmation phrase before proceeding:

```tsx
<ConfirmationDialog
  isOpen={isOpen}
  title="Delete User"
  description="This action cannot be undone."
  confirmText="alice"  // User must type this exact word
  onConfirm={() => deleteUser()}
  onCancel={() => setIsOpen(false)}
/>
```

Keyboard workflow:
1. Dialog opens → input auto-focuses
2. User types `"alice"` into the input field
3. Confirm button becomes enabled once input matches
4. User can press Enter OR click Confirm button
5. Action executes

**Input Label**: "Type \"{confirmText}\" to confirm" (e.g., "Type "alice" to confirm")

---

## UserMenu

The user menu dropdown appears in the header (top-right corner) and provides access to profile and sign-out actions.

### Keyboard Behavior

| Key | Action | Conditions |
|-----|--------|-----------|
| **Escape** | Close menu | Only when menu is open |
| **Tab** | Move focus to next menu item | DOES NOT cycle; moves to next element on page |
| **Shift+Tab** | Move focus to previous menu item | DOES NOT cycle; moves to previous element on page |
| **Click Outside** | Close menu | Always; click outside the menu closes it |

### Focus Management

#### Menu Trigger Button

The avatar button that opens the menu has ARIA attributes:
- `aria-haspopup="true"` — indicates a menu is attached
- `aria-expanded={true|false}` — reflects the open/closed state
- `aria-label="{username} — open user menu"` — accessible label

#### Menu Items

Once open, the menu contains two items:
1. **Profile** — navigates to `/profile`
2. **Sign out** — triggers logout (can be busy with `logoutBusy` prop)

Both items have `role="menuitem"` and are keyboard-accessible via Tab.

#### Focus Restoration

When the menu closes:
- Focus returns to the avatar trigger button
- This is achieved via Escape key handler that closes the menu and the browser's natural focus behavior

### Escape Behavior

**When menu is open**:
- Pressing Escape closes the menu by calling `setOpen(false)`
- Focus returns to the trigger button
- No action is executed

### Click-Outside Behavior

The menu also closes when the user clicks anywhere outside the menu container:
- Click on menu → menu stays open
- Click on trigger button → menu toggles (via button click handler)
- Click outside container → menu closes (via `handleClickOutside`)

---

## Focus Trap Architecture

Both ConfirmationDialog and UserMenu implement focus trapping via document event listeners.

### Implementation Pattern

```ts
// Attach listener when component is open
useEffect(() => {
  if (isOpen) {
    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }
}, [isOpen]);

// Detach listener when component closes
// This prevents event listener accumulation and ensures focus returns to page
```

### Keyboard Event Handling

- **ConfirmationDialog**: Handles Escape, Tab trapping, and Enter outside native buttons on
  `keydown`; native buttons handle Enter themselves to avoid duplicate confirmation
- **UserMenu**: Handles Escape on `keydown` event
- Both use document-level listeners for predictable capture (keyboard events bubble)

---

## Design System Integration

### Color Variants (ConfirmationDialog)

The confirmation dialog supports three variants, each with specific visual styling:

| Variant | Use Case | Button Color | Icon Color |
|---------|----------|--------------|-----------|
| **danger** | Delete, remove, destructive actions | Error red | Error red |
| **warning** | Caution required, non-destructive irreversible actions | Warning orange | Warning orange |
| **default** | Confirmations, neutral actions | Primary blue | Primary blue |

All variants use the same keyboard behavior (Escape, Enter, Tab) regardless of visual styling.

---

## Accessibility Standards (WCAG 2.2)

These keyboard patterns implement WCAG 2.2 Level AA accessibility:

- **2.1.1 Keyboard** — All functionality available via keyboard
- **2.4.3 Focus Order** — Focus order is logical and predictable
- **2.4.7 Focus Visible** — Focus indicator is visible (via browser default + Tailwind focus-ring styles)
- **3.2.1 On Focus** — No unexpected context switches when focus moves
- **3.2.2 On Input** — Changes only occur on explicit key press (not auto-triggered)

---

## Developer Notes

### Adding New Modal/Menu Components

When creating a new interactive component that should support keyboard navigation:

1. **Implement Escape handling** — Attach `keydown` listener when component is open
2. **Implement focus trap** — Use `useEffect` to attach/detach listener
3. **Implement focus restoration** — Ensure focus returns to trigger element when closing
4. **Test with keyboard** — Verify Tab cycle, focus trap, and Escape behavior
5. **Document in this file** — Add a section describing keyboard behavior

### Testing Keyboard Navigation

Use `userEvent` from React Testing Library to simulate keyboard interactions:

```ts
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';

it('closes dialog on Escape', async () => {
  const user = userEvent.setup();
  render(<ConfirmationDialog isOpen={true} onCancel={mockCancel} />);
  
  await user.keyboard('{Escape}');
  expect(mockCancel).toHaveBeenCalled();
});
```

### Debugging Focus Issues

If focus does not trap as expected:

1. Check that component is truly open (verify `isOpen` or `open` state)
2. Verify `useEffect` is attaching listener (check browser DevTools console for event listener count)
3. Ensure focus-cycled elements are keyboard-accessible (buttons, inputs, links)
4. Check for competing event listeners that may be stopping propagation

---

## Slice G additions (issue #766)

Slice G lands the cross-cutting accessibility / responsive / loading-state
sweep. The pieces that touched keyboard semantics are documented below;
every other slice G change is colour, motion or contract (no new
keyboard surface).

### Skip link — first focusable element on every authenticated page

`SkipLink` is rendered as the first focusable child of `<Layout>`. On a
fresh `Tab` press from a cold page load the link is the first stop in the
tab order; activating it (Enter) moves focus to `<main id="main-content"
tabindex="-1">` and scrolls the main landmark into view. The link is
visually hidden with `sr-only` until focused, then revealed with
`focus:not-sr-only`, `z-200`, and a 2 px surface-coloured halo plus 4 px
accent ring (`ds-focus-ring`). The label resolves through
`common:layout.skipToContent` so both EN and ES locales ship.

| Key | Action | Conditions |
|-----|--------|-----------|
| **Tab** (cold) | Reveal skip link and focus it | Page just loaded and no element is focused |
| **Enter** | Move focus + scroll to `<main id="main-content">` | Skip link is focused |
| **Tab** (warm) | Move focus to the next interactive element | Skip link has already been revealed this session |

### Drawer → bottom sheet collapse (compact viewports)

The `<ReviewChangesPanel>` rendered on the configuration surface stays as
a centered modal at `≥ sm` (640 px) and collapses to a bottom sheet
anchored to the bottom edge below that breakpoint:

- `items-end` + `rounded-t-2xl` + `max-h-[85vh]` below `sm`.
- Focus trap and Escape behaviour are unchanged.
- Keyboard users on a 320 px viewport get the same Escape / Tab
  semantics inside the sheet as on the desktop modal. The sheet is
  scrollable so long diff lists remain reachable.

### Sidebar drawer overlay

The sidebar drawer overlay shipped with an `<label aria-label="...">`
which axe flagged as an ARIA prohibited-attribute combination (the
`<label>` element already carries an implicit accessible name from its
`for` attribute; ARIA forbids layering an additional `aria-label` on a
generic `<label>`). The fix replaces the aria-label with a
`<span class="sr-only">` child carrying the new i18n key
`common:layout.closeSidebar`. Keyboard behaviour is unchanged: clicking
or pressing Enter while the label is focused still untoggles the
drawer's hidden checkbox and closes the sidebar.

### prefers-reduced-motion (W4-a)

A global `@media (prefers-reduced-motion: reduce)` block at the bottom
of `src/index.css` collapses every Tailwind transition-* utility and
animate-* utility to a 0.001ms sentinel, plus kills the custom
`spin-smooth` keyframes via `animation-name: none`. This applies to:

- the `Skeleton` component's pulse (W1);
- the `LoadingBoundary` pending state (W1);
- the confirmation dialog's colour transition;
- any per-component `motion-reduce:animate-none` utility (defensive).

When a user has prefers-reduced-motion: reduce set, the loading
indicators stop pulsing but remain visible, focus rings stay at full
contrast, and screen-reader announcements stay intact. The
`prefers-reduced-motion` browser test in `e2e/a11y.spec.ts` verifies
the gate end-to-end.

### Contrast auditor (W4-a)

`scripts/check-contrast-tokens.mjs` reads the colour tokens declared in
`tailwind.config.js` and `src/index.css`, computes the WCAG 2.1
relative-luminance ratio for every documented foreground / background
pair in both themes, and fails when any pair drops below the AA
threshold (4.5:1 for body text, 3:1 for UI components). It runs in two
modes:

- `pnpm check:contrast` — report-only, exits 0 even on drift; intended
  for local development.
- `pnpm check:contrast:strict` — exits 1 on AA drift; intended for CI
  once the pre-existing design drift in corporateLight muted text is
  fixed (see the W4 evidence section in the ODD task).

The auditor's unit test lives in
`scripts/check-contrast-tokens.test.mjs` and covers the happy path,
report-only drift, `--strict` drift, missing-token resolution, and a
calculator identity check.

### axe-core browser gate (W4-b)

`e2e/a11y.spec.ts` runs `@axe-core/playwright` with the default AA
rule set (`wcag2a`, `wcag2aa`, `wcag21a`, `wcag21aa`) on the four most
trafficked surfaces — `/login`, `/overview`, `/configuration`,
`/security` — and fails when any violation carries `critical` or
`serious` impact. The first run of the gate surfaced the sidebar
overlay aria-label issue documented above; W4-b fixes it. A future
slice can layer in the best-practice / WCAG 2.2 rule set once the open
design backlog items are addressed.

---

## Changelog

| Date | Change | Phase |
|------|--------|-------|
| 2026-05-19 | Initial documentation created | Phase 6 |
| 2026-05-19 | ConfirmationDialog keyboard patterns documented | Phase 6 |
| 2026-05-19 | UserMenu keyboard patterns documented | Phase 6 |
| 2026-05-19 | Focus trap and restoration documented | Phase 6 |
| (slice G) | Skip link, drawer to bottom sheet, sidebar overlay a11y fix, prefers-reduced-motion gate, contrast auditor, axe-core browser gate | issue #766 slice G |

---

## Related Issues

- **Issue #116** — UX Baseline Standardization (this documentation)
- **Access Control** — WCAG 2.2 Level AA compliance baseline

---

## Questions?

For questions about keyboard navigation:
- Check this document first
- Review component implementations in `frontend/src/shared/components/`
- Consult the spec at `sdd/issue-116-ux-baseline/spec` for detailed requirements
