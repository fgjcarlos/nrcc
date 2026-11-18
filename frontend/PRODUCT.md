# Product

<!-- impeccable:product-schema 1 -->

## Platform

web

## Users

Primary: the technical operator who maintains their own Node-RED 5 instance day to day. They come back often to change `settings.js` configuration, check runtime health, manage environment variables, and take or restore backups. They know Node-RED, but they should not have to hand-edit JSON or shell into the container to do any of this.

Secondary (served, not prioritized): integrators who install NRCC for someone else, and small teams sharing one instance. Roles are `user` and `admin`; user management, updates, and libraries are admin-only.

## Product Purpose

NRCC (Node-RED 5 Control Center) is a visual control plane for one Node-RED 5.x instance at a time. It discovers the deployment, imports `settings.js` losslessly, guides edits with a redacted diff, validates, snapshots a backup, applies the change transactionally, and rolls back when the readiness check fails.

Success means the operator changes a risky setting with confidence: they see what will happen before it happens, and a bad change can always be undone.

## Positioning

NRCC explains what it is about to do before it does it. Every configuration change goes through the same gated sequence (discover → edit → review redacted diff → validate → backup → apply → readiness check / rollback). It configures one instance; it does not orchestrate fleets, manage sibling containers, or duplicate the Node-RED flow editor.

## Operating Context

- Deployed as a single stack: one NRCC container, one Node-RED container, one persistent volume. Multiple instances mean multiple stacks on distinct ports.
- Runs behind a firewall on the operator's own infrastructure, often without internet access.
- Beta / hardening phase: feature-complete for the Node-RED 5 scope, not 1.0.
- Node-RED compatibility: `>=5.0 <6.0` full editing; `4.x` detection, read-only inspection, and migration guidance; future majors read-only.

## Capabilities and Constraints

- Surfaces: setup and login, Overview (runtime health, metrics, restart), Configuration (typed settings catalog, advanced escape hatches, review-changes panel, topology), Security (NRCC UI/API, Node-RED `adminAuth`, Dashboard auth as separate surfaces), Environment variables, Recovery (backups, schedules, retention), Bootstrap (host/environment info), Profile, and admin-only Users, Updates, and Libraries. A command palette is available.
- Secrets are masked before any diff is shown; nothing the operator types in plain text reaches logs.
- Must work fully offline: fonts and all assets are served locally, no CDN or third-party requests at runtime.
- Interface languages: English and Spanish, with catalog parity enforced. User-visible copy lives in i18n catalogs.
- Light and dark themes are both supported.
- Stack: React + TypeScript + Vite + Tailwind, served from the Go binary.

## Brand Commitments

- The product name **NRCC** (Node-RED 5 Control Center) is fixed.
- Everything else in the visual identity, including the current mark (`docs/assets/nrcc-mark.svg`), may be redesigned.

## Evidence on Hand

- Product and workflow description: `README.md`, `docs/control-plane.md`, `docs/handbook/`.
- Current mark and favicon: `docs/assets/nrcc-mark.svg`, `frontend/public/favicon.svg`.
- No customer testimonials, case studies, usage metrics, or certifications exist. Do not fabricate them.

## Product Principles

1. Show before doing: every change is previewed, explained, and reversible before it touches the instance.
2. Safety over speed: risky actions are gated, confirmed, and recoverable; the safe path is the easy path.
3. One instance, done well: depth on a single Node-RED deployment, never fleet management.
4. Respect the operator's expertise: precise, technical, honest language; no hand-holding and no hiding of state.
5. Self-contained: everything works on an isolated network.

## Accessibility & Inclusion

WCAG 2.2 AA is required in both themes and both languages (enforced today by token contrast checks and axe in e2e). Full keyboard operation and reduced-motion support are required.
