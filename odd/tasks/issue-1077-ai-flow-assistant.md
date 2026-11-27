# Issue #1077 — AI providers and flow assistant

Source: https://github.com/fgjcarlos/nrcc/issues/1077
Risk: item 2 (provider credentials, redaction, and outbound endpoint policy).

## Specs

S1. OpenAI-compatible, Anthropic and Ollama native providers can be configured, tested and used. Each has `httptest`-based tests for request shape, response parsing and error mapping.
S2. `analyze/flow` returns parsed text and an optional validated candidate flow, never a raw provider envelope.
S3. Provider 5xx and timeouts are not reported as 400.
S4. Plain `http` endpoints are accepted only for loopback and private addresses with an explicit opt-in.
S5. The flows UI exposes the assistant actions, review-first, with no auto-apply.
S6. The OpenAPI contract and the router agree for every `/api/ai/*` route.
S7. Map errors correctly: provider 5xx → 502, timeout → 504, auth failure → a clear "provider rejected credentials" error. Add a configurable timeout.
S8. Fix the `anthropic` mismatch in env-contract docs. Consider `NRCC_AI_API_KEY_FILE` (listed as TODO in env-contract).
S9. Disabled state with a link to settings while the provider is not ready (per #727).
S10. i18n en/es for the settings card and assistant.

## Tasks

- [x] T1 — S1–S10 — Implement in parallel: backend and contract (S1–S4, S6–S8); frontend (S5, S9–S10 and provider settings for S1/S4) — focused Go tests and 31 Vitest tests passed; commits pending.
- [x] T2 — S1–S10 — Check the integrated branch and obtain independent security verification — done.

## Evidence

- Commits: 44e3e3d (provider adapters and analysis), a76aacf (config, endpoint policy, timeout, handler), 2d8c0f6 (contract, env docs, generated schema), 9d9bd65 (Flows assistant and settings).
- Go: `go test ./...` green except pre-existing `TestSaveConfig_WritesExplicitRuntimeSettingsAndRestarts` (`internal/handler/config_test.go:654`), reproduced unchanged on `origin/main` in a scratch worktree; focused AI suites green in service and handler.
- Frontend: Vitest 98 files / 789 tests green; `npx tsc --noEmit` clean; Playwright suite green (60 passed, 3 skipped, 1 pre-existing flaky i18n case) plus `e2e/ai-flow-assistant.spec.ts` passing on chromium-dark.
- Contract: `npx @redocly/cli lint docs/openapi.yaml` valid (7 pre-existing style warnings).
- Independent verifier: met on S1–S10, no blocker; its three suggested probes (no redirect follow, preview body equality, malformed upstream) are covered in `internal/service/ai_provider_test.go` and `internal/service/ai_test.go`.
- Slice boundary: 4 commits, ~1995 authored lines; largest slices 632 and 451 lines. Publication shape (single PR vs chained) is the maintainer's decision.

## Log

L1. Reuelve la issue #1077

L2. Approved issue body (verbatim):

### Pre-flight checks

- [x] I have searched existing issues and this is not a duplicate.
- [x] I understand this issue needs `status:approved` from a maintainer before any PR can be opened.
- [x] I am proposing a feature request (not a bug report). Bug reports use the Bug Report template.

### Problem description

The AI provider foundation is in place: encrypted persisted config (#733), a connection test and capability status (#725), and redaction before requests. The feature is still not usable end to end:

- Only `offline` and `openai` are accepted (`internal/service/ai_config.go`, `isSupportedAIProvider`). `docs/configuration/env-contract.md` lists `anthropic` as an example id, but the code rejects it.
- `POST /api/ai/analyze/flow` does not parse provider responses. `resp.Summary` is the raw response body (`internal/service/ai.go`), so the user would get the provider's JSON envelope as text. Reads are silently truncated at 1 MiB.
- Error mapping in `internal/handler/ai.go` turns upstream non-2xx responses into `400 AI_REQUEST_ERROR`, and the timeout check never matches a client timeout. Provider outages look like client mistakes.
- No frontend surface calls `analyze/flow`. The flow AI actions from #145/#727 are no longer routed, so configuring a provider has no visible effect.
- `/api/ai/analyze/patterns` and `/api/ai/patterns/{id}/*` are documented in `docs/openapi.yaml` as `x-status: stub`, but they are not routed (404, not 501).
- Endpoints must be `https`, so a local Ollama on the same host or LAN cannot be used.
- `AIProviderSettings.tsx` hardcodes English labels, which bypasses i18n.

### Proposed solution

**Providers** (adapter per provider: request builder, response parser, probe):
1. **OpenAI-compatible**: Chat Completions with a configurable base URL, covering OpenAI, OpenRouter, LM Studio, vLLM, and Ollama's `/v1` API.
2. **Anthropic**: native Messages API (`x-api-key`, `anthropic-version`, `system` + `messages`).
3. **Ollama native**: `/api/chat`, no API key. Allow `http://` only for loopback and private-network endpoints, behind an explicit opt-in, and keep `https` mandatory otherwise.

**Backend**
- Parse each provider response into `Summary` and an optional `CandidateFlow`, and validate candidate flow JSON before returning it.
- Map errors correctly: provider 5xx → 502, timeout → 504, auth failure → a clear "provider rejected credentials" error. Add a configurable timeout.
- Either implement or remove the `patterns` stubs from OpenAPI so the contract matches the router.
- Fix the `anthropic` mismatch in env-contract docs. Consider `NRCC_AI_API_KEY_FILE` (listed as TODO in env-contract).

**Frontend: flow assistant**
- From the flows view: explain, suggest improvements, audit (obsolete or insecure nodes), and generate candidate flow JSON.
- Review-first: show the redacted payload that will be sent and the result. Generated flows are never auto-applied and need explicit import (per #145).
- Disabled state with a link to settings while the provider is not ready (per #727).
- i18n en/es for the settings card and assistant.

### Affected area

Backend (Go services / API)

### Alternatives considered

- OpenAI-compatible only (Anthropic via a proxy such as LiteLLM): fewer adapters, but adds a moving part and loses native features.
- Keep the AI feature hidden: the config UI already exists, so an inert feature is confusing.

### Additional context

- Related: #145, #725, #727, #733, ADR 0002 (AI off unless explicitly configured; redact before external requests).
- Acceptance criteria:
  - [ ] OpenAI-compatible, Anthropic and Ollama native providers can be configured, tested and used. Each has `httptest`-based tests for request shape, response parsing and error mapping.
  - [ ] `analyze/flow` returns parsed text and an optional validated candidate flow, never a raw provider envelope.
  - [ ] Provider 5xx and timeouts are not reported as 400.
  - [ ] Plain `http` endpoints are accepted only for loopback and private addresses with an explicit opt-in.
  - [ ] The flows UI exposes the assistant actions, review-first, with no auto-apply.
  - [ ] The OpenAPI contract and the router agree for every `/api/ai/*` route.
