---
spec: "AI-039-copilot-settings-merge"
verdict: "PASS"
reviewed_sha: "c4c43d61522f6124ddbd4451d0870addae13bdfb"
reviewer: "nan/deepseek-v4-flash"
date: "2026-08-29"
---

## Adversarial review

**Scope**: AI-039-copilot-settings-merge (PR #1365 `feat/copilot-settings-merge`, **merged** 2026-08-29T02:34Z into main, merge commit `3261d04`).
**Sources**: `specs/AI-039-copilot-settings-merge/{proposal,tasks,verification,features}.json`; merged diff of PR #1365 (`cli/internal/deploy/deploy.go` + `merge_test.go`, `cli/internal/cmd/deploy.go` + `deploy_requires_test.go`, `cli/internal/doctor/checks_deploy_manifest.go` + test, `ai/deploy.json`, `ai/copilot/*`, `setup-linux.sh`/`setup-windows.ps1`, `tests/copilot-config.bats`, `tests/env-contract.bats`).

Confirmation of independence: this session runs as `nan/deepseek-v4-flash`, the pool's **primary** reviewer — the only Anthropic-free, reasoning-class entry the pool will sign, and the exact id `review-request.json` records. The implementer's session was a different run. Evidence below was re-produced from the merged tree (HEAD `c4c43d6`), not taken on trust from `verification.md`.

### Spec and task alignment

- Proposal is complete and specific; all five acceptance criteria (AC1–AC5) are testable and no open questions remain (Risks section resolves each with a decision). Frontmatter `status: implementing`; no `review:` waiver — a review is required, matching this pass.
- `tasks.md`: all implementation and verification boxes ticked `[x]`, no `[ ]` remain. Each `[P]` (failing-test-first) task names the mutation it guards.
- `verification.md`: per-AC named tests, mutation evidence, and a box run (Windows work box, Copilot CLI 1.0.81) with before/after key dumps.
- `features.json`: five features, one per AC, each `state: "verified"` with evidence, plus a per-AC reproducible command.
- No `[AGENT-DRAFT]` / `[AGENT-SUGGESTION]` tags remain in proposal/tasks/verification/features.json (the only matches are inside this run's own `review-transcript.jsonl`).

### Adversarial pass

The merge strategy is the heart of the change and got the closest scrutiny. Verified against the merged code, not the spec's claims:

- **AC1** — `mergeInto` (deploy.go) writes only the source's top-level keys, preserves every other key (tested: `TestDeploy_MergePreservesUnmanagedKeys`), creates an absent destination (`TestDeploy_MergeCreatesAnAbsentDestination`), tolerates the `//` header and leaves an in-sync file byte-identical never-rewritten (`TestDeploy_MergeToleratesTheCLIHeaderAndDoesNotChurnIt` + mtime assertion in `TestDeploy_MergeIsIdempotentAndDoesNotRewrite`). A non-object destination (incl. `null`, which would otherwise panic on nil-map assignment) errors by name instead of being silently replaced (`TestDeploy_MergeRejectsANonObjectDestinationByName`).
- **AC2** — `PlanConfig` and the dry-run path call no `MkdirAll`; the compare is hoisted above staging for non-rendered entries (`TestPlanConfig_And_DryRun_CreateNoDestinationDirectory`, both strategies). `ParseManifest` rejects an unknown strategy and a rendered merge by name, and a rendered config is refused by `PlanConfig` (`TestParseManifest_ValidatesStrategyByName`, `TestPlanConfig_RefusesARenderedConfig`).
- **AC3** — `ai/copilot/settings.json` = exactly `{model, includeCoAuthoredBy:false, autoUpdate:false}`; `config.json` = exactly `{trustedFolders}`; `mcp-config.json` replace. Manifest declares `copilot-settings`/`copilot-config` merge, `copilot-mcp` replace, all `requires: copilot`. Both setups copy `copilot-instructions.md` explicitly and no longer glob `ai/copilot/*`. `tests/copilot-config.bats` 11/11 verifies the documented-key set (frozen, 1.0.81), the doctrine values, no `powershellFlags`/`telemetry`, manifest shape, and the explicit copy. All assets verified by reading the merged files directly.
- **AC4** — `checkDeployManifest` reports PASS-with-counts / WARN-naming-entry-and-remedy / SKIP-without-repo, and never creates a destination directory (`TestCheckDeployManifest_ByStatus`, all sub-cases); registered in doctor.go after `checkDeployDrift`. Gated (requires-absent) and rendered entries are excluded from comparison and counted, so doctor cannot WARN a box for a tool it does not carry (#843).
- **AC5** — box sequencing in verification.md is coherent and was exercised against the real CLI header; `copilot -p` smoke passes.

Rationale for not elevating the two theoretical risks below to majors: merge deliberately preserves box-owned keys, and the only realistic way that guarantee breaks is a concurrent writer racing a deploy — which the single-operator setup/converge context does not hit. Both are flagged honestly and are non-gating.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Minor | THEORETICAL | concurrency | `mergeInto` computes the merged object from a read-time snapshot of the destination; `commit` then renames that snapshot over the file. A co-writer (Copilot `/model`, an editor) changing an **unmanaged** key between the plan-read and the rename loses that write — the staged file holds the stale unmanaged value. This is the one place merge's "preserve unmanaged keys" has no atomicity (vs `replace`, whose whole file is owned). Never observed; window is negligible in non-concurrent setup runs. | code read: `mergeInto` reads dst once, `PlanConfig`→`stage`→`commit` renames the snapshot; `commit` (deploy.go) has no re-read or lock | UNTESTED (no concurrency test exists) | code (optional, non-gating): re-read+re-merge at commit, or accept as a documented single-operator limitation |
| Minor | SPECULATIVE | edge | An existing but empty (0-byte) or truncated destination errors the merge ("destination is not a JSON object") instead of being recovered/created. This is the fail-safe side of the design (erroring beats the silent replace the strategy exists to prevent) and Copilot never writes an empty/truncated `config.json`, so it is not expected in practice — but it is untested. | code read: `mergeInto` unmarshal of `""` fails before the ErrNotExist branch | UNTESTED | tests (optional): add an empty-destination case to pin the fail-safe behavior |
| Question | THEORETICAL | spec | A managed key removed from a future repo source is never removed from existing boxes — it becomes an unmanaged key owned by the box. Consistent with the intended `telemetry`-keeps behavior (proposal "A box that already has it keeps it"), not a defect; but retiring a managed key in future will silently leave it on boxes, so a future retirement should carry an explicit note. | proposal "Out of scope" + manifest comment "an unmanaged key is the box's"; code: `mergeInto` never deletes destination keys | UNTESTED (no key-retirement test) | spec / docs (note for future maintainers, non-gating) |

Non-findings worth stating so they are not read as gaps:
- **Storing the merged file without the `//` header** on an actual rewrite is intended (comment: "the tool that wants a header puts it back"); since the CLI rewrites `config.json` on every launch it re-adds its header, and the no-rewrite-when-in-sync path prevents a deploy/header flapping loop (`TestDeploy_MergeToleratesTheCLIHeaderAndDoesNotChurnIt`).
- **`requires`** is the one scope addition beyond the original proposal's letter; it is documented in "Decisions made during implementation" and is a real requirement — it keeps the integration guard "~/.copilot never created without copilot" (#1312) true (`TestDeployCmd_SkipsAnEntryWhoseRequiredCommandIsAbsent`, verify-setup). Not creep.
- **Parallel/related work** (manifest-v2 PR #1369, AI-042 trustedFolders-render, CLI-063) is explicitly out of scope and tracked as separate specs; no conflict within this change.

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | B | All five ACs verified by passing named negative-path tests; the merge's preservation guarantee carries one untested THEORETICAL race (no concurrency coverage) |
| Verification       | A | Reproducible per-AC commands + mutation evidence + box dump, all independently re-run green here |
| Scope              | A | Merged diff matches the proposal exactly; the only addition (`requires`) is documented and required by #1312 |
| Reliability        | A | Error paths named and fail-safe, atomic rename, mode enforcement, manifest fail-fast, idempotent no-rewrite |
| Maintainability    | A | Clear naming, small functions, why-comments; `mergeInto` readable under limits |
| Handoff-readiness  | A | All spec artifacts complete, features.json per AC, decisions + promotion candidates recorded, archive checklist present |

### Verdict
**PASS**

Aggregation: no D, no C; rubric is all B/A. Findings are Minor/THEORETICAL, Minor/SPECULATIVE and one Question — none is REAL, none is a Blocker or Major, and neither Minor moves the verdict below PASS (the SPECULATIVE one explicitly cannot).

### Recommended next steps (before archive)
- `dotf spec archive AI-039-copilot-settings-merge` is **advisable** in the current state — it will pass the review gate and there is no residual blocker.
- Before/at archive, close bitácora **#1322** with the PR reference (it is still OPEN; the change is merged).
- Optional, non-gating follow-ups (do not block archive): (1) decide whether the merge TOCTOU is an accepted single-operator limitation and — if so — record it once in the manifest comment so no future maintainer "fixes" it by deleting it as noise; (2) add an empty-destination merge test to pin the fail-safe error behavior; (3) add a one-line note that retiring a managed key leaves it on existing boxes.
- No vault promotion needed: the decisions here extend CLI-039's manifest under ADR-020 C7 and are already recorded in the spec — not a new ADR or pattern.
