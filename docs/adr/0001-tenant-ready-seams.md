# ADR 0001 — Tenant-ready seams for stores, JWT claims, and process management

## Status

Proposed

## Context

NRCC is currently a mono-tenant, single-instance control center. Most runtime services are built around one `DATA_DIR`, one JSON store set, one local Node-RED process, and one JWT authority:

- `internal/store/json_store.go` persists typed JSON documents at concrete file paths.
- `internal/service/auth.go` signs access tokens with user identity and role claims only.
- `internal/service/process.go` owns one `ProcessManager` with one `dataDir`, one log buffer, and one Node-RED lifecycle.
- `main.go` wires services from the process-wide configuration and passes a single `dataDir` through the app.

That is the right default for Raspberry Pi, NAS, and single-node self-hosted deployments. However, future multi-tenant or SaaS-style work would be risky if tenant boundaries are added late by branching inside every service.

## Decision

Keep NRCC mono-tenant by default, but add future seams around a `TenantContext` concept before introducing multi-tenant behavior.

The initial seam should be intentionally small:

```go
type TenantID string

const DefaultTenantID TenantID = "default"

type TenantContext struct {
    ID TenantID
}
```

Recommended boundaries:

1. **Store boundary**
   - Keep existing JSON store APIs for current code.
   - Add a factory or resolver that maps `TenantID` to store paths.
   - Preserve the current layout as the compatibility path for `default`.
   - If tenant persistence is enabled later, use `data/tenants/<tenant-id>/...` and migrate only through explicit upgrade code.

2. **Service boundary**
   - Avoid adding global tenant branching inside services.
   - Prefer constructors that can be created per tenant or receive a resolved tenant-specific dependency set.
   - Keep single-tenant constructors as wrappers around `DefaultTenantID`.

3. **JWT boundary**
   - Keep current claims unchanged while NRCC is mono-tenant.
   - When tenant-aware APIs exist, add a `tenantId` claim and validate that every request tenant matches the token tenant.
   - Treat missing `tenantId` as `default` only during a documented compatibility window.

4. **Process boundary**
   - Do not make the existing `ProcessManager` tenant-aware internally.
   - Introduce a manager registry later, for example `map[TenantID]map[InstanceID]*ProcessManager`, so process isolation stays explicit.
   - Each managed process must have separate data directory, env store, log buffer, and lifecycle lock.

## Mono-tenant compatibility model

Current deployments must remain simple:

- `DATA_DIR=./data` continues to work.
- Existing files such as `users.json`, `config.json`, backups, flows, logs, and settings remain valid.
- The default tenant is implicit and should not be visible in the UI until a real multi-tenant feature is introduced.
- No existing route should require a tenant parameter.

If a future migration introduces `data/tenants/default`, it should either:

- leave existing files in place and resolve `default` to the legacy root; or
- move files through a reversible migration with a backup and clear release notes.

## First safe implementation slice

A first PR should avoid behavior changes and introduce only the seam:

- Add `internal/model/tenant.go` with `TenantID`, `DefaultTenantID`, and `TenantContext`.
- Add a `TenantPathResolver` or `StoreResolver` with tests proving `default` resolves to the current paths.
- Keep all current constructors and routes unchanged.
- Add unit tests for path resolution and invalid tenant IDs.

This gives future work a stable place to attach without forcing multi-tenancy into the product now.

## Security risks to address before real multi-tenancy

- **Storage isolation:** tenant IDs must be validated and must never allow path traversal.
- **JWT confusion:** requests must not be able to override token tenant identity with URL/body parameters.
- **Process isolation:** one tenant's Node-RED process, env vars, logs, and backups must not be shared with another tenant.
- **Secrets:** per-tenant encryption keys or key derivation need a design before tenant secrets are stored.
- **Admin scope:** define whether `admin` means tenant admin or platform admin.
- **Audit trails:** tenant-sensitive actions should include actor, tenant, target instance, and result.

## Consequences

Positive:

- Future tenant work has clear seams.
- Current mono-tenant deployment remains the optimized path.
- Store, auth, and process boundaries can evolve independently.

Trade-offs:

- Adds a small amount of architecture vocabulary before the product exposes tenants.
- Requires discipline to avoid half-implemented tenant branching in UI or handlers.

## Related issue

Resolves #149.
