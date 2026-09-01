---
spec: "HARNESS-093-reviewer-pool-random"
verdict: "PASS"
reviewed_sha: "acca2dabf3ff636eb0ea88be846166bf06c7c3fa"
reviewer: "nan/glm5.3-flash"
date: "2026-08-29"
---

## Adversarial review

**Scope**: HARNESS-093-reviewer-pool-random (PR #1372, squash-merged to `origin/main` as `acca2da`)
**Sources**: `specs/HARNESS-093-reviewer-pool-random/{proposal,tasks,verification,features.json}`; `git show acca2da` (11 files, +365/−18); `harness/reviewer-pool.json`; `ai/pi/models.json`; vault source `00_meta/skills/adversarial-review/SKILL.md`.

Independence note: this review is the **first drawn under the new policy** — the launcher drew `nan/glm5.3-flash` at random (`review-request.json`, 2026-08-29T06:04:06Z) and launched this detached session with a thin prompt. The implementer worked in a different session/worktree (`dotfiles-wt-pool`, `feat/reviewer-pool-random`). The pin was verified from inside: `PI_PROVIDER=nan`, `PI_MODEL=glm5.3-flash`.

### Spec and task alignment

- Diff matches the proposal's "What" list exactly — launcher draw + launch line, two pool entries, bats guard, skill line — no scope creep. Spec artifacts complete; no `[AGENT-DRAFT]`/`[AGENT-SUGGESTION]` tags.
- All 4 ACs re-verified first-hand in this session, not trusted from `verification.md`:
  - **AC1**: `go test ./internal/spec/ ./internal/cmd/ -run 'Reviewer|SpecReview' -count=1` → both `ok`. Named tests: `TestDrawReviewerReachesEveryMemberAndResolveHonoursAnExplicitChoice` (both indexes reachable, out-of-range draw errors, empty name refused, explicit choice honoured), `TestResolveReviewerRefusesAModelOutsideThePool`, `TestResolveReviewerRefusesWhenThereIsNoPool` (now through `DrawReviewer`). Box-level: `--reviewer nan/not-in-pool` → `Error: reviewer "nan/not-in-pool" is not in harness/reviewer-pool.json` + full pool listing. `DrawReviewer` guards `len==0` **before** calling draw, so `rand.IntN` can never see 0 (no panic path); out-of-range results error instead of wrapping.
  - **AC2**: `TestSpecReviewDrawsAPoolMemberAndSaysSo` (pinned draw → `(pi, random draw)`; `--reviewer` → `(pi, requested)`). The 8 pre-existing `TestSpecReview*` tests keep their meaning via the package `init` pin to index 0, with in-test save/restore around the one test that unpins — checked for order-of-restore correctness (restores the pinned func, not the production one; no `t.Parallel`, so no global race).
  - **AC3**: `bats tests/reviewer-pool.bats` → 4/4 (unique ids; all four NaN models present; every `pi` member `reasoning: true` in `ai/pi/models.json` — both new models verified directly: `glm5.3-flash reasoning=true ctx=1048576`, `qwen3.8-flash reasoning=true ctx=1048576`; launcher + committed skill record say "drawn at random"). Vault source line 48 verified: `# a pool member drawn at random (the launch line names it)`.
  - **AC4**: six `--dry-run` launches from this session drew **five different members across all five pool entries** (glm5.3-flash, agy, deepseek, agy, qwen3.8, mimo) — stronger than the "≥2 distinct" the AC asks. `--reviewer nan/glm5.3-flash --dry-run` prints `(pi, requested)` and the command carries `'--model' 'glm5.3-flash'`.
- `--dry-run` returns **before** `WriteReviewRequest`, so evidence dry-runs can never clobber the launch record — checked specifically because AC4's evidence consists of six dry-runs run against the real spec.
- Untracked `review-request.json` is normal mid-flight state: transcripts are gitignored (`specs/**/review-transcript.jsonl`), request files are tracked in every archived spec and ride the archive commit.
- Security: the API key reaches the runner via `dotf secrets run --only NAN_API_KEY --` env injection; the key never appears in the printed command or stdout. The prompt is a single ShellJoin-quoted argv element — no injection surface. No new dependencies.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Minor | REAL | verification evidence | `features.json` f4's recorded command `dotf spec review <id> --dry-run \| grep '^Reviewer:' \| sort -u \| wc -l \| grep -qvE '^1$'` cannot see the Reviewer line: cobra's `cmd.Printf` writes **stderr**, so the pipeline feeds grep an empty stdout, `wc -l` prints `0`, and `grep -qvE '^1$'` exits 0 — a broken launcher (missing pool, hard error) would also "verify". The assertion `!= 1` is weaker than the intended `>= 2`. | Reproduced this session: `dotf ... --dry-run 2>/dev/null \| grep '^Reviewer:'` → grep exit 1. The underlying behavior is nonetheless real: the six draws above hit all five members (observed on stderr). | UNTESTED (the evidence string is a spec artifact, not code) | spec artifacts (`features.json` f4: `2>&1` before the pipe and assert `grep -qE '^[2-9]$'`). Note: editing `features.json` now would stale this review (`reviewed_sha`) — record here and correct the pattern the next time an AC4-style box check is recorded, or fix + re-review if preferred. |
| Minor | REAL | skill-record drift | The deployed generated record on this box (`~/.pi/agent/skills/adversarial-review/SKILL.md`) still reads "the pool's primary reviewer" — the `generated_sha` refresh is deferred to the next Linux `compile-harness.sh --refresh` (ENGINE-001), so agents on this box read the stale usage line until then. | Observed at this session's start; disclosed in `verification.md` (AC3). The repo's committed record (`harness/skills/adversarial-review/SKILL.md`) IS updated and bats-guarded (test 4), so the repo SSOT is correct. | UNTESTED (machine-local derived state) | vault/ops — run `--refresh` on a Linux box; no code change. |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | A | All 4 ACs verified first-hand incl. negative paths (empty pool, out-of-range draw, unknown id, empty name); no panic path (`IntN` guarded); single caller, no hidden consumer of the old empty→first-entry contract. |
| Verification       | B | Three of four AC evidences are reproducible and re-ran clean (`go test`, `bats 4/4`, lint 0 issues); AC4's recorded command is tautological on empty output (finding 1), though the behavior itself was independently reproduced here. |
| Scope              | A | Diff matches the proposal exactly; spec artifacts complete; out-of-scope items (calibration, weighting, CPU load) untouched. |
| Reliability        | A | Error paths handled loudly (bad draw, empty pool, unknown id, tmux-missing degrade); request-write failure warns instead of refusing, with recorded rationale; dry-run is side-effect-free. |
| Maintainability    | A | `DrawReviewer` is ~12 lines, CC ≤ 3; randomness is a documented seam; pool entries state honestly what is and is not established; `role` kept as provenance, not selection. |
| Handoff-readiness  | A | features.json per AC, decisions recorded, promotion candidates answered, archive checklist present; both minors disclosed in-session. |

### Verdict

PASS

### Recommended next steps (before archive)

- None are gating. Two Minors are recorded above; neither blocks archive.
- Optional: correct the f4 evidence-command pattern (next box-evidence author should use `2>&1` and assert ≥2 distinct) — doing it now would stale this review by touching `features.json`; the pragmatic path is to carry this review as the audit trail.
- Refresh the deployed skill record on the next Linux `--refresh` run.
- `dotf spec archive HARNESS-093-reviewer-pool-random` is **advisable** in the current state: fresh passing review at `acca2da`, no blockers or majors, rubric all B or above.
