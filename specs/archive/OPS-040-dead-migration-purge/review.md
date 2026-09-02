---
spec: "OPS-040-dead-migration-purge"
verdict: "PASS"
reviewed_sha: "d0c8b16b96d1baf0291cc971e652353070c39095"
reviewer: "nan/glm5.3-flash"
date: "2026-09-01"
---

## Adversarial review

**Scope**: OPS-040-dead-migration-purge — branch `chore/ops-040-dead-migration-purge`, commits `1e86f48` + `d0c8b16` against merge base `16d8f96` (origin/main at review start).
**Sources**: `specs/OPS-040-dead-migration-purge/{proposal,tasks,verification,features.json}`; `git diff origin/main...HEAD` (14 files); full `bats tests/*.bats` run; targeted suites; three guard mutations (reverted).

Independence: reviewer drawn from `harness/reviewer-pool.json`, not the implementing session. Every claim below was verified by execution on this tree, not accepted from `verification.md`.

### Spec and task alignment

- The diff maps 1:1 to the proposal's nine deleted blocks and two kept blocks. Blocks 1–9 are absent from both scripts (features f2/f3 loops re-executed by this review: no match); HIVE-118 (setup-linux.sh:1025, setup-windows.ps1:563) and MEM-002 (setup-linux.sh:~1258, setup-windows.ps1:~884) are untouched — no diff hunk falls inside either region — and `.zshrc`/`.bashrc` retain their `BUN_INSTALL` exports (verified at :134/:153).
- AC1 holds repo-wide: zero executable `dotf secrets show OPENROUTER_API_KEY` call sites remain; the only consumers of the variable are the interactive `opencode`/`pi` wrappers (`.zshrc:91-92`, `.bashrc:113-114`, `powershell/profile.ps1:297-298`), `dotf review`'s runtime `os.Getenv` (`cli/internal/review/review.go:52` — user shell, not setup), and registry/docs prose. Nothing reads a setup-process export.
- Block 4's premise verified independently: `harness/manifest.json` `.doctrine.deploy[]` resolves to `.gemini/GEMINI.md` **and** `.codex/AGENTS.md`; `~/.gemini/GEMINI.md` present on msi at 12029 bytes (stat only); `tests/compile-harness.bats` "doctrine injection preserves user content and is idempotent" asserts exactly the contract the deleted `rm -f` defeated. The doc claims check out: `tests/antigravity.bats:83` still pins the `.zshrc` endpoint export; `cli/internal/doctor/checks_deploy.go:801` defaults to production and skips when `agy` is absent.
- Fail-first reproduced by this review, stronger than the implementer's scratch-tree proof: three mutations — `rm -f "$GEMINI_HOME/GEMINI.md"` in setup-linux.sh, `Remove-Item -LiteralPath "$GeminiHome\GEMINI.md"` in setup-windows.ps1, and `rm -f "$HOME/.codex/AGENTS.md"` (the **second** manifest target, proving the list is manifest-driven) — each turned the guard red at the right line; tree restored clean after each.
- AC6 re-executed: `bash -n` clean; shellcheck 19 findings on merge base → 16 here (all info, none introduced); full suite `1..1535`, ok=1528 (77 skips) **not_ok=7**, and the 7 are exactly the claimed pre-existing set — re-confirmed red on a clean detached worktree at the merge base `16d8f96` (the implementer baselined at the older `3a11d97`; this review verified at the actual base). `shellcheck -S warning` silent.
- AC5 verified: `secrets-show-callsites.bats` reports `ok … # skip no 'dotf secrets show <id>' call sites in the tree — nothing to resolve`; AC8 verified: lessons 256/257 exist, are dated, and are indexed in `docs/lessons/_index.md:276-277`; `specs/archive/BUG-014-claude-mem-marketplace-register/` exists as the cited prior art. No `[AGENT-DRAFT]`/`[AGENT-SUGGESTION]` tags in any spec file.
- **Mid-review event, recorded for the archive gate**: while this review ran, a background fetch showed origin/main advanced to `06c3b7a` — this branch squash-merged as PR #1433. Verified content-identical: `git diff origin/main HEAD` is empty, and both setup-script blobs match (`4854fb8`, `d55f560`). The review therefore remains valid for the merged state. `tasks.md`'s "PR opened" checkbox is unchecked in the merged contract files — see finding F6; do **not** tick it post-review, that edit would re-trigger the staleness gate against this review.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Minor | REAL | verification artifacts | Recorded numbers do not reconcile: verification.md says `setup-windows.ps1` 2341→2301 (−40); actual 2341→**2305** (numstat 17+/53−, net −36). Suite tally "ok=1526 not_ok=7 skipped=76" sums to 1609 ≠ TAP plan 1535 (actual: ok=1528 incl. 77 skips, not_ok=7). Material conclusions (net deletion; the 7-failure set) are unaffected and independently reproduced. | `git diff --numstat origin/main...HEAD`; `wc -l` = 1686/2305; `bats tests/*.bats` plan line `1..1535` | UNTESTED | spec artifacts (verification.md) |
| Minor | REAL | scope/enumeration | `rm -f "$GEMINI_HOME/config/.migrated"` (setup-linux.sh:463) survives unclassified. `.migrated` has **zero** other references in the repo — no recorded producer, consumer, or expiry — which is precisely the "answer not in the code" class this spec exists to retire. Probe: file absent on msi, so ownership is inconclusive; if agy writes it, setup deletes agy's own migration marker every run (the inverse-dead pattern). Pre-existing line, not introduced here. | repo-wide grep = single hit; `ls ~/.gemini/config/.migrated` → absent | UNTESTED | follow-up ticket: probe the owner, then classify or delete |
| Minor | THEORETICAL | guard breadth | The guard matches only `(rm -[rf]+|Remove-Item)[^|;]*` + basename: flag-less `rm`, `rm --force`, `unlink`, and pwsh aliases (`del`/`ri`/`rd`) evade it; basename-only matching can false-positive on a legitimate removal of a *different* file named `GEMINI.md`/`AGENTS.md`. The wiring itself is proven (three mutations red this session); the residual gap is match breadth, not resolution from the manifest. | mutation runs recorded above | `tests/guard-doctrine-target-not-deleted.bats` @test "guard: no setup script deletes a harness doctrine deploy target" (mutations); @test "guard: the shipped manifest resolves at least one doctrine deploy target" | tests (widen ERE or document the accepted contract) |
| Minor | REAL | AC6 evidence precision | Two AC6 clauses lack discriminating evidence. (a) "passes under both bash and zsh": the shell invoking bats does not change test semantics (bats re-execs its own bash), so the clause cannot fail as written and one run is recorded. (b) "PSScriptAnalyzer is clean" is asserted but no local run exists (pwsh absent on this box) — it is enforced only by CI (`.github/workflows/ci.yml:115-121`), which verification.md does not say. | verification.md AC6 block; `command -v pwsh` → absent; ci.yml | UNTESTED | spec artifacts (reword the clause; cite CI as the evidence source) |
| Minor | SPECULATIVE | classification wording | Blocks 3/5 are labelled "dead code (skipping costs nothing on any machine)" while the operative justification is "leftovers are inert": on an unconverged machine skipping leaves the stale file in place, and the Windows twin of block 5 was kept on exactly the "leftovers measured present on a real box" logic. Consistent outcome, but the spec applies a one-line distinction everywhere else and omits it here. | proposal table vs the WIN-013 rationale added in the diff for the kept Windows twin | UNTESTED | spec artifacts (wording) |
| Question | REAL | contract bookkeeping | `tasks.md` "PR opened referencing this spec folder" is unchecked while PR #1433 (squash `06c3b7a`) is already merged with tree-identical content. This review cannot edit contract files (staleness watch), so the checkbox state vs reality is left to the human at archive time; ticking it now would post-date this review and re-trigger staleness. | `git diff origin/main HEAD` empty; `git log 16d8f96..origin/main` | UNTESTED | human decision at archive (accept #1433 as the referenced PR) |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | B | All eight ACs verified by execution including mutation reds and the merge-base baseline; the deduction is for the unclassified `.migrated` cleanup (F2) and wording precision (F5). |
| Verification       | B | features.json commands re-executed green and the 7-failure baseline reproduced at the true merge base, but two recorded numbers are off (F1) and one AC6 clause silently rests on CI (F4). |
| Scope              | A | 14 files, every hunk maps to a classified block, AC, lesson, or its guard; no creep; kept blocks byte-identical to main. |
| Reliability        | A | Deletion-only change; C15 skip path pinned; guard fail and skip paths proven by mutation; the child-pwsh invariant correctly narrowed to the surviving installer. |
| Maintainability    | A | Replacement comments state what was removed and on what evidence; the guard is manifest-driven so future doctrine targets are covered without edits; net −88 lines. |
| Handoff-readiness  | B | Lessons 256/257 written and indexed, #1431 spun out, owner decisions surfaced rather than landed; deduction for F1 arithmetic slips and the F6 checkbox drift. |

### Verdict
PASS

### Recommended next steps (before archive)

- None of the findings blocks archive: no Blocker/Major, rubric all B or above.
- File a small follow-up ticket for F2 (owner of `~/.gemini/config/.migrated`; classify or delete — it is a ready-made instance of lesson 257).
- Optionally correct the two numbers in verification.md (F1) and reword the AC6 clauses (F4) — note both edits are safe (verification.md is not staleness-watched), but any edit to `proposal.md`/`tasks.md`/`features.json` after this review re-triggers staleness and would require a re-review.
- At archive time, resolve F6 by accepting PR #1433 as the referenced PR rather than editing `tasks.md`.
