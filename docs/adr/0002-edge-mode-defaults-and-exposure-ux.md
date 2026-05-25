# ADR 0002 — Edge-mode defaults and exposure UX

## Status

Proposed

## Context

NRCC is aimed at self-hosted and edge-like deployments: Raspberry Pi, NAS, home labs, small industrial boxes, and local automation nodes. These machines often have limited CPU, memory, disk, and operator attention.

Current relevant areas:

- `README.md` documents self-hosted installation and the Portless integration.
- `internal/service/host.go` already detects local tools such as Node.js, npm, Node-RED, Docker, Docker Compose, and Portless.
- `internal/service/config.go` persists Node-RED configuration in the selected data directory.
- `frontend/src/features/settings/` is the natural place to surface deployment and exposure controls.

The project should support edge-friendly defaults without surprising non-edge deployments or making public exposure too easy to misconfigure.

## Decision

Introduce an explicit edge-mode design before implementation. Edge mode should be opt-in first, conservative, and visible to operators.

Primary activation:

```bash
EDGE_MODE=true
```

Future auto-detection may suggest edge mode, but should not silently enable it. Candidate signals include ARM architecture, low memory, small root disk, Raspberry Pi model files, or NAS/container labels.

## Edge-mode defaults

When `EDGE_MODE=true`, NRCC should prefer resource-safe behavior:

1. **Logs and metrics**
   - Lower in-memory log and metric retention.
   - Prefer bounded ring buffers over unbounded history.
   - Surface retention limits in the UI.

2. **Backups**
   - Prefer compressed backups.
   - Keep a conservative default retention count.
   - Warn before restore operations that can overwrite local Node-RED state.

3. **Update checks**
   - Avoid noisy or frequent network checks.
   - Make update checks manual or low-frequency by default.
   - Clearly distinguish check-only actions from install/update actions.

4. **AI/network features**
   - Keep AI features disabled unless a provider is explicitly configured.
   - Redact secrets before any external request.
   - Provide an offline/no-key state that does not look broken.

5. **Process operations**
   - Keep start/stop/restart explicit.
   - Show warnings for externally managed Node-RED processes before taking lifecycle actions.

## Exposure UX: Portless, Tailscale, and public URLs

NRCC may display guided exposure status for local HTTPS tunnels or tailnet URLs, but exposure must be treated as a security-sensitive operation.

Recommended UX:

- Show current exposure state as **Local only**, **Tailnet/private**, or **Public**.
- Show detected Portless aliases from `HostService.ReadPortlessAliases()` when available.
- Add future Tailscale/Funnel detection as read-only status first.
- Require admin role for any exposure configuration.
- Require strong authentication before showing public exposure actions.
- Display a warning that exposing Node-RED or NRCC can expose flows, credentials, environment variables, logs, and backup operations.

Public exposure prerequisites:

- `JWT_SECRET` must be set to a production value.
- Admin password must exist and should pass minimum strength checks.
- Security headers and CSRF/session protections must remain enabled.
- MFA or an equivalent second factor should be designed before one-click public exposure is offered.
- Node-RED admin auth should be enabled when exposing Node-RED itself.

## Non-goals

- Do not silently change runtime behavior for existing users.
- Do not auto-open public tunnels.
- Do not bundle SaaS tenant management into edge mode.
- Do not make AI/network features mandatory for edge deployments.
- Do not hide advanced settings; edge mode should set safer defaults, not remove control.

## First implementation slice

A first PR should be small and testable:

- Add an `EDGE_MODE` configuration flag with default `false`.
- Add `edgeMode` to the backend status/config response consumed by Settings or System UI.
- Add a read-only UI badge: `Edge mode: enabled/disabled`.
- Document the flag in `.env.example` and `README.md`.
- Add tests proving default behavior is unchanged when `EDGE_MODE` is unset.

A later PR can tune retention defaults and expose Portless/Tailscale status cards.

## Acceptance checks for future implementation

- Existing non-edge deployments behave the same by default.
- Edge mode can be enabled explicitly through environment configuration.
- UI states clearly separate local, private/tailnet, and public exposure.
- Public exposure controls are admin-only and blocked without strong auth prerequisites.
- Resource-sensitive defaults are documented and covered by tests.

## Related issue

Resolves #148.
