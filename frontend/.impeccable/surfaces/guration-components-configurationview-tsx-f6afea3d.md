---
version: 1
slug: "guration-components-configurationview-tsx-f6afea3d"
primary_target: "src/features/configuration/components/ConfigurationView.tsx"
related_targets: ["src/features/configuration/components/ReviewChangesPanel.tsx"]
---

# Surface brief: Configuration

Scope: `/configuration` (settings.js catalog, advanced editor, review-changes flow). Visitor mode: Operate. This surface is the proof of the new NRCC visual world; every other surface inherits it.

Audience and job: the operator of their own Node-RED 5 instance changes risky settings and must see what will happen before it happens. Frequent, focused sessions; light and dark used equally.

Constraints: WCAG 2.2 AA in both themes and en/es; fully offline (self-hosted fonts and assets); standard web controls; keep density (the user rejects a result that "pierde densidad" or looks like "marketing / juguete"); everything else may change.

Memorable moment: the change circuit closing contact by contact, or opening onto the rollback branch.

Unresolved: final faces (workhorse UI sans + mono, self-hosted) and the mark redesign are decided during the build.

## Direction contract

THESIS: A settings change is a circuit that only closes when every gate passes, and the operator watches it close. It refuses the category default: form cards, a Save button, and a toast.

OWN-WORLD: IEC schematic language. Drawing-white ground (dark theme: drawing negative), hairline ink conductors, stable designators (CS-0042 change set, BK-0017 backup, settings keys as terminal labels) with cross-references to settings.js. Conductor colours do jobs: neutral blue = action and selection, green-yellow = verified, brown = pending, red only for critical and rollback. Ink navigation rail. Three type sizes; rank by weight, case, and rule.

STORY: The operator edits, sees each change register on the circuit live, reviews the redacted diff, then watches validate, backup, apply, and readiness close in order. A failure opens its contact and shows the rollback branch taken. Every state carries its proof (probe result, backup designator).

FIRST VIEWPORT: 1440 wide. Ink nav rail left; page header with instance designator and one runtime status; settings form centre; the change circuit as a right column: six contacts top to bottom (Edit, Diff, Validate, Backup, Apply, Ready), pending total and restart cost right-aligned at its foot, primary action "Review N changes" directly under it.

FORM: Wiring diagram, position 3 of 7 on the grounded list. seed:a927e97550a2032a

FINISH: unreviewed and undocumented is unfinished; this build ends with the finish review, the verdict, DESIGN.md, and every shipping raster carrying its provenance
