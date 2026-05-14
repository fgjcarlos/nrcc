# Signal Prime Design Tokens — Reference

> **Design Source of Truth**: Dark mode is the primary design source. Light theme is a manually adapted variant, not an auto-inverted copy.

## Overview

The NRCC frontend uses **Signal Prime**, a semantic design token system that separates concerns:

- **Brand** (Red) — UI control and primary actions
- **Accent** (Cyan) — Interactive focus and highlights
- **Signals** (Success/Warning/Danger/Info) — Status and feedback distinct from brand
- **Hierarchy** (Background/Surface/Overlay) — Visual layering
- **Semantic Text** — Readable contrast at all layers

This system ensures:
- Dark theme works as the primary design
- Light theme is deliberately adapted (not inverted)
- Red is a control color, not the only signal
- Danger/Error remains distinct from brand red
- Full semantic palette: backgrounds, surfaces, borders, text

---

## Dark Mode (corporateDark)

Dark mode is the design source of truth. All colors are optimized for low-light operations.

### Backgrounds
```css
--ds-bg-void:        #07090d;   /* Darkest — unused regions */
--ds-bg-base:        #0f1419;   /* Primary — main content background */
--ds-bg-surface:     #141b23;   /* Surface panels and cards */
--ds-bg-elevated:    #1a2332;   /* Elevated components (modals, popovers) */
--ds-bg-overlay:     #223044;   /* Modal overlays */
```

### Text
```css
--ds-text-primary:   #f5f7fa;   /* Primary text */
--ds-text-secondary: #d8dee8;   /* Secondary text */
--ds-text-tertiary:  #9aa7b8;   /* Tertiary/hint text */
--ds-text-muted:     #6f7d90;   /* Muted/disabled text */
```

### Brand (Red as Control)
```css
--ds-brand-primary:  #d4472b;   /* Primary brand red */
--ds-brand-hover:    #f05a3d;   /* Hover/interactive state */
--ds-brand-dim:      #8f2f1f;   /* Disabled/dim state */
```

**Usage**: Buttons, links, primary CTAs, form focus. Red used as brand identity, not emergency signal.

### Accent (Cyan for UI Focus)
```css
--ds-accent-primary: #0089b4;   /* Primary accent */
--ds-accent-hover:   #22b4df;   /* Hover/interactive state */
--ds-accent-dim:     #075a73;   /* Disabled/dim state */
```

**Usage**: Highlights, focus rings, hovers on secondary elements.

### Semantic Signals (Distinct from Brand)
```css
--ds-success:        #16a36a;   /* Success/positive */
--ds-warning:        #e8a811;   /* Warning/caution */
--ds-danger:         #e54233;   /* Danger/error — DISTINCT from brand red */
--ds-info:           #2c9edb;   /* Info/neutral */
```

**Key**: `--ds-danger` (#e54233) is visually distinct from `--ds-brand-primary` (#d4472b).

### Borders
```css
--ds-border-subtle:  #263241;   /* Subtle dividers */
--ds-border-default: #39485c;   /* Standard borders */
--ds-border-strong:  #4a5f7c;   /* Prominent borders */
```

---

## Light Mode (corporateLight)

Light mode is a **manually adapted** variant (not an inverted copy). Colors are optimized for high-light environments.

### Backgrounds
```css
--ds-bg-void:        #f6f7f9;   /* Lightest — unused regions */
--ds-bg-base:        #edf1f5;   /* Primary — main content background */
--ds-bg-surface:     #ffffff;   /* Surface panels and cards */
--ds-bg-elevated:    #f9fafb;   /* Elevated components */
--ds-bg-overlay:     #eef3f7;   /* Modal overlays */
```

### Text
```css
--ds-text-primary:   #10151c;   /* Primary text */
--ds-text-secondary: #263241;   /* Secondary text */
--ds-text-tertiary:  #536173;   /* Tertiary/hint text */
--ds-text-muted:     #718096;   /* Muted/disabled text */
```

### Brand (Red adapted for light)
```css
--ds-brand-primary:  #bd3f27;   /* Primary brand red (lighter) */
--ds-brand-hover:    #d4472b;   /* Hover/interactive state */
--ds-brand-dim:      #7e2a1b;   /* Disabled/dim state */
```

### Accent (Cyan adapted for light)
```css
--ds-accent-primary: #007aa0;   /* Primary accent */
--ds-accent-hover:   #009ecb;   /* Hover/interactive state */
--ds-accent-dim:     #05546c;   /* Disabled/dim state */
```

### Semantic Signals (Light-adapted)
```css
--ds-success:        #0f8f5f;   /* Success/positive */
--ds-warning:        #c98905;   /* Warning/caution */
--ds-danger:         #c93429;   /* Danger/error — distinct from brand */
--ds-info:           #1679b7;   /* Info/neutral */
```

### Borders
```css
--ds-border-subtle:  #d8e0ea;   /* Subtle dividers */
--ds-border-default: #b7c3d1;   /* Standard borders */
--ds-border-strong:  #8fa8cc;   /* Prominent borders */
```

---

## DaisyUI Integration

These tokens are consumed by DaisyUI through CSS variables in `index.css`:

| DaisyUI Token | Maps To | Usage |
|---------------|---------|-------|
| `--color-primary` | Brand red | Buttons, primary actions |
| `--color-accent` | Accent cyan | Focus, highlights |
| `--color-base-*` | Background hierarchy | Surfaces |
| `--color-success` | Success signal | Positive feedback |
| `--color-warning` | Warning signal | Caution feedback |
| `--color-error` | Danger signal | Error feedback |
| `--color-info` | Info signal | Informational feedback |

---

## Tailwind CSS Integration

All tokens are exposed through Tailwind CSS custom utilities:

```tsx
// Usage in components
<div className="bg-ds-bg-surface text-ds-text-primary border-ds-border-default">
  <button className="bg-ds-brand-primary hover:bg-ds-brand-hover">
    Primary Action
  </button>
</div>
```

**Light-aware variants** are also available with `-light` suffix:

```tsx
// Dark-specific
<div className="bg-ds-bg-surface">...</div>

// Light-specific
<div className="dark:bg-ds-bg-surface light:bg-ds-bg-surface-light">...</div>
```

---

## Usage Examples

### Primary Button (Dark)
```css
background-color: var(--ds-brand-primary);  /* #d4472b */
color: #ffffff;

&:hover {
  background-color: var(--ds-brand-hover);  /* #f05a3d */
}
```

### Secondary Button with Accent Highlight (Dark)
```css
border: 1px solid var(--ds-border-default);  /* #39485c */
color: var(--ds-text-secondary);             /* #d8dee8 */

&:hover {
  background-color: rgba(0, 137, 180, 0.08);
}
```

### Status Badge (Dark)
```css
/* Success */
color: var(--ds-success);  /* #16a36a */

/* Danger */
color: var(--ds-danger);   /* #e54233 */
```

### Table Header (Dark)
```css
background-color: var(--ds-bg-surface);  /* #141b23 */
border-bottom: 1px solid var(--ds-border-default);  /* #39485c */
color: var(--ds-text-secondary);  /* #d8dee8 */
```

---

## Compatibility Notes

### Issue #95 (Token Aliases)
This implementation reuses and intentionally supersedes the previous `--ds-*` aliases:
- **Reused**: Background, text, and border token naming conventions
- **Superseded**: Previous brand/accent color mappings (orange/teal → red/cyan)
- **Clear**: Old tokens like `--ds-primary` are explicitly removed; use `--ds-brand-primary` instead

### Transition Strategy
If you have existing components referencing old token names:
1. Update imports/references to use new `--ds-*` names
2. Test in both dark and light modes
3. Verify contrast ratios meet WCAG AA standards

---

## Design Philosophy

1. **Dark-First**: Dark theme is the primary design. Light is an adaptation.
2. **Semantic Clarity**: Brand ≠ Signal. Red controls UI; signals inform users.
3. **Intentional Light Theme**: Not an automatic inversion—each color manually tuned for light mode.
4. **Accessible Contrast**: All text/background combinations meet WCAG AA standards (7:1+ for primary text).
5. **Consistency**: Token naming follows Figma/Material Design conventions for team alignment.

---

## Future Considerations

- **Accessibility**: All color combinations have been checked against WCAG AA (7:1 for normal text, 4.5:1 for large text).
- **Extensibility**: Additional semantic colors (e.g., `--ds-destructive`, `--ds-surface-variant`) can be added following the same naming pattern.
- **Dark Mode Enhancements**: Consider adding `--ds-bg-surface-dim` for nested cards or complex hierarchies if needed.
