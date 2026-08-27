---
spec: "CLI-050-knowledge-crystallize-cutover"
verdict: "FAIL"
reviewed_sha: "c1a3d84fe71387e80995386bfd42a84962efa887"
reviewer: "nan/deepseek-v4-flash"
date: "2026-08-27"
---

## Adversarial review

**Scope**: CLI-050-knowledge-crystallize-cutover (branch `feat/cli-050-crystallize-cutover`, HEAD `c1a3d84`)
**Sources**: `specs/CLI-050-knowledge-crystallize-cutover/{proposal,tasks,verification,features.json}`; diff `merge-base(HEAD, origin/main)..HEAD` (31 files, +352/−1451)

### Spec and task alignment
- Acceptance criteria AC1–AC5 map to tasks and to features f1–f5 with non-vacuous verification commands. AC2 names every former caller and each is touched in the diff (confirmed by inspection of the diff and by running the f2 grep).
- Out-of-scope clarity is good: the two tangential doc-staleness items are ticketed (DOCS-014 / #1271), and the vault-side skill SSOT edit is correctly sequenced as a post-merge vault commit rather than silently skipped.
- No `[AGENT-DRAFT]` / `[AGENT-SUGGESTION]` tags remain in any contract file (checked proposal/tasks/verification/features.json).
- I independently reproduced the build/vet/test claims: `go build ./...` ok, `go vet ./...` ok, `go test ./...` all `ok`, `GOOS=windows go vet ./...` clean, `golangci-lint` clean, and `tests/vault-maintenance-weekly.bats`, `tests/setup-windows.bats`, `tests/knowledge-crystallize-go-parity.bats` all pass (12/12 parity cases green). No coverage gap in the deleted-bats story: `TestIsYAMLBlockScalar`, `TestProcessProjectRefusesYAMLAndLeavesFileIntact`, and the go-parity golden cases genuinely cover the behavioral assertions of the deleted `-yaml-guard.bats` / `-golden.bats` files; the only excluded golden (`help`) is a documented, defended CLI-framework difference covered by the parity-hygiene test.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Major | REAL | reliability/invocation | `scripts/vault-maintenance-weekly.sh` now calls bare `dotf vault crystallize --all` (PATH-resolved) instead of the old absolute-path `"$SCRIPT_DIR/knowledge-crystallize.sh"`. Under cron's default minimal PATH this resolves to `dotf: command not found` and the crystallize step silently no-ops (`|| true`). The cron entry `setup-linux.sh` installs (`7 10 * * 0 …/vault-maintenance-weekly.sh`) sets no PATH, the script does not export `$HOME/.local/bin`, and this machine's crontab has no `PATH=` line — so the primary *automated* consumer of the cutover regresses to doing nothing while logging an error. | Reproduced: ran the script with `PATH="/usr/bin:/bin"` (cron-like); log shows `dotf: command not found` for the crystallize step and exit 0. Old script used an absolute path (per diff) so no PATH was needed before. Cron `PATH=` count = 0. | UNTESTED — `vault-maintenance-weekly.bats` sandboxes by prepending the shim dir to the *full interactive* PATH (`PATH="$TMP:$PATH"`), so `dotf` always resolves and the cron-minimal-PATH case is never exercised. | code + tests (export `PATH="$HOME/.local/bin:$PATH"` or resolve dotf absolutely in the script / cron line; add a named bats case asserting the crystallize step runs under a minimal PATH) |
| Minor | REAL | spec artifact (features.json f2) | The f2 verification command as recorded is not literally reproducible: as written it returns two unexcluded references to `knowledge-crystallize.sh` in `specs/HARNESS-063-spec-gate-adjacency/proposal.md` (historical text, not production callers). The AC2 *intent* holds (no production caller references remain), but the recorded self-check does not pass on the tree, so the f2 "state: passing" claim is overstated. | Ran the literal f2 grep from features.json; it flags `specs/HARNESS-063…/proposal.md:70` and `:92`. | The grep itself is the check; it fails as written (exit 1). | spec artifact (`features.json` — a contract file, so the implementer must edit; say so, do not require it in this review) |
| Minor | THEORETICAL | reliability/invocation (Windows) | `scripts/vault-maintenance-weekly.ps1` has the same class of PATH dependency: `& dotf vault crystallize --all` requires `dotf` on the scheduled-task environment's PATH. Lower bar than cron (user-context tasks usually inherit the user PATH), and there is no pwsh on this box to reproduce, but it is the same unresolved-dependency pattern with zero test coverage for the `.ps1` maintenance path. | Code read; Task Scheduler description in setup-windows.ps1; no `.ps1` maintenance test exists (pre-existing gap, not introduced here). | UNTESTED — no bats/Pester case exercises `vault-maintenance-weekly.ps1` at all. | code + tests (same PATH hardening as the `.sh`, plus a structural/execution guard if feasible) |

Spec-vs-code mismatch check: AC2/AC5 are satisfied as written (caller text repointed; coverage preserved by named Go tests). The two findings above are not literal AC violations, but the first is a functional regression in the change's own automated deployment path, which is why it cannot be waved off.

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | C | Criteria met and tests green, but the delivered weekly-automation path reproducibly fails to invoke `dotf` under its own cron environment — a substantial negative-path gap in the change's primary consumer. |
| Verification       | B | Strong, reproducible build/vet/test evidence I re-ran and confirmed; but f2's recorded command is not literally reproducible (returns 2 historical-text hits). |
| Scope              | A | Diff matches proposal exactly; tangential doc-staleness correctly ticketed out and deferred skill edit sequenced, no scope creep. |
| Reliability        | C | Bare-`dotf` dependency silently swallowed by `|| true` with no PATH hardening or test for the cron environment; Windows twin has the same unresolved dependency. |
| Maintainability    | A | Clean deletions, accurate re-framed doc comments (crystallize.go, lib.sh, ORACLE), no dead code left (dead `shell` mode and `GC_ORACLE_SH` plumbing removed). |
| Handoff-readiness  | B | proposal/tasks/verification complete with candid promotion notes and archive checklist; blocking on the real reliability finding and the f2 self-check gap. |

### Verdict
FAIL

### Recommended next steps (before archive)
- **Code (required to flip verdict):** harden `scripts/vault-maintenance-weekly.sh` (and `.ps1`) so the crystallize step resolves `dotf` independently of the calling environment's PATH — e.g. `export PATH="$HOME/.local/bin:$PATH"` at the top, or resolve the installed binary path explicitly; or write a `PATH=` line into the cron entry in `setup-linux.sh`.
- **Tests (required):** add a named bats case that runs `vault-maintenance-weekly.sh` under a cron-minimal PATH (without `~/.local/bin`) and asserts the crystallize section actually produced output — the current test masks this by preserving the interactive PATH.
- **Spec artifact (required):** amend the f2 verification command in `features.json` (implementer, not this review) so it prunes `specs/HARNESS-063…` or documents those two references as intentional historical text, making f2 literally reproducible.
- **Then re-review:** with the Major resolved and a named regression test added, this can flip to PASS WITH GAPS or PASS.
- **Archive:** `dotf spec archive` is **NOT advisable** in the current state — the reproduced Major and the stale self-check must be addressed first.
