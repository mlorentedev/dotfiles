---
spec: "HARNESS-186-named-lanes-sticky-personas"
verdict: "PASS-WITH-GAPS"
reviewed_sha: "d6c552a67003b22e1142ace204315d2e20fb3624"
reviewer: "agy/gemini-3.8-flash-high"
date: "2026-11-08"
---

## Adversarial review

**Scope**: HARNESS-186-named-lanes-sticky-personas
**Sources**: `specs/HARNESS-186-named-lanes-sticky-personas/{proposal,tasks,verification}.md`, `features.json`, `git diff 4fbf735a0caaad93d347c110a74fdf830d328bc2...HEAD`

### Spec and task alignment

The implementation covers all 4 layers specified in `proposal.md` and fulfills the core requirements across 18 tracked tasks:
- **Layer 1 (Persistent Named Lanes & In-Place Park/Recycle)**: `cli/internal/lane/named.go`, `cli/internal/worktree/done.go`, `cli/internal/worktree/sweep.go`. Idempotent creation of `feat-lane-primary`, `feat-lane-secondary`, and `fast-track-lane` under `WORKTREE_ROOT/<repo>/<lane>` (`changed=0` on re-run). Deterministic branch prefix routing with 2-feature WIP limit. In-place park detaches at remote default branch, verifies `branchLanding` tip SHA containment, deletes merged local branch, clears prompt file, releases claims, and archives memory threads under `LockHandoffMemory`.
- **Layer 2 (Sticky Persona Gate & Windows Atomic State)**: `cli/internal/harness/gate.go`, `cli/internal/harness/dispatch.go`, `cli/internal/cmd/harness_gate.go`. Exempts read-only discovery tools (`Read`, `Grep`, `Glob`, `LS`, `read_file`, `grep_search`, `find_by_name`, `list_dir`, `view`). Binds sticky persona per session and branch to prevent mid-session thrashing when authoring ADRs/lessons/specs. In-process mutex `atomicWriteMu` and 15-attempt Windows rename retry.
- **Layer 3 (Ephemeral Fruit Workers & Pi Fallback)**: `cli/internal/agent/fruits.go`, `cli/internal/agent/backends.go`, `cli/internal/agent/process.go`. 64-fruit shuffle without replacement for `meeseek-<role>-<fruit>`, role resolution in `effectiveRole()`, and `piBackend` passes `--provider nan` while classifying missing provider authentication as `StatusPoolUnavailable`.
- **Layer 4 (Cross-OS Forge Provenance & Scoped Pre-commit Guard)**: `cli/internal/forgewrite/provenance.go`, `git-hooks/lib/lane-guard.sh`, `git-hooks/lib/board-pickup.sh`. Non-attributed `Environment: <machine-id> (<goos>/<goarch>) · <lane>` footer. Pure POSIX `sh` pre-commit guard keyed strictly on `.dotf/lanes.json` marker with 10 verified test scenarios. Backgrounded `board-pickup.sh` detaches CWD before network calls.

No `[AGENT-DRAFT]` or `[AGENT-SUGGESTION]` markers remain in the spec directory.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Major | THEORETICAL | gate / sticky persona | `currentGateWorktreeContext` in `cli/internal/cmd/harness_gate.go:498-508` nests `mem.RepoIdentity(workDir)` inside `if meta, merr := worktree.LoadMetadata(workDir); merr == nil && meta != nil`. In normal repository checkouts or non-worktree clones where `.dotf-worktree.json` does not exist, `branch` returns `""`. Consequently, `BindStickyPersona` records `Branch: ""` and subsequent calls to `LoadStickyPersona` cannot detect git branch switches, preventing sticky persona invalidation when switching branches in non-lane checkouts. | Code inspection of `currentGateWorktreeContext` lines 498-508; `TestHarnessGateStickyPersonaAndMeeseekResolution` tested with empty branch string `""`. | UNTESTED | code (`cli/internal/cmd/harness_gate.go`) |
| Minor | THEORETICAL | resilience / cleanup | `sweepStrandedDispatchTemps` in `cli/internal/harness/dispatch.go:263-280` only sweeps files matching `.dispatch-*.tmp`. `atomicWriteFile` creates temporary files using `.atomic-*.tmp` for persona and ledger writes (line 214). A process killed mid-write (via hook timeout kill or SIGKILL) strands `.atomic-*.tmp` files in `$STATE_DIR/gate/`. | Code inspection of `atomicWriteFile` lines 214-217 and `sweepStrandedDispatchTemps` line 271. | UNTESTED | code (`cli/internal/harness/dispatch.go`) |
| Minor | REAL | maintainability / code quality | Functions `Park` in `cli/internal/worktree/done.go` (92 lines), `parkSingleNamedLane` in `cli/internal/worktree/sweep.go` (54 lines), and `switchLaneLocked` in `cli/internal/lane/named.go` (49 lines) exceed the repository's `< 40 lines` function length threshold. | Line counts in `cli/internal/worktree/done.go`, `cli/internal/worktree/sweep.go`, and `cli/internal/lane/named.go`. | `TestLaneParkInPlaceAndTipVerification`, `TestLaneSwitchRoutingAndWIPCap` (exercises code, line count not gated) | code (`cli/internal/worktree/done.go`, `cli/internal/worktree/sweep.go`, `cli/internal/lane/named.go`) |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | B | AC1–AC6 verified clean in lanes; minor branch resolution gap in non-lane checkouts |
| Verification       | A | Reproducible tests across all 6 ACs, real git worktree fixtures, and 10 POSIX guard scenarios |
| Scope              | A | Implementation adheres strictly to the 4-layer specification without feature creep |
| Reliability        | B | Concurrency-safe on Windows with bounded retry, mutex serialization, and fail-closed tip verification |
| Maintainability    | B | Architecture documented in ADR-060, guide runbook, and lessons 567–569; a few functions exceed 40 LOC |
| Handoff-readiness  | A | All tasks complete with evidence, promotions completed, and review history documented |

### Verdict
PASS WITH GAPS

### Recommended next steps

The contract set (`proposal.md`, `tasks.md`, `features.json`) is closed and verified. As this is a PASS WITH GAPS verdict, recommendations are for the implementer to disposition in `verification.md` or track as follow-up work, rather than editing contract files:
- **Disposition Finding 1 in `verification.md` or ticket on bitácora**: Un-nest `mem.RepoIdentity(workDir)` in `currentGateWorktreeContext` (`cli/internal/cmd/harness_gate.go:498-508`) so git branch resolution functions in main repository checkouts and non-worktree clones as well as named lanes:
  ```go
  if meta, merr := worktree.LoadMetadata(workDir); merr == nil && meta != nil {
      wtPersona = strings.TrimSpace(meta.Persona)
  }
  if id, ok := mem.RepoIdentity(workDir); ok {
      branch = strings.TrimSpace(id.Branch)
  }
  ```
- **Disposition Finding 2 in `verification.md`**: Update `sweepStrandedDispatchTemps` pattern matching to sweep `.atomic-*.tmp` alongside `.dispatch-*.tmp` after `dispatchTempTTL`.
- **Disposition Finding 3 in `verification.md`**: Refactor `Park` and `switchLaneLocked` into helper routines in a follow-up hygiene pass to comply strictly with the `< 40 lines` function length rule.
- `dotf spec archive HARNESS-186-named-lanes-sticky-personas` is **advisable** once the implementer dispositions the tracked gaps in `verification.md`.
