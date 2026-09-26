---
spec: "GUARD-006-agent-tier-agreement"
verdict: "PASS-WITH-GAPS"
reviewed_sha: "519cf34fdeae47d67c495cd9cf374cbd2bacce30"
reviewer: "nan/deepseek-v4-flash"
date: "2026-09-25"
---

## Adversarial review

**Scope**: `GUARD-006-agent-tier-agreement` — the whole change, `git diff 2116a58434eae0b9819c39b509f60cb241c2e7a0...HEAD` (commits `e54263c` + `519cf34`, PR mlorentedev/dotfiles#1174).
**Sources**: `specs/GUARD-006-agent-tier-agreement/{proposal,tasks,verification}.md`, `features.json`; `cli/internal/doctor/{checks_agent_tiers.go,checks_agent_tiers_test.go,checks_model_map.go,report.go}`; `scripts/compile-harness.sh` (`skill_targets_agent`, `skill_field`, `deploy_agents`); `harness/{manifest,model-map}.json`, `harness/agents/curator/AGENT.md`, `.gitattributes`.

### Spec and task alignment

AC-by-AC, each verified by running something in this session:

| AC | Verdict | Evidence |
|---|---|---|
| Resolving record passes; pass line carries the count | met | `TestAgentTiersResolve/{a_tier_the_deploy_target_can_answer,every_pair_counts_toward_the_pass_line}`; real tree prints `[ OK ] every declared agent tier resolves for its deploy targets (1 checked)` |
| A target that cannot answer FAILs, naming record + tier + harness | met | `TestAgentTiersResolve/a_tier_one_deploy_target_cannot_answer` asserts `top`, `opencode`, `AGENT.md` |
| A tier no `tiers` block declares FAILs the same way | met | `TestAgentTiersResolve/a_tier_no_tier_block_declares_at_all` |
| No declared tier is not drift | met | `TestAgentTiersResolve/a_record_declaring_no_tier_is_not_drift` |
| `targets:` scope honoured | met | `.../a_record_scoped_by_targets_is_judged_only_against_those` + `TestRecordTargetsAgreesWithTheRender` (real shell predicate) |
| Absent `targets:` = every harness, own test | met | `TestRecordTargetsDefaultsToEveryHarness`, and the differential test would catch an inversion |
| Absent/unparseable manifest, no deploy targets → no FAIL | met | `TestAgentTiersMissingInputsAreNotFailures` (3 subtests); the unparseable branch warns rather than saying nothing |
| Verified against the real tree | met — reproduced | `DOTFILES_DIR=<worktree> dotf doctor --verbose` from a binary built here; the tier line above |

Also reproduced: `go build ./...`, `go vet ./...`, `GOOS=windows go vet ./...` (exit 0), `go test ./... -count=1` (all packages ok), `golangci-lint run` → `0 issues.` (v2.12.2), `gofmt -l internal/` → empty, all 7 `features.json` verifiers exit 0. Tasks are ticked against real work: PR #1174 exists, is merged, and its body names `specs/GUARD-006-agent-tier-agreement/`. Board matches reality (mlorentedev/dotfiles#1164 closed by #1174, 2026-08-22T04:57:25Z). No `[AGENT-DRAFT]`/`[AGENT-SUGGESTION]` tags remain.

The late fix (`519cf34`) is judged against the early commit and it interacts correctly: the check now runs the render's rule (`Contains` on the first `targets:` line) and proves it against the real shell predicate, so the previous round's two refuted points (quoted entry hiding drift, block-style list raising a FAIL) are closed — `TestRecordTargetsAgreesWithTheRender` fails when `recordTargets` is mutated to exact matching, which I verified rather than assumed.

### Findings

Mutation results are from a copy of `cli/` at `/tmp` (whole repo left clean): each mutant was applied to `checks_agent_tiers.go` and the package's tests re-run. **Caught**: absent-`targets` inversion, exact-match `targets`. **Survived**: first-wins removal, indent guard removal, `record_dir` default removal, pass-line guard removal, delimiter strictness.

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location |
|----------|---------|------|---------|----------|---------------------------|--------------|
| Major | THEORETICAL | fidelity of the second parser | `TestRecordTargetsAgreesWithTheRender` pins only the **`targets`** predicate. The `model` read path — the value `ResolveTier` actually consumes — is never compared against the render's `skill_field`, and the reader is more permissive than awk in three reproduced shapes. CRLF: `readAgentFrontmatter` → `model="top"`, `skill_field` → `top\r`, and `dotf harness resolve-tier "top\r"` exits 1, so the render hard-fails while the check reports OK. `model : top`: Go reads a tier, awk's `/^model:/` reads none. Indented `---`: Go treats it as the delimiter and reads the body's `model:` as the tier; awk sees no frontmatter. All three are false negatives or false positives in the direction the check exists to prevent (C15: a positive-looking line standing in for a check that read different bytes). | `/tmp`-copy probe of 6 record shapes; `bash -c 'skill_field …'` on the CRLF record → `top\r`; `dotf harness resolve-tier "top\r" --harness claude` → `tier "top\r" is not declared`, exit 1 | UNTESTED | tests (extend the differential test to run `skill_field` beside `readAgentFrontmatter` for `model`) |
| Major | THEORETICAL | untested rule added by the fix | The first-wins rule (`seen()`), added by `519cf34` specifically to mirror the render's `exit`-on-first-match, is pinned by a comment only. Both mutants that restore last-wins survive the entire package. A record carrying `targets:` twice would then be judged on the wrong line — the same class as the defect this round fixed. | mutants `A_last_wins` and `H_firstwins_dropped_helper_always_new` survive `go test ./internal/doctor/ -run 'TestRecordTargets\|TestAgentTiers'` | UNTESTED | tests |
| Minor | THEORETICAL | untested reader branches | Two more branches of the reader survive deletion: the leading-whitespace key guard (which is what stops a nested `  model:` from being read as the tier — probe confirms `nested_model` → `""` with it, and it is the only thing that gets that right) and the `record_dir` default `harness/agents`. Neither has a named test. | mutants `F_indent_guard_removed`, `D_recorddir_default_removed` survive; probe `nested_model` | UNTESTED | tests |
| Minor | THEORETICAL | output invariant | Nothing asserts that a FAIL suppresses the pass line: `if checked > 0 && failed == 0` can be reduced to `if checked > 0` with no test failing. The result would be `[ OK ] every declared agent tier resolves for its deploy targets` printed beside a `[FAIL]` — the contradictory pair this check's own doctrine treats as the failure mode worth preventing. | mutant `E_passline_no_failed_guard` survives | UNTESTED | tests |
| Minor | REAL | verification artifact is stale | `verification.md` says `gofmt -l internal/` is "clean apart from `internal/doctor/report.go`, which is **#1154**, pre-existing on main and **untouched here**". Both halves are wrong at HEAD: `report.go` **is** touched by this change (commit `e54263c` drops the trailing blank line gofmt flags), and `gofmt -l internal/` now prints nothing. As written the sentence describes a state that no longer exists, and it names a touch as untouched. | `git show --stat e54263c` lists `cli/internal/doctor/report.go | 1 -`; `gofmt -l` on the base copy flags it, on HEAD it does not; `gofmt -d` shows exactly that blank line | n/a (documentation) | spec artifact (`verification.md`, outside the contract set) |
| Minor | REAL | the count does not cover the zero case | `verification.md` claims the count "so a future reader can tell 'everything resolved' from 'nothing was looked at'". At zero pairs the check prints **nothing at all** — the test `TestAgentTiersResolve/a_block-style_targets_list_is_judged_as_the_render_reads_it` asserts the absence of `checked` — and the count is only visible under `--verbose` (non-verbose `dotf doctor` on the real tree shows the section header and the two INFO lines, not the tier line). So "nothing was looked at" is exactly the state the reader cannot distinguish. | non-verbose run output; the block-style subtest's `wantNot: ["checked"]` | `TestAgentTiersResolve` (asserts the count only at 1 and 2) | spec artifact (`verification.md` wording); emitting an INFO at zero is a judgement call, not a defect |

Not findings, recorded so they are not re-litigated: the substring rule (`copilot` contains `pi`) and block-style `targets:` deploying nowhere are the **render's** defects, faithfully predicted by the check and correctly excluded from its scope; mlorentedev/dotfiles#1733 exists, is open, and is titled exactly as the comment describes. The `render` column of `agents.deploy` being ignored by `deploy_agents` is pre-existing and would make a future `render: adapter` target fail both the deploy and this check — agreement, not drift.

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | B | Every acceptance criterion holds and the `targets` rule is now provably the render's, but the reader's fidelity on the field it resolves (`model`) is asserted in prose, with three reproduced divergences. |
| Verification       | C | Evidence re-runs green except one claim that does not survive re-running (`report.go` "pre-existing on main and untouched here" — it is touched by this diff); reproducible elsewhere, so not a D. |
| Scope              | B | Matches the proposal; the sole side-change is a one-line trailing-blank-line trim, which the verification artifact mis-describes rather than the diff hiding it. |
| Reliability        | B | Absent/unparseable manifest, missing record dir and empty deploy list are all handled; read-only diagnostic with no `--fix` by design; nothing to roll back. |
| Maintainability    | B | Three small functions (well under 40 lines, complexity low), lint and gofmt clean at the pin, comments say WHY; three branches of the reader have no test. |
| Handoff-readiness  | B | Spec set complete and ticked, promotion candidates dispositioned with reasons, resolution evidence carries reproducible commands; the one stale sentence should be corrected when the gaps are dispositioned. |

### Verdict
PASS WITH GAPS

### Recommended next steps

The contract set (`proposal.md`, `tasks.md`, `features.json`) is closed by this verdict — do not edit it, or the archive gate will refuse a review that describes a state that no longer exists. Everything below lands outside it and can be applied now.

- **Apply** (tests, no contract edit): extend `TestRecordTargetsAgreesWithTheRender` into a real differential test over **both** fields the check consumes — run the render's `skill_field` beside `readAgentFrontmatter` on the same bytes, including a CRLF record and `model : top`. That single test closes the Major above and makes the reader's fidelity a property the suite enforces instead of a comment.
- **Apply** (tests): a table case for a duplicated `targets:` key (first wins, as the render's `exit` does), and one for the leading-whitespace guard (a nested `  model:` must not become the tier).
- **Apply** (tests): assert that the pass line is absent when a pair fails, by extending the FAIL fixture with `wantNot: ["(1 checked)"]` or equivalent.
- **Apply** (`verification.md`, non-contract): correct the `gofmt`/`report.go` sentence to what is true at `519cf34` (the file was touched here; `gofmt -l internal/` is clean), and either drop or qualify the claim that the count separates "everything resolved" from "nothing was looked at", given the zero case is silent and the count is verbose-only.
- **Decline with a reason** if the zero-checked silence is deliberate: it is defensible (silence is not a pass line), and the finding is about the sentence that over-claims, not about the code.
- **No action** on the render's own substring/block-style defects (#1733) or on the pre-existing `report.go` gofmt debt — the first is tracked and out of scope, the second is now paid.

### Closing pass (Definition of Done)

- **Debt** — the two observations I could not fix from here are filed above as findings with root cause and options; the render's defects are already ticketed (#1733, #1170), so no new ticket is needed for them.
- **Knowledge** — no lesson or ADR proposed: the `gofmt`/verification staleness is a one-off artifact defect, and the fidelity gap is captured by these findings and their tests rather than by a new pattern.
- **Board** — read, not assumed: #1164 closed by #1174; the spec itself is what remains, and archiving it is what this review unblocks.
- **Review** — the PR is already merged and its round-1 findings were dispositioned in `519cf34` (the commit message names both). No new review comments were waiting when this ran; `gh issue view` hit a GraphQL rate limit on the first attempt, so the issue states were re-read over the REST API instead.
- **Evidence** — every claim above was produced by a command run in this session; the outputs that matter are quoted inline.
