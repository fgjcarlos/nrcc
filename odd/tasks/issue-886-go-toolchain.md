# Issue #886 — Patched Go toolchain and ordered integration
Baseline: main 58d922ab; existing child 8f9c60f. Authorized: fix, commits, non-force pushes, PR publication and ordered merges only after green checks. RDD off.
Delivery: focused #886 security slice first, then update the existing #893 -> #892 tracker chain. Engram mirror unavailable (no callable memory tools).

## Specs
- S1 User authorization: "Ok resuelve eso en ese orden".
- S2 Accepted order: "Resolver #886 y actualizar la cadena con esa corrección."; "Confirmar CI verde."; "Mergear **#893 hacia `feat/issue-875-tracker`**, sin borrar todavía esa rama base."; "Verificar el tracker ya integrado, pasar **#892** de draft a listo y mergearlo a `main`."
- S3 Issue #886: "Update pinned Go versions to a supported patched version and synchronize the documented contract."
- S4 Issue #886: "Re-run govulncheck on the exact candidate commit and audit remaining imported/module advisories."
- S5 Issue #886: "Inspect/scan the actual binary and image toolchain; Docker's different builder version must not be assumed equivalent."
- S6 Issue #886: "Record security results and relevant reachability/exposure decisions before release."

## Tasks
- T1 [S3,S5] inline, implemented and focused checks passed: registry multiarch digest verified, contract RED observed on all six surfaces; GREEN plus existing workflow tests passed after synchronizing pins/docs. Commit: pending artifact checks.
- T2 [S3-S6] inline checks + delegated gentle-ai-verify (risk items 2/5), pending: build/scan exact candidate and delivered artifacts; publish focused security PR and integrate only after green required checks. Commit: pending.
- T3 [S1-S2] inline, pending: update tracker/child with verified security fix without force; check exact child head and squash #893 into tracker. Integration commit: pending.
- T4 [S1-S2] inline, pending: check resulting tracker head, mark #892 ready, squash into main and confirm final state. Integration commit: pending.

## Log
- L1 User: "Ok resuelve eso en ese orden".
- L2 Read-only map mvarw4uq-1-6euv: five CI Go selections across four workflows use1.26.8; support matrix matches. go.mod1.26.0 is the language/minimum contract, not compiler patch pin. Docker executable builder is1.27-alpine@sha256:4c9fe60190a2a3350ddc51de80d0224b8a6698d12bdfc999fee45ea9d6c46dbc, while its comment says1.26. Actual digest content must be inspected, not inferred. Registry/provenance job-17 pending.
- L3 Route: narrow dependent write inline (no independent writer units); independent verifier mandatory after own checks for security/build delivery changes. Local explorer fallback used because original strict extension-tool allowlist previously failed; no repeat of the failed agent. Existing #875 child is318 lines; keep #886 independently reviewable rather than inflate it past the selected review budget. Separate security PR targets main first, preserving #893 -> tracker -> main order for the feature.
- L5 Explicit follow-up approval: user selected "Sí, aprobar #886 e integrar primero su PR de seguridad", authorizing status:approved and security-PR merge with green checks before #893/#892. Label applied and read-back confirmed; ADMIN and squash merge available. Remote auto-delete enabled; never delete the tracker branch before its main PR is merged.
- L6 Registry job-17 found multiarch Go1.26.9-alpine index sha256:3082400e369fa24d5fc60bca20edab3f6d604e0c5a690ec66b295eff4dd87ade (amd64, arm64 and other platform manifests). Job exited1 only because the separate old-image cache lookup found no image, not because registry resolution failed. No provenance claim for old digest. Fresh main fetch succeeded, security branch created from origin/main.
- L7 RED: TestPatchedGoToolchainContract failed on CI1.26.8, Docker1.27/tag+digest and documented1.26.8. GREEN: go test ./tools/langscan/workflow -count=1 exited0 (0.007s), including existing language-workflow regressions; git diff --check passed. Module minimum1.26.0 and all unrelated Node/runtime/tool versions untouched. Artifact/full-suite/security checks pending.
- L4 Initial RED evidence already observed remotely on both #892 and #893: govulncheck fails for ten standard-library advisories on Go1.26.8; issue886 records fixes in1.26.9. Local contract RED and actual patched-registry availability still pending. Pre-existing release cross-platform/Linux-only documentation discrepancy is out of scope; do not silently alter release targets.
