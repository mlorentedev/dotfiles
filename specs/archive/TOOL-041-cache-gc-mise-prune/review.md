---
spec: "TOOL-041-cache-gc-mise-prune"
verdict: "PASS"
reviewed_sha: "d282b9dfa43f4b60000b655dba1736024c24eeef"
reviewer: "nan/deepseek-v4-flash"
date: "2026-11-15"
---

## Adversarial review

**Scope**: `TOOL-041-cache-gc-mise-prune` — `cache-gc` also prunes old mise tool versions, with python held back
**Sources**: `specs/TOOL-041-cache-gc-mise-prune/{proposal,tasks,verification,features}.json`; `git diff 49639be42f196b71f55556cdd2c8ae09806944fa...HEAD` (commits `4c7688f5`, `aef3f12a`, `d282b9df`); `cli/internal/converge/cachegc.go`, `cli/internal/converge/cachegc_test.go`, `docs/adr/adr-036-install-channels.md`, `docs/runbooks/guide-self-deploy-timer.md`, `docs/lessons/lesson-605-…md`

### Spec and task alignment

- **AC1** (apply lists the installed tools, then `mise prune --tools --yes` naming every installed tool but python, sorted) — implemented in `misePrune`/`cmdsFor`/`commands` (`cachegc.go:271-320`), covered by `TestCacheGC_MisePruneLeavesPythonAlone` and by the extended `TestCacheGC_ApplyRunsEveryPresentPrunerInOrderAndWritesTheStamp`. **Mutation-proven**: with `miseHeld` emptied both tests go red (`mise prune --tools --yes age http:zsh-autosuggestions python` where the test wants no `python`), and reversing the sort reddens both ordering assertions.
- **AC2** (python alone runs no prune; a failed listing prunes nothing and fails the step) — covered by `TestCacheGC_MisePruneRunsNothingWhenOnlyPythonIsInstalled` and `TestCacheGC_MiseListFailureFailsTheStepAndKeepsItDue`. **Mutation-proven**: deleting the `len(names) == 0` guard makes the first fail with `mise prune --tools --yes` (the bare form that prunes everything). I probed the three neighbouring inputs with a throwaway test file, deleted after the run: unparseable JSON → `mise: reading mise ls --installed --json: invalid character 'd'…`, nothing pruned; `{}` and `null` → listing only, nothing pruned; a failed listing leaves the stamp unwritten, so a second Reconcile is still `due (no previous run recorded)` and retries.
- **AC3** (runbook and ADR-036 name the pruner and the python hold) — `docs/runbooks/guide-self-deploy-timer.md:69` row and `docs/adr/adr-036-install-channels.md:156` amendment; the `features.json` f3 grep command passes.
- `tasks.md`: every implementation and closing box is ticked and each tick has diff evidence. `verification.md` carries evidence for all three criteria; its test-status block reproduced (see below). No `[AGENT-DRAFT]` / `[AGENT-SUGGESTION]` tags in the folder.
- `contract_digests` in `review-request.json` match `dotf spec digest TOOL-041-cache-gc-mise-prune` byte for byte (features.json `68a6a0f2…`, proposal.md `5e369383…`, tasks.md `b199ffd2…`), so the contract I reviewed is the contract that was launched.

**What I reproduced, not read** (fresh, in this worktree, at `d282b9df`):

- `cd cli && go test ./internal/converge -count=1` → `ok … 27.8s`; `go vet ./...` and `go build ./...` → exit 0; `golangci-lint run ./internal/converge/...` (v2.12.2, go1.26.10) → `0 issues`.
- The f1/f2 verification commands from `features.json` → both PASS.
- The real command the code builds, against the Mac's mise 2026.11.6: `mise ls --installed --json` (46 tool keys, one non-bare `http:zsh-autosuggestions`) then `mise prune --tools --yes --dry-run <every tool but python>` → **exit 0, 34 prunable versions, zero `python@` lines**, and the same output on a second run (deterministic, no hidden state). The proposal says 33; 34 is the same measurement one version later.
- End-to-end integration: a locally built `dotf converge --plan --only cache-gc` with a scratch `XDG_STATE_HOME` prints `would run … mise prune --tools (every installed tool but python) …` and runs nothing (no state dir created).
- The hold's premise is real, not asserted: `~/.local/share/uv/tools/hive-vault/pyvenv.cfg` and `~/Projects/kubelab/.venv/pyvenv.cfg` both carry `home = /Users/manu/.local/share/mise/installs/python/3.13.16/bin`.
- The flags are not a new mise floor: `-y/--yes` is a mise **global** flag (`mise --help`) and the repo already runs `mise prune --yes --tools` in the `tools` step (`cli/internal/tools/mise.go:697`). `mise prune --help` documents `--tools` and positional `[INSTALLED_TOOL]…`, which is exactly the form `misePrune` builds.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Minor | REAL | tests | The pruner's *unparseable listing* branch (`json.Unmarshal` error, `cachegc.go:305`) has no test of its own; only the exec-failure half of "the listing failed" is covered. | Probe: `miseInstalled = "deepseek is not json"` → `err=mise: reading mise ls --installed --json: invalid character …`, `log=["mise ls --installed --json"]`, no prune. Behaviour is correct; the branch is unexercised. | UNTESTED (nearest named test, `TestCacheGC_MiseListFailureFailsTheStepAndKeepsItDue`, fails at `r.run`, never at `json.Unmarshal`) | tests |
| Minor | REAL | maintainability | `plan` is a hand-written literal that duplicates `miseHeld` and the command shape, so it can drift from behaviour with nothing catching it; the `cmdsFor != nil` branch of `plannedCommands` is exercised by no test. | End-to-end plan output: `mise prune --tools (every installed tool but python)`; `TestCacheGC_PlanWhenDueListsThePresentPrunersAndRunsNothing` presents only uv+npm, so the new branch never executes in the suite. | UNTESTED (add a plan case with `mise` present, or derive the plan from `miseHeld`) | code + tests |
| Minor | THEORETICAL | resilience / spec coverage | The python hold guards one deletion path only. The `tools` step's orphan prune (`mise.go:697`, fed by `mise ls --prunable --json`, `mise.go:517`) runs the same `mise prune --yes --tools <names>` with no hold, so a python that ever loses its pin is deleted there, breaking exactly the venvs this spec protects. | Code read; `mise ls --prunable --json` is a global listing like `--installed` (34 prunable versions measured, none python today because python is pinned). No repro: needs `PYTHON_VERSION` removed from `versions.conf`. | UNTESTED | code (hold in the shared prune) or an ADR-036 sentence stating the assumption |
| Minor | THEORETICAL | maintainability / SSOT | Two listings now answer "what is prunable": the new `mise ls --installed --json` (every tool, then `mise prune` filters) and the existing `mise ls --prunable --json` already parsed by `Orphans`, which also carries the `http:<name>` handling lesson 463 records. The second would name only tools with something to prune. | Code read of `cachegc.go:296` vs `mise.go:517-536`; lesson 463 documents the shared key shape. | UNTESTED | code (reuse `ls --prunable`), no behaviour change required |
| Question / assumption | SPECULATIVE | correctness | `miseHeld` matches the exact key `python`. A python installed under any other key (a non-core backend, a renamed tool) would not be held. | Measured on this Mac: the key is `python` (3 installed versions); the only non-bare key among 46 is `http:zsh-autosuggestions`. No evidence of a non-`python` key exists. | UNTESTED | confirm; no change unless confirmed |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | A | All three criteria verified, negative paths (listing failure, empty list, python hold) covered and mutation-proven; no defect observed; the real `mise` command run end to end. |
| Verification       | A | Every command in `verification.md` reproduced in this session (tests, vet, lint, real dry run, plan), and the spec's own count (33) matches mine (34) one version later. |
| Scope              | A | Nine files, exactly the proposal's surface: pruner + tests + ADR-036 line + runbook row + lesson 605 + the spec folder. No creep. |
| Reliability        | A | Failures fail the step loudly and leave it due; an empty tool list runs nothing (the dangerous bare `mise prune` is guarded); dry runs are deterministic. |
| Maintainability    | B | `misePrune` is ~25 lines at low complexity and reads well, but the plan literal can drift from `miseHeld` and a second "what is prunable" listing path now exists. |
| Handoff-readiness  | A | ADR-036 amended in place, runbook row added, lesson 605 written, `verification.md` filled with reproducible evidence. |

### Verdict
PASS

### Recommended next steps

Each is outside the contract set (`proposal.md`, `tasks.md`, `features.json` are closed by this verdict — do not edit them, or the archive gate refuses this review as stale).

- **Disposition in `verification.md`** (or a follow-up ticket) for the two untested branches: the `json.Unmarshal` failure path and the `cmdsFor` arm of `plannedCommands`. One table-driven case each in `cachegc_test.go`; the behaviour is already correct, so this is regression insurance, not a fix.
- **Follow-up ticket (code)**: consider deriving the dry-run `plan` from `miseHeld` (or asserting it against the built command) so the two cannot disagree.
- **Follow-up ticket (code or ADR-036 sentence)**: decide whether the `tools` step's orphan prune should also hold python. If the answer is "not for now", say so in ADR-036 so the next reader does not assume the hold covers every deletion path.
- **Suggestion (code)**: `mise ls --prunable --json` already exists in this repo with its parsing and `http:<name>` handling (`mise.go:517`); reusing it would name only tools that have something to prune and keep one listing vocabulary.
- **UNVERIFIED in this run** (stated, not silently skipped): `golangci-lint` under `GOOS=windows` and mise's Windows runtime behaviour (I ran darwin only; no Windows host here); the full `go test ./...` across every `cli` package (I ran the changed package plus `go vet ./...` and `go build ./...`); the `dotf spec archive` gate implementation, which lives outside this repository; PR and bitácora state, which this review did not touch.
