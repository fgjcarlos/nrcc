# Issue 1027: integration review

The tracker is draft-only while its child PRs are reviewed. Merge the children in order into the integration branch; only then consider promoting the tracker to `main`. Keep issue #1027 open for the unresolved findings.

## Specs

S1. Abre PR
S2. PRs encadenados con rama integradora

## Tasks

- T1 (S1–S2): Verify publishing gates and prepare focused branches. Route: inline.
- T2 (S1–S2): Open the draft tracker and immediate-parent child PRs. Route: inline.
- T3 (S1–S2): Verify the published bases, scopes, and CI status. Route: inline.

## Review order

1. MFA enrollment and refresh cookies.
2. Flow history deduplication at startup.
3. Managed runtime readiness and shutdown.
4. Login, script policy, and npm selector hardening.
5. Backup summaries and the follow-up register.

## Log

- L1: “Abre PR”
- L2: “PRs encadenados con rama integradora”
- L3: Preserve the original work-unit commits; the tracker adds no production code. This sequence does not authorize merging.
