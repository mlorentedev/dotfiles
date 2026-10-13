---
spec: "AI-045-nan-catalog-alignment"
verdict: "FAIL"
reviewed_sha: "081374d32c532bf568c164bde7843b586f2a4cae"
reviewer: "nan/glm5.3-flash"
date: "2026-10-13"
---

## Adversarial review

**Scope**: AI-045-nan-catalog-alignment — AC1–AC10 (retire mimo-v2.5, align NaN consumers with the catalog, preflight, canary, cross-provider review pool, repository-wide review queue)
**Sources**: `specs/AI-045-nan-catalog-alignment/{proposal,tasks,verification,features.json}`; `git diff a94ae3465a245b4a29af25e4b8a056d0f4cd62d0...HEAD` (base stated by the launcher; the diff spans two weeks of merged main history, so per-file review focused on the spec's own contract sites: `harness/{model-map,reviewer-pool,nan-quotas,model-pins}.json`, `.pr_agent.toml`, `.github/workflows/{pr-agent,model-canary}.yml`, `scripts/pr-agent-*.sh`, `ai/pi/*`, `ai/opencode/opencode.jsonc`, `cli/internal/{nanprobe,harness,spec,cmd}`, and the named test files)

### Spec and task alignment
- All ten ACs map to implemented, named tests; tasks.md is fully ticked and matches the diff sites.
- Independently re-run this session, all green: `bats tests/reviewer-pool.bats tests/opencode.bats` (50 ok), `tests/pr-agent-model-preflight.bats` (10 ok), `tests/pr-agent-route.bats tests/pr-agent-route-real.bats` (37 ok, 0 not ok), `tests/pr-agent-publish-guard.bats` (24 ok), `tests/model-canary.bats` (9 ok), the AC5 focused filters on `tests/opencode.bats`/`tests/pi-config.bats` (4 ok), `go test ./internal/nanprobe/ ./internal/cmd/ -run HarnessCanary` (ok), and the three AC4 focused Go tests (ok). `shellcheck` on the three pr-agent scripts and `git diff --check` are clean.
- Mutation check (AC2): changing the workflow's `DECLARED_MODEL` off the toml's value turns "the chain the preflight probes equals the toml's, model and fallbacks" red with a message naming both sides; reverted, green again. Working tree left clean.
- AC4's declared values match the proposal exactly: per-model `7/7/7` (deepseek-v4-flash, glm5.3-flash, qwen3.8-flash), `5/5/5` (mimo-v2.6-flash, qwen3.6, gemma4), key-wide cap 7. AC9's pool weights sum to 100 (27+19+19+35), Sonnet as `route: risk` weightless. AC5: `enable_thinking` appears nowhere in the live configs (grep exit 1).
- No `[AGENT-DRAFT]`/`[AGENT-SUGGESTION]` tags remain in any spec file.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Major | REAL | features.json (contract set) | Four `verification` commands have drifted from the current test names, so the harness-facing contract cannot prove the behaviors they name. f6 `every attempt streams its NaN calls` and f7 `the job outlives both attempts` match **zero** tests in `tests/pr-agent-config.bats` (f7 exits 0 with `1..0` — a vacuous pass; f6 exits 1). f9's third filter `one pinned model with bounded` matches zero, so `grep -c '^ok '` returns 2, not the required 3 (exit 1). f3's second clause `preflight runs wherever PR-Agent runs` runs 0 tests and exits 0 vacuously. The behaviors themselves ARE implemented and pinned — by `the preflight and the draw run wherever PR-Agent runs, and NaN down hands over to Anthropic`, `the NaN attempt streams its calls, and an Anthropic attempt carries no NaN transport`, `the job outlives both bounded Actions and the probes before them`, and `the Anthropic attempts run the routed model, allowlisted, with bounded input and output` — but tasks.md's ticked claim "Every acceptance criterion has a matching entry in features.json with a non-vacuous verification command" no longer holds. This is the file the archive gate's harness runs; a contract that vacuously passes or fails as written defeats that gate. | Reproduced this session: `bats -f 'the job outlives both attempts' tests/pr-agent-config.bats` → `1..0`, exit 0; f6 filter → exit 1; f9 filter count = 2; f3 clause → `1..0`, exit 0 | The behaviors are proven by the four named tests above; the defect is in the contract's commands, not the code. The fix needs no new test — re-run the corrected commands for evidence | spec (features.json — contract set: fix, then re-review) |
| Minor | REAL | CI lint | `actionlint .github/workflows/pr-agent.yml` exits 1: `queue: max` on the job-level `concurrency` block is an unexpected key for actionlint 1.7.12. The proposal declares this deviation openly and the live run (36835280780, plus the three-run queue evidence on #1923) proves GitHub accepts it, but tasks.md's ticked "Lint passes (shellcheck, actionlint)" is no longer accurate as stated, and nothing in CI or the git hooks runs actionlint, so the claim is checked by no gate. | Reproduced this session: exit 1, `unexpected key "queue" for "concurrency" section`; no actionlint invocation found in `.github/workflows/`, `scripts/`, or `~/.dotfiles/git-hooks` | UNTESTED | code (add an `.actionlint.yaml` ignore for the job-level `queue` key, or pin an actionlint version that knows it — outside the contract set); disposition the stale tasks.md line in `verification.md` |
| Question / assumption | — (resolved during review) | ai/pi/nan-package-windows.json | Still contains `mimo-v2.5: 1048576` and `qwen3.8-flash: 262144`, which look like AC1/AC5 violations on a grep. Resolved: it is a committed snapshot of the pinned `@gtrabanco/pi-nan-provider@0.7.0` package's own declarations, explicitly not deployed (ai/pi/README.md), parity-tested against the installed package by `pi-nan-package: ai/pi/nan-package-windows.json equals the installed package's windows`. Not a live binding; AC1's file list does not include it. No action needed; recorded so the next grep-driven reviewer does not re-open it. | File header comment, README table, `tests/pi-nan-package.bats` | `tests/pi-nan-package.bats`, `tests/nan-context-windows.bats` | none |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | B | All AC behaviors verified by named tests plus one live mutation this session; no behavioral defect found. |
| Verification       | C | verification.md's narrative evidence is extensive and reproducible, but 3 of 10 machine-verifiable features.json commands are stale — vacuously passing or failing — so the harness-facing proof surface is broken. |
| Scope              | B | The launcher's base predates two weeks of merged main, inflating the diff; the spec's own PR set maps cleanly to its ACs with no creep found at the contract sites. |
| Reliability        | B | Fallbacks, the publish guard, the queue and the canary are tested and live-verified, including a real second-attempt handover (run 38429799619). |
| Maintainability    | B | shellcheck/vet clean, focused test files per concern; the twice-declared chain is held equal by a mutation-verified test. |
| Handoff-readiness  | B | Spec artifacts complete, promotion lines answered, lessons recorded (320/327/369/370); docked for the stale features.json contract. |

### Verdict
FAIL

One **REAL Major** in the contract set: `features.json`'s verification commands for f3/f6/f7/f9 have drifted from the current test names, so the machine-readable contract the archive gate relies on vacuously passes (f7, f3's second clause) or fails as written (f6, f9). Per the severity × reality rule this forces a fix and a re-review; the rubric's C in Verification corroborates it. No behavioral, security or correctness defect was found in the shipped code — the failure is in the contract's proof commands, not the change.

### Recommended next steps
- Fix `specs/AI-045-nan-catalog-alignment/features.json` (contract set — this is the point of the FAIL, and a re-review follows it): point f3's second filter at `preflight and the draw run wherever PR-Agent runs`, f6 at `the NaN attempt streams its calls`, f7 at `the job outlives both bounded Actions`, and f9's third filter at `routed model, allowlisted, with bounded input and output` — then run all four corrected commands and record exit 0.
- Outside the contract set, for the implementer to disposition in `verification.md`: commit an `.actionlint.yaml` ignore for the job-level `queue` key (or justify its absence), so the lint claim becomes gate-checked rather than aspirational.
- No change requested to `proposal.md` or `tasks.md`; the tasks.md lint line and the features.json non-vacuous-command claim are stale as *descriptions of the current tree*, and the findings above are the record of that.
