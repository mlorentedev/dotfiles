---
tags: [spec, verification, templates]
created: "2026-08-27"
---

# Verification - CLI-050-knowledge-crystallize-cutover

## Evidence

- [x] AC1 (scripts deleted) -> `features.json` f1; `git rm scripts/knowledge-crystallize.{sh,ps1}`
- [x] AC2 (every caller repoints) -> `features.json` f2; commits touching
      `session_start_injectors.go`, `vault-maintenance-weekly.{sh,ps1}`, `setup-linux.sh`,
      `setup-windows.ps1`, `vault.sh`, `utils.ps1`, `.claude/CLAUDE.md`, the distillation runbook,
      `ci.yml`, `adr-008`, `vault.go`, `mem.go`
- [x] AC3 (diff scope) -> `features.json` f3
- [x] AC4 (build/test/lint green) -> `features.json` f4
- [x] AC5 (no coverage gap) -> `features.json` f5

## Test status

- `cd cli && go build ./... && go vet ./... && go test ./...` -> all packages `ok`
- `GOOS=windows go vet ./...` -> clean
- `golangci-lint run` (pinned 2.12.2, matches `versions.conf`) -> 0 issues
- `shellcheck --severity=error setup-linux.sh scripts/*.sh` (CI's actual invocation, not default
  severity) -> exit 0
- `bats tests/*.bats` -> 1462 tests, 1462 ok, 0 not ok
- Manual smoke test: `dotf vault crystallize --help` against the already-installed
  `~/.local/bin/dotf` confirms `--all` exists before deleting the twin (two-tier deploy
  verification, lesson-150)
- No regressions in existing test suite: yes, after fixing two tests the deletion broke (see
  Decisions below) — both are documented, not silently patched

## Post-review fixes (round 1, reviewed sha `c1a3d84`, verdict FAIL)

- **Major/REAL**: bare `dotf vault crystallize --all` in `vault-maintenance-weekly.sh` regressed
  the weekly cron job — cron's minimal PATH excludes `~/.local/bin` (install-dotf.sh's install
  target), unlike the old absolute-path shell call, so the step would silently no-op every Sunday
  behind `|| true`. Fixed: `export PATH="$HOME/.local/bin:$PATH"` at the top of the script;
  applied the same hardening to the `.ps1` twin for symmetry. New regression test
  `"vault-maintenance-weekly.sh resolves dotf under a cron-minimal PATH"` — verified it actually
  fails without the fix (`dotf: command not found` in the log) before committing it as green.
- **Minor**: `features.json` f2's verification command wasn't literally reproducible — it flagged
  2 unexcluded hits in `specs/HARNESS-063-spec-gate-adjacency/proposal.md`, a different, unrelated
  active spec that names the old filename only as a worked example in its own design rationale.
  Patching the exclusion list turned out fragile: this same PR's own post-review fix commit added
  a new historical comment mentioning the retired filename, breaking the "fixed" version again
  before it was even pushed. Rewrote f2 to match invocation SHAPE (sourcing, relative-path exec,
  `Copy-Item`, `&`-invoke) scoped to production files, rather than any textual mention anywhere —
  AC2 claims "no production file still invokes it", not "the string never appears in prose", and
  the new command is durable against future documentation legitimately naming the retired script.
- **Minor/THEORETICAL**: no test coverage exists for `vault-maintenance-weekly.ps1` at all (not
  introduced by this PR). Applied the PATH fix there too, but did not build new Pester coverage in
  this PR — no pwsh on this box to validate it, and it's a separable unit of work. Ticketed as
  TEST-002 (#1277) instead of building it unverified.

## Decisions made during implementation

- **Two bats files broke on deletion and needed real fixes, not just deletion.**
  `tests/setup-windows.bats`'s "deploys knowledge-crystallize.ps1" test string-matched the
  filename and kept passing (false positive) because the new explanatory comment in
  `setup-windows.ps1` still names the retired script — replaced with a
  "no longer deploys ... (CLI-050)" test asserting the specific `Copy-Item` pattern is absent
  plus the file's non-existence, following the existing CLI-020/MEM-002 precedent in the same
  file. `tests/vault-maintenance-weekly.bats`'s sandbox (`_prep_sandbox`) placed a stub script
  literally named `knowledge-crystallize.sh` next to a copy of the maintenance script — since the
  script now calls bare `dotf vault crystallize --all` (PATH-resolved), the stub was rewritten as
  a `dotf` shim that branches on `$1 vault $2 crystallize`, and the log-content assertions were
  updated from the old invocation string to the new one.
- **Kept `tests/golden/crystallize/ORACLE` rather than deleting it with `capture.sh`.** It is data
  (a provenance record of which shell bytes produced the goldens), not executable code that can no
  longer run — unlike `capture.sh`, which references a deleted oracle and would error if invoked.
  Amended its header to say so explicitly.
- **Removed the now-dead `shell` mode and `GC_ORACLE_SH` plumbing from
  `tests/golden/crystallize/lib.sh`** rather than leaving it unreachable: the sole remaining
  caller (`knowledge-crystallize-go-parity.bats`) always sets `GC_IMPL_MODE=go` before calling
  `gc_run_case`, so the shell branch could never execute again. Dead code that only *looks*
  reachable is worse than no code.
- **Deferred the vault-side skill SSOT edit** (`00_meta/skills/crystallize/SKILL.md`, which
  already says "or `dotf vault crystallize`") to a direct-to-vault-master follow-up commit after
  this PR merges, per CLI-021's own sequencing note and this repo's standing vault-commit
  convention — not left unticketed, just sequenced correctly.
- **Ticketed two unrelated doc-staleness findings (DOCS-014 / #1271)** noticed while editing the
  distillation runbook instead of fixing them here, to keep this diff to the crystallize cutover.

## Promotion candidates

- [ ] Lesson for the repo's `docs/lessons/`? **yes** — "a bats fixture that stubs a sibling script
  by literal filename silently stops testing anything once the production code stops calling that
  filename; a false-pass (string still present in an unrelated comment) is worse than a red test,
  because it reads as coverage." Candidate for a lesson file if this pattern recurs (first
  occurrence here; not promoted to `docs/lessons/` yet per this repo's convention of promoting
  after a repeat, not on first sight — logged here so it's not lost).
- [ ] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? no — this is CLI-021/ADR-020's
  strangler-fig contract executing exactly as designed, not a new decision.
- [ ] New pattern candidate for `00_meta/patterns/`? no — single-project occurrence so far.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/CLI-050-knowledge-crystallize-cutover/` -> `specs/archive/CLI-050-knowledge-crystallize-cutover/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
