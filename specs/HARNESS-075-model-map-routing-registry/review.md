---
spec: "HARNESS-075-model-map-routing-registry"
verdict: "FAIL"
reviewed_sha: "1b1cfb36ace8cc2ac6397a7db42d557e03c26706"
reviewer: "nan/deepseek-v4-flash"
date: "2026-08-20"
---

## Adversarial review

**Scope**: HARNESS-075-model-map-routing-registry (branch `feat/harness-075-model-map` @ `1b1cfb3`)
**Sources**: `specs/HARNESS-075-model-map-routing-registry/{proposal,tasks,verification,features}.json`,
`harness/model-map.json`, `harness/model-map.schema.json`,
`cli/internal/harness/model_map.go` + `model_map_test.go` + `model_map_fuzz_test.go`,
`cli/internal/doctor/checks_model_map.go` + `checks_model_map_test.go`, `tests/model-map.bats`,
`harness/reviewer-pool.json`, PR #1143 (merged as `e22a4d0`), PR #1136 (ADR-035, merged).

### Spec and task alignment

- **AC1–AC9 map cleanly to named tests, and I re-ran the entire AC9 loop at HEAD**: `go build ./...`,
  `go vet ./...`, `GOOS=windows go vet ./...`, `go test ./...` (17 packages ok, 0 FAIL),
  `golangci-lint run` (0 issues, v2.12.2 = the `versions.conf` pin), full bats suite
  (`BATS_EXIT=0`, 1400 ok / 0 not ok — the record's count "1394" has drifted as main gained tests),
  and `go test -fuzz` (30 s, 557,920 executions, no crash). All eight named features.json commands
  exit 0, and I confirmed the `grep -q '--- PASS: <Name> '` form exits 1 when the test name matches
  nothing (non-vacuous).
- **Rounds 1–4 findings are closed at HEAD.** I re-injected the round-3 mutation (`"minItems": "5"`)
  and round-4 mutations (`properties: {"a": 5}`, `required: ["a", 5]`, `concurrency: -5`) against
  the shipped schema: all loud errors from the library. Blank ids, whitespace ids, `"nan:"` chains
  and ghost references in `chains`/`services` are all rejected by the shipped schema.
- **AC6 verified live through the binary**: absent / unparseable / schema-invalid → three distinct
  loud FAIL outputs, none containing "no pools"/"0 pools"; the deployed copy at `~/.dotfiles`
  correctly reports `[FAIL] not found` because the repo is ahead of the deploy dir (C15-correct).
- **THE MERGED CHANGE IS NOT THE REVIEWED CHANGE — Blocker below.** PR #1143 (which `tasks.md`
  marks as the spec's PR) merged to main as `e22a4d0` carrying the ORIGINAL, pre-fix
  implementation. Every fix from rounds 1–4 exists only on this local branch (`1b1cfb3`), which is
  not an ancestor of `e22a4d0` and is not on main. I verified the merged tree directly (temp
  worktree at `e22a4d0`) and ran the round-1..4 regression probes against it: **all still fail
  there** — ghost pool in `chains` accepted, blank model ids accepted, `concurrency: -5` accepted,
  malformed `properties` element silently skipped, no `x-tiersHaveChains` rule at all, no fuzz
  guard, no library. The verification.md narrative ("all findings closed") describes the branch,
  not anything on main.
- `tasks.md` closing boxes: the PR box is ticked against a merged PR that lacks the fixes; the
  review box is unchecked (this review is the round it awaits).
- `verification.md` AC2 section is stale: it still asserts "No schema-engine dependency was added;
  `cli` still has three direct dependencies" and shows the deleted native validator's error wording
  ("declares \"oneOf\" at (root), which this validator does not implement") as current evidence —
  neither can occur under the library-backed code at HEAD.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Blocker  | REAL | handoff / scope | The change merged to main is not the change reviewed. PR #1143 (merged `e22a4d0`, on `origin/main`) contains the ORIGINAL implementation: native validator with `implementedKeywords`, no `minLength`/`minimum`/`pattern`/`x-tiersHaveChains` in the shipped schema, `checkPoolReferences` walking only `harnesses`, no blank-id guard, no fuzz test, no library. The reviewed HEAD (`1b1cfb3`, all four rounds' fixes) is not an ancestor of `e22a4d0` and is absent from main. I ran the round-1..4 regression probes against the merged tree: ghost-in-chains, blank-id, `-5` budget and malformed-`properties` all still validate cleanly there. Archiving on HEAD's green tests would certify a model-map whose merged form carries every defect rounds 1–4 documented as closed. | Reproduced against a worktree at `e22a4d0` (temp probe test, since removed); `git rev-list --left-right --count origin/feat/harness-075-model-map...HEAD` = 11/18; `git show e22a4d0:cli/internal/harness/model_map.go` | UNTESTED (no gate checks merged-vs-reviewed code; the staleness gate watches only `proposal.md`/`tasks.md`/`features.json`) | code-repo (merge the fixed branch; re-review at the merged SHA) |
| Major  | REAL | spec vs code | AC2 as written is not satisfied by the shipped code. proposal.md AC2 still binds "The validator is native (no schema-engine dependency)" and "encountering a construct it does not implement is a loud error, never a silent pass" — but commit `8d510ce` replaced the native interpreter with `santhosh-tekuri/jsonschema/v6` (a direct dependency), and under draft-2020-12 unknown keywords are annotations, silently tolerated. The Risks section documents the pivot; the acceptance criterion itself was never amended. The schema's own `description` repeats the dead claim ("an unimplemented keyword is a loud error"). | Direct read of proposal.md AC2 (lines 127–131) vs `cli/go.mod` and `model_map.go`; probe: adding `"x-bogus-review-keyword": true` to the shipped schema validates silently | UNTESTED — the named tests assert behavior, not the AC2 wording | spec (amend AC2 + schema `description` to the library direction; the watched contract files were left contradictory) |
| Major  | REAL | validation | A typo'd custom-rule name silently disables a cross-block rule. `ValidateModelMap` looks up exactly `"x-poolReferencesResolve"`/`"x-tiersHaveChains"` in the schema; a misspelled key (e.g. `x-poolReferenceResolve`) is treated by the library as an annotation and by the Go walk as absent → `continue` → the rule never runs, while a document containing a ghost pool validates with nil error. Rounds 1–3 closed the "wrong TYPE" variant (`"true"` string, malformed values); the "renamed/missing flag" variant is unguarded — and the old allow-list would have caught it loudly, so this is a regression of the exact loudness property the spec fought four rounds to establish. | Reproduced: schema with `x-poolReferenceResolve` (typo) + `harnesses.pi.pools:["ghost"]` → `ValidateModelMap` returns nil | UNTESTED — `TestValidatorRejectsMalformedPoolReferenceFlag` covers a wrong-typed flag, not a missing/renamed one | code + tests (error when an expected custom rule name is absent; add a named regression test) |
| Minor  | REAL | module hygiene | `go.mod`/`go.sum` are not tidy. `go mod tidy -diff` moves `santhosh-tekuri/jsonschema/v6` from the `// indirect` block to the direct block (it IS imported directly by `model_map.go`) and adds `dlclark/regexp2` lines to `go.sum`. The proposal's recorded accounting ("direct requires 3 → 4") does not match the committed file (3 direct + the library mislabeled indirect). Fresh-cache build works, so not build-breaking; CI does not run `go mod tidy`, so nothing enforces it. | `go mod tidy -diff` output at HEAD | n/a (module metadata) | code (run `go mod tidy`, commit the diff) |
| Minor  | REAL | maintainability | Dead documentation block at the top of `model_map.go` still describes `implementedKeywords` ("Adding a keyword to the schema therefore requires adding it here") — the variable and allow-list were deleted in `8d510ce`. `rg implementedKeywords` finds only the comment. A reader would hunt for a symbol that no longer exists and for a contract that is now the library's. | Direct read + `rg -n implementedKeywords cli/` | n/a (comment) | code (delete the stale comment) |
| Minor  | REAL | spec artifact | `verification.md` AC2 section is stale and internally contradicted by its own Round-4 section: "No schema-engine dependency was added" / "three direct dependencies" and the `oneOf` "not implemented" mutation output describe the deleted native validator and cannot recur. | Direct read of verification.md AC2 vs the round-4 section; probe: `oneOf: []` in the shipped schema now fails with a library message, not the recorded one | n/a (doc; the f2 gate command still passes either way) | spec (update the AC2 evidence block) |
| Minor  | REAL | spec artifact | `tasks.md` closing boxes stale: the PR box is ticked against a PR whose merged content lacks the fixes, and the independent-review box awaits this round. | `gh pr view 1143` (MERGED, `e22a4d0`); `git log origin/main -- cli/internal/harness/model_map.go` → only the merge | n/a (workflow state) | spec artifact — reviewer must not edit `tasks.md`, so flagged |
| Minor  | SPECULATIVE | schema | A whitespace-only pool NAME (a key, not a value) validates: `pools: {"  ": {...}}` is accepted, and a harness referencing it would resolve. Pool keys are unconstrained where model ids (values) are pattern-guarded. No routing exists yet (level 1 only), so the impact is cosmetic until a dispatcher dereferences keys. | Probe against shipped schema at HEAD: `pools["  "]` accepted | UNTESTED | schema (`patternProperties` on pool keys) — surface only, does not move the verdict |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | C  | HEAD's code meets the ACs and passes every named test and probe I ran, but AC2 as written (native, loud-on-unimplemented) is not what ships, and a typo'd custom-rule name silently disables a cross-block check. |
| Verification       | A  | Every criterion has reproducible commands + outputs; I re-ran build/vet/windows-vet/test/lint/full-bats/fuzz and both the three broken states and the merged-state probes live. |
| Scope              | C  | The committed diff at HEAD is exactly the proposal's files, but the change that merged to main is a materially different, pre-fix state — scope diverges between reviewed and merged. |
| Reliability        | B  | Error paths handled, reads are pure/idempotent, no mutation, C15 honored; the missing-flag fail-open and the unknown-keyword tolerance are the exceptions. |
| Maintainability    | B  | ≤40-line functions, deliberate naming, why-comments, table-driven tests; pulled from A by the dead `implementedKeywords` comment and the untidy go.mod. |
| Handoff-readiness  | D  | The reviewed fixes are not on main; PR #1143 merged the unfixed original; verification.md/tasks.md AC2 and boxes are stale. |

### Verdict
FAIL

### Recommended next steps (before archive)
- **Merge the fixed branch, then re-review at the merged SHA.** The Blocker is that main's
  model-map (`e22a4d0`) still carries every round-1..4 defect — ghost pools in `chains`/`services`
  accepted, blank ids accepted, negative budgets accepted, malformed `properties` silently skipped,
  no `x-tiersHaveChains`, no fuzz guard. The fix rounds 1–4 are present only on
  `feat/harness-075-model-map` (`1b1cfb3`), which has never been merged and is not an ancestor of
  the merged PR. Land those commits (or an equivalent PR), then re-run the pool review on the
  merged commit; the review here certifies the branch, and the branch is not the change on main.
- **Amend AC2 and the schema description to the library direction** (spec artifact, implementer-owned):
  "native, no schema-engine dependency" and "loud on an unimplemented construct" no longer describe
  the code; the Risks section already records the pivot, the acceptance criterion does not.
- **Guard the custom rule names** (code + tests): error loudly when an expected `x-` rule name is
  absent from the schema, so a typo cannot silently disable the check; add a named regression test.
- **Housekeeping**: run `go mod tidy` and commit the diff; delete the dead `implementedKeywords`
  comment; refresh verification.md's AC2 evidence block; tick the tasks.md boxes (PR and review).
- **`dotf spec archive` is NOT advisable in the current state** — FAIL verdict, and the merged
  change on main does not contain the reviewed code, which the staleness gate cannot see (it
  watches only the spec's contract files, not the diff that merged).
