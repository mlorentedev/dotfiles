---
spec: "AI-032-pi-settings-field-sync"
verdict: "PASS"
reviewed_sha: "6a791df5eef96a0f20bd4c4ea652bf49f1bcf802"
reviewer: "nan/deepseek-v4-flash"
date: "2026-08-27"
---

## Adversarial review

**Scope**: AI-032-pi-settings-field-sync (implementation `d26b238`, spec folder `f31ea39`, review-fix `833d506`, merge `6a791df` = HEAD)
**Sources**:
- `specs/AI-032-pi-settings-field-sync/{proposal,tasks,verification,features}.json`
- Per-commit file lists for `d26b238` / `f31ea39` / `833d506` / `6a791df`
- Live runs: `bats tests/pi-config.bats` (18/18), `bats tests/*.bats` (1521 results, exit 0), `bash -n`, `shellcheck`, all five `features.json` verification commands
- Mutation battery (this session): 6 mutations across `setup-linux.sh` + `setup-windows.ps1` against the exact test-17/test-18 logic
- Negative probes (this session): malformed dst, src missing key, empty src array, tmp-file hygiene, `set -euo pipefail` execution of the real block
- Independent BOM check: `~/.local/lib/node_modules/@earendil-works/pi-coding-agent/dist/core/settings-manager.js` (`JSON.parse(stripBom(...))`, lines 200/379)
- Reviewer = pool primary (`harness/reviewer-pool.json` entry 0); reviewer ≠ implementer

This is a **re-review**: the prior round (this same session stream) returned FAIL on a Windows null-guard defect, the fix (`833d506`) landed, and this review verifies the fix and re-audits the whole change. No PR exists yet (`tasks.md` "PR opened" is the one unchecked task); the review is against HEAD on the feature branch.

### Spec and task alignment

- All five ACs map to named coverage (below). The two contract claims that previously failed now hold: the Windows src null-guard **is** fixed and regression-pinned, and proposal.md's "byte-for-byte" overstatement is corrected to "value-for-value" (both in `833d506`, both verified in the current tree).
- `features.json`: 5 entries, all `pending` with empty `evidence` — none falsely `passing`. Every verification command exits 0 when run exactly as written.
- No `[AGENT-DRAFT]`/`[AGENT-SUGGESTION]` in any contract file (grep-verified). `proposal.md` AC checkboxes are all `[ ]` and frontmatter `status: draft` while tasks/verification are fully ticked — the same contract-file drift the AI-033 review flagged; a Minor finding below, to fix at archive time only.
- Diff scope: the AI-032 commits touch exactly `setup-linux.sh`, `setup-windows.ps1`, `tests/pi-config.bats`, the spec folder, `docs/lessons/lesson-233-...`, and proposal.md's wording fix. The HARNESS-067 / AI-033 folders and `cli/` changes visible in `main...HEAD` arrive via shared-history commits (`d7e5ddc`, `cba6be2`) and the `6a791df` merge — not part of this change.
- `git diff HEAD -- specs/AI-032-.../` is empty: contract files match the reviewed sha, so this review cannot go stale the moment it is written.

### What I actually ran (not read off the page)

- `bats tests/pi-config.bats` → 18/18 ok, exit 0 (both new tests #17, #18).
- `bats tests/*.bats` (full suite) → 1521 results, **0 failures**, exit 0. Skips: 78 (65 "only runs inside integration test container" + 13 "pwsh not available"). **This does not match verification.md's "1519 ok + 2 skips"** — see Finding 1.
- `bash -n setup-linux.sh` clean; `shellcheck setup-linux.sh` → 22 pre-existing findings, **none** in the sync block (lines 930–980).
- Proper ASCII check (PCRE) on the extracted `setup-windows.ps1` sync block → 0 non-ASCII (verification.md's claim holds); whole file has 10 pre-existing non-ASCII elsewhere.
- All five `features.json` verification commands exit 0.
- Mutation battery (proves the tests catch regressions, not just pass):
  - Linux (#17): wrong-field write (`.theme = "dark"` instead of `.enabledModels`) → **caught**; equality check neutralized (`'true'`) → **caught**; extraction anchor removed → **caught**.
  - Windows (#18): **null-guard reverted to the pre-fix shape → caught** (the prior Major is now regression-pinned); `-InputObject` → pipe → **caught**; **write lines deleted entirely → NOT caught** (Finding 3).
- Negative probes on the real extracted Linux block: malformed dst left untouched + no stray tmp; src missing `enabledModels` key → skip, dst untouched; second run idempotent at value level; `set -euo pipefail` execution of the shipped block converges and stays converged.
- Empty-src probe: Linux **writes `[]` into dst** when src `enabledModels` is `[]` (Finding 2).

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Minor | REAL | verification artifact | verification.md's "1521 results, 1519 ok + 2 skips" does not reproduce in the current tree. Actual: 1521 results, 0 failures, 78 skips (65 container-gated + 13 pwsh), exit 0. Every AI-032 test passes and the exit-0 claim holds — only the arithmetic is stale (merged main added skip-tagged tests after the verification was written). | `bats tests/*.bats` this session (1521/0/78); count of `# skip` lines by reason (65 + 13) | UNTESTED (doc-only fix; the check that would validate it is the suite run itself, reproduced: exit 0) | spec (verification.md — do not edit during this review; fix at archive time) |
| Minor | THEORETICAL | parity (empty src) | Empty `enabledModels` in the repo src diverges: Linux `[]` passes the guard (`-z`/`"null"` checks both miss `[]`) and writes an empty array into a live dst — removing every model from the user's picker; Windows `$piSrcModels.Count -eq 0` skips with a warning. Unreachable from current repo state (src is CI-held non-empty; README/set-equality tests would have to be emptied too), so no real machine is at risk today, but the two platforms promise AC4 parity and differ on a defined input. The proposal's own rationale ("adding a repo-curated entry … can never surprise or destroy anything") argues for skip-on-empty, not write-empty. | N3 probe reproduced the Linux write of `[]` into dst with a synthetic src; code read of both guards | UNTESTED — no fixture covers an empty src array on either side | code + tests (align Linux with Windows' skip-on-empty, or decide and pin one policy; add a fixture test) |
| Minor | REAL | Windows test strength (AC4/AC5) | The structural Windows test does not pin the write itself: deleting `$piDstJson.enabledModels = $piSrcModels` and the `Set-Content` line leaves test #18 green — the block could stop syncing entirely and no test would notice. The current block is correct by static analysis (write present, braces/parens balanced, `-InputObject` at both comparison sites, null-check before `@()` wrap), so this is a coverage gap, not a live defect; it matches the repo's documented grep-parity pattern for PowerShell, and no pwsh exists here to execute it. | M6 mutation: write removed → test-18 logic still passes (rc 0) | UNTESTED — no named test asserts the write line or the `Set-Content` call (add `grep -qF '$piDstJson.enabledModels = $piSrcModels'` and the `Set-Content` line to #18, or run the block under pwsh where available) | tests |
| Minor | THEORETICAL | reliability (Linux write) | `mv "$PI_SETTINGS_TMP" "$PI_SETTINGS_DST"` is unguarded under `set -euo pipefail`: if it fails (dst became a dir, directory not writable), setup aborts and a `.settings.json.XXXXXX` temp is left in `~/.pi/agent/`. Matches the script's pre-existing idiom (neighbouring `cp`/`mv` are unguarded too), and the failure would be loud, so severity is low. | code read of the else branch; consistent with adjacent blocks | UNTESTED | code (fold `mv` into the guarded `if jq … && chmod` chain) |
| Minor | REAL | spec artifact hygiene | `proposal.md` AC checkboxes all unticked, frontmatter `status: draft`, while tasks/verification are fully ticked and all five ACs are demonstrably met — drift that will confuse the next reader and `/spec check`-style tooling. Same class as the AI-033 review's Minor. | direct read of the three files | UNTESTED (spec-artifact concern; the archive gate does not validate it) | spec (tick ACs and set `status:` at archive time only — editing now would invalidate this review's staleness check) |
| Question | — | test extraction fragility | Test #17's `sed -n '/^# Field-level sync (AI-032/,/^fi$/p'` ends at the first column-0 `fi`; a future edit that dedents an inner `fi` would truncate the extracted block. The seed-block tests guard this with a "range never closed" size check; #17 has none. In practice loud (a truncated block fails `eval` or the jq assertions), so it is a robustness note, not a defect. | code read of #17 vs the seed tests' size guard | UNTESTED | tests (optional: add a line-count sanity guard to #17) |

**Resolved since the prior FAIL review** (each verified this session, not assumed):
- Windows src null-guard (prior Major): fixed in `833d506` — `$piSrcModelsRaw` null-checked *before* the `@()` wrap, covering both a missing key and `ConvertFrom-Json '[]' → $null` on PowerShell Core (PowerShell/PowerShell#13595), and empty-on-5.1 via `@().Count -eq 0`. Regression-pinned by #18's `$null -eq $piSrcModelsRaw` grep; mutation M4 confirms the test fails on a revert. Braces/parens balance verified (9/9, 18/18).
- "byte-for-byte" wording (prior Minor): corrected to "value-for-value" in `proposal.md`; value-preservation independently confirmed (N4 fixture: mode preserved, values intact).
- `mktemp` hygiene (prior Minor): temp now written beside the destination (same-filesystem atomic rename) with `chmod --reference` mode copy; N1/N4 confirm no stray temp and preserved mode. GNU-only `chmod --reference` is safe here — `setup-linux.sh` has no macOS/darwin branch.
- `-Depth 10` truncation hazard: non-issue — the live destination's max nesting depth is 2 (verified on this machine); the committed seed file is depth ≤2.

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | B | AC1–AC3 executed and mutation-proven; AC4 correct by structural test + full static analysis (no pwsh to execute); no observed defect; minor negative-path gaps (empty-src divergence, M6 unpinned write) |
| Verification       | B | Implementer's claims largely reproducible this session (suite exit 0, both new tests, shellcheck-clean block, ASCII-clean block, all features commands); one stale count ("1519+2") and the Windows behavioral side is executed nowhere |
| Scope              | A | AI-032 commits touch exactly the three code files + spec folder + lesson-233 + proposal wording fix; HARNESS-067/AI-033 material arrives via shared history and the merge, not this change |
| Reliability        | B | Error paths handled (malformed dst, missing key, no jq, PS catch), idempotent by construction, atomic same-fs Linux write; theoretical gaps: unguarded `mv`, non-atomic Windows `Set-Content` |
| Maintainability    | A | Small WHY-commented blocks mirroring the packages-reconcile idiom; CC ≈5–6 per block (manual count, well under 10); shellcheck/bash -n/ASCII clean; no dead code |
| Handoff-readiness  | A | Spec updated in-session (wording fix), lesson-233 captured, promotions assessed, verification filled, archive checklist present; PR + archive steps correctly open |

### Verdict
PASS

The severity axis: no Blocker, no Major — the prior round's Major is fixed and regression-pinned. The rubric is all B or above (no C, no D), which per the mechanical aggregation rule yields PASS; the five Minor findings and one Question are tracked below and none blocks the deploy contract. The two findings a skeptic should weight most are Finding 3 (Windows write line unpinned — a coverage gap on the exact platform where the prior review found a real bug) and Finding 2 (empty-src parity divergence, unreachable from repo state), and both are listed with their named test/fix paths.

### Recommended next steps (before archive)
- [ ] **Before/at archive**: fix Finding 1 — correct verification.md's suite arithmetic to the current truth (1521 / 0 failures / 78 skips) and re-verify. Do **not** edit `proposal.md`/`tasks.md`/`features.json` now (any edit invalidates this review's staleness check; the AC ticks belong at archive time per Finding 5).
- [ ] **Recommended (tests)**: pin the Windows write line in test #18 (`$piDstJson.enabledModels = $piSrcModels` + the `Set-Content` call) so the block cannot silently stop syncing (Finding 3); optionally run the block under pwsh on a Windows machine once, to execute rather than grep the parity guarantee.
- [ ] **Recommended (code)**: decide the empty-src policy — skip-on-empty on Linux to match Windows (or document why write-empty is intended) — plus a fixture test (Finding 2).
- [ ] **Optional**: guard the Linux `mv` (Finding 4); add a line-count guard to test #17's extraction (Question).
- [ ] Open the PR referencing the spec folder — the one unchecked task; the review is against HEAD, which a PR must not change.
- [ ] At archive time: tick proposal.md ACs, set `status: archived`, move the folder, close #1247 with the PR link.

**Archive advisability**: `dotf spec archive` is **advisable** in the current state. Gate pre-flights verified against `cli/internal/spec/review.go`: `spec` equals the folder id; `verdict: PASS` is recognized and non-blocking; `reviewed_sha` equals HEAD and the contract files are clean vs HEAD; provenance matches `review-request.json` (reviewed_sha `6a791df…` and reviewer `nan/deepseek-v4-flash` identical; the digest-before check passes because this file differs from the prior round's); `reviewer: nan/deepseek-v4-flash` is pool entry 0; no unresolved `[AGENT-DRAFT]`/`[AGENT-SUGGESTION]` in the contract files. Finding 1's verification.md correction should land with or before the archive step so the artifact matches reality.
