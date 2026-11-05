---
spec: "CLI-129-initrepo-lane-contract"
verdict: "PASS"
reviewed_sha: "81030700182ccbaf4e39b3b64bcfb43b2d1564a5"
reviewer: "nan/deepseek-v4-flash"
date: "2026-11-04"
---

## Adversarial review

**Scope**: `CLI-129-initrepo-lane-contract` — `dotf init lane` + the lane step in `dotf init`
**Sources**: `specs/CLI-129-initrepo-lane-contract/{proposal.md,tasks.md,verification.md,features.json}`; `git diff c8cd9487718ba1331859fe257499707c2575e03b...HEAD` at `8103070`; branch commits `52461005` (this spec), plus `730cc9d5` (#3161) and `81030700` (#3162), which are other PRs' work carried in the range and are not judged here.

### Spec and task alignment

Contract digests in `review-request.json` match `dotf spec digest CLI-129-initrepo-lane-contract` exactly
(`features.json` `e42acc…`, `proposal.md` `c8f614…`, `tasks.md` `809b48…`, `spec.md` empty): the contract
files have not drifted since the review was requested. No `[AGENT-DRAFT]` / `[AGENT-SUGGESTION]` tags
remain in any spec file.

Every acceptance criterion is met, and I re-ran the command that proves it rather than reading the
claim in `verification.md`:

| AC | Named test (re-run, `--- PASS`) | What it actually proves |
|---|---|---|
| AC1 | `TestBootstrapLane_ConfigLoads` | scaffolded `.dotf/lane.json` parses through the real `lane.LoadConfig` (`DisallowUnknownFields` on), queue empty, `fix` / `origin/main` / `.dotf/lane-prompt.md` |
| AC2 | `TestBootstrapLane_PromptRenders` | the scaffolded prompt renders through the real `lane.RenderPrompt` (`missingkey=error`) with `PromptData`. |
| AC3 | `TestBootstrapLane_AgentsSection` (2 subtests) | creates `AGENTS.md`; appends after prior content, prefix preserved |
| AC4 | `TestBootstrapLane_Idempotent` | second run byte-identical for every file under the root, all three fields `unchanged` |
| AC5 | `TestBootstrapLane_KeepsExisting` | pre-existing config, prompt and section untouched without `--force` |
| AC6 | `TestBootstrapLane_Force` | prompt and section replaced in place, `lane.json` byte-identical to the pre-run file |
| AC7 | `TestInit_LaneStep` (2 subtests), `TestInitLane_DryRun`, `TestBootstrapLane_DryRunWritesNothing` | step index `lane > agents`; `--skip-lane` / `--skip-agents` leave no `.dotf`; dry-run writes nothing at library and cobra level |
| AC8 | `gh issue view 3154` | issue exists, `OPEN`, "CLI-129 follow-up: roll the lane contract out to kubelab first…", linked from `verification.md` |

Red-green was re-established by mutation, not assumed — each mutation turned exactly the named test red,
and each was reverted (`git status` clean apart from this file):

- unknown `"bootstrap"` key added to `templates/lane.json` → `TestBootstrapLane_ConfigLoads` FAIL (AC1's guard is real);
- `BootstrapLane`'s config call switched to `force` → `TestBootstrapLane_Force` FAIL (AC6's "never touches lane.json");
- the `hasSection && !force` early return removed → `TestBootstrapLane_Idempotent` FAIL (AC4);
- the orchestrator's dry-run argument hardcoded to `false` → `TestInitDryRun` FAIL (`DryRun wrote files to root: [.dotf/ AGENTS.md]`) (AC7's wiring, which my own new test would not have caught).

Build/lint evidence produced in this session: `go build ./...` clean, `go vet ./...` clean,
`golangci-lint run ./internal/initrepo/... ./internal/cmd/...` → `0 issues.`,
`go test ./internal/initrepo ./internal/cmd -count=1` → `ok` for both.

The one thing the proposer's own words do not cover: the two commits above the spec commit in the range
(#3161, #3162) are unrelated features that arrived on this branch; they do not touch `init`/`initrepo`
and do not interact with the lane work, so they are out of judgement here.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Minor | REAL | tests / CLI plumbing | The new `--skip-lane` flag has no test at the CLI boundary: the flag-to-field wiring can be broken without any test noticing. I mutated `SkipLane: skipLane` → `SkipLane: skipAgents` in `cli/internal/cmd/init.go` and the whole `go test ./internal/cmd` suite stayed green. Shipped wiring is correct (read and run), so this is coverage, not a defect. | mutation applied, suite `ok internal/cmd 32.8s`; `TestInit_LaneStep` sets `InitOptions.SkipLane` directly, never through cobra | UNTESTED (`TestInit_LaneStep` covers options, not the flag; no test runs `dotf init --skip-lane`) | tests — add a `dotf init --skip-lane` / `--skip-agents` case to `cli/internal/cmd/init_test.go` asserting `.dotf` absent and `AGENTS.md` present (and the converse) |
| Minor | REAL | spec artifacts | `--skip-lane` and the coupling "`--skip-agents` also skips the lane step" are new public CLI semantics, but they appear only in `verification.md` (Decisions) and the flag help — not in `proposal.md`'s What/ACs nor in `tasks.md`. A reader of the contract cannot see the flag exists. | `proposal.md` "What"/AC1–AC8 mention neither; `verification.md` line "`--skip-agents` also skips the lane step"; `init.go` help text | `TestInit_LaneStep/--skip-lane_and_--skip-agents_leave_the_contract_out` proves the behaviour, not the contract's record of it | spec — disposition in `verification.md` (or a follow-up ticket). Contract set is closed under this verdict; do not edit `proposal.md`/`tasks.md` now. |
| Minor | REAL | docs / handoff | `docs/adr/adr-022-dotf-init-flagship.md` "Subcommand surface" still lists only `dotf init`, `init agents`, `init github` — the ADR a reader consults for the init surface now omits `dotf init lane`. | `grep -n "init lane" docs/adr/adr-022-*` → no hit; table at adr-022 lines ~69–74 | UNTESTED (docs; no gate) | docs — add the row (ADR-022 is outside the contract set, so this can be fixed without re-review) or state in `verification.md` that ADR-050 §6 is the current surface and ADR-022's table is a snapshot. |
| Minor | SPECULATIVE | resilience | `hasSection` matches a full-line `## Lanes` anywhere in `AGENTS.md`, including inside a fenced code block; with `--force`, `replaceSection` would then drop everything from that line to the next `## ` heading — i.e. edit the wrong region of an `AGENTS.md` that merely *quotes* the heading. Not reproduced; the SDD section has had the identical behaviour since before this change. | code read, `agents.go`/`lane.go`; no reproduction attempted | UNTESTED | code + tests, only if a repo is ever found quoting the heading; do not gate on this. |

Nothing in the diff hits the injection, secrets, auth, performance or concurrency rows of the code-level
checklist: the writes are `os.WriteFile` to paths built from a constant relative name joined to the repo
root (no user-controlled path segment reaches the write), the templates are embedded compile-time content,
and no network or `gh` call is added.

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | A | All eight ACs proven by named tests I re-ran; negative paths (existing files, force, dry-run, skips) covered; four mutations each caught by the right test. |
| Verification       | B | Reproducible commands and honest mutation notes; the one unmentioned gap is the flag-level coverage of `--skip-lane`. |
| Scope              | B | Diff matches the proposal for the contract itself; two new flags are side-work documented only in `verification.md`, and the range carries two unrelated PRs (#3161, #3162) that no part of this spec authored. |
| Reliability        | A | Read/write errors wrapped and returned, per-step warn-not-fatal in the orchestrator, idempotent, dry-run writes nothing at both the library and orchestrator level. |
| Maintainability    | A | Four small functions (≤ ~35 lines, nesting ≤ 4), `golangci-lint` clean, comments carry the *why* (why `lane.json` is never forced, why the section is shared with the SDD surgery), no dead code. |
| Handoff-readiness  | B | `verification.md` filled with named tests and decisions, follow-up #3154 filed and linked; ADR-022's subcommand table is left stale and the new flags are not in the contract. |

Aggregation: no D, no C → **PASS**; findings alone are Minor, one SPECULATIVE, so nothing escalates the verdict.

### Verdict

PASS

### Recommended next steps

- **Contract set is closed.** Do not edit `proposal.md`, `tasks.md` or `features.json` now — that would
  invalidate this verdict. Record the disposition of the three REAL Minors in `verification.md` (applied /
  ticketed / declined with a reason); `verification.md` is outside the staleness check.
- Add the CLI-level test for `--skip-lane` / `--skip-agents` (the surviving mutation is the only concrete
  coverage hole): a `dotf init --skip-lane` case next to `TestInitLane_DryRun` in `cli/internal/cmd/init_test.go`.
- Add the `dotf init lane` row to `docs/adr/adr-022-dotf-init-flagship.md`'s subcommand table, or note in
  `verification.md` that ADR-050 §6 supersedes that table.
- `dotf spec archive CLI-129-initrepo-lane-contract` is **advisable** in this state: the review is fresh for
  the current contract digests, the verdict is PASS, and no `[AGENT-DRAFT]` tag remains.
