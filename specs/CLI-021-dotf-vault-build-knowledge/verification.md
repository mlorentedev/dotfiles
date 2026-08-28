---
id: "CLI-021-dotf-vault-build-knowledge"
type: spec
status: draft
created: "2026-08-08"
issue: "mlorentedev/dotfiles#490"
tags: [spec, verification]
template_version: "1.0"
---

# Verification — CLI-021-dotf-vault-build-knowledge

> Increments 1 and 2 landed (build-beside, cut over separately). Filled per increment as they land.

## Evidence

**Increment 1 — `dotf vault crystallize`:** landed in #882, byte-identical to the shell oracle on
the full golden corpus. Cut over separately in #1276 (CLI-050), which deleted the shell/PowerShell
twin and repointed every caller.

**Increment 2 — `dotf vault health`:** landed — `cli/internal/vault/health.go` +
`cli/internal/cmd/vault_health.go` (`dotf vault health`). Byte-identical to the shell oracle on all
16 golden cases (`tests/vault-health-go-parity.bats`), including the one exit path a golden alone
cannot exercise cleanly (an unresolved backlog-scripts location, which the port fails loudly on
rather than silently skipping — unit-tested in `health_test.go`). Not yet cut over: the shell twin,
its callers, and `session_start.go`'s own separate `vault-health.sh` exec (SessionStart brief) are
untouched, per this increment's build-beside scope.

**Increment 3 — `dotf vault maintain`:** not started

## Test status

- Golden characterization (#672 / CLI-031): captured for both crystallize and health.
- Go/shell parity suites: `tests/knowledge-crystallize-go-parity.bats` (13/13, `help` excluded —
  documented), `tests/vault-health-go-parity.bats` (16/16).
- Table-driven units: `cli/internal/vault/crystallize_test.go`, `cli/internal/vault/health_test.go`.
- `test-windows` CI: unchanged by this spec so far — `GOOS=windows go vet ./...` passes for both
  increments' code, but neither has been run on an actual Windows box.

## The proof that matters most here

Because this is a **build-beside** PR, the load-bearing check is negative: `git diff --stat` must
touch only `cli/` and `specs/`. Any hit in `scripts/`, `setup-*.{sh,ps1}`, or the vault means the
cutover leaked into the build PR, which is precisely the risk AUDIT-007 split PR5 from PR7 to
avoid.

## Promotion candidates

- [ ] Lesson? Candidate from BUG-060, likely promoted at *that* spec's archive rather than here:
      *a maintenance script that has never run is not "safe", it is untested — its first execution
      is a deployment.*
- [ ] ADR-worthy? No. `docs/adr/audit-007-cli-convergence-state.md` already owns the decision; this
      is its PR5.
- [ ] Pattern candidate? Possibly — "characterization-test the oracle before you replace it" as a
      twin-port discipline, if CLI-022..028 repeat the shape. Decide after the third port, not the
      first.

## Archive checklist

- [ ] All three increments landed, or the remainder explicitly descoped with a reason
- [ ] `proposal.md` frontmatter -> `status: archived`
- [ ] Folder moved to `specs/archive/CLI-021-dotf-vault-build-knowledge/`
- [ ] Issue #490 closed with PR links
- [ ] Flip checklist handed to CLI-023 (PR7) — not silently dropped
