---
spec: "OPS-028-bw-folder-taxonomy"
verdict: "PASS"
reviewed_sha: "e5973a7a238584bce8e304d34cc0fb7add43f079"
reviewer: "nan/deepseek-v4-flash"
date: "2026-08-14"
---

## Adversarial review

### Spec and task alignment

Diffs unchanged from the prior review (spec artifacts were not touched by the fix commit). The six spec files — `proposal.md`, `tasks.md`, `verification.md`, `features.json`, and the archivable `review.md` — remain at c13a8b0. The only changed files are three source files in `cli/internal/secrets/` (+39/-5 LOC total), addressing the two Minor/THEORETICAL findings from the prior review.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test | Fix location |
|----------|---------|------|---------|----------|------|-------------|
| PASSED | THEORETICAL | Validation | **Plane↔folder correspondence now enforced.** The `planeFolder` map maps `"app"`→`"Dotfiles/apps"` and `"infra"`→`"Dotfiles/infra"`. The new check in `validate()` (registry.go:211-213) runs AFTER the ratified-set check: first catches bogus folders like `Dotfiles/typo`, then checks plane match. Planes absent from `planeFolder` (personal, floor) return `want=""` → guard `want != ""` skips → no false rejection for unfoldered personal/floor secrets. | Code in registry.go: `planeFolder` var (lines 74-77), validation block (lines 208-214). | `TestParseRegistry_BwFolder_MustMatchPlane` (registry_test.go:262-269): `plane: app` + `folder: Dotfiles/infra` → parse error. | registry.go: new `planeFolder` map + `if want := planeFolder[s.Plane]; want != "" && s.BW.Folder != want` block. Test added. |
| PASSED | THEORETICAL | Concurrency | **TOCTOU race documented honestly.** `ResolveFolder`'s comment no longer claims "idempotent" unconditionally — it now reads "idempotent for SEQUENTIAL calls" and adds a full paragraph naming the prior review finding, the race mechanism (two processes, same absent name), and the acceptance justification (single-operator CLI, no concurrent-writer story elsewhere). The re-read-after-create alternative is explicitly considered and rejected as narrowing but not closing the race. | `bw.go` lines 252-265: complete, honest documentation. | N/A (documentation-only fix). | bw.go: comment replaced. |
| Minor | THEORETICAL | Test coverage | **No test for symmetric plane↔folder mismatch (`infra → Dotfiles/apps`).** The code handles it correctly — `planeFolder["infra"]` returns `"Dotfiles/infra"`, `s.BW.Folder != "Dotfiles/infra"` → error — but only `app → Dotfiles/infra` carries a test. The symmetric case is the same code path (`!=` is symmetric), so this is a coverage gap, not a defect. | `registry_test.go` line 262-269 tests `app`→`Dotfiles/infra` only. No `infra`→`Dotfiles/apps` case. | Missing test for the reverse direction. | registry_test.go: add `plane: infra` + `folder: Dotfiles/apps` case. Not a blocker. |
| THEORETICAL | REAL | Field validation | **Pre-existing: `plane` field itself is not validated for allowed values (`app | infra | personal | floor`).** This is NOT introduced by this fix commit, but interacts with it: a secret with `plane: ""` (empty/unset) and `folder: Dotfiles/apps` passes the plane↔folder check because `planeFolder[""]` returns `""` and `want != ""` skips it. In practice the registry has no empty-plane entries (all 33 carry one of the four valid values), and the `plane` field is consulted downstream by `github.go:79` (`s.Plane == "floor"`) where an empty string would simply not match any branch. | `registry.go` `validate()` (lines 129-226): no `switch s.Plane` anywhere in the method. `github.go` line 79 checks `s.Plane == "floor"` only. | No test for invalid plane. | `registry.go` `validate()`: add `switch s.Plane { case "app", "infra", "personal", "floor": default: return error }`. Not introduced by this fix commit; not a blocker for archiving. |

### Evaluator rubric

| Dimension | Grade | Notes |
|-----------|-------|-------|
| **Correctness** | A | Both prior findings are correctly addressed. The plane↔folder check is sound: ratified-set check first, then plane match, skips for planes with no entry. The TOCTOU documentation is honest and complete. Symmetric (`infra→Dotfiles/apps`) is handled by the same `!=` code path. |
| **Verification** | B | The new test proves the rejection case. Full suite green (build, vet, test, lint all pass with exit code 0). One minor coverage gap: no test for the symmetric `infra→Dotfiles/apps` case. No test verifying the specific error message content (only `err == nil`). |
| **Scope** | A | The fix commit touches exactly the three files needed: `registry.go` (validation), `bw.go` (documentation), `registry_test.go` (test). Zero scope creep. |
| **Reliability** | A | The plane↔folder check is a pure read-only addition to an existing validation pass — no new mutation paths. No existing registry entry triggers it (all 21 app entries have `Dotfiles/apps`, all 5 infra have `Dotfiles/infra`, personal/floor have no folder). |
| **Maintainability** | A | The `planeFolder` map is a self-contained, testable var next to the existing `validBWFolders`. The comment on `ResolveFolder` is thorough and cross-references the review finding. The validation block groups both checks in the natural order (ratified-set first, then plane match). |
| **Handoff-readiness** | A | Both prior Minor findings are resolved. No blocker remains. The pre-existing plane-validation gap is a separate concern (track separately or as follow-up). |

### Verdict

**PASS**

Both Minor/THEORETICAL findings from the prior review (`c13a8b0`) are correctly addressed:

1. **Plane↔folder enforcement** (previously Minor): Fixed with the `planeFolder` map + validation check. The test proves rejection. The code handles the symmetric case, planes with no entry, and the correct ordering relative to the ratified-set check.
2. **TOCTOU race** (previously Minor): Documented with an honest, detailed comment that explains the limitation, names the prior review, and justifies the "don't fix" decision. The reasoning is sound for a single-operator CLI.

The only new observation is a pre-existing gap (plane field unvalidated for allowed values) that is out of this spec's scope. One Minor coverage gap (symmetric test missing) is a documentation-level observation, not a defect.

### Recommended next steps (before archive)

1. **No blockers.** `dotf spec archive` IS ADVISABLE right now — both prior findings are resolved, the CI suite is green, and the pre-existing plane-validation gap is separate work.
2. **Optional but recommended**: Add the symmetric test case (`plane: infra` + `folder: Dotfiles/apps`) for completeness — it's one line of YAML, makes the test suite's anti-regression coverage symmetric, and prevents a future refactor from dropping the `!=` check or changing it to a non-symmetric form. Open a fast-follow issue or inline into the existing test.
3. **Track the plane-validation gap separately** (not blocking this archive): Add a `switch s.Plane` in `validate()` rejecting anything outside `{app, infra, personal, floor}`. Not required by this spec, not introduced here, but a cheap hardening pass that prevents a future registry typo from silently creating a semantically invalid entry.
4. **Run `dotf spec archive`** after confirming no urgent work depends on this branch.