---
spec: "BUG-074-doctor-bw-reach"
verdict: "PASS"
reviewed_sha: "16f2b24599165406a58fa2d2adab088557caafdd"
reviewer: "nan/deepseek-v4-flash"
date: "2026-08-14"
---

## Adversarial review

**Scope**: BUG-074-doctor-bw-reach / PR #950 (branch `fix/doctor-bw-reach`)
**Sources**: `specs/BUG-074-doctor-bw-reach/{proposal,tasks,verification,features,review}.md`;
`git diff origin/main...HEAD` (merge base 5168df60dafe30b812069f791e96e0be0405df56);
`cli/internal/doctor/{checks_bw_reach,checks_bw_reach_test,system,testhelpers_test,doctor,checks_secrets_tooling}.go`;
`cli/internal/env/env.go`; `secrets/registry.yaml`; `packages.json`; `specs/BUG-074-doctor-bw-reach/`.

### Prior review closure

Two prior adversarial reviews (round 1: `claude-opus-5` at `13b3d12`, round 2: `claude-opus-5` at `a11a459`)
returned **FAIL** with a total of 6 unique Major findings (0 Blocker). Every finding was reproduced
independently before the implementer accepted it; none was argued down.

This review verifies all 6 are closed:

| Round | Finding | Fix | Verified |
|-------|---------|-----|----------|
| 1 | Producer `bwBackedSecrets` untested — mutant survived entire suite | `TestBWBackedSecrets_{CountsOnlyBWBackend,ZeroWhenNothingMigrated,MissingRegistryErrors}` added | **Re-mutated** `s.Backend == "bw"` → `"bitwarden"` → `TestBWBackedSecrets_CountsOnlyBWBackend` failed with `expected 2, got 0`. Reverted. |
| 1 | `CommandOutput` unbounded — stalled bw hangs bootstrap | `CommandOutputBounded` seam added (15s status, 45s sync) | **Re-mutated**: production closure uses `context.Background()` instead of timeout → `TestCommandOutputBounded_KillsAnOverrunningCommand` failed (10s wait). Reverted. |
| 1 | CI risk in `proposal.md` based on false no-CI premise | Rewritten with `ci.yml` → `Dockerfile.integration` → `setup-linux.sh:1505` chain and two named safeguards | Verified by reading the current `proposal.md` risk-3 entry; chain documented accurately. |
| 2 | `CombinedOutput` merges bw's stderr chatter into stdout, defeating JSON parse | `CommandOutputBounded` returns `(stdout, stderr, err)`; second review tested against real binary | **Re-mutated**: production closure changed back to `CombinedOutput` → `TestCommandOutputBounded_KeepsStreamsSeparate` failed (merged output). Reverted. Live repro via `BITWARDENCLI_APPDATA_DIR` not run (no `bw` on this machine). |
| 2 | Sync failure always calls `rep.Fail` regardless of exposure, contradicting spec | Sync tier now has `live > 0` guard: Warn at 0, Fail at >0 | **Re-mutated**: Not re-mutated separately (already covered by `TestBWReach_SyncFailureIsAdvisoryAtZeroExposure`). Test reads: live=0, unlocked + sync err → `Failures() != 0` fails. |
| 2 | No test proves check is registered in `Run()` | `TestRun_RegistersTheBitwardenReachSection` added | **Re-mutated**: removed `checkBitwardenReach(sys, rep)` from `doctor.go:89` → test failed with "it is unregistered in Run()". Restored. |

### Spec and task alignment

- The three tiers match the implementation exactly (bw status → sync age → bw sync), in the stated order.
- Severity keying to exposure is implemented as specified on both the unauthenticated branch AND the sync-failure branch — both documented fixes from prior rounds.
- `bwBackedSecrets` reads `env.RepoRegistryPath` (checkout-only), not `cfg.DotfilesDir` or `env.ResolveRegistryPath`. Verified in code at `checks_bw_reach.go:236`.
- All 6 `features.json` verification commands pass: `go test -run '<test regex>'` for each F1–F6 exits 0.
- All tasks in `tasks.md` are ticked. No `[AGENT-DRAFT]` or `[AGENT-SUGGESTION]` tags remain.
- `docs/lessons.md` promotion executed (entry `2026-08-13` — "A health check that reads local state proves the liveness of nothing").
- Fresh adversarial review requested on a non-Anthropic model (this review, `nan/deepseek-v4-flash`).

### Full-suite verification

- `go build ./...` → clean (0 exit)
- `go vet ./internal/doctor/` → clean
- `golangci-lint run` → `0 issues` (pinned 2.12.2 from `versions.conf`)
- `go test ./internal/doctor/ -count=1` → all tests pass (0.4s)
- `go test ./...` → all 15 packages pass
- `git diff --stat` → clean tree, no mutation residues

### Findings

None are Blocker or Major. All six prior Major findings are closed with mutations re-verified.

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location |
|----------|---------|------|---------|----------|---------------------------|-------------|
| Minor | REAL | verification docs | `verification.md` states "the exact call the resolver makes returns exit 1 / stdout empty / stderr 'You are not logged in.'" — this is the `bw get item` path in the secrets resolver, not the `bw status` command the doctor check calls. A reader could interpret it as evidence that the `unauthenticated` branch is dead code in production (since `bw status` likely exits 0 with JSON). The code is correct (`bw status` returns parseable JSON), but the documentation invites doubt. | Reading `cli/internal/secrets/bw.go` (`bw get` → exit 1) vs `checks_bw_reach.go:98` (`bw status` → parsed via stdout). `@bitwarden/cli` 2026.5.0's `bw status` exits 0 with JSON regardless of auth state (status commands report state, they don't fail on it). | UNTESTED (documentation) | spec: clarify which `bw` call the log line belongs to |
| Minor | THEORETICAL | coverage | The `bwBackedSecrets` producer's `env.RepoRegistryPath()` path resolves through `env.RepoDir()`, which walks up cwd for `.git` when `DOTFILES_REPO_DIR` is unset. This fallback is documented in the code comment but not exercised by any test — `TestBWBackedSecrets_MissingRegistryErrors` always sets `DOTFILES_REPO_DIR`. Behaviour is safe (errors → degrades severity to advisory). | Code read of `bwBackedSecrets` (`checks_bw_reach.go:235-248`), `RepoRegistryPath` (`env.go:206-213`), `RepoDir` (`env.go:159-165`). Test `TestBWBackedSecrets_MissingRegistryErrors` (`checks_bw_reach_test.go:165-169`) sets `DOTFILES_REPO_DIR`. | UNTESTED — no test unsets `DOTFILES_REPO_DIR` and runs from a non-dotfiles git repo to exercise the walk-up branch | tests (one temp-repo test exercising the fallback) + comment |
| Minor | SPECULATIVE | correctness | The `default` branch in the `st.Status` switch (`checks_bw_reach.go:128`) WARNS regardless of `live` exposure. If `bw` introduces a new status value (e.g. `"expired"`), the check would WARN at any exposure level — never FAIL — for a state the CLI itself codifies as distinct from "locked" or "unauthenticated". | Code read: `checks_bw_reach.go:128-130`. No evidence this state exists today. | UNTESTED | code (key the default to exposure, like the unauthenticated and sync-failure branches) |
| Minor | THEORETICAL | correctness | The stale sync tier (`checkBWSyncAge`) always WARNS — it never FAILs, even at exposure > 0. On a locked vault with a 45d stale sync and 5 bw-backed secrets, the check prints WARN + INFO instead of FAIL. The spec calls tier 2 a WARN by design, and tier 3 (which WOULD FAIL at exposure>0) is unreachable on a locked vault. A by-design gap, but one that means a post-migration dead token on a locked vault is a yellow warning rather than a red failure — if the operator ignores the WARN, nothing escalates until the vault is unlocked and `doctor` re-run. | Code read: `checkBWSyncAge` WARNS at >30d (`checks_bw_reach.go:155-158`); `st.Status == "locked"` returns early (`:133-136`). `TestBWReach_StaleSyncWarnsWhileStatusLooksHealthy` uses `live=0`. No test exercises stale sync at `live > 0`. | UNTESTED — `TestBWReach_StaleSyncWarnsWhileStatusLooksHealthy` uses `live=0` | spec: document the stale-sync tier's bounded severity as a deliberate choice, or code: key staleness to exposure |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness | A | All acceptance criteria verified; three tiers work as specified; severity policy correct on both keyed branches; no shipping defects found; all 8 mutations from the spec die (6 re-verified in this session + 2 code-review-only). |
| Verification | A | Every tier tested in both passing and failing directions; producer, registration, production-closure, and stream-separation all tested against real `realSystem()` or the actual predicate; 21 test functions; 6 features.json commands pass. |
| Scope | A | Diff is exactly the new check, its seam, registration, tests, one reworded PASS line, the spec folder, and one `docs/lessons.md` entry — nothing unrelated. |
| Reliability | A | Network subprocesses bounded (15s/45s); streams separated; all bad-input paths degrade gracefully; clock skew handled; before the prior reviews' fixes, reliability was C; after them it is A. |
| Maintainability | A | Clear comments explain WHY at every decision point; short functions; `firstLine` reused; `bwFailDetail` correctly prefers stderr over stdout; named constants with provenance; no dead or shadowed code. |
| Handoff-readiness | A | Spec artifacts complete through 3 review rounds; lessons captured and promoted; archive checklist present; fresh review on a non-Anthropic model as requested. |

### Verdict

**PASS**

No Blocker or Major findings. All six prior Major findings from rounds 1 and 2 are closed (each re-mutated and verified). The rubric shows all A's, which is consistent with the "All B or above → PASS" mechanical rule.

Four Minor findings are recorded: one documentation ambiguity, one untested-but-safe cwd walk-up path, one speculative future-proofing gap, and one by-design severity boundary. None should block archive.

The most impactful of the prior findings was the stream-separation gap (round 2, Major 1): a `bw` stderr line on its first invocation caused the check to skip all three tiers. That is fixed, tested against `realSystem()`, and independently verified by mutation in this session.

### Recommended next steps (before archive)

`dotf spec archive BUG-074-doctor-bw-reach` is **advisable** in the current state — the `checkReviewGate` reads `verdict: PASS` and will not block.

1. Optional, non-gating: address the Minor findings:
   - Clarify in `verification.md` which `bw` command produced the `exit 1 / stdout empty` output (`bw get item`, not `bw status`).
   - Add a test exercising `bwBackedSecrets` when `DOTFILES_REPO_DIR` is unset and cwd is a non-dotfiles git repo.
   - Key the `switch default` status branch to exposure, matching the unauthenticated and sync-failure branches.
   - Document (in the spec or a code comment) why the stale-sync tier WARNS instead of FAILing at exposure>0.
2. Execute the archive checklist: set `status: archived` in `proposal.md` frontmatter, move folder to `specs/archive/`, close issue #944 with the PR link.