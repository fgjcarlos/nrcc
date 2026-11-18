---
name: NRCC
description: Node-RED 5 Control Center, drawn as an IEC wiring diagram; a settings change is a circuit that closes only when every gate passes.
colors:
  # Light theme (corporateLight, "drawing white") — src/index.css
  conductor-blue: "#2b5fb4"
  conductor-blue-hover: "#234e94"
  conductor-blue-dim: "#dbe4f3"
  conductor-blue-text: "#2959a8"
  pending-brown: "#7a4a2a"
  verified-green: "#2d6a22"
  verified-green-text: "#2c6722"
  verified-fill: "#c9b81a"
  critical-red: "#b3261e"
  critical-red-text: "#a8251e"
  warning-amber: "#e8a317"
  warning-amber-text: "#8a5a00"
  drawing-white: "#f6f5f0"
  drawing-sunken: "#eeede6"
  sheet: "#fbfaf7"
  sheet-raised: "#ffffff"
  hairline: "#d6d4cb"
  control-border: "#838891"
  ink: "#15171c"
  ink-secondary: "#3d424b"
  ink-muted: "#5c626c"
  rail-text: "#e9e7df"
  # Dark theme (corporateDark, "drawing negative") — src/index.css
  conductor-blue-dark: "#7fa6ea"
  conductor-blue-hover-dark: "#9ab9ef"
  conductor-blue-dim-dark: "#24375a"
  pending-brown-dark: "#c99a6e"
  verified-green-dark: "#8cc66b"
  critical-red-dark: "#f08a80"
  warning-amber-dark: "#e8b04a"
  drawing-negative: "#111317"
  drawing-sunken-dark: "#0c0e11"
  sheet-dark: "#171a1f"
  sheet-raised-dark: "#1d2128"
  hairline-dark: "#2a2f37"
  control-border-dark: "#6b717c"
  ink-dark: "#e9e7df"
  ink-secondary-dark: "#c3c1b8"
  ink-muted-dark: "#9a9ea6"
  rail-dark: "#0a0b0d"
typography:
  title:
    fontFamily: "Overpass Variable, system-ui, sans-serif"
    fontSize: "1.25rem"
    fontWeight: 700
    lineHeight: 1.4
  body:
    fontFamily: "Overpass Variable, system-ui, sans-serif"
    fontSize: "0.875rem"
    fontWeight: 400
    lineHeight: 1.43
  body-strong:
    fontFamily: "Overpass Variable, system-ui, sans-serif"
    fontSize: "0.875rem"
    fontWeight: 600
    lineHeight: 1.43
  label:
    fontFamily: "Overpass Variable, system-ui, sans-serif"
    fontSize: "0.75rem"
    fontWeight: 400
    lineHeight: 1.33
  designator:
    fontFamily: "Overpass Mono, ui-monospace, monospace"
    fontSize: "0.75rem"
    fontWeight: 600
    lineHeight: 1.33
    fontFeature: "tnum"
rounded:
  selector: "2px"
  field: "4px"
  box: "4px"
  full: "9999px"
spacing:
  xs: "4px"
  sm: "8px"
  md: "12px"
  lg: "16px"
  xl: "24px"
components:
  button-primary:
    backgroundColor: "{colors.sheet}"
    textColor: "{colors.conductor-blue-text}"
    rounded: "{rounded.field}"
    padding: "8px 16px"
    height: "36px"
  button-solid:
    backgroundColor: "{colors.conductor-blue}"
    textColor: "{colors.sheet-raised}"
    rounded: "{rounded.field}"
    padding: "8px 16px"
  button-solid-hover:
    backgroundColor: "{colors.conductor-blue-hover}"
  button-secondary:
    backgroundColor: "{colors.sheet}"
    textColor: "{colors.ink}"
    rounded: "{rounded.field}"
    padding: "8px 16px"
    height: "36px"
  button-ghost:
    textColor: "{colors.ink}"
    rounded: "{rounded.field}"
    padding: "8px 12px"
    height: "36px"
  input:
    backgroundColor: "{colors.sheet-raised}"
    textColor: "{colors.ink}"
    typography: "{typography.body}"
    padding: "8px 12px"
  terminal-label:
    textColor: "{colors.ink}"
    typography: "{typography.designator}"
  tab:
    textColor: "{colors.ink-muted}"
    typography: "{typography.body}"
    padding: "8px 12px"
    height: "36px"
  tab-selected:
    textColor: "{colors.conductor-blue-text}"
  change-circuit:
    backgroundColor: "{colors.sheet}"
    width: "20rem"
  lock-banner:
    textColor: "{colors.ink-secondary}"
    rounded: "{rounded.box}"
    padding: "8px 12px"
---

# Design System: NRCC

## Overview

**Creative North Star: "The Wiring Diagram"**

NRCC is drawn the way a panel builder draws a control cabinet: IEC schematic language on a drawing-white sheet (the dark theme is the drawing negative), hairline ink conductors, and conductor colours that each do one job. The stylesheet names the world "Esquema eléctrico". The operator configures one Node-RED instance and must see what will happen before it happens, so the interface reads as a technical drawing they can trust, not as a dashboard of cards.

The signature is the change circuit: the gated sequence Edit → Diff → Validate → Backup → Apply → Ready drawn as six normally-open contacts on one conductor. A contact closes when its gate passes; a failure leaves it open and, when the server rolled back, draws the rollback branch. Every state is said in words and shape, never by colour alone. Density is a feature: three type sizes, ranked by weight, case and rule rather than by size.

Identity lives in designators, not decoration. The settings revision is `REV a1b2c3d4`, contacts are `K1`–`K6`, the rollback branch is `RB`, and every settings field carries its `settings.js` key as a terminal label. Light and dark are equal citizens; WCAG 2.2 AA holds in both.

**Key Characteristics:**
- Flat drawing sheet, 1px hairlines, at most a 1px-offset shadow.
- Conductor colours with fixed jobs: blue acts and selects, brown is pending, green-yellow fills a verified contact, red is critical and rollback only.
- Three type sizes (20, 14, 12px) in Overpass; designators in Overpass Mono.
- Stable mono designators (REV, K1–K6, RB, settings keys) instead of icons and badges.
- Self-hosted fonts and assets; nothing loads from the network.

## Colors

Conductor colours on a warm drawing sheet; every hue has one job, and the job holds in both themes. Each colour exists in a light value and a dark value (the `-dark` keys), declared per theme in `src/index.css`.

### Primary
- **Neutral Conductor Blue** (`conductor-blue` / `conductor-blue-dark`): action and selection only. The primary button border and label, the selected tab underline, focus outlines, the checked toggle, row hover tints and the text caret. Text on a plain or tinted surface uses **Conductor Blue Text** (`conductor-blue-text`; dark uses the blue itself) so labels keep 4.5:1 on an 18% tint. Hover deepens to `conductor-blue-hover`; `conductor-blue-dim` is the quiet selection ground.

### Secondary
- **Pending Brown** (`pending-brown` / `pending-brown-dark`): a step that is energising or waiting for the operator, such as the Diff contact before review and the server contacts while the transaction runs. It is the `info` role in daisyUI terms; it never means "information" decoration.
- **Verified Green** (`verified-green` / `verified-green-dark`, text `verified-green-text`): a passed gate and the "Editable" runtime status.
- **Green-Yellow Verified Fill** (`verified-fill`, same in both themes): the small square drawn beside a closed contact. It is a fill, never text; it carries an ink stroke so it holds on both grounds.

### Tertiary
- **Critical Red** (`critical-red` / `critical-red-dark`, text `critical-red-text`): a failed contact, the dashed rollback branch, the `RB` designator, destructive buttons and validation errors. Nothing else.
- **Warning Amber** (`warning-amber` / `warning-amber-dark`, text `warning-amber-text`): a gate held by the operator's situation (read-only runtime, invalid fields: the "Open" contact state) and the read-only notice.

### Neutral
- **Drawing White / Drawing Negative** (`drawing-white` / `drawing-negative`): the page ground.
- **Sunken** (`drawing-sunken`, dark `drawing-sunken-dark`): table headers, code blocks, the off toggle track.
- **Sheet** (`sheet` / `sheet-dark`): panels, cards, the change circuit column, button grounds.
- **Raised Sheet** (`sheet-raised` / `sheet-raised-dark`): inputs, dialogs, overlays.
- **Hairline** (`hairline` / `hairline-dark`): decorative dividers and panel edges.
- **Control Border** (`control-border` / `control-border-dark`): every control edge and the circuit conductor; it clears 3:1 against the grounds (WCAG 1.4.11).
- **Ink** (`ink` / `ink-dark`), **Ink Secondary**, **Ink Muted**: primary, secondary and muted text. Light `ink` is also the navigation rail ground; `rail-dark` is the dark rail and `rail-text` its label colour.

### Named Rules
**The One Job Rule.** A conductor colour means one thing everywhere: blue acts or selects, brown is pending, green and green-yellow are verified, red is critical or rollback, amber is a held gate. Never use a conductor colour as decoration.

**The Red Is Rare Rule.** Red appears only for a failure, a rollback, a destructive action or an error. A screen with no fault shows no red.

**The Tint Text Rule.** Text on a semantic tint or a plain surface uses the `*-text` value (`--ds-on-tint-*`), never the solid fill colour; the solid colour is for borders, fills and button grounds.

## Typography

**Body Font:** Overpass Variable (with system-ui, sans-serif)
**Label/Mono Font:** Overpass Mono (with ui-monospace, monospace), weights 400, 500 and 600

**Character:** Overpass descends from highway signage, so it reads like the lettering on a drawing title block: plain, upright, slightly engineered. The mono cut carries every designator, so codes line up and never wrap mid-word. Both are self-hosted through `@fontsource` in `src/main.tsx`.

### Hierarchy
- **Title** (700, 20px, 28px line): the page title only, one per view.
- **Body** (400, 14px, 20px line): form controls, tabs, descriptions, runtime status. Section headings and contact names use **Body Strong** (600, 14px).
- **Label** (400–600, 12px, 16px line): field labels, help text, contact state words, proof lines, totals.
- **Designator** (Overpass Mono 600, 12px, tabular figures): REV, K1–K6, RB, settings keys, server error codes, version numbers.

### Named Rules
**The Three Sizes Rule.** Use only 20, 14 and 12px. Rank by weight, case and a rule line, never by adding a fourth size.

**The Tabular Data Rule.** Tables, code, counts and totals use tabular numerals so columns and digits never shift.

## Layout

Dense, column-based operating layout. From `lg` (1024px) the ink rail (16rem) sits beside the content; below `lg` it is a drawer opened from the header menu button, and the header controls are icon-only (the command palette below `lg`, the theme and update labels below `xl`) so the header never needs a second row. At `xl` (1280px) and up, Configuration is a two-column grid: the settings form takes the remaining width and the change circuit is a sticky right column of 20rem, with a 24px gap. Below `xl` the circuit becomes a sticky horizontal strip above the form: six contact symbols on a horizontal conductor and a one-line summary ("Step 3 of 6 · Validate · Waiting"). The column waits for `xl` because at `lg` the rail already takes 16rem and a column would leave the form narrower than a phone.

**Breakpoints.** `md` (768px) is the touch-target switch, `lg` (1024px) the rail, `xl` (1280px) the circuit column. Below `md` every standalone target (buttons, text controls, tabs, standalone links, icon buttons) is at least 44×44px; from `md` up at least 24×24px. A global rule in `src/index.css` holds the 44px height for buttons and text controls below `md`, so a call site's `min-h-*` utility only applies from `md` up; other targets use `min-h-11 md:min-h-6`-style pairs.

**Nothing scrolls sideways.** The document never scrolls horizontally at any width from 320px (`e2e/responsive.spec.ts` checks 320 and 768). Grid and flex children that hold text carry `min-w-0`; rows wrap instead of scrolling (tab lists wrap onto a second line); tables stack into labelled rows below `sm` (640px) through the shared `<Table stacked>`; paths, ids and package names break anywhere (`break-all`) rather than truncate; content takes no fixed pixel widths.

Spacing runs on a 4px base: 4px inside label rows, 8px between a control and its neighbours and inside the circuit foot, 12px between contacts and between header items, 16px between page blocks, 24px between columns. The page header sits on a hairline rule (12px below it), and sections are separated by hairline rules rather than by cards.

## Elevation & Depth

The system is flat. Depth comes from tonal grounds (sunken, sheet, raised sheet) and 1px hairlines; a surface may add at most one shadow, the raised elevation. There are no gradients, glows or glass blur.

### Shadow Vocabulary
- **Raised** (light `box-shadow: 0 1px 2px rgb(21 23 28 / 0.08)`, dark `0 1px 2px rgb(0 0 0 / 0.4)`, `--ds-shadow-raised`): panels, cards and the legacy `glow` utility names, which all resolve to it.
- **Focus ring** (`box-shadow: 0 0 0 2px var(--ds-bg-base), 0 0 0 4px var(--ds-accent-primary)`): a sheet-coloured halo plus a conductor-blue ring for keyboard focus. Controls without their own focus style get a 2px solid conductor-blue outline at 2px offset.

### Named Rules
**The Hairline Rule.** Elevation is a 1px border plus at most `--ds-shadow-raised`. If a surface needs more separation, use a ground change or a rule line, not a bigger shadow.

## Motion

Motion explains a state change or an arriving overlay; it never decorates. One curve, two durations, defined once in `src/index.css` (`--motion-ease`, `--motion-state`, `--motion-overlay`) and wired into Tailwind's default transition, so every `transition-*` and `ease-out` utility uses them.

- **Curve:** exponential ease-out, `cubic-bezier(0.16, 1, 0.3, 1)`. No bounce, elastic or spring easing or keyframes.
- **State changes, 150ms:** a contact closing, chips, disclosure chevrons, toggles, and colour or border changes on buttons, fields, rows and tabs. Animate colour, opacity and transform only; never width, height or position (the rail collapses instantly).
- **Overlays, 150–200ms:** dialogs, the navigation drawer, the review drawer and the command palette enter with opacity plus a 4–8px translate toward their resting place (`.motion-overlay`, 180ms; direction from `--overlay-from`). Their scrim fades in over 150ms (`.motion-scrim`). Exit is instant.
- **The authored moment:** the change circuit closes its contacts in sequence, 60ms per contact index. It is the only staggered motion in the product.
- **Loops:** only loading spinners (1s linear) and skeleton shimmer run indefinitely. A pending action shows a spinner, not a pulsing icon.
- **Reduced motion:** under `prefers-reduced-motion: reduce` one global rule at the end of `src/index.css` collapses every transition and animation to 0.001ms and every delay to 0s, inline styles included, so the contact stagger and the overlay entrances are instant. State still changes; only the movement goes. `pnpm run test:appearance` pins the grammar and the reduced-motion path.

## Shapes

Drafting geometry: near-square corners, straight 1px conductors, and IEC symbols drawn with a 1.5px square-capped stroke. Selectors (checkbox, radio) take 2px corners; fields, buttons, panels and banners take 4px (`--radius-field`, `--radius-box`). Status chips take 2px corners like selectors; only the toggle track and knob are fully round. Dashed strokes mean "not confirmed" (an unknown contact blade) or the rollback return path; solid strokes mean a drawn conductor.

### The mark

The NRCC mark is the closed IEC normally-open contact: a conductor entering two terminal rings bridged by a heavier blade, on a 16-unit grid. Keep one terminal-ring diameter (4 units, a quarter of the glyph) clear on every side, and never draw the glyph below 16px, where the rings fill in. The favicon sets it in ink on a drawing-white tile.

## Components

### Buttons
Flat sheet, conductor-coloured edge; the colour is the job.
- **Shape:** 4px corners (`{rounded.field}`), 36px minimum height (44px below `md`).
- **Primary (action):** sheet ground, conductor-blue 1px border and label, 8px 16px padding, 500 weight. Hover lays a 10% blue tint over the sheet. "Review N changes" under the circuit is the canonical instance.
- **Solid primary:** conductor-blue fill with white (dark: drawing-sunken) label, 600 weight; hover goes to `conductor-blue-hover`.
- **Secondary:** sheet ground, control-border edge, ink label; hover lays a 6% ink tint over the sheet. "Unlock to edit settings.js" is the canonical instance.
- **Ghost:** transparent border, ink label; hover is a 10% muted tint. "Discard changes" under the circuit uses it with secondary ink.
- **Danger:** sheet ground with a critical-red border and label; light-theme semantic buttons use a solid fill, dark-theme ones a 28% tint.
- **Disabled:** 50% opacity, default cursor.

### Inputs / Fields
- **Style:** raised-sheet ground, 1px control-border edge, ink text, muted placeholder, 14px body text.
- **Terminal label:** every settings field is labelled by its `settings.js` key in Designator type (`uiPort`), a middle dot, then the human label in 12px secondary ink. The key may wrap only after a dot (`editorTheme.codeEditor.` / `options.theme`). The control's accessible name reads "UI Port (uiPort)", so speech users hear the plain label first.
- **Focus:** the border turns conductor blue under a 3px 24% blue glow; keyboard focus adds a 2px conductor-blue outline at 2px offset.
- **Error / Disabled:** a 1px critical-red ring with a 3px 24% red glow; disabled controls drop to 50% opacity with a not-allowed cursor.
- **Toggle:** a 44×24px round track. Off is a control-border outline on the sunken ground with a muted knob; on is a conductor-blue fill with a sheet-coloured knob. It exposes `aria-pressed`.

### Navigation
- **Tabs (section tablist):** a WAI-ARIA tablist with roving tabindex on a hairline baseline. Each tab is 36px high (44px below `md`), 14px at 500 weight, with an icon. Tabs that do not fit wrap onto another line; the tablist never scrolls sideways. Unselected tabs are muted ink with a transparent 2px underline that turns control-border on hover; the selected tab is conductor-blue text over a 2px blue underline.
- **Sidebar links:** the active link takes a 12% blue tint, a control-border edge and a 2px inset blue bar on its leading edge.
- **Ink rail:** `--ds-rail-bg` and `--ds-rail-text` define an ink navigation rail (light: ink; dark: `rail-dark`). The app shell applies it: the sidebar sits on the rail ground and the selected link carries an action-blue edge bar.

### Lock banner
The raw `settings.js` editor is locked by default. The locked state is a 4px-cornered banner with a control-border edge, 14px secondary text explaining the lock and the backup, and a secondary "Unlock to edit settings.js" button beside the text. Unlocking requires an acknowledgement dialog. The section states plainly that direct editing bypasses the change circuit.

### Page header and designators
The page title (Title type) shares a baseline with its designator `REV` plus the first 8 characters of the settings revision fingerprint (Designator type, full fingerprint in the tooltip) and exactly one runtime status: `Node-RED <version>` in mono, then "Editable" in verified green or "Read-only" in warning amber. Host details follow in 12px secondary type, and a hairline rule closes the header.

### Change Circuit (signature)
The gated change sequence as a single-line schematic.
- **Structure:** an ordered list of six contacts, Edit, Diff, Validate, Backup, Apply and Ready, numbered `K1`–`K6` in Designator type. They hang on one continuous 1px conductor drawn in the control-border colour on whole pixels: vertical in the `xl` column, horizontal in the narrow strip.
- **Symbol:** an IEC normally-open contact, 20×20px. It has two terminal stubs and a blade that pivots on the lower terminal, drawn with a 1.5px square-capped stroke. Open: the blade is rotated −30° off the line. Closed: the blade lies on the line, a 4×4px green-yellow square with an ink stroke sits beside it, and the stroke turns to ink.
- **State vocabulary (text, symbol, colour):**
  - **Waiting** (`idle`): not reached; open blade, muted.
  - **Pending** (`active`): energising or awaiting the operator (unreviewed diff, transaction running); open blade, pending brown.
  - **Closed** (`passed`): the gate passed; closed blade plus the verified fill, label in verified green.
  - **Open** (`blocked`): the gate holds until the operator acts (read-only runtime, invalid fields); open blade, warning amber.
  - **Failed** (`failed`): the stage that failed; open blade, critical red, with the server error code in mono underneath.
  - **Not confirmed** (`unknown`): the outcome was not observed; dashed blade, muted.
- **Proof:** each contact carries one 12px proof line ("3 changes", "Diff reviewed", "Taken before write", "Readiness OK") and, when relevant, a mono designator line with the error code or the invalid settings keys.
- **Rollback branch:** when the server restored the backup after a failed restart or readiness check, a dashed critical-red branch runs from Apply back to Backup (drawn from `xl` up). A note under the circuit carries the rollback glyph, the `RB` designator and the outcome ("Rollback: restored from backup" or "Rollback failed: runtime recovery unsuccessful") at every width.
- **Foot:** the pending total and restart cost, right-aligned in 12px tabular type ("3 changes · 1 needs restart"), then the primary "Review N changes" button and, while edits are pending, the ghost "Discard changes".
- **Motion:** closing or opening a contact is the one authored motion on the surface (see Motion). The blade rotates over 150ms with `cubic-bezier(0.16, 1, 0.3, 1)`, with a 60ms stagger per contact index so contacts close in sequence. Under reduced motion it is instant, with no stagger.
- **Announcement:** a polite live region speaks the frontier step and its proof ("Validate: Failed. Rejected; nothing written").

## Do's and Don'ts

### Do:
- **Do** give every colour one job from the One Job Rule, and say every state in text as well as colour and shape.
- **Do** report the outcome of a gated configuration change on the change circuit and its polite live region. **No toasts on the gated flow**: applying, holding, failing and rolling back a change never raise a toast.
- **Do** label settings with their `settings.js` key in mono, and identify things with stable designators (`REV`, `K1`–`K6`, `RB`).
- **Do** use the control-border colour (`--ds-border-strong`) for every control edge and conductor, and the hairline only for decorative dividers.
- **Do** keep to the three sizes (20, 14, 12px) and rank with weight, case and rule lines.
- **Do** derive every circuit state from a signal the product really observes. The transaction is one request: while it runs, Validate, Backup, Apply and Ready are all Pending together, and an answer without a known error code leaves them Not confirmed.
- **Do** keep fonts and assets self-hosted, and both themes at WCAG 2.2 AA.

### Don't:
- **Don't** invent a backup designator. The apply API returns no backup id, so the Backup contact's proof is "Taken before write" with no `BK-` number until the API exposes one.
- **Don't** animate per-step progress inside the server transaction. The frontend sees only its start and its end, so no live per-step progress exists to draw.
- **Don't** use red for anything but failure, rollback, destructive actions and errors.
- **Don't** add gradients, glows, glass blur or shadows deeper than `--ds-shadow-raised`.
- **Don't** wrap a settings key mid-word; break only after a dot.
- **Don't** introduce a fourth type size.
- **Don't** fade text with opacity (`text-base-content/50`) to rank it; use the three text tokens (ink, ink-secondary, ink-muted). Opacity mixes escape the token contrast check, and `pnpm run check:faded-text` rejects them below `/65`.
