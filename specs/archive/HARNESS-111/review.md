---
spec: "HARNESS-111"
verdict: "PASS-WITH-GAPS"
reviewed_sha: "8a0eb12c78bbc15645332f1d13d3f8cbccc182d6"
reviewer: "nan/glm5.3-flash"
date: "2026-09-30"
---

## Adversarial review

**Scope**: HARNESS-111
**Sources**: `specs/HARNESS-111/{proposal,tasks,verification}.md`, `specs/HARNESS-111/features.json`, `specs/HARNESS-111/review-round-1.md` (FAIL, dispositions recorded), `git diff a76ea184ea20f8f54903df15c7d558074f22e241...HEAD` — spec-relevant surfaces: `scripts/compile-harness.sh` (`fold_to_ascii`, `non_ascii_hex`, `migrate_legacy_preamble`, `deploy_doctrine`), `tests/compile-harness.bats`, `tests/skills-pipeline.bats`.

### Spec and task alignment
- All six ACs were re-verified **by execution in this session**, not by reading `verification.md` (results below). `tasks.md` boxes all carry diff evidence; the round-1 fix commit (`8a0eb12c`) matches its stated scope: both cap warnings now print `characters / bytes` and compare against the larger, AC3 is amended in the contract with the #1685 rationale, and f3/f6 now run tests that can fail.
- Round-1 Blocker (code folded accents/section signs against the then-AC3): resolved by contract amendment, not by code change — the correct set, since #1685 widened the fold deliberately on 2026-09-24 and the rationale lives at `fold_to_ascii`. The invariant AC3 was protecting (fixed table, no catch-all, survivors reported by bytes) holds and is test-named.
- Round-1 Major (cap warning printed one unit): applied in code; both over-cap warnings print both units and the comparison uses the larger.

### Verification run in this session (evidence, fresh)
- **AC1**: `--deploy` into a temp `HOME`; both capped targets measure **8111 chars == 8111 bytes** (pure ASCII by construction), under the 12000 cap in both units.
- **AC2**: mutation `fold_to_ascii "$payload"` → `if false; then …` — see findings: the `skills-pipeline` byte assertion **passes** under the mutation (payload far under cap); the pure-ASCII guard `doctrine: a capped surface is folded to pure ASCII, marker and preamble included` goes **red** under the same mutation (reproduced), then green after revert.
- **AC3**: `bats -f 'a character the fold does not know survives|a capped surface is folded to pure ASCII'` → 2/2 ok; unknown byte `e29c93` is reported, not guessed at.
- **AC4**: `shellcheck` rc 1 on both branch and `main` with the **same 7 SC2016 infos**, **0 SC1112** on both — no new findings. (`verification.md` says "6 SC2016"; the count is now 7 on both sides, so the claim it supports — identical to main — still holds; the number is stale.)
- **AC5**: `bash -n` and `zsh -n` clean; two consecutive `--deploy` runs into fresh temp homes produce **byte-identical** trees (`diff -r` clean).
- **AC6**: both over-cap-warning tests pass, including the case asserting the file's bytes exceed its characters and the warning names both.
- `bats tests/compile-harness.bats` 83/83; named skills-pipeline cap test ok. No `[AGENT-DRAFT]`/`[AGENT-SUGGESTION]` tags in the spec's contract files.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Minor | REAL | AC2 / test sensitivity | AC2's mutation proof is content-dependent and no longer reproduces at HEAD: with the fold disabled, `HARNESS-056: the compact doctrine payload carries it and stays under its cap` **passes**, because the committed payload is 8111 chars/bytes — far under the 12000 cap — so the byte assertion cannot fire. The proof recorded in `verification.md` (2026-09-05, payload at 11974/12047) was genuine then, but nothing marks it as content-bound. The protection is not lost: the byte assertion is now a backstop kept unreachable by two stronger named guards (the 8000-character budget and the pure-ASCII assertion on the committed records, both in `tests/compile-harness.bats`), and the fold itself **is** mutation-verified — the pure-ASCII test went red under this session's mutation. Note the skills-pipeline comment "reintroduce a multi-byte character and the byte count … lands here first" is inaccurate: the pure-ASCII test lands first. | Reproduced this session: mutation → skills-pipeline test ok; same mutation → `doctrine: a capped surface is folded to pure ASCII, marker and preamble included` not ok; revert → green | `HARNESS-056: the compact doctrine payload carries it and stays under its cap` (the insensitive assertion); `doctrine: a capped surface is folded to pure ASCII, marker and preamble included` (the mutation-verified guard) | verification.md (disposition: AC2's evidence is historical and content-dependent; the current guard chain is 8000-budget + pure-ASCII + byte backstop) and tests (correct the "lands here first" comment). Both outside the contract set — no contract edit required. |
| Minor | THEORETICAL | quality | `deploy_doctrine` is ~100 lines with ~12 branches — over the repo's <40-line function law and past CC 10 (within the rubric's ≤15). The structure is clear and every branch is warning/emission logic, so this is a split-and-delegate follow-up, not a defect. | Code read, `scripts/compile-harness.sh` lines ~1239–1340 | UNTESTED | code (follow-up refactor) |
| Minor | SPECULATIVE | robustness | `cap` is read via jq `// 0` and compared with `[[ "$cap" != 0 ]]` plus arithmetic; a string-valued `char_cap` in `harness/manifest.json` would break the arithmetic. The manifest is authored in-repo and currently numeric, so no live path. | Code read; manifest values are JSON numbers | UNTESTED | code (only if the manifest ever gains a schema guard) |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | B | All six ACs verified by execution; the one gap is AC2's mutation demonstration no longer reproducing (content-dependent), with an equivalent stronger guard verified red. |
| Verification       | B | Evidence is command-backed and I reproduced AC1/AC3/AC4/AC5/AC6 fresh; AC2's evidence is historical and `verification.md` does not mark it content-bound (and its SC2016 count is stale by one). |
| Scope              | B | The spec-relevant surfaces match the proposal and the AC3 amendment is recorded in the contract; the launcher-resolved range also spans ~770 files of prior merged work that this review verifies only through the shared suite, not re-certifies. |
| Reliability        | A | Idempotent (verified byte-identical), `set -e`/`pipefail` traps handled explicitly (`if` vs `(( )) &&`, `non_ascii_hex`'s `|| true`), exact-line preamble adoption protects user content. |
| Maintainability    | B | Fixed table with WHY comments and self-naming hex escapes; `deploy_doctrine` exceeds the function-length law. |
| Handoff-readiness  | A | AC3 amendment recorded in the contract with rationale, round-1 dispositions written into `verification.md`, promotion candidates answered. |

### Verdict
PASS WITH GAPS

All findings are Minor; no Blockers, no REAL Majors, no rubric D or C. The three minors are tracked above, each with a disposition path that stays **outside the contract set** (`verification.md` dispositions and follow-up work), so this verdict permits archive without invalidating itself.

### Recommended next steps
- **Disposition in `verification.md` (implementer, outside contract set):** record that AC2's mutation evidence is historical and content-dependent — the byte assertion cannot fire while the 8000-character budget and the pure-ASCII assertion hold — and that the live mutation-verified guard for the fold is `doctrine: a capped surface is folded to pure ASCII, marker and preamble included`. Correct the stale SC2016 count (7, not 6, identical on both sides).
- **Follow-up ticket (tests):** fix the "lands here first" comment in `tests/skills-pipeline.bats` — the pure-ASCII test in `tests/compile-harness.bats` fires first; the byte assertion is a backstop.
- **Follow-up ticket (code):** split `deploy_doctrine` (warnings into a helper) to get back under the repo's function-length law.
- Archive is **advisable**: `dotf spec archive HARNESS-111` can run against this review; the reviewer string matches `harness/reviewer-pool.json` and the contract digests are unchanged since the launcher recorded them.
