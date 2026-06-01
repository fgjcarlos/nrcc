# Spec: prometheus-metrics — Expose Prometheus Metrics (Issue #142)

## Change

`prometheus-metrics`

## Domains

| Domain | Type | Spec Path |
|--------|------|-----------|
| `metrics-endpoint` | New | `specs/metrics-endpoint/spec.md` |

## Summary

A single new domain covers all observable behaviour introduced by this change:
the `/metrics` scrape endpoint and the eight instrumentation points wired into
existing NRCC operations (login, backup, restore, Node-RED process state,
update, library operations).

Key constraints carried into the spec:
- No dynamic label values (zero cardinality risk).
- No secret material in metric output.
- Custom prometheus.Registry mandatory for test isolation.
- Endpoint is public, registered before the SPA fallback.

## Requirements Count

| Domain | Added | Modified | Removed | Scenarios |
|--------|-------|----------|---------|-----------|
| metrics-endpoint | 9 | 0 | 0 | 19 |
| **Total** | **9** | **0** | **0** | **19** |
