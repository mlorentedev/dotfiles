---
spec: "CLI-128-lane-claim-ref"
verdict: "FAIL"
reviewed_sha: "0aa64cc2764445b07f237d44d48bba308a3eccd4"
reviewer: "nan/mimo-v2.6-flash"
date: "2026-10-16"
---

## Adversarial review

**Scope**: CLI-128-lane-claim-ref — `git diff 6f7aba20bc75959fe58e843915a63ea61babbfc9...HEAD` (base resolved by the launcher), i.e. the three spec commits (#2624, #2625, #2627) plus three unrelated PRs (#2622, #2623, #2582) interleaved on main inside that range.

**Sources**: `specs/CLI-128-lane-claim-ref/{proposal,tasks,verification}.md`, `features.json`, `review-request.json`; `cli/internal/lane/{claim,handoff,queue}.go` + tests; `cli/internal/cmd/{lane,lane_claim,lane_claims}.go` + tests; `docs/adr/adr-050-*.md`; `docs/lessons/459,460`.

### Spec and task alignment

- **Contract integrity**: the normalised digests of `proposal.md`, `tasks.md`, `features.json` recomputed with the repo's own `ContractDigests` match `review-request.json` byte-for-byte — the contract set is exactly what the launcher recorded at launch (probe test written, run, removed).
- **AC1–AC3, AC5, AC6**: each has a real-git named test and passes on demand — all four `features.json` verification commands run green in this session (`TestClaimAcquire`, `TestClaimedErrorNamesTheHolder`, `TestClaimConcurrent` (+5 repeat runs), `TestClaimIsInvisibleToADefaultClone`, `TestLaneRelease`).
- **AC4**: proven at the `lane.Handoff` layer by seven named tests (skip-on-loss, fail-closed, no-claim-on-existing-worktree, dry-run, give-back, keep-after-worktree-exists, ordering `acquire → create`). The **production** half — the single line that hands the claimer to `lane next` — is unguarded (F1, below).
- **AC7**: the probe test exists, skips without `DOTF_CLAIM_PROBE_REMOTE` (verified), and passed against a local bare remote in this session's suite runs; ADR-050 decision 2 is amended in the diff to "Decided 2026-10-17" naming GitHub and Gitea; `git ls-remote origin 'refs/dotf/*'` on the live origin is empty, corroborating the "no ref litter" claim. The live GitHub/Gitea probe runs themselves are implementer-asserted only (6.9s / 14.9s) — **UNVERIFIED** by this reviewer (no authorization to push probe refs; see F3 for the contradiction inside `verification.md`).
- **Non-goals respected**: autopilot and other harness skills untouched by the spec's commits; no expiry/takeover logic; queue order unchanged (`nextFree` only skips); no release-on-land hook (`Release` callers are handoff give-back and the `lane release` verb only).
- **Verification evidence run in this session**: `go build ./...`, `go vet ./...`, `GOOS=windows go vet ./internal/{lane,cmd}/...`, `golangci-lint run ./internal/{lane,cmd}/...` (repo config) → 0 issues, full `go test ./...` → ok (all packages), `bats tests/roster-consistency.bats tests/herdr-config.bats` → 12/12, `bats tests/skills-pipeline.bats tests/compile-harness.bats` → exit 0, `bats tests/verify-setup.bats` → 73 ok / 0 fail. `tests/copilot-native-skills-smoke.ps1` (in range from #2623) is Windows-only → **UNVERIFIED** on this host.
- **Mutation checks (all reverted; `git status` clean apart from the launcher's `review-request.json`)**:
  - M1 — disable the claim in `Handoff`'s path → 7 handoff tests fail (AC4 wiring inside the lane package is guarded).
  - M2 — drop the claim nonce → `TestClaimAcquire` fails (lesson 459 is load-bearing).
  - M3 — drop the exact-ref match in `remoteSha` → `TestClaimIgnoresARefThatOnlyEndsLikeTheClaim` fails.
  - M4 — delete `Claims: laneClaimer(cfg, top, parent)` from `doLaneNext` (`cli/internal/cmd/lane.go:147`) → **entire `go test ./internal/cmd/` suite still green** (F1).
- **Manual probe**: with a `pre-push` hook exiting 1 installed, a plain `git push` was refused ("PRE-PUSH HOOK RUNNING") while `dotf lane claim 2551` succeeded — the `--no-verify` bullet holds (no named test covers it, F5).
- **Draft tags**: none remain in the spec's contract or verification files.
- **tasks.md / proposal.md checkbox drift**: PR 1's seven task boxes (and the branch box) are unticked although PR 1 shipped and `verification.md` marks AC1/AC2/AC3/AC6 proven; proposal AC1–AC6 are unticked while AC7 is ticked (F2).

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Major | REAL | tests / wiring | The **sole production construction** of `Options.Claims` — `doLaneNext`, `cli/internal/cmd/lane.go:147` — has no test that fails when it is removed. Delete the line and `lane next` (and `lane apply`'s `Next` verb, which funnels through `doLaneNext` at `lane_apply.go:74`) silently stop claiming, restoring exactly the two-sessions-one-lane race this spec exists to remove — with a green suite. This is the shape lesson 457 was written from (CLI-127 review round 2 returned FAIL for `Sessions:`/`Forge:` deletions leaving the suite green), and lesson 457 is cited by this spec's own proposal | Mutation M4 this session: line deleted → `go test ./internal/cmd/` → `ok` (whole package); `grep Claims:` shows one production site; lesson 457 "the arguments easiest to drop are the ones whose absence is silent"; the new dry-run stub (`ls-remote origin refs/dotf/claims/5`) is consumed *if asked* but never asserted in `asked` | UNTESTED (closest: `TestLaneNext_DryRunNamesTheNextIssueAndChangesNothing` pins the command shape only when the wiring already runs; compare `TestRealLaneVerbsSweepHonoursTheForgeAndTheSessions`, which does pin the CLI-127 wiring) | tests — one positive assertion that `laneNextDryRun`'s `asked` contains the `refs/dotf/claims/` probe, or a `realLaneVerbs`-style wiring test on `doLaneNext` |
| Minor | REAL | spec (contract set) | Checkbox state misreports completed work: `tasks.md` PR 1's seven boxes and the branch box are `[ ]` although the primitive shipped, and `proposal.md` leaves AC1–AC6 unticked while AC7 is ticked — a reader or gate treating boxes as status sees the opposite of `verification.md` | `tasks.md` and `proposal.md` as read in this session; `verification.md` marks the same items `[x]` with evidence | n/a (document state) | spec — fold into the re-review round this FAIL already requires; do **not** touch after a passing verdict (contract set) |
| Minor | REAL | spec / verification | `verification.md` contradicts itself on AC7: the evidence block records `TestClaimRefFormOnARemote` passing on GitHub (6.9s) and Gitea (14.9s) with the ADR amended, while the closing paragraph says "GitHub and Gitea are still unprobed and AC7 stays open until the owner runs it" | both passages quoted verbatim in `verification.md`; the ADR diff (`Decided 2026-10-17`) sides with the evidence block | n/a | verification.md — outside the staleness set, free to edit; delete or date-stamp the stale paragraph |
| Minor | REAL | quality | `handoffIssue` is 57 physical lines (>40) with gocyclo complexity **18** (>10 per AGENTS.md Code Quality Rules, and >15, the rubric's C row); `Claimer.List` is 11 (>10). Repo lint does not catch it (`.golangci.yml` runs gocyclo only on `internal/cmd`, threshold 22), so this passes CI while breaking the repo's stated Law | gocyclo via temp config (min-complexity 10): `handoff.go:110: cyclomatic complexity 18 of func handoffIssue`; `claim.go:174: ... of func (Claimer).List`; `awk` line counts 57 / 39 | n/a (measured) | code — outside the contract set; split `handoffIssue` before/after the claim step |
| Minor | UNTESTED | hooks | The `--no-verify` claim-push (proposal: "skips the repository's pre-push hooks") has no named test | reviewer probe this session: failing pre-push hook refused a normal push, `dotf lane claim 2551` still succeeded | UNTESTED (manual probe shown above) | tests — a bare repo with a failing `pre-push` hook and one `Acquire` |
| Minor | SPECULATIVE | identity | `who()` maps control characters to `?`, so two worktrees whose paths differ only in a control character collide as one holder and one could release the other's claim without `--force` | `claim.go who()`; `TestClaimHolderWithControlCharactersRoundTrips` proves round-trip, not collision freedom | `TestClaimHolderWithControlCharactersRoundTrips` (partial) | code — surface only; claims are coordination, not an authz boundary |
| Minor | THEORETICAL | resilience | `Claimer.List` aborts the whole `lane claims` listing if a claim ref is deleted on the remote between `ls-remote` and `fetch` | code read of `List` (readSeen error → `return nil, err`); not reproduced | `TestClaimList` (happy path) | tests — surface only |
| — | — | scope (disclosure) | The review range contains three merged PRs unrelated to this spec (#2622 secrets, #2623 roster, #2582 deploy) because they interleaved on main between the launcher's base and HEAD. The spec's own three commits touch only `cli/internal/{lane,cmd}`, spec files, ADR and lessons — no harness/autopilot edits, matching the out-of-scope declaration. Their tests were run and pass (go suite + bats above); the Windows-only `.ps1` is UNVERIFIED | `git show --stat` per commit; test runs above | n/a | none — disclosed, not a defect |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | B | Shipped behavior meets AC1–AC7 with strong negative paths (fail-closed, look-alike refs, force-release, control chars), but AC4's command-level wording is only proven against an injected fake, not the command. |
| Verification       | B | Every AC has a named test and all four `features.json` commands reproduce on demand, but the evidence does not cover the one production line wiring `lane next` (F1). |
| Scope              | B | The spec's three commits match the proposal exactly; three unrelated merged PRs ride the launcher-set range (disclosed, their tests green). |
| Reliability        | B | Fail-closed everywhere, release is lease-pinned and idempotent; minor gaps are `List` aborting wholesale on a mid-read deletion and one `gh` call per claim in `lane claims`. |
| Maintainability    | C | New production `handoffIssue` measures 57 lines / gocyclo 18 — above the rubric's 15 and the repo Law's 10/40 — though naming, comments and test structure are otherwise exemplary. |
| Handoff-readiness  | B | ADR amended, lessons 459/460 added with index entries, decisions recorded; blemished by the self-contradictory AC7 paragraph and the unticked contract checkboxes. |

### Verdict

**FAIL** — one **REAL Major** (F1): the production wiring of the spec's central behavior is mutation-undetected, reproduced in this session (M4) and matching the class that failed CLI-127's review round 2 (lesson 457, cited by this proposal). The test-traceability gate independently bars PASS: the finding is **UNTESTED** and needs a named regression test. The rubric floor is PASS WITH GAPS (two Cs, no D), so the severity path governs. No Blocker: the shipped code is correct as written, and no acceptance criterion fails when run.

### Recommended next steps

**Minimum set that flips this to PASS (fix, then re-review — a contract-set edit is legitimate on a FAIL):**

1. **tests**: add the wiring guard — the cheapest correct form is one positive assertion in `laneNextDryRun` that `asked` contains the `refs/dotf/claims/5` probe (the stub already exists), or a dedicated `doLaneNext`-level test in the spirit of `TestRealLaneVerbsSweepHonoursTheForgeAndTheSessions`. Re-verify with the M4 mutation: deleting `Claims:` must now turn it red.
2. **spec (contract set, same round)**: tick `tasks.md` PR 1's completed boxes and `proposal.md` AC1–AC6 (or annotate why a box stays open); optionally reconcile proposal's "`lane status` shows it" clause with tasks PR 3's documented replacement by `lane claims`.
3. **verification.md (free — outside the staleness set)**: resolve the AC7 contradiction by deleting or date-stamping the stale "still unprobed" paragraph.

**For a plain PASS rather than PASS WITH GAPS:** clear the Maintainability C — split `handoffIssue` (gocyclo ≤15, ideally ≤10; ≤40 lines).

**Track as dispositions (not gating):** F5 (hook-skip named test), F6 (`List` mid-read deletion), F7 (`who()` collision) — record each in `verification.md` as applied / ticketed / declined with a reason, or fold F5 into the fix round since the probe is two lines.

**(a) Verdict:** FAIL. **(b) `dotf spec archive` / `/spec archive` is NOT advisable** in the current state — the gate will refuse (and a FAIL verdict must not be overridden). **(c) Minimum to flip:** item 1 above plus a fresh review round; items 2–3 ride that round, and the Maintainability split (for full PASS) can ride it too since the code files are outside the contract set.
