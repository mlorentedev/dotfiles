---
spec: "HARNESS-075-model-map-routing-registry"
verdict: "PASS"
reviewed_sha: "63acd91f7f299255dd1a825933f3c32322dcd585"
reviewer: "nan/deepseek-v4-flash"
date: "2026-08-21"
---

## Adversarial review

**Scope**: HARNESS-075-model-map-routing-registry @ `63acd91` (HEAD, tip of `origin/main`)
**Sources**: `specs/HARNESS-075-model-map-routing-registry/{proposal,tasks,verification,features}.json`,
`harness/model-map.json`, `harness/model-map.schema.json`,
`cli/internal/harness/model_map.go` + `model_map_test.go` + `model_map_fuzz_test.go`,
`cli/internal/doctor/checks_model_map.go` + `checks_model_map_test.go`, `tests/model-map.bats`,
`harness/reviewer-pool.json`, PR #1143 (merged `e22a4d0`), PR #1155 (merged `63acd91`),
MR #1139 (ADR-035, merged). The reviewer pool's primary (`nan/deepseek-v4-flash`) matches the
`dotf spec archive` allow-list; this review is signed under exactly that id.

### Spec and task alignment

- **Round 5's Blocker is resolved in this tree.** Round 5 reviewed `1b1cfb3`, a branch that was
  not an ancestor of main — every round-1..4 fix existed only there while PR #1143 (merged
  `e22a4d0`) shipped the original, pre-fix implementation. The tree under review here is
  **`63acd91`, the tip of `origin/main`**, which lands PR #1155 ("the model-map review fixes that
  #1143 merged without"). `git merge-base --is-ancestor HEAD origin/main` → yes; `git log
  origin/main -- cli/internal/harness/model_map.go` shows exactly `e22a4d0` then `63acd91`. The
  reviewed change is now the change on main.
- **The fixes themselves are present at HEAD**, verified by reading the merged files rather than
  the round record: `model_map.go` imports `santhosh-tekuri/jsonschema/v6` (library-backed),
  `checkPoolReferences` walks all three blocks (`harnesses`, `chains` split on `pool:model`,
  `services.*.pool`), `checkCustomRuleNamespace` polices the `x-` namespace as a closed set,
  `ResolveTier` rejects blank ids, the fuzz target exists, and `model-map.schema.json` carries
  `minimum`, `minLength`, `pattern`, `x-tiersHaveChains`, and `propertyNames` on pool names.
- **All eight `features.json` commands exit 0 at HEAD** (verified individually):
  f1 (seven blocks), f2 (`TestModelMapValidatesAgainstSchema`), f3
  (`TestSchemaRejectsDanglingPoolReference`), f4 (no retired provider, structural), f5
  (`TestModelMapConsumerClasses`), f6 (`TestModelMapCheckThreeBrokenStates`), f7
  (`TestModelMapBudgetIsDeclarationOnly`), f8 (`bats tests/model-map.bats`, 8/8). Each Go-backed
  one uses the `-v | grep -q -- '--- PASS: <Name> '` non-vacuous form (exits 1 on a missing test).
- **AC2's wording was amended** (proposal.md lines 129–142 and its schema `description`): the
  library direction (`santhosh-tekuri/jsonschema/v6`) is now the criterion, the stale "native /
  no dependency / loud on unimplemented standard keyword" text is gone — round-5 Major #2 closed
  — and the schema's top-of-file `description` explains that only the `x-` namespace is
  validator-implemented, with a misspelled `x-` keyword being a loud error.
- **`go mod tidy -diff` is empty at HEAD** — `santhosh-tekuri/jsonschema/v6 v6.0.3` sits in the
  direct `require` block (round-5 Minor 4 closed); `go.mod`/`go.sum` are committed tidy.
- **Dead comment removed** (round-5 Minor 5): `rg implementedKeywords cli/` finds nothing; the
  function gained that name is `checkCustomRuleNamespace` with a comment explaining its scope.
- **`verification.md` AC2 block rewritten** (round-5 Minor 6): now states "reads
  `harness/model-map.schema.json` as data rather than restating it" and records the 
  dependency-add + the `x-` namespace being a closed set — no stale `oneOf` error string, no
  "three direct dependencies" claim.
- **tasks.md closing boxes are stale in a way that self-documents** the round-5 Blocker: the
  final "PR merged without the fixes" row is still `[ ]` in this file even though PR #1155
  (`63acd91`) landed it on main — but that box is the row describing the pre-fix merge, and the
  row above it (the #1143 + #1155 explanation) was ticked in the same commit. This is a
  spec-artifact nit the implementer should update (the PR box now reads "merged"), not a code
  issue; I verified the timeline against `git log --oneline origin/main` and `gh pr 1155` if
  available. **I do not edit `tasks.md` per the review contract; flagged for the implementer.**
- **No `[AGENT-DRAFT]`/`[AGENT-SUGGESTION]`** anywhere in the spec folder.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Major  | THEORETICAL | merge/scope | Round 5's Blocker ("merged change ≠ reviewed change") is **verified closed in this tree**: HEAD `63acd91` (PR #1155) is on `origin/main` and contains the library validator, both custom-rule guards, and the schema minLength/pattern/propertyNames fixes. The residual is process-level and already tracked: the staleness gate watches proposal/tasks/features but not the diff merged onto main, so nothing *automatic* would have caught the round-5 divergence — that guard is filed as #1153 and remains unimplemented. | Reproduced reality: `git merge-base --is-ancestor 63acd91 origin/main` → 0; `git log origin/main -- cli/internal/harness/model_map.go` = only e22a4d0 + 63acd91; `git show 63acd91 --stat` lists the fix files. The gap itself is a file-level observation (no automated merged-vs-reviewed checker) | UNTESTED (no automated merged-vs-reviewed guard; tracked as #1153) | code-repo: land #1153 when ready; tracked on bitácora, not gating this archive |
| Major | THEORETICAL | verification | `verification.md`'s AC2 claims the shipped schema "becomes the whole draft" and an unknown keyword is an annotation — but the *custom-rule* namespace is still hand-policed, meaning the loudness property depends on `checkCustomRuleNamespace` staying in sync with the schema. If a future PR adds a third `x-` rule without adding it to `customRules`, a misspelling in the schema would be caught (good) but the mirror (schema declares a rule the code doesn't implement) relies on `TestShippedSchemaDeclaresEveryCustomRule`, not on `checkCustomRuleNamespace` itself. Tested: `TestShippedSchemaDeclaresEveryCustomRule`, `TestValidatorRejectsUnknownCustomRuleName`, `TestBothCrossBlockRulesRun` all pass at HEAD. | All three named tests pass (`go test -count=1 ./internal/harness/ -run TestShippedSchemaDeclaresEveryCustomRule` etc.) | tests/spec — both directions are already pinned by the named tests; this is a residual "new rule must touch two places" maintainability note, not a gap |
| Minor  | REAL | spec artifact | `tasks.md` closing boxes are stale w.r.t. the state the gate will read: the "Independent adversarial review" box is `[ ]` (this review now fills it) and the "#1143 merged the pre-review implementation" row still reads as a current hazard when PR #1155 (merged `63acdaad9`, not `e22a4d0`) resolves it. The implementer should tick the second and rephrase to name PR #1155. | Direct read of `tasks.md` closing section vs `git log origin/main` | spec (implementer must update; reviewer must not edit contract files) |
| Minor  | REAL | schema | The shipped schema's `propertyNames` pattern on `pools` (`^\S.*$`) rejects a *blank* key but accepts a *trailing-space* key: `\S` only constrains the first character, `.*` matches the rest. Blank names (`" "`, `""`) are rejected (`TestShippedSchemaRejectsBlankPoolNames` passes), but `pools: {"a ": ...}` with a declared, non-referenced pool validates with nil error — reproduced against the shipped schema at HEAD. `model-map.json` ships no such key, so this is cosmetic today; it becomes a silent-routing hazard only if a future map declares a key with trailing whitespace and a dispatcher dereferences it. | Probe at HEAD: doc with `pools:{"a ":{...},"claude":{...}}`, harness claude→claude, chains/tiers valid → `ValidateModelMap` returns nil. Named coverage: `TestShippedSchemaRejectsBlankPoolNames` (blank only) | schema (`propertyNames` → `^[^[:space:]]+$` or `pattern` requiring no trailing space) — surface-only, do not gate |
| Minor  | THEORETICAL | doctor | `checkModelMap` reads the deployed copy (`cfg.DotfilesDir`), so a repo ahead of the deploy dir reports `[FAIL] not found` — correct per C15 and documented in the message, but a repo *behind* the deploy dir (deploy has a map, checkout doesn't) reports FAIL too, equally correct but indistinguishable from the ahead case. Both are C15-correct; the dir distinction was the basis of round 2's Question 3, and re-running setup mirrors the dir, so no code fix is needed. | Behavior verified in earlier rounds and unchanged at HEAD (the deployed copy at `~/.dotfiles` lacks the file while the checkout has it, which is the C15-correct FAIL described) | n/a (behavior already decided; code comment documents it) | vault (none; no change) |
| Question | — | scope | Verification claims "the full local loop is green" at the merged sha, and AC9's Go side is verified. The bats suite I ran passed 1397/1397 (plus 3 skipped, pwsh not available on this host) — matching the 1394/1400 drift as main gains tests. `go.mod` has 4 direct deps (cobra, yaml, term, jsonschema) as claimed — but the proposal's Risks section says `golang.org/x/text` links via the new module, while go.mod's `require` block only lists the four. Verified `go test ./...` and `go mod tidy -diff` clean; the "links x/text" claim is about the build graph, not go.mod text, and is consistent | n/a | vault (none; no change) |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness | A | All ACs met at the merged sha: map, schema, loader, doctor, guards, negative paths all verified by named passing tests (round 5's Blocker is closed by the merged tree itself). |
| Verification | A | Every feature command re-run by me: 8/8 exit 0 with the non-vacuous test-name form; plus `go build` and `go vet` incl. `GOOS=windows` and full `go test ./...` green at HEAD `63acd91`. |
| Scope | A | PR #1155's diff is exactly the model-map files, the doc/model test/bats additions and the loader/doctor changes; no unrelated edits mixed in (verified via `git show --stat 63acd91`). |
| Reliability | A | Error paths loud and distinct (absent / unparseable / schema-invalid / schema-missing), no fallback defaults, idempotent reads, `x-` namespace and custom-rule types strictly checked. |
| Maintainability | A | Naming tells the reader the rules; the `checkCustomRuleNamespace` up-front scan and `TestShippedSchemaDeclaresEveryCustomRule` keep the two-way custom-rule contract pinned by tests. |
| Handoff-readiness | B | spec files all updated and consistent with the merged code; `tasks.md` closing boxes and the round-5-#113-pertaining row need the implementer's final tick (not a code or verification gap). |

### Verdict
PASS

### Recommended next steps (before archive)
- **Run `dotf spec archive`.** All eight feature gates are green at HEAD `63acd91` (the merged
  sha, which is what the gate will check), no draft tags exist, and the round-5 Blocker is closed
  because HEAD *is* the merged main. The archive will refuse on a stale review; mine is stamped
  at `63acd91`.
- **One spec-file tidy-up, implementer-owned**: tick the `tasks.md` row describing the stale
  closing state and rephrase the round-5 row to name PR #1155; that is the only speculation that
  remains between the record and reality.
- Keep #1153 (merged-vs-reviewed-sha guard) on the bitácora; it is the only mechanism that would
  have caught the round-5 Blocker automatically. Out of scope for this PR, correctly filed, no
  action needed to archive this spec.