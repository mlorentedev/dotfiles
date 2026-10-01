---
spec: "AI-044-move-off-qwen38-quota"
verdict: "FAIL"
reviewed_sha: "6d1393d7ec4fc3a2c0b8d2c7454354f63fc0f3d2"
reviewer: "nan/deepseek-v4-flash"
date: "2026-09-30"
---

## Adversarial review

**Scope**: AI-044-move-off-qwen38-quota (opencode/model-map NaN routing change, #1762 / PR #1772)
**Sources**: `specs/AI-044-move-off-qwen38-quota/{proposal,tasks,verification,features}.md/json`; `git diff 2e73eab34510fdfe0573f207e783bd352ebc02f7...HEAD` (launcher-resolved base = 2026-09-26 `ci: require every pull request to name the knowledge it produced (#1759)`); the spec's own commit `75eb855` (9 files) plus the ~45 later commits the resolved base pulls in. Local commands run: `bats -f 'default model' tests/opencode.bats`, `bats -f 'context window' tests/opencode.bats`, `bats tests/opencode.bats`, the `features.json` f3 `jq` check, and a reverted mutation of `ai/opencode/opencode.jsonc`.

### Spec and task alignment

- **AC1** (opencode `model` = `nan/glm5.3-flash`, `small_model` = `nan/qwen3.6`): implemented in `ai/opencode/opencode.jsonc` (`model`, `small_model`, `agent.plan`, provider `options.model`) and held by a test whose recorded command still works: `bats -f 'default model' tests/opencode.bats` → `1..1 ok 1`, exit 0. **Verified.**
- **AC3** (no tier/chain head on `qwen3.8-flash`; rerank service named `rerank`): `harness/model-map.json` `tiers.low.nan = qwen3.6`, `chains.low = [nan:qwen3.6, nan:glm5.3-flash, claude:haiku]`, `services.rerank.model = "rerank"`; the recorded `jq -e` returns `true`, exit 0. Grep finds `qwen3.8-flash` in the tree only as picker/catalog entries, `$comment` text, quotas and the reviewer pool — no routing head. **Verified.**
- **AC2** (opencode and pi declare the same context window for every NaN model both carry): the *substance* survives but the *recorded proof* does not. The test named in `features.json` f2 was moved by AI-046 (`c114094`) out of `tests/opencode.bats` into `tests/pi-nan-package.bats`, under the same `@test` name. The contract's own verification command now selects **zero** tests. See findings F1/F2.
- `tasks.md`: all implementation boxes `[x]`, and each maps to lines in the diff; the branch-naming line is cosmetic (the work is on a launcher-created worktree, not `fix/nan-quota-default-model`). No `[AGENT-DRAFT]`/`[AGENT-SUGGESTION]` tags remain in the spec folder — the archive's own tag gate is satisfied.
- The 233-file diff is **not** the spec's change: `git show --stat 75eb855` is 9 files, matching the proposal exactly. Everything else is mainline work the resolved base drags in (AI-046, AI-047, secrets curation, ADRs 041/042). I reviewed the AI-044 surface plus the later changes that interact with it — which is where the one blocking defect lives.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Major | REAL | Verification integrity (contract set) | AC2's recorded evidence command selects nothing and exits 0. `features.json` f2 claims `state: passing` on `bats -f 'context window' tests/opencode.bats`, but AI-046 moved that test to `tests/pi-nan-package.bats` and left a tombstone comment in its place. The recorded command cannot fail and the recorded `state` is therefore unfalsifiable. | Reproduced at HEAD: `bats -f 'context window' tests/opencode.bats` prints `1..0`, exit 0 (a filter matching no test). `grep -rn "context window" tests/` → only `tests/pi-nan-package.bats:96` carries the assertion; `tests/opencode.bats:158` is a comment. This is the repo's own lesson `lesson-309-a-test-filter-that-matches-nothing-passes.md`, added in this same diff. | `tests/pi-nan-package.bats` → `@test "pi-nan-package: opencode.jsonc and the package snapshot declare the same context window for every NaN model both carry"` — named, but CI-only (`PI_BIN` + network; `.github/workflows/ci.yml:329` runs it with `PI_NAN_PACKAGE_REQUIRED=1`). The recorded command reaches neither. | **contract set**: `features.json` f2 `verification` (edit ⇒ re-review) |
| Major | REAL | Guard coverage (test deletion) | The parity property is no longer guarded by anything that runs in the default local suite. A reverted mutation of `ai/opencode/opencode.jsonc` setting all three `"context": 262144` to `999999` leaves the whole opencode suite green. AI-046's deletion bar (`adversarial-review` § Test deletions, field 2: "the stronger proof that remains, **shown**") is met only in CI; nothing at repo level fails when opencode's NaN windows drift. | Mutation run and reverted (`git checkout -- ai/opencode/opencode.jsonc`; tree clean afterwards): `bats tests/opencode.bats` → `1..43`, no `not ok`, exit 0. The package test cannot be run in this session (it installs `@gtrabanco/pi-nan-provider` from npm), so its own passing state is **UNVERIFIED** here. | Same named test as F1 — CI-only, so locally `UNTESTED`. | **tests** (add an offline parity guard, e.g. against the pinned package's committed snapshot or a checked-in snapshot) |
| Minor | REAL | Spec/code mismatch (non-contract) | `verification.md` AC2 names the wrong artefact as pi's side: "test `opencode.jsonc and ai/pi/models.json declare the same context window…`". AI-046 removed the entire `nan` provider block from `ai/pi/models.json` (this diff); pi's NaN ids now come from the pinned `@gtrabanco/pi-nan-provider` package in `ai/pi/packages.json`. A reader re-verifying AC2 is pointed at a file that no longer carries the data. | This diff, `ai/pi/models.json`: the 125-line `providers.nan` block is deleted; `grep -n '\"nan\"' ai/pi/models.json` finds no provider. `tests/opencode.bats:158` already redirects to `tests/pi-nan-package.bats`. | Same named test as F1. | **spec** (`verification.md` — outside the contract set, editable without invalidating this review) |
| Minor | SPECULATIVE | Reliability | The new `model-map` low chain falls back `nan:qwen3.6 → nan:glm5.3-flash → claude:haiku`; if `glm5.3-flash`'s 2B monthly quota is spent mid-month the chain falls to `claude:haiku`, and no alarm exists yet to signal either event (AI-047, #1766, declared out of scope). | Read of `harness/model-map.json` `chains.low` + `proposal.md` "Out of scope". No reproduction. | `UNTESTED` | none — surfaced only; explicitly deferred to AI-047, do not gate |
| Question | — | Scope | The launcher-resolved base covers ~45 commits beyond the spec's own, so the reviewed diff (233 files, ~9.6k insertions) is much wider than the change under review. I attributed the AI-044 surface from `75eb855`; if that attribution is wrong, say so. | `git log --oneline 2e73eab..HEAD` (46 commits) vs `git show --stat 75eb855`. | n/a | none (process observation for the launcher, not the spec) |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | B | AC1 and AC3 verified by commands that run and pass; AC2 holds in substance but its recorded proof is vacuous. |
| Verification       | C | One of three recorded evidence commands matches zero tests (`1..0`), and the AC2 evidence names data that moved; AC1/AC3 are reproducible, so not a D. |
| Scope              | B | The spec's own commit is 9 files matching the proposal exactly; the wider diff is an artefact of the resolved base, not creep by the author. |
| Reliability        | B | Config/routing change with no error paths of its own; the low-chain fallback is sane, its quota alarm deliberately deferred to AI-047. |
| Maintainability    | A | `opencode.jsonc` comments record the *why*, the date and the successor model; the deleted test left a tombstone naming where it moved. |
| Handoff-readiness  | C | Spec updates were included but have gone stale under a later spec: the recorded evidence command and the `ai/pi/models.json` pointer both misdirect a re-verifier. |

### Verdict
FAIL

Rubric alone (two Cs, no D) would floor this at PASS WITH GAPS; the severity × reality axis escalates it: **F1 is a REAL Major inside the contract set**, and the skill's rule is that a REAL Major in `proposal.md`/`tasks.md`/`features.json` means fix-then-re-review, not track-and-pass. The change itself is sound — the routing move does what the proposal says, and the parity check it relies on exists and is stronger than the one it replaced. What fails is the evidence chain: the archive would record "AC2 passing, exit 0" via a command that can no longer fail.

### Recommended next steps

1. **Contract set (the blocker, ⇒ re-review)** — `features.json` f2: replace `bats -f 'context window' tests/opencode.bats` with the test that actually carries the assertion, and record where it runs (the `pi-nan-package` CI job; it skips without `PI_BIN`). Editing `features.json` changes its digest, so the next round of this review is the mechanism, not an inconvenience.
2. **`verification.md` (non-contract, free to edit)** — rewrite AC2's evidence line to name `tests/pi-nan-package.bats` and the pinned `@gtrabanco/pi-nan-provider` package in `ai/pi/packages.json`, not `ai/pi/models.json`. Add the mutation evidence (three `262144 → 999999` windows, 43/43 green) as the reason a local guard is still wanted.
3. **Tests (follow-up ticket, not a blocker)** — add an offline guard that fails when `ai/opencode/opencode.jsonc`'s NaN context windows drift from a committed snapshot of the pinned package, so the property is enforceable without network and without CI. Today nothing in the default suite notices.
4. **`dotf spec archive` is NOT advisable** in this state: the archive gate is satisfied mechanically (pool-built review, no `[AGENT-DRAFT]` tags, promotion candidates answered), but the verdict is FAIL, so the archive would persist a contract whose AC2 evidence is unfalsifiable. Flip to PASS by completing step 1 and re-running the review; step 2 can land with it.
5. **Unreached here (UNVERIFIED, not asserted either way):** the full-suite claim in `verification.md` (`1655/1656 ok`, test 1257 red on main too) was not re-run inside the time budget; `bats tests/opencode.bats` (43/43) was run and passed. Separately, `dotf pr triage-queue` reports pending CodeRabbit output on **PR #1824**, which is unrelated to AI-044 and was not dispositioned by this review.
