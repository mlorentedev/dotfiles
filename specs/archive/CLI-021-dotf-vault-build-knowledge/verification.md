---
id: "CLI-021-dotf-vault-build-knowledge"
type: spec
status: verifying
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
  - **Sharpened at review (finding 4):** the toast's script is assembled with Go `%q`. Today's
    fixed title and body contain nothing `%q` would escape, so it works; a title or body carrying
    `"` or `\` would be escaped Go-style and PowerShell would not read it identically. Theoretical
    now, a real drift risk the moment those strings stop being literals. Fix is either a Windows
    case or PowerShell-native escaping — neither in this spec's scope, both cheap at CLI-023.

## Divergences of increment 3, recorded at review

`tasks.md` §4 lists the three known before implementation (timestamp shape, the `Out-File` BOM, the
issue regex's substring over-count). The reviewer found a fourth by running both implementations
against an identical empty HOME and vault and diffing the logs:

- **The Go log carries no ANSI.** The shell's log has `\e[0;34m[1/7]\e[0m`, `\e[1;33mSKIP\e[0m`,
  `\e[0;31m[ERROR]\e[0m` — colour from `utils.sh` — and the Go log has none. Content is otherwise
  **byte-identical** after stripping ANSI and the timestamps.
- This is the deliberate no-ANSI decision already recorded for increment 1, but increment 3 is the
  one place it is **not normalised away**: crystallize and health compare through goldens that strip
  ANSI, and maintain has no golden. So it is the first place a human diffing the two logs meets it.
  Recorded here rather than left discoverable only by running both.
- It is not a defect for the cutover: the log is read by humans and by `grep -ciE`, and neither is
  affected. It is written down so the CLI-023 flip is not surprised by a diff it cannot explain.

## The proof that matters most here

Because this is a **build-beside** PR, the load-bearing check is negative: **nothing under
`scripts/`, `setup-*.{sh,ps1}`, or the vault.** A hit there means the cutover leaked into the build
PR, which is precisely the risk AUDIT-007 split PR5 from PR7 to avoid.

**Corrected at review (finding 2).** This sentence read *"`git diff --stat` must touch only `cli/`
and `specs/`"*, which every increment broke honestly — a twin port needs its characterization tests,
and increment 3's diff touches `tests/vault-maintenance-weekly.bats`. Worse, `proposal.md` had
already been corrected in the same PR with the claim that it was being aligned *"to what
`verification.md` has actually been enforcing"* — and `verification.md` said no such thing. The
assertion was made about a document that was not re-read. The reviewer found it by comparing the two
artifacts against the actual diff, which is exactly the check the claim should have survived before
being written.

## Acceptance criteria, dispositioned at close-out

**`proposal.md`'s boxes stay unticked, and that is the gate working rather than an oversight.**
They were ticked here first; `dotf spec archive` refused, because a review's staleness check watches
`proposal.md` and `tasks.md` and its own comment names the bypass exactly: *"get a passing review,
rewrite the acceptance criteria in the working tree, archive."* That is what ticking them after a
PASS is, however innocent the intent. The reviewer's step 2 offered this route explicitly (*"or
state in `verification.md` that they are ticked at close-out"*), and its launch instructions had
already said not to touch the contract files. So the dispositions live here, against the independent
review's evidence (`review.md`, `nan/deepseek-v4-flash`, `52035f1`) rather than self-asserted.

| # | Criterion | Disposition |
|---|---|---|
| 1 | The three subcommands exist with `--help`, `crystallize` takes `--all` + a positional dir | **Met.** Reviewer ran all three `--help`. |
| 2 | Golden characterization, byte-identical to the shell | **Met for crystallize and health** (14/14, 17/17 parity). Maintain deliberately excluded per `tasks.md` §4; the reviewer accepted it as a recorded decision, not an omission. |
| 3 | HARNESS-029 holds, proven against a naive append | **Met.** Re-verified independently at review by mutation — 3 cases plus a parity case go red. |
| 4 | Table-driven units for encode/decode and section insertion | **Met.** |
| 5 | No twin deleted, no caller repointed | **Met by every diff of this spec.** Nothing under `scripts/`, `setup-*.{sh,ps1}` or the vault in any increment. |
| 6 | The shell scripts remain canonical | **Met for health and maintain. Not for crystallize** — see below. |

### Reviewer's Q1 — the crystallize cutover ran ahead of CLI-023

**Answered: deliberate, and it did not jump the queue — it was moved out of it.**
`scripts/knowledge-crystallize.{sh,ps1}` are gone and every caller repointed, done by **CLI-050
(#1269, PR #1276)**, a sibling ticket, not by CLI-023. AUDIT-007 row 5 sequences the *build* of all
three here and row 7 the *cutover* of the cluster; neither requires the three cutovers to move
together. Crystallize's was separable because its twin had no caller outside the setup scripts.

The other two are not separable the same way, which is why they are still standing:

- **health** — `cli/internal/mem/session_start.go:135` still execs `vault-health.sh` for the
  SessionStart brief, so removing the script is a Go change, not a shell one.
- **maintain** — the crontab (`setup-linux.sh:1642`) and Task Scheduler (`setup-windows.ps1:2240`)
  entries name the scripts by path.

So criterion 6 reads as written for the two increments still awaiting CLI-023, and is
superseded-by-ticket for the third. Recorded rather than left looking like a silent violation.

### Close-out, from `tasks.md` §5

- **Diff scope proven** — nothing under `scripts/`, `setup-*.{sh,ps1}` or the vault, all three
  increments. (`tasks.md` words this as "only `cli/` and `specs/`", which every increment broke
  honestly: a twin port needs its tests. See §"the proof that matters most here".)
- **PRs merged, CI green including `test-windows`** — #882 (increment 1), #1489 (increment 3,
  `52035f1`).

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
