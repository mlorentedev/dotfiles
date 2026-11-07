---
spec: "TOOL-034-manage-ts-bridge-client-routing"
verdict: "PASS"
reviewed_sha: "6b359af3477102fa53ab37334585111182832f12"
reviewer: "agy/gemini-3.8-flash-high"
date: "2026-11-07"
---

## Adversarial review

**Scope**: TOOL-034-manage-ts-bridge-client-routing
**Sources**: `specs/TOOL-034-manage-ts-bridge-client-routing/{proposal,tasks,verification,features}.json` + `git diff 45baa19ea8d3fc02f1410368ba4231a6249f192b...HEAD`

### Spec and task alignment
- `proposal.md` acceptance criteria (AC1–AC8) and explicit non-goals are clearly aligned with the delivered changes across PR 1 (#3231), PR 2 (#3236), and PR 3 (#3219).
- Checksum verification strictly precedes single-member extraction for `.zip` and `.tar.gz`/`.tgz` assets; malformed/duplicate/directory/symlink/unsupported entries fail closed without replacing destination binaries.
- Old-reader rollout safety is preserved: unknown source types remain explicit skips, and role filtering landed before any role-bound archive tool entry was published in `packages.json`.
- Role convergence deterministically owns and retires `~/.config/git/ts-bridge.gitconfig` with the `socks5h://127.0.0.1:1081` Gitea URL proxy, while foreign/unmanaged files at that path are strictly protected against clobbering.
- `dotf doctor` diagnostics verify pinned binary version, listener readiness, and destination reachability with verified TLS without printing or storing credentials.
- All implementation tasks across PR 1, PR 2, and PR 3 in `tasks.md` are completed with evidence. `tasks.md` accurately left `[ ] Lint passes` unchecked, matching the minor `gofmt` CRLF line-ending finding below.
- Contract digests in `review-request.json` match `dotf spec digest TOOL-034-manage-ts-bridge-client-routing` exactly.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Minor | REAL | code-quality | `golangci-lint` reports 3 `gofmt` issues in `cli/internal/tools/{archive,catalog,install}.go` due to CRLF (`\r\n`) line endings instead of LF (`\n`). Correspondingly, `tasks.md` task `- [ ] Lint passes` was left unchecked. | `cd cli && golangci-lint run ./internal/tools/...` flags line 1 of `archive.go`, `catalog.go`, and `install.go` with `File is not properly formatted (gofmt)`. | `golangci-lint run ./internal/tools/...` | code (run `gofmt -w` / dos2unix on the 3 files) |
| Minor | THEORETICAL | resilience | On Windows, attempting to upgrade `ts-bridge.exe` while an active instance is running will fail in `placeBinary` with an OS file-sharing violation (`ERROR_ACCESS_DENIED` on rename). | Windows PE loader holds an exclusive execution lock on active process images. Documented in `docs/runbooks/ts-bridge-client.md`. | UNTESTED | code / follow-up ticket (tracked in TOOL-035/#3244 for managed lifecycle) |
| Minor | THEORETICAL | configuration | `checkTsBridge` hardcodes `ListenerHealthURL = "http://127.0.0.1:9081/health/ready"`. Overriding `TS_HEALTH_ADDR` to another local port causes doctor to report listener not ready on 9081. | Hardcoded default in `cli/internal/bridge/config.go`. Mitigated by operational runbook instruction to maintain 9081. | `TestRouteDefaultsShareOneHostAndSocksEndpoint` | docs / follow-up ticket |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | A | All 8 acceptance criteria independently proven; comprehensive negative-path tests for malformed/corrupted archives and foreign git drop-ins. |
| Verification       | A | Machine-readable features in `features.json` pass fresh; live Windows consequence evidence recorded in `verification.md`. |
| Scope              | A | Cleanly bounded per proposal non-goals: no native Tailscale, no system proxy, no DNS changes, no unattended daemon. |
| Reliability        | A | Idempotent second runs verified (zero changes); atomic placement with fsync/chmod; strict timeouts prevent doctor hangs. |
| Maintainability    | B | Clean modular structure, all functions under 40 lines and CC <= 10; Minor CRLF formatting issue in 3 tools files prevents grade A. |
| Handoff-readiness  | A | Comprehensive ADR-058, operator runbook `docs/runbooks/ts-bridge-client.md`, and repo lessons 565 and 566 captured. |

### Verdict
PASS

### Recommended next steps
1. **Code formatting (code)**: Run `gofmt -w` on `cli/internal/tools/archive.go`, `cli/internal/tools/catalog.go`, and `cli/internal/tools/install.go` to convert Windows CRLF line endings to LF and satisfy `golangci-lint`.
2. **Verification disposition (outside contract set)**: Record the disposition of the minor formatting finding in `specs/TOOL-034-manage-ts-bridge-client-routing/verification.md`. Do not modify `proposal.md`, `tasks.md`, or `features.json` as that would invalidate contract digests.
3. **Lifecycle tracking (follow-up ticket)**: Continue tracking unattended process lifecycle and stop semantics in TOOL-035 (#3244).
4. **Archive clearance**: `dotf spec archive TOOL-034-manage-ts-bridge-client-routing` is advisable once PR 3 lands.
