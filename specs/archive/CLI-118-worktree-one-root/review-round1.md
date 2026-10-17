---
spec: "CLI-118-worktree-one-root"
verdict: "FAIL"
reviewed_sha: "c14994becc37f730a79af400dfaf97d52d81e465"
reviewer: "nan/qwen3.8-flash"
date: "2026-10-16"
---

## Adversarial review

**Scope**: CLI-118-worktree-one-root / worktree one-root + `dotf worktree` orchestration
**Sources**: `specs/CLI-118-worktree-one-root/{proposal,tasks,verification,features}.md`; diff `58909de7...c14994be` (59 commits — the spec's 16 worktree/herdr/trust/doc commits, plus interleaved landed work reviewed below under Scope)

Fresh evidence produced in this session (not quoted from `verification.md`):
- `go build ./... && go vet ./... && go test -count=1 ./...` — 42 packages ok, 2 no-test-files, zero failures.
- `GOOS=windows go vet ./...` — exit 0 (AC11's Go leg, re-verified fresh).
- `bats tests/herdr-config.bats` (6 ok, incl. the root-equality test #2503) and `tests/env-contract.bats` — pass fresh.
- Mutation ×2, reverted, tree clean: (a) `slugifyBranch` without `ToLower` → `TestResolveWorktreePath`/`TestAcceptedBranchesSlugToThemselves` red; (b) `branchLanding` ancestor check forced true → `TestDoneKeepsAnUnlandedBranch*` red. Both killed, both reverted green.
- Named-test existence sweep over every test cited in `verification.md`: all exist except `TestClassify` (AC5) and `TestClassifyUnadopted` (features.json) — see finding F3.

### Spec and task alignment
- PR 1–7 map cleanly onto the commits (`ff0b1612 #2527`, `fdfef1f3 #2539`, `ac5208a3 #2556`, `57355aa2 #2562`, `1c6a204d #2563`, `6153adf7 #2564`, `d9a8ad9d #2565`, `6543c830 #2575`, `1ac92409 #2568`, `f0deadf3 #2536`, `00163516 #2580`, `94c74b9f #2581`, …). `tasks.md` boxes match the diff.
- AC2/AC3/AC5/AC6/AC7/AC8: verified by named passing tests plus my two fresh mutations. `done` prints the deleted SHA (`cmd/worktree.go:644`); `add` preflights herdr before any git mutation (`cmd/worktree.go` RunE order: `prepareAgent` → `worktree.Add` → `herdr.Launch`); slugify is traversal-safe by construction.
- **AC1 is not met as written** — see F1, the Blocker.
- AC9 partially verifiable from this repo (AGENTS.md §9 states the new layout ✓; vault-sourced skills are outside the repo — **UNVERIFIED** from here). AC10 ✓ (ADR-032 carries the cockpit/dispatch paragraph, line 397).

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Blocker | REAL | AC1 / deploy | `WORKTREE_ROOT` is never rendered into herdr's config: the `herdr` entry in `ai/deploy.json` has no `paths:` key, `ai/herdr/config.toml` hardcodes `directory = "~/Projects/worktrees"`, and the manifest is version 4 while TOML paths are gated at 6. On any machine where `WORKTREE_ROOT` is overridden via `machine.json` (ADR-025's own mechanism), dotf and herdr again create two different trees — the exact divergence the spec exists to remove — and the pinning test compares only the *default*, so it stays green. `verification.md` AC1 is left `[ ]` ("Rendering into the three configs is PR 7") and `features.json` records `state: "pending"` for exactly this behavior; PR 7 shipped the reader and the two JSON trust lists, but not the herdr wiring. | `grep WORKTREE_ROOT ai/herdr/config.toml` → none; deploy.json `herdr` entry lacks `paths`; `expandPathsAs` only runs when `c.Paths != ""` (`deploy.go:771`, gate at 281); bats test 6 reads `.default.linux` only. | `tests/herdr-config.bats` "WORKTREE_ROOT and herdr's worktrees.directory name the same root" covers the default only; the override path is **UNTESTED** | code (`paths` on the herdr entry + manifest v6 bump + an override test) **or** spec — amend AC1 to "kept equal to the default by a test" with the owner's waiver recorded. Both routes touch the contract set (AC1/features) → fix, then re-review. |
| Major | THEORETICAL | AC7 | `--ignore-recent-session` exists on `done` only; AC7's wording gives it to `done`, `sweep` **and** `migrate` ("refuse … unless --ignore-recent-session is passed"). The implementation fail-closes (sweep/migrate skip with the reason printed) and the deviation is an explicit recorded decision in `verification.md`. Safer than spec, but spec says X, code does Y — flagged, not silently accepted. | `cmd/worktree.go:655` (flag registered on done); `sessionSkipReason` comment "names no override, since neither command has one" | `TestMigrateSkipsWhatItMustNotMove` asserts no skip *mentions* the flag (fails closed on the old wording); the asymmetry itself is pinned by `TestWorktreeDoneRefusesARecentSessionUntilIgnored` | spec (align AC7 wording to the recorded decision — contract edit, belongs with the F1 re-review round, not with this verdict) |
| Minor | REAL | verification artifacts | `features.json` verification command `go test -run 'TestAdopt\|TestClassifyUnadopted'` names a test that does not exist — the regex silently runs fewer tests than the contract claims. `verification.md` AC5 cites `TestClassify`; the actual functions are `TestClassifyActiveStates` / `TestClassifyReapableAndUnmerged` / `TestClassifyNamesIgnoredContent`. The UNADOPTED *behavior* is covered (`worktree_test.go:152` region, `TestWorktreeAdoptTakesAGitCreatedWorktree`), so this is citation drift, not a coverage hole. | session sweep: `grep -rl 'func TestClassifyUnadopted'` → none | named coverage exists under other names | spec (features.json = contract; verification.md is free) — fold into the F1 round |
| Minor | THEORETICAL | herdr drift guard | The proposal's risk mitigation "dotf doctor should report a herdr whose `api schema` lacks `WorktreeOpenParams.path`" is implemented nowhere (`grep -r WorktreeOpenParams cli/ docs/ tests/` → 0 hits) and no `tasks.md` PR carries it. Version-drift on herdr's side is currently caught by nothing. | grep above; tasks.md has no doctor row | UNTESTED | code + tests, or ticket it (track-or-fix) |
| Minor | SPECULATIVE | add --agent | A `--prompt-file` that exists but trims to empty makes `Launch` silently skip the prompt step (`o.Prompt == ""` → return after start) — `add` validated the file, then sends nothing, and reports only "Started … agent". Also `AgentName`'s 32-char cut can collide for two long slugs; that leg fails closed via the name-taken refusal, so it is a false-refusal annoyance, not a hazard. | `herdr/agent.go` Launch (prompt=="" branch), `cmd/worktree.go:354-360` | `TestLaunchWithoutAPromptOnlyStarts` pins the no-prompt behavior *without* `--prompt-file`; the empty-file leg is UNTESTED | code + tests (refuse or warn on an empty prompt file) |
| Minor | SPECULATIVE | sessions TOCTOU | No check closes the window between reading a session store and removing the directory; `verification.md` states this honestly ("it only narrows it"). The process-ancestor check covers live sessions; a session that starts in the window after all checks is stranded. | done.go check order (session check last, under the dotf lock) | UNTESTED (window not deterministically reproducible) | — (surface only; documented, lowest-cost mitigation already taken) |
| Question | — | scope of the window | The launcher's base puts 59 commits in scope, of which ~15 belong to other specs that landed on main during CLI-118's window: CLI-119/CLI-127 lane work (both archived with their own `review.md`), the 0.71.0/0.72.0 releases, and `65140ae1 #2601` — the ignored-content change to `Dirt`/`done` that this proposal explicitly lists under "Out of scope … unchanged here". I graded CLI-118 on its own commits; #2601 arrived with its own PR trail and test (`TestClassifyNamesIgnoredContent`), so I read the window as a base-resolution artifact, not as spec creep. Confirm the owner agrees the window is coarser than the spec. | `git log 58909de7..HEAD` | n/a | — (human confirmation) |

Guardrail check: no `[AGENT-DRAFT]`/`[AGENT-SUGGESTION]` tags in any spec file. Working tree apart from the launcher's `review-request.json`: clean.

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness | C | AC1's core mechanism (rendering into herdr's config) is not wired; every other criterion held under fresh tests, my two mutations and negative-path reading. |
| Verification | B | Named tests + mutation evidence I independently reproduced (2/2 killed, suite green fresh); but stale test citations (F3) and open AC checkboxes admit the AC1 gap. |
| Scope | B | The spec's 16 commits match the proposal; other specs' landed work shares the window (F7), and one window commit touches a declared non-goal under its own PR. |
| Reliability | B | Consistently fail-closed (unreadable store refuses, ambiguous slug refuses, preflight before mutation, branch kept unless landing verified); the session TOCTOU window is documented, not closed (F6). |
| Maintainability | B | Small functions, WHY-comments throughout, table-driven tests, planning-token regex kept in step with the enforced rule by test; no dead code found (`agentNameShape` used in tests). |
| Handoff-readiness | C | verification.md leaves 7 of 11 AC lines unchecked, the Promotion candidates block is unanswered template placeholders, and the archive checklist is unticked — `dotf spec archive` will refuse on those even before this review does. |

### Verdict
**FAIL** — one REAL Blocker (F1: AC1's herdr rendering unwired; a Blocker stops archive at any reality), plus two C rubric lines. F2 (Major, THEORETICAL) and the minors would otherwise have supported PASS WITH GAPS.

### Recommended next steps
Minimum to flip to PASS (contract-set edits ride on this one re-review round):
1. **Resolve F1 either way, by owner decision:** wire `paths` into the `herdr` entry of `ai/deploy.json` (`{WORKTREE_ROOT}` in `config.toml` + manifest bump to version 6, accepting the documented stale-reader consequence), **or** amend AC1 to "herdr's directory is kept equal to the contract *default* by test" with the override divergence explicitly waived — and in both cases make `tests/herdr-config.bats` speak to the override path (currently UNTESTED). Re-review after.
2. Tick or negate the unchecked AC lines in `verification.md` (free, outside the contract set) and answer the Promotion candidates block — the archive gate refuses placeholders regardless of this review.
3. Correct `TestClassify`→`TestClassifyActiveStates` in `verification.md`; fix the `TestClassifyUnadopted` regex in `features.json` in the same round as (1) so it does not cost an extra invalidation.
4. Ticket the missing `dotf doctor` herdr schema guard (proposal Risks, F4) and the empty-`--prompt-file` leg (F5) rather than leaving them verbal.
5. F2: fold the AC7 wording alignment into the round from (1); do not edit the contract files alongside a passing verdict later.

**Is `dotf spec archive` advisable now: NO.** FAIL verdict on F1; fix (or formally waive) AC1, then re-review.
