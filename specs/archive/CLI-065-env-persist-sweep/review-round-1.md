---
spec: "CLI-065-env-persist-sweep"
verdict: "FAIL"
reviewed_sha: "6d1393d7ec4fc3a2c0b8d2c7454354f63fc0f3d2"
reviewer: "nan/deepseek-v4-flash"
date: "2026-09-30"
---

## Adversarial review

**Scope**: CLI-065-env-persist-sweep / mlorentedev/dotfiles#1363 (implementation commit `8de1cd1`, PR #1378). Stated diff range: `git diff 1f9e195b1fc3afd4d87c359e3f9b96eaca62cc5b...HEAD` (980 files, 76k insertions).
**Sources**: `specs/CLI-065-env-persist-sweep/{proposal,tasks,verification}.md`, `features.json`; `cli/internal/env/persist.go`, `persist_windows.go`, `persist_sweep_test.go`, `persist_test.go`; `cli/internal/cmd/env_persist.go`, `env_persist_test.go`; `cli/internal/doctor/checks_env_persist.go`, `checks_env_persist_test.go`; `docs/lessons/lesson-244-*`.

### Scope note (read this with the findings)

The launcher-resolved base `1f9e195` is exactly the parent of this spec's implementation commit `8de1cd1` — confirmed with `git log --format='%h %p %s' -1 8de1cd1` → `8de1cd1 1f9e195 feat(env): … (#1378)`. The range therefore contains 227 further commits that landed on `main` afterwards and belong to other specs. I reviewed the CLI-065 change **in full** (the whole delta of `8de1cd1`, plus the current state of the files it owns at HEAD) and did **not** review the unrelated 227 commits in the stated range; that bulk is outside this spec's declared scope and is not covered by this verdict.

### Spec and task alignment

- `proposal.md` lists AC1–AC6 and each maps to the code: marker (`ManagedMarker`/`MarkerValue`), sweep (`Leftovers`+`Persist`), `--check` (`Retired`/`MarkerStale`), doctor row, no-marker/no-op path, box transcript. No `[AGENT-DRAFT]`/`[AGENT-SUGGESTION]` tags remain.
- `tasks.md` boxes are all `[x]`; each names a real behaviour present in the diff. The three sub-decisions claimed in `verification.md` (delete-before-write, `Delete`-of-absent-succeeds, reader/store split) are genuinely present, not just asserted.
- `features.json` verification commands are non-vacuous and run; I re-ran them (evidence below).
- The stated non-goals (pre-marker values, `PATH`, machine scope, rc layer) are honoured by the code: with no marker `owned` is empty and nothing is deleted; `Leftovers` never sees a name absent from the marker.
- One artifact mismatch: `verification.md` attributes every AC to commit `a1878e0`, which is not an object in this repository (`git cat-file -t a1878e0` → `fatal: Not a valid object name`). The real commit is `8de1cd1`. See finding F3.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Major | REAL | correctness / reserved name | A contract variable named `DOTF_MANAGED_ENV` collides with the ownership marker. `checkNames` refuses a name containing `;`, but not the marker's own name, so `Persist` writes the variable and then immediately overwrites that same store key with the marker. Result: the variable's declared value is silently replaced by `A;DOTF_MANAGED_ENV` on every run, AC1's "a second run changes nothing" is violated, `--check` can never come back clean (the value always differs), and because `Leftovers` skips `ManagedMarker` such a name can never be swept when the contract retires it. | Reproduced this session: probe `TestAdv_MarkerNameCollision` (synthetic contract) → run 1 `DOTF_MANAGED_ENV="A;DOTF_MANAGED_ENV"`, run 2 `ops=[set:DOTF_MANAGED_ENV]` → `NOT idempotent`. No shipped contract names the marker today, so the trigger is a contract edit; the code path is real and nothing rejects the input. | UNTESTED | code (`checkNames` also refuse `strings.EqualFold(v.Name, ManagedMarker)`) + tests (a table case beside `TestPersist_RefusesASeparatorInAName`, and an AC1 `--check`-stays-clean assertion) |
| Minor | REAL | consistency / read path | The separator guard is enforced only in `Persist`. The read-only surface accepts the same contract, so `dotf env persist --check` reports "record does not match the contract … run `dotf env persist`" for a contract whose `persist` then refuses with `contract variable name "A;B" contains the marker separator`. The remedy `--check` prints cannot be carried out. | Probe this session: `MarkerStale` on `{A;B}` → `stale=true err=<nil>`; `Persist` → `contract variable name "A;B" contains the marker separator ";" and cannot be persisted`. | UNTESTED | code (validate names once, at resolve, so read and write paths agree) + tests |
| Minor | REAL | evidence provenance | `verification.md` cites commit `a1878e0` for all six ACs; that object does not exist in the repository, so the stated provenance cannot be checked by anyone reading the spec later. | `git cat-file -t a1878e0` → `fatal: Not a valid object name`; `git log --oneline -- specs/CLI-065…` → `8de1cd1`. | UNTESTED (documentation) | spec artifact (`verification.md` — outside the contract set, editable without invalidating this review) |
| Minor | THEORETICAL | reliability / concurrency | `--check` is not a snapshot: `checkPersisted` calls `Drift`, `Retired` and `MarkerStale`, each re-reading the marker, and `Retired` issues one `Get` per leftover. A concurrent `persist` between reads makes `--check` report a combination of states that never existed together; there is no lock around the marker read-modify-write in `Persist` either. | Code read; no repro. Vault/scale is one box with one writer. | UNTESTED | code (optional; low value at this scale) |
| Minor | THEORETICAL | verification gap | The doctor row emits two separate WARN entries when a box has both drift and retired names. The test table (`TestCheckPersistedEnv_ByStatus`) covers drift-only and retired-only, never the combined state. | `checks_env_persist.go` warns once per non-empty slice; test table in `checks_env_persist_test.go`. | `TestCheckPersistedEnv_ByStatus` (covers each state alone; combined UNTESTED) | tests |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | B | AC1–AC5 hold on every state I probed (0 divergences in an exhaustive mirror sweep) and AC6 has a box transcript, but one unguarded reserved name breaks AC1 and can never be swept. |
| Verification       | B | Named, reproducible tests exist for every AC and all re-ran green this session; provenance sha in `verification.md` does not resolve, and AC6 is box-only (unverifiable here). |
| Scope              | B | The spec's own commit `8de1cd1` matches the proposal exactly; the stated launcher range additionally carries 227 unrelated commits (not the implementer's doing). |
| Reliability        | B | Error paths wrapped and named, delete-before-write ordering pinned, `Delete`-of-absent promised and mirrored; unguarded reserved name and the non-snapshot `--check` are the gaps. |
| Maintainability    | B | Small functions, one shared `Leftovers` for all three callers, comments explain the why (order, case-insensitivity); `checkNames` incomplete. |
| Handoff-readiness  | A | `verification.md` filled, `tasks.md` ticked, lesson 244 written in this PR, promotion candidates reasoned rather than deferred. |

### Verdict
FAIL

One **REAL Major** (reserved-name collision, reproduced, **UNTESTED**) — under `severity × reality` that is what fails, independent of the rubric, which is all-B/A and would otherwise read PASS. None of the other findings is a Blocker.

### Recommended next steps

- **Code + tests (fixes the FAIL)**: extend `checkNames` in `cli/internal/env/persist.go` to also refuse a contract name equal (case-insensitively) to `ManagedMarker`, and add the test case; the same guard should cover the read path so `--check` and `persist` agree (finding F2). Because this changes executable code and adds tests, run the review again after the fix — a Major in code is "fix, then re-review".
- **Contract set (available on a FAIL, and only on a FAIL)**: if the reserved-name rule is to be recorded as a task, add the line to `tasks.md` now; that edit invalidates this verdict, which is the expected cost, not an accident.
- **`verification.md` (safe to edit; not in the contract set)**: replace the unresolvable `a1878e0` provenance with `8de1cd1` (finding F3).
- **Disposition in `verification.md`** for the remaining tracked gaps: combined drift+retired doctor case (add a table row), and the non-snapshot `--check` reads — apply, ticket, or decline with a reason.
- **Not for this spec**: do not re-litigate the declared non-goals (pre-marker values, `PATH`, machine scope). They are stated, tested and bounded.
