# ADR 0001: Restart-count semantics

Status: Accepted
Date: 2026-06-01
Related: #244

## Context

NRCC exposed the name `restartCount` with two different meanings:

- `GET /api/health.restartCount` used the durable cumulative auto-restart counter.
- `GET /api/runtime/history.status.restartCount` used the in-session backoff/give-up counter.
- `GET /metrics` exported `nrcc_nodered_restarts_total` from `RuntimeStatus.RestartCount`, so its semantics followed whichever meaning `RuntimeStatus` carried.

This made API consumers likely to compare fields with the same name but different reset behavior.

## Decision

Use `restartCount` consistently for the durable cumulative auto-restart count for this NRCC installation.

Expose the separate backoff/give-up counter as `consecutiveFailures` on `RuntimeStatus`. This counter resets on user-initiated starts and remains useful for supervision/backoff visibility without overloading `restartCount`.

`nrcc_nodered_restarts_total` exports the durable cumulative count via `RuntimeStatus.RestartCount`.

## Compatibility

- `restartCount` remains present and numeric, but `/api/runtime/history.status.restartCount` now changes from the backoff counter to the durable cumulative counter.
- `consecutiveFailures` is additive and optional for existing consumers.
- Current frontend code does not render `/api/runtime/history.status.restartCount`, so the user-visible impact is expected to be low.

## Non-goals

- Rename the existing Prometheus metric.
- Change the internal `pm.restartCount` field name in this slice; it remains the private backoff counter.
- Change restart-history event semantics.
