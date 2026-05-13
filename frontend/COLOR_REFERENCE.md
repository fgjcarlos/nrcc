# NRCC Color Reference — Quick Lookup

## Dark Theme (corporateDark)
```css
/* Primary (Signal Red) */
#ff3b30  primary
#ffffff  primary-content
#4f1110  primary-container
#fff1f0  on-primary-container

/* Secondary (Navy) */
#10242a  secondary
#dff7f3  secondary-content

/* Accent (Sky Cyan) */
#38bdf8  accent
#041016  accent-content

/* Base Colors */
#06080a  base-100  (main background)
#0c1116  base-200  (secondary background)
#151c24  base-300  (tertiary background/inputs)
#eef4f8  base-content (text)

/* Semantic */
#2dd4bf  success (Teal 400)
#fbbf24  warning (Amber 300)
#ff3b30  error (Red 500)
#38bdf8  info (Sky 300)

#1a242d  neutral
#eef4f8  neutral-content
```

## Light Theme (corporateLight)
```css
/* Primary (Corporate Red) */
#d92d20  primary
#ffffff  primary-content
#fff1f0  primary-container
#4f1110  on-primary-container

/* Secondary (Teal Tint) */
#e4f4f1  secondary
#0f3d3a  secondary-content

/* Accent (Cobalt) */
#0369a1  accent
#ffffff  accent-content

/* Base Colors */
#f7f9fb  base-100  (main background)
#edf2f6  base-200  (secondary background)
#dce5ec  base-300  (tertiary background/inputs)
#111827  base-content (text)

/* Semantic */
#0f766e  success (Teal 800)
#b45309  warning (Amber 700)
#d92d20  error (Red 700)
#0284c7  info (Sky 700)

#334155  neutral
#f8fafc  neutral-content
```

## Shadow Definitions
```css
/* Custom shadows in tailwind.config.js */
shadow-glow:       0 24px 60px rgba(0, 0, 0, 0.42), 0 0 0 1px rgba(148, 163, 184, 0.08)
shadow-glow-light: 0 24px 48px rgba(15, 23, 42, 0.08), 0 0 0 1px rgba(15, 23, 42, 0.04)

/* ⚠️ MISSING: shadow-sm, shadow-md, shadow-lg, shadow-xl scales */
```

## Background Gradients
```css
/* Dark Mode Body Background */
linear-gradient(90deg, rgba(56, 189, 248, 0.04) 1px, transparent 1px),     /* Grid X */
linear-gradient(180deg, rgba(56, 189, 248, 0.03) 1px, transparent 1px),    /* Grid Y */
radial-gradient(circle at 8% 0%, rgba(255, 59, 48, 0.12), transparent 28%),   /* Red glow */
radial-gradient(circle at 92% 12%, rgba(45, 212, 191, 0.08), transparent 26%), /* Teal glow */
linear-gradient(180deg, #0a0d10 0%, #06080a 48%, #040607 100%);              /* Base gradient */

/* Light Mode Body Background */
linear-gradient(90deg, rgba(15, 23, 42, 0.035) 1px, transparent 1px),     /* Grid X */
linear-gradient(180deg, rgba(15, 23, 42, 0.03) 1px, transparent 1px),    /* Grid Y */
radial-gradient(circle at 8% 0%, rgba(217, 45, 32, 0.08), transparent 28%),   /* Red glow */
radial-gradient(circle at 92% 12%, rgba(15, 118, 110, 0.07), transparent 26%), /* Teal glow */
linear-gradient(180deg, #fbfcfd 0%, #f7f9fb 48%, #edf2f6 100%);              /* Base gradient */
```

## Opacity Rules (Dark Mode)
```
Borders:           14-16% opacity
Hover backgrounds: 6-8% opacity
Card backgrounds:  68-92% opacity (gradient stop)
Input backgrounds: 100% (base-300 color)
Text secondary:    60% opacity
Placeholder:       50% opacity
Glass panel:       68-78% opacity (gradient stop)
```

## Border Radius Values
```css
--radius-selector: .375rem  (6px)    /* Button/card corners — UNUSED */
--radius-field:    .25rem   (4px)    /* Input corners — UNUSED */
--radius-box:      .5rem    (8px)    /* Card/panel corners — UNUSED */

/* ACTUAL usage: Tailwind classes */
rounded-md   →  6px  (inputs)
rounded-lg   →  8px  (buttons, cards, panels)
rounded-2xl  →  16px (warning banner)
```

## Typography CSS Utilities
```css
.text-display       font-weight: 700, letter-spacing: -0.02em, line-height: 1.1
.text-headline      font-weight: 600, letter-spacing: -0.01em, line-height: 1.25, margin-top: 3rem
.text-title         font-weight: 500, line-height: 1.3
.text-body-secondary line-height: 1.6, color: #9aa8b5 (dark) or rgba(17, 24, 39, 0.62) (light)
.text-label         font-weight: 500, font-size: 0.75rem (12px), letter-spacing: 0.05em, text-transform: uppercase

/* ⚠️ MISSING UTILITIES */
.text-body          (default paragraph text)
.text-caption       (small auxiliary text)
.text-code          (monospace code blocks)
```

## Component Class Variants

### Button Classes
```css
.btn-primary    Red gradient (dark) / Red solid (light)
.btn-secondary  Ghost border with sky/blue hover
.btn-tertiary   Text-only accent (cyan)
.btn-success    Teal background with semantic color
.btn-warning    Amber background with semantic color
.btn-error      Red background with semantic color
.btn-info       Cyan background with semantic color
.btn-ghost      Transparent with minimal hover
```

### Semantic Surface Classes
```css
.glass-panel       Glassmorphism: blur 18px + gradient + glow shadow
.surface-panel     Filled panel: gradient background + border
.surface-card      Lighter than panel: lower opacity gradient
.app-shell         Hero container: radial gradients + fade
.auth-shell        Auth background: red + blue radials
```

### Input Focus States
```css
.input-focus-glow  4px box-shadow with primary color
.input-error-glow  4px box-shadow with error red
```

## Overlay & Modals
```css
.modal-overlay      Backdrop: 75% dark (dark), 25% dark (light)
.modal-inner        Inner panel: subtle background
.ghost-divider      Border: 14-16% opacity
```

## Key Properties Used in Components

| Component | Primary Hex | Accent Hex | Border | Shadow |
|-----------|------------|-----------|--------|--------|
| Button (dark) | #ff3b30 | N/A | rgba(255,59,48, 0.32) | glow |
| Button (light) | #d92d20 | N/A | rgba(217,45,32, 0.18) | glow-light |
| Input (dark) | #38bdf8 (focus) | N/A | none | input-focus-glow |
| Input (light) | #0369a1 (focus) | N/A | none | input-focus-glow |
| Card (dark) | N/A | N/A | rgba(148,163,184, 0.14) | glow |
| Card (light) | N/A | N/A | rgba(15,23,42, 0.08) | glow-light |

---

**Last Updated**: 14/05/2026  
**Analysis**: See `frontend/FRONTEND_ANALYSIS.md` for full report
