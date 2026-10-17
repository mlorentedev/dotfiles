---
spec: "CLI-118-worktree-one-root"
verdict: "PASS-WITH-GAPS"
reviewed_sha: "6c2c3bcde9a7b126cd4c96cd8cec9bda26622c4e"
reviewer: "nan/qwen3.8-flash"
date: "2026-10-16"
---

## Adversarial review

**Scope**: CLI-118-worktree-one-root / round 2. Whole change `58909de7...6c2c3bcd` (274 files, 59 commits), with the
delta since the round-1 FAIL at `c14994be` (`378581a8..6c2c3bcd`) reviewed as the part that must be judged against
the early work it interacts with.
**Sources**: `specs/CLI-118-worktree-one-root/{proposal,tasks,verification,features}.md`, `review-round1.md`;
diff `git diff 58909de7...HEAD`; production delta `ca267544` (#2582, the F1 fix) and `6c2c3bcd` (spec close-out).

Fresh evidence produced in this session (nothing below is quoted from `verification.md` without a re-run):
- `go build ./...` exit 0; `go test -count=1 ./...` — **42 packages ok, 0 FAIL**.
- `bats tests/herdr-config.bats` — 6 ok, including test 1 (`herdr config check` accepts the source that now carries the
  literal `{WORKTREE_ROOT}`) and test 6 (the F1 guard).
- **Mutation M1 (the F1 regression, reverted, tree clean):** deleted `"paths": "native"` from the `herdr` entry of
  `ai/deploy.json` → `TestDeploy_TheShippedHerdrEntryKeepsHerdrsOwnKeys` fails with
  `worktrees.directory = {WORKTREE_ROOT}, want WORKTREE_ROOT rendered as "/r/worktrees"` **and** bats test 6 fails
  (`herdr entry renders no paths: null`). Both green again after `git checkout`. The round-1 Blocker is fixed with a
  guard that actually bites.
- **Probe (new, temporary test file, deleted after):** rendered `WORKTREE_ROOT` = `C:\Users\runneradmin\Projects\worktrees`
  through `expandPathsAs(tomlPaths, …, PathsNative, …)`. go-toml emits a TOML *literal* string
  `directory = 'C:\Users\runneradmin\Projects\worktrees'` and it re-parses to the same value — so the `paths: native`
  choice on a TOML source does **not** produce the `\U` escape corruption a basic string would. UNTESTED in the committed
  suite on Windows for the *shipped* value (`TestDeploy_TheShippedHerdrEntryKeepsHerdrsOwnKeys` asserts
  `filepath.FromSlash("/r/worktrees")`, which exercises the round trip on `windows-latest`, but through the fake root).
- Version-floor claim checked against the repo, not believed: `git show v0.72.0:cli/internal/deploy/deploy.go` →
  `const ManifestVersion = 6` and `git show v0.72.0:ai/deploy.json` → `"version": 4`;
  `git merge-base --is-ancestor 1ac92409 a8f251b9` → YES (the reader shipped in 0.72.0), and the manifest bump
  `ca267544` is after the 0.72.0 release commit. So no box running an older dotf is handed a manifest it refuses —
  and the refusal is fail-closed anyway: `ManifestVersion`/`MinManifestVersion` gate (`deploy.go:246`), plus the
  per-entry gate `paths` on TOML needs version 6 (`deploy.go:279-281`), pinned by
  `TestParseManifest_PathsOnATOMLSourceNeedsVersion6` (v3/v4/v5 refused naming 6, v6 accepted).
- One oracle on both sides, verified in code: `dotf deploy` renders with `env.ResolvePath` (`cmd/deploy.go:142`) and
  `dotf worktree add|done|migrate` resolve the root with `env.ResolvePath("WORKTREE_ROOT")`
  (`cmd/worktree.go:116,290`), which consults `machine.json` before the contract default
  (`env.ResolvePathUnder`, and `$env:USERPROFILE` is expanded for Windows — `env/resolve.go:40`).
- Contract digests reproduced with `spec.ContractDigests` at HEAD: `features.json 32aabc83…`, `proposal.md 861ae6cc…`,
  `tasks.md 31c844ce…` — identical to `review-request.json`, so this verdict is against the state the launcher asked for.
- Test-name sweep for the features.json regexes fixed in `6c2c3bcd`: `TestClassifyActiveStates`,
  `TestClassifyReapableAndUnmerged`, `TestClassifyNamesIgnoredContent` exist (the bare `TestClassify` prefix matches
  them), so the AC5 verification command no longer silently runs fewer tests than it claims.
- Working tree apart from the launcher's `review-request.json` / `review-round1.md` and my own `review.md`: clean
  (`git checkout` restored the mutated file).

### Round-1 finding dispositions (verified, not asserted)

| Round-1 | Status now | Evidence |
|---|---|---|
| F1 Blocker — `WORKTREE_ROOT` never rendered into herdr | **FIXED** | `ca267544`: `directory = "{WORKTREE_ROOT}"`, entry `paths: native`, manifest v6; M1 kills it twice over (Go + bats); override path (root ≠ default) now asserted by `TestDeploy_TheShippedHerdrEntryKeepsHerdrsOwnKeys` |
| F2 Major — AC7 wording vs `--ignore-recent-session` on `done` only | **FIXED** | `proposal.md` rule 7 and AC7 now say the override is on `done`, `sweep`/`migrate` skip with the reason and no override; matches `cmd/worktree.go:655` (registered on `done` only) |
| F3 Minor — stale test citations (`TestClassifyUnadopted`) | **FIXED** | `features.json` regex `TestAdopt\|TestClassify\|TestSweepReapsAnAdoptedWorktreeOnly`; functions exist (sweep above) |
| F4 Minor — `dotf doctor` herdr schema guard (proposal Risks) | **OPEN** | `grep -rn WorktreeOpenParams cli/ docs/ tests/` → 0 hits; no ticket recorded in `verification.md` — see N2 below |
| F5 Minor — empty `--prompt-file` silently sends no prompt | **OPEN** | `herdr/agent.go:134` `if o.Prompt == "" { return nil }`; `cmd/worktree.go:355` `os.ReadFile(promptFile)` with no emptiness check; no named test for the empty-file leg |
| F6 Minor — session-store TOCTOU window | **OPEN, documented** | `verification.md`: "No check closes the gap … it only narrows it" |
| F7 Question — window coarser than the spec | unchanged; treated as a base-resolution artifact again (the other specs in the window each carry their own `review.md`) |

### Spec and task alignment
- AC1 is now met as written: declared once (`env-contract.json`, `WORKTREE_ROOT`, linux/windows defaults), rendered into
  herdr (`paths: native`), agy (`paths: slash`, `trustedWorkspaces` = the root itself) and Copilot (`paths: native`,
  `{WORKTREE_ROOT}/*/*`), all three carrying the token rather than a home literal, so they cannot diverge in the source;
  bats tests 5 and 6 fail if any of them drifts.
- AC2/AC3/AC5/AC6/AC7/AC8 hold as graded in round 1 and are unaffected by the delta: the delta touches
  `ai/deploy.json`, `ai/herdr/config.toml`, `cli/internal/deploy/merge_toml_test.go`, `tests/herdr-config.bats` and the
  spec folder only (`git diff --stat c14994be..HEAD -- cli/ tests/ ai/` shows no change under
  `cli/internal/worktree/` or `cli/internal/herdr/`), so round 1's fresh mutations and green runs still stand.
- AC4/AC9/AC10 unchanged from round 1 (live `herdr` run and the vault-sourced skills remain UNVERIFIED from here,
  as stated there). AC11: I re-verified `go test ./...` green here; the Windows leg runs in `cli.yml` (matrix
  `ubuntu-latest`/`windows-latest`); `GOOS=windows go vet` from the repo root is a no-op error (`./cli/...` is not a
  module path) — run from `cli/` it is the CI leg; my build+test run above is the fresh evidence.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|---|---|---|---|---|---|---|
| Major | THEORETICAL | AC / Windows long path | The proposal's own pre-condition for the Windows default — "Measure the deepest path of the largest repo against 260 before setting the Windows default", with "no admin, so long paths cannot be enabled" — has **no recorded measurement anywhere in the repo**, while `env-contract.json` already sets the Windows default `$env:USERPROFILE\Projects\worktrees` and `worktree.go` appends `<repo>/<branch-slug>`. The new `<root>/<repo>/<slug>` layout is deeper than the old `<parent>/<repo>-wt-<slug>` sibling by design (the layout is herdr's), so the risk the spec names is the one the change actually increases. No observed failure: the harm needs a deep branch slug on the corporate box. | `proposal.md` Risks line 1 vs `grep -rn "260" specs/CLI-118-worktree-one-root/ docs/lessons/` → nothing; `env-contract.json` windows default present; `worktree/root.go:15` `filepath.Join(root, repo, slugifyBranch(branch))` | UNTESTED (no guard measures depth; `TestResolveWorktreePath` pins shape only) | verification.md (record the measurement, per OS) **or** a ticket for a `dotf doctor` path-depth check — code/tests, not the contract set |
| Minor | REAL | round-1 debt not disposed | F4 (the declared `dotf doctor` herdr-schema guard) and F5 (empty `--prompt-file`) are neither fixed nor filed in `verification.md`. The track-or-fix standing order needs each detected defect to be fixed or ticketed; a mention in a closed review round is not an exit. F4's absence is verified (0 grep hits); F5's code path is verified present and unguarded. | `grep -rn WorktreeOpenParams cli/ docs/ tests/` → 0; `herdr/agent.go:134`, `cmd/worktree.go:355`; `verification.md` has no line naming either | F4 UNTESTED; F5: `TestLaunchWithoutAPromptOnlyStarts` pins the *no* prompt-file leg, the *empty* file leg UNTESTED | code + tests (F5: refuse or warn on a prompt that trims to empty); ticket (F4); dispositions written to `verification.md` (free, outside the contract set) |
| Minor | REAL | scope of the fix commit | `ca267544` bundles an orthogonal behaviour change into the F1 fix: the herdr entry also gains `listKeys: {"keys.command": "key"}` and the source drops its "Until #2455 …" caveat. That is the merge-identity mechanism (#2561, ADR-051), not AC1's rendering; nothing in `tasks.md` PR 7 or `verification.md` AC1 names it. It is tested (`TestParseManifest_ListKeys`, `TestMergeInto_ListKeysOnADottedKeyIsRefused`) and ADR-documented, so the risk is a reviewer attributing a herdr key-binding behaviour change to the worktree-root PR. | `git show ca267544 --stat`; `ai/deploy.json` herdr entry; `docs/adr/adr-051-a-merge-entry-records-what-it-manages.md` in the same window | named: `TestParseManifest_ListKeys`, `TestMergeInto_ListKeysOnADottedKeyIsRefused` | verification.md (one line naming the bundling) — code already covered |
| Minor | SPECULATIVE | deploy-time rendering vs later override | If `WORKTREE_ROOT` changes after a deploy (env or `machine.json`), `dotf worktree add` follows at once while herdr keeps the value rendered into `config.toml` at the last `dotf deploy` until the next deploy. Both sides read the same oracle (`env.ResolvePath`), so the window is a stale deployed file, not a different resolution rule, and the existing deploy-drift check should surface it — **I did not verify that this particular entry is in drift's scope** (UNVERIFIED). | `cmd/deploy.go:142` vs `cmd/worktree.go:290`; render happens inside `Deploy`/`PlanConfig` | UNTESTED for the "root changed after deploy" sequence (the tests render once) | tests (a drift case with a changed root), or confirm the doctor check covers it and record it in `verification.md` |
| Minor | SPECULATIVE | carry-over from round 1 | Session-store TOCTOU: no check closes the window between reading a store and removing the directory. Documented; the process-ancestor floor covers live sessions; mitigation cost exceeds the harm. | `verification.md` decision line; `done.go` check order | UNTESTED (not deterministically reproducible) | — (surface only; do not gate) |
| Question | — | fleet ordering | The v6 manifest is safe only because every box is at ≥0.72.0 (verified: 0.72.0 reads 6, the bump is after the release commit). A box still on ≤0.71 now refuses `dotf deploy` **entirely** — loud and fail-closed, and `dotf update` does not read the manifest so it heals — but that fleet state is a claim about machines, not about this diff, and I cannot check it from here. | `git show v0.72.0:…`, `tasks.md` PR 7 line, `deploy.go:246-247` | named: `TestParseManifest_RefusesWhatItCannotFullyRead` (the refusal shape) | — (owner confirms the fleet floor; already stated in `tasks.md`) |

Guardrail check: no `[AGENT-DRAFT]` / `[AGENT-SUGGESTION]` tag in `proposal.md`, `tasks.md`,
`verification.md` or `features.json` (the only grep hit is the launcher's own transcript of this skill).
All 11 AC boxes ticked; the Promotion-candidates block is answered (lessons 436/446/447, ADR-032, pattern `no:` with a
reason); the archive checklist is the only unticked block, which is `dotf spec archive`'s own to perform.

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|---|---|---|
| Correctness | B | All 11 ACs now have named passing tests; AC1 re-verified this round by a mutation that turns two independent guards red; residual gaps are the unmeasured Windows path depth and the empty-prompt leg. |
| Verification | B | Suite green fresh (42 ok, 0 FAIL), bats 6 ok, digests reproduced, my M1 + the TOML native-backslash probe all run here; but F4/F5 and the 260 measurement have no recorded disposition. |
| Scope | B | The delta is exactly the fix plus the close-out; one orthogonal `listKeys` behaviour change rides in the fix commit, tested and ADR-documented; the window still carries other specs' work (round-1 F7). |
| Reliability | B | Fail-closed at every new gate (per-entry v6 gate, whole-manifest floor, unresolvable token named as an error, ambiguous slug refused); the session TOCTOU window is narrowed, not closed, and stated. |
| Maintainability | B | Small functions, WHY-comments, table-driven tests, source-inspection guards in bats; golangci-lint not re-run this session, so the CC≤10 claim is carried from round 1 rather than re-measured. |
| Handoff-readiness | B | Round-1's C fixed: AC lines ticked, promotion block answered, lessons and ADR present, the override asymmetry written into AC7; what remains open is ticket hygiene on F4/F5, not lost knowledge. |

### Verdict
**PASS WITH GAPS** — no Blocker (F1 is fixed and its fix is guarded by a mutation I killed myself), no REAL Major;
one open Major is THEORETICAL (Windows path-depth evidence) and the rest are Minor, each tracked with a disposition
line above. Rubric: all B, no C, no D — the verdict is set by the open Major, not by the rubric.

`dotf spec archive` is **advisable** in the current state: the review is fresh (contract digests unchanged since
launch, `reviewed_sha` = HEAD), the reviewer is in `harness/reviewer-pool.json`, and no `[AGENT-DRAFT]` tag remains.

### Recommended next steps
All routes below are **outside the contract set** (`verification.md`, code, tests, tickets). On this verdict
`proposal.md`, `tasks.md` and `features.json` are closed: editing any of them invalidates this review, so carry them
into a follow-up ticket if one is genuinely needed rather than touching them now.

1. Disposition the open Major: record the Windows deepest-path measurement (repo + branch slug under
   `$env:USERPROFILE\Projects\worktrees`) in `verification.md`, or file a `dotf doctor` path-depth check with the root
   cause noted. Decide which, in writing.
2. File F4 (herdr `api schema` / `WorktreeOpenParams.path` guard) as an issue, and fix-or-ticket F5: refuse (or at
   minimum warn) when `--prompt-file` trims to empty, with a named test — `TestLaunchWithoutAPromptOnlyStarts` does
   not cover that leg.
3. Add one line to `verification.md` naming the `listKeys` bundling in `ca267544`, so a future reader of the
   worktree-root change does not attribute herdr key-binding behaviour to it.
4. Close the UNVERIFIED items if cheap to do: the drift story for a `WORKTREE_ROOT` changed after deploy (a test that
   renders twice with different roots and expects drift), and the vault-side of AC9 (readable only from the vault).
5. Re-run the pool's `--second` signature only if `proposal.md` declares `risk: high` — it does not; not required here.
