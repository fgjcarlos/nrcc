# P3 CI repair and final acceptance

- Branch: `fix/nrcc-tailwind-alignment`; PR #866 stays draft, no merge.
- Scope: identical ANSI regex raw literal in `internal/service/version.go` only.
- Budget: 900 lock / 450 other / 1300 total, including all existing tracking.
- Preserve older trackers, `.codegraph/`, security floors, schema and production.
- Use new owned snapshots, Corepack pnpm 11.12.0, owned stores and Go 1.26.0.
## Tasks
- [ ] C1: RED to GREEN; lint 2.12.2 has 0 issues; 11 version cases pass (Go 1.26.0).
  Evidence: `/tmp/nrcc-c1-red-sVG7Ql/green-snapshot`; C1 commit/push authorized.
- [ ] C2: Independently verify frozen graph, CSS, suite, parser, build and Go.
- [ ] C3: Verify corrected default image, live UI and three persisted reload windows.
## Delivery
- Only C1 commit/push is authorized; final P3 acceptance and merge remain pending.
