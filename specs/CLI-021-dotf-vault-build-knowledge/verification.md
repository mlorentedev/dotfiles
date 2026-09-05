---
id: "CLI-021-dotf-vault-build-knowledge"
type: spec
status: implementing
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
16 golden cases (`tests/vault-health-go-parity.bats`). A golden alone cannot exercise one exit path
— an unresolved backlog-scripts location, which the port fails loudly on rather than silently
skipping — because a shell always knows its own `$SCRIPT_DIR`; that path is covered separately by
`health_test.go`, not by the golden/parity suite. Not yet cut over: the shell twin, its callers, and
`session_start.go`'s own separate `vault-health.sh` exec (SessionStart brief) are untouched, per this
increment's build-beside scope.

**Increment 3 — `dotf vault maintain`:** landed — `cli/internal/vault/maintain.go` +
`cli/internal/cmd/vault_maintain.go`. **Not golden-characterized, deliberately** (rationale at the
head of `tasks.md` §4): the twin is a 52-line wrapper around two subcommands whose byte-parity is
already proven by increments 1 and 2, so a third fixture scheme would re-measure the same thing.
The wrapper's own behaviours are covered by 22 table tests
(`cli/internal/vault/maintain_test.go`) and 4 end-to-end bats cases against the built binary
(`tests/vault-maintenance-weekly.bats`, 17 total in the file alongside the 13 shell cases).

Composition is **in-process**, which removes the twin's documented cron failure mode rather than
reproducing it, and the fourth bats case asserts that structurally by running under
`PATH=/usr/bin:/bin`.

An end-to-end run against an empty home and an empty vault, with no `obsidian` on `PATH`, produces
both sections in order, the GNU-`date`-shaped stamps, `Vault health: FAILED — one or more checks`
on stdout, and **exit 0** — the exit-code decision working, not asserted.

## Test status

- Golden characterization (#672 / CLI-031): captured for crystallize and health. **Not** for
  maintain, by the reasoning above — recorded as a decision, not an omission.
- Go/shell parity suites: `tests/knowledge-crystallize-go-parity.bats` (13/13, `help` excluded —
  documented), `tests/vault-health-go-parity.bats` (16/16).
- Table-driven units: `crystallize_test.go`, `health_test.go`, `maintain_test.go`.
- **Mutation-verified for increment 3**, with the harness proving each mutation LANDED before
  believing the red (lesson 267 — a mutation that silently fails to apply reads as a passing test):
  four mutations, four kills, each by the test named for it. Detail in `tasks.md` §4.
- `GOOS=windows go vet ./...`, `go vet ./...`, `go build ./...`, and the pinned
  `golangci-lint` (v2.12.2, matching `versions.conf`) all clean.
- `test-windows` CI: none of the three increments has been run on an actual Windows box. The
  Windows log path (`%LOCALAPPDATA%`) is unit-tested from Linux via `logFileFor`, and the
  PowerShell toast in `NotifyDesktop` is **unexercised anywhere** — it is fire-and-forget by
  design, so a failure is silent by construction. Stated as a gap, not covered.

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
