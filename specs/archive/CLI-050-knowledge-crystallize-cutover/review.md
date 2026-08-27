---
spec: "CLI-050-knowledge-crystallize-cutover"
verdict: "PASS"
reviewed_sha: "b2f14350f823069420794a320ff757a34b0eed9c"
reviewer: "nan/deepseek-v4-flash"
date: "2026-08-26"
---

## Adversarial review

**Scope**: CLI-050-knowledge-crystallize-cutover (branch `feat/cli-050-crystallize-cutover`, HEAD `b2f1435`)
**Sources**: `specs/CLI-050-knowledge-crystallize-cutover/{proposal,tasks,verification,features.json}`; diff `merge-base(HEAD, origin/main)..HEAD` (27 files, +144/−1451)

### Spec and task alignment

Round 2 re-review after the FAIL at `c1a3d84`. This round reads the post-review fix commit `b2f1435` and re-verifies everything from scratch rather than trusting the round-1 disposition.

- Acceptance criteria AC1–AC5 map to tasks and to features f1–f6; each verification command is non-vacuous and I ran it. AC2 names every former caller and each is touched (confirmed by diff inspection and by running the f2 command).
- Out-of-scope is crisp: the two doc-staleness items are ticketed (DOCS-014 / #1271) and the vault-side skill SSOT edit is correctly sequenced as a post-merge vault commit. No scope creep.
- No `[AGENT-DRAFT]` / `[AGENT-SUGGESTION]` tags remain in any contract file (proposal/tasks/verification/features.json — the only matches are inside the review transcript artifact and the review body itself, both non-contract).

**Independent reproduction (all verified by executing, not by reading assertions):**
- `go build ./...`, `go vet ./...`, `GOOS=windows go vet ./...` — clean.
- `go test ./...` — every package `ok`.
- `golangci-lint run` — 0 issues.
- `shellcheck --severity=error setup-linux.sh scripts/vault-maintenance-weekly.sh scripts/vault.sh` — exit 0.
- Full `bats tests/*.bats` — **1464 ok, 0 not ok, exit 0** (the f4 evidence cites 1462, a snapshot variance from test additions; zero failures is what matters).
- `tests/vault-maintenance-weekly.bats` passes including the new cron-minimal-PATH case; `tests/knowledge-crystallize-go-parity.bats` passes 14/14 (incl. the parity-hygiene case asserting full-case coverage except the documented `help` exclusion); `tests/setup-windows.bats` passes.
- The installed `~/.local/bin/dotf` serves `vault crystallize --help`, confirming the two-tier deploy premise (lesson-150).
- **Mutation check (the load-bearing verify):** I removed the `export PATH="$HOME/.local/bin:$PATH"` line from `vault-maintenance-weekly.sh` and re-ran the cron-minimal test — it fails with `dotf: command not found` in the log, exactly the round-1 Major. Restored the line; the test passes. The round-1 Major finding is genuinely fixed and genuinely guarded.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Minor | REAL | spec artifact (features.json f3) | f3's verification command `git diff --stat main -- . ':!…'` depends on where the local `main` ref points at run time. Here `main` is one commit ahead of `origin/main` (HARNESS-045 #1272 added the harness-gate files *after* this branch forked at `84400ae`), so the literal command reports 41 files (spurious "deletions" of files the branch never contained) instead of the true 27. The AC3 *assertion* holds — `merge-base(HEAD, origin/main)..HEAD` is a clean 27-file scope with no unrelated changes — but the recorded self-check is fragile against a moving `main` ref, the same reproducibility class round 1 flagged for f2. | Ran `git diff --stat main` literally on this tree → 41 files / 3371 deletions; confirmed via `git log main..HEAD -- cli/internal/harness/` (empty) that the branch did not delete those files and `git cat-file -e 84400ae:…/gate.go` (absent) that they postdate the fork. | The recorded command is the check; it is not robust to a stale-`main` environment (I reproduced the divergence). | spec artifact (`features.json` — implementer to edit, not this review): either pin the base (`git diff --stat origin/main...HEAD`) or document that `main` must sit at the fork point; the evidence field as written already documents the true 27-file scope. |
| Minor | THEORETICAL | reliability/invocation (Windows) | `.ps1` hardening marked resolved-by-symmetry, but there is still no automated test for `vault-maintenance-weekly.ps1` at all. The PATH prepend (`$env:USERPROFILE\.local\bin`) is the correct class of fix and removes the round-1 concern, but the `& dotf` resolution under a Scheduled-Task environment remains unverified. Not introduced by this PR (the `.ps1` had zero coverage before too) and ticketed as TEST-002 (#1277). | Code read of the fix; `pwsh` absent on this box so no execution test was possible. Parity with the `.sh` guard is structural only. | UNTESTED — no bats/Pester case exercises `vault-maintenance-weekly.ps1`. | tests (separate ticket TEST-002 / #1277); not a blocker for this archive |

**Spec-vs-code mismatch check:** AC1–AC4, AC6 are satisfied as written and reproduced. The round-1 Major/REAL finding (cron PATH) is fixed in code and now guarded by a named, mutation-verified regression test; the round-1 Minor f2 finding is resolved (f2 is now literally reproducible, exit 0); round-1's Windows finding is mitigated by the `.ps1` hardening and ticketed. No spec says something the code does not do.

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | A | All ACs met; the one real negative path (cron PATH) is now covered by a mutation-verified regression test; no observed defects. |
| Verification       | B | I re-ran build/vet/test/lint/full-bats and the mutation check, all reproduce; but f3's recorded command is inherently sensitive to the local `main` ref and did not reproduce the documented 27-file scope on this tree. |
| Scope              | A | Clean 27-file diff against the true base; tangential staleness ticketed out, deferred skill edit sequenced; no creep. |
| Reliability        | A | Error/invocation path hardened on both twins with a genuine (fails-without-it) regression test on the `.sh`; residual `.ps1` test gap is pre-existing and ticketed. |
| Maintainability    | A | Clinical deletions, accurate frozen-contract re-framing (crystallize.go, lib.sh, ORACLE), dead `shell` mode / `GC_ORACLE_SH` plumbing removed, no orphaned references. |
| Handoff-readiness  | A | proposal/tasks/verification complete with candid decisions, promotion notes and a full archive checklist; only mechanical archive steps remain. |

### Verdict
PASS

### Recommended next steps (before archive)
- **Spec artifact (follow-up, not blocking):** tighten f3's verification command to pin the diff base (`origin/main...HEAD` or an explicit fork-point ref), or add a note that `main` must equal the fork point, so the recorded self-check is not environment-dependent. Implementer to edit `features.json`, not this review.
- **Tracked (already filed):** TEST-002 (#1277) remains the right home for adding `vault-maintenance-weekly.ps1` coverage; nothing new required by this archive.
- **Archive:** `dotf spec archive` is **advisable** — the round-1 blocker and majors are resolved with evidence and a mutation-verified regression test, there are no open blockers/majors, and the rubric is all B-or-above. The two Minor findings above are non-blocking and do not gate archive.
