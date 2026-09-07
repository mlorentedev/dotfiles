---
spec: "HARNESS-120"
verdict: "PASS"
reviewed_sha: "9d3a7eea30f86df19100f177d77e1d6efc1de41c"
reviewer: "nan/deepseek-v4-flash"
date: "2026-09-06"
---

## Adversarial review

**Scope**: HARNESS-120
**Sources**: `specs/HARNESS-120/{proposal,tasks,verification}.md`, `specs/HARNESS-120/features.json`, `git diff 39754ec80efecd772570d9c2c2262dd738a3b7d2...HEAD`

### Spec and task alignment

- `tasks.md` maps cleanly to the diff: pure `harness` functions (`route.go`, `preamble.go`) land before the command wiring (`agent_auto.go`, the `agent.go` refactor), and each task carries the `[AC<n>]` tag to its criterion. Every implementation box is ticked; the one unticked box is the adversarial review itself, which this run closes.
- The round-1 Blocker (AC7 unverified) has been addressed by commit `9d3a7ee`: `verification.md` now quotes a real end-to-end dispatch. The round-1 Minor (dictated `--tier` not naming legal tiers) was fixed in `ResolveChain` (commit `6d8db91`) and pinned by `TestDictatedTierRefusalNamesTheLegalOnes`. Both round-1 findings are dispositioned in `verification.md`.
- `features.json` lists all seven features with non-vacuous verification commands, and its `f7` command now **asserts** (a `jq -e` predicate) rather than only printing. All `state` values remain `pending` (correct: only the harness sets `passing`).
- No `[AGENT-DRAFT]`/`[AGENT-SUGGESTION]` tags remain in any spec contract file.

### Verification performed

Built and tested the full Go layer (`go build`, `go vet`, `go test ./...` — all pass; `GOOS=windows go vet` clean; `golangci-lint` at the `versions.conf` pin `2.12.2` reports `0 issues`). Ran the bats guard `guard-review-verdict-honours-reality.bats` (7/7) and `guard-lesson-numbers-unique.bats` (4/4). Reproduced AC1–AC6 at the CLI with the built binary against the shipped roster and `model-map.json`, and AC4 with a fixture persona (illegal tier and empty model). Confirmed the AC7 deterministic half by dry-run (`--role reviewer` resolves tier `mid` inferred, pool `claude:sonnet`). Mutation-checked two paths and reverted: making the ambiguous case return its first candidate, and sending the bare task instead of the composed preamble — both fail the relevant tests, so the tests are non-vacuous.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Minor | REAL | UX / error reporting | `auto --role hermes-nan` refuses with `no persona named "hermes-nan"`, but `hermes-nan` **is** declared — it is just `kind: autonomous`. Behaviour is correct (fail-closed: an autonomous steward is not a dispatchable phase), but the message is factually imprecise and tells an operator who knows the persona exists to look for a missing record. | `dotf agent auto --role hermes-nan --backend dry-run` → `Error: no persona named "hermes-nan" ... The roster declares: architect, builder, curator, planner, reviewer, shipper` | `TestAgentAuto_RefusesAPersonaTheRosterDoesNotDeclare` covers the nil-persona path (potato); the autonomous-kind message is not pinned by a distinct assertion — **UNTESTED** for that specific case | code (reword the refusal to distinguish "not declared" from "declared but not invocable") + tests |
| Minor | REAL | Verification accuracy | The AC1 output quoted in `verification.md` is not byte-exact: it omits `"duration_ms":0` and `"truncated":false`, which the real command emits. The criterion (role/tier) is correctly shown, so this is cosmetic, but the quote is not verbatim as AC7's is. | Real CLI output includes `..."exit":0,"duration_ms":0,"output":"would dispatch..."` — the quoted block skips the two fields. | `TestAgentAuto_DerivesBothRoleAndTierFromTheTask` proves the behaviour; the discrepancy is docs-only | `verification.md` (outside the contract set — free to edit) |
| Minor | REAL | Maintainability | `resolvePersonaForTask` (`cli/internal/cmd/agent_auto.go`) is ~48 lines, over the repo's 40-line function-length rule. Cyclomatic complexity is 9 (within limits); the inflation comes largely from the multi-line `fmt.Errorf` prose, not control flow. | `awk` function-length count; `gocyclo` reports CC 9 for the function. | `TestAgentAuto_*` exercises the function | code (optional; extract the two error branches) |
| Minor | SPECULATIVE | Reliability | `taskDelimiter = "===== TASK ====="`. If a user task text or a record body ever contains that exact string, the preamble's task framing becomes ambiguous (the task could be read as instruction, or the split point lost). Deliberately chosen as unlikely; no repro. | Code read of `preamble.go`; the comment records the reasoning. No observed occurrence. | UNTESTED | code (surface only; do not gate) |
| Question / assumption | — | Verification | AC7's live dispatch (status `ok`, pool `claude`, model `sonnet`, 12.7 s, output `OK`) is documented in `verification.md` and `features.json`. I verified the deterministic half (dry-run resolves reviewer → mid inferred → claude:sonnet, matching the evidence's `role_from: dictated` / `tier_from: inferred`) but did **not** independently re-run the live dispatch — that spends pool quota and is a pool-spend decision. The evidence is internally consistent and the machine is identified (the identity gate passes). | Dry-run of the AC7 command reproduces `{"tier":"mid","role":"reviewer","resolution":{"role_from":"dictated","tier_from":"inferred"},"pool":"claude","model":"sonnet"}` | `TestAgentAuto_SendsThePersonasOwnRecordToTheBackend` + the deterministic AC7 dry-run | — (confirmation only) |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | B | All seven criteria verified; negative paths (AC2/AC3/AC4 refusals, illegal/absent tier, autonomous persona) covered; no observed functional defect, but the autonomous-persona refusal message is imprecise. |
| Verification       | A | AC1–AC6 reproduced independently with reproducible commands + outputs; AC7 documented with a real dispatch whose deterministic half I re-verified; one cosmetic quote discrepancy. |
| Scope              | B | Authored diff matches the proposal exactly; the diff as reviewed also carries upstream HARNESS-111 SKILL.md/bats/lesson-275 changes brought in by the `main` merge (commit `511c77d`/`53e9c84`), which are not this PR's additions. |
| Reliability        | A | Fail-closed refusals with actionable error messages, no state mutation, dispatch idempotent, rollback-safe. |
| Maintainability    | B | New functions CC ≤ 9, well-named, comments explain WHY; one function (`resolvePersonaForTask`) slightly over the 40-line rule. |
| Handoff-readiness  | A | proposal/tasks/verification/features updated in-session; lesson 276 captured; PR #1546 references the spec; AC7 and follow-ups (#1547, #1548) documented. |

### Verdict
PASS

`dotf spec archive` is **advisable** in the current state: the review is fresh (reviewed_sha = HEAD = `9d3a7ee`, the latest change to the contract set), signed by a pool member (`nan/deepseek-v4-flash`), and non-blocking. No Blocker and no REAL Major; the rubric is all A/B with no C/D. The Minor findings above are tracked, not fixed — the contract set stays closed.

### Recommended next steps

- **Code** (optional): reword the `--role` refusal to distinguish "not declared" from "declared but `kind: autonomous`", and add a named assertion pinning the autonomous-kind case (currently UNTESTED). Low priority.
- **`verification.md`** (outside the contract set): make the AC1 quoted output byte-exact (add `"duration_ms":0` / `"truncated":false`), matching how AC7 is quoted. Low priority.
- **Code** (optional): consider trimming `resolvePersonaForTask` toward the 40-line rule. Low priority.
- **Surface only, do not gate**: the `===== TASK =====` delimiter-collision possibility — no repro, documented.
- **Question to confirm**: if independent confirmation of AC7's live dispatch is wanted, re-run `dotf agent auto --role reviewer --task 'Reply with the single word OK and do nothing else.' --timeout 5m` on an identified machine (pool spend). The deterministic half is already verified.
