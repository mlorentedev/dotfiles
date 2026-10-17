---
spec: "DX-005-zsh-completions"
verdict: "PASS-WITH-GAPS"
reviewed_sha: "9b9e99f0d8c76d1552eeaed0f1b29832df4cd103"
reviewer: "nan/glm5.3-flash"
date: "2026-10-17"
---

## Adversarial review

**Scope**: DX-005-zsh-completions (`.zshrc` completion dump rule + fpath + `menu select`; the `shell-completions` converge step; zsh-autosuggestions through mise's http backend, AC6; the 0.73.0-gated pin mark)
**Sources**: `specs/DX-005-zsh-completions/{proposal,tasks,verification,features.json}`; `git diff 43f550261efc8e278b5092ae97f24547b2c70372...HEAD` (launcher-resolved base)

### Spec and task alignment

- Proposal declares 6 ACs, all ticked in `tasks.md`; every tick maps to diff evidence (PR1 commit `20e68a3` for AC1–AC5, PR2 `f9524886` plus the mark commits `2788fc14`/`9b9e99f0` for AC6).
- Every AC has a named test and a non-vacuous command in `features.json`. The commands were re-run in this review, fresh, all green:
  - `cd cli && go build ./... && go vet ./...` — exit 0.
  - `go test ./internal/converge ./internal/tools ./internal/cmd ./internal/doctor` — ok.
  - `bats tests/zshrc-guards.bats` — 17/17; `bats tests/versions-conf.bats` — 24/24 incl. the "marks no file tool while DOTF_VERSION predates fileTools" order guard; guard-zsh bats green.
  - `golangci-lint run` (2.12.2, pinned) — 0 issues.
  - Live AC4: built-tree converge in a scratch HOME printed `[CHANGE] shell-completions wrote _dotf, _herdr` / `1 changed`, second run `[ OK ] completions up to date` / `0 changed, 1 ok`; `_dotf` starts `#compdef dotf`.
  - AC5 spot check: medians of 12 `zsh -ic exit` under scratch ZDOTDIR, main `.zshrc` 0.05 s vs branch 0.05 s (`/usr/bin/time`, 10 ms resolution) — no regression; the spec's 20-run medians (44.2–44.7 ms vs 44.4–45.5 ms) stand.
- Mutation checks run by this reviewer, then reverted (tree left clean):
  - Forced `compinit -C` in the rc → `@test "a completion newer than a fresh dump rebuilds it (dump)"` and the brew-before-compinit source-inspection test went red. AC1's rule is enforced by named tests on both sides.
  - Removed the `isFileTool` fast path in `runsAtPin` → `TestSync_ADanglingLatestLinkIsRepairedNotAccepted` (and the install/verify path) failed with "not running at their pin: zsh-autosuggestions". The file-tool probe is load-bearing and tested.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Minor | REAL | tests | `verification.md` claims the stdout-only runner "keeps a tool's stderr warning out of the file", but no named test exercises `ExecStdout` at all; swapping it for `tools.ExecRunner` (which appends stderr) would turn no test red and would silently embed a tool's warning inside `_dotf`/`_herdr`, corrupting the completion script compinit loads. | `grep -rn ExecStdout cli/internal/converge/*_test.go cli/internal/cmd/*_test.go` → no reference; go suite green on the unchanged code | UNTESTED | tests (add a converge case whose fake tool writes to stderr and assert the written file bytes), disposition in `verification.md` |
| Minor | THEORETICAL | shell-rc | The 24 h / newer-completion dump rule lives only in the non-oh-my-zsh `else` branch. On a machine where omz is installed, dump policy is omz's own (omz does not run `compinit -C`, so completions are not lost — they surface on omz's next full run — but the rc's rule is not what governs there). The proposal's "What" does not carve this out; the bats sandbox never sources omz. | Code read of `.zshrc` if/else; no omz path in `tests/zshrc-guards.bats` | UNTESTED | spec artifacts (disposition in `verification.md`); a follow-up ticket if omz machines are in scope |
| Minor | THEORETICAL | security | The full rebuild runs `compinit -u`, which skips the insecure-directory audit and loads completions from group-writable site-functions (stock Debian/Ubuntu). Deliberate (CI fails otherwise, lesson recorded in `verification.md`), and the trust level matches the old unconditional `compinit -C` — but it widens the audit-free path to every full rebuild, a local-privilege consideration worth a line if the policy is ever revisited. | `@test "an insecure completion directory neither prompts nor blocks the rebuild (dump)"` codifies the behavior; `-u` flag in `.zshrc` | same named test (codifies, does not bound) | spec artifacts (`verification.md` decision note) |
| Question | SPECULATIVE | scope | The review range `43f5502...HEAD` contains other specs' landed work (CLI-131, TOOL-030, #2651/#2657/#2660/#2663, the 0.73.0 release commit). DX-005's own footprint (`.zshrc`, `shellcompletions*.go`, `mise_files*.go`, `mise.go`, the four test files, `versions.conf`, its spec folder) was reviewed in depth; the rest was scanned for interactions only — the registry-order test was updated correctly for `shell-completions` and no cross-feature interaction was found. Confirm the launcher intended the range to include that merged work. | `git log 43f5502..HEAD --oneline` (14 commits, several unrelated) | — | — |

Not findings, argued against: `mise where` + `latest/<file>` double check in `fileToolInstalled` is tested (`TestSync_ADanglingLatestLinkIsRepairedNotAccepted`); the `http:`-prefixed orphan mapping is tested both directions (`TestSync_DoesNotPruneAPinnedFileToolAndPrunesAnUnpinnedOne`); Windows skip is by `runtime.GOOS` in render/plan/orphans and GOOS-vetted; the digest gate fails closed at parse (`TestParseMisePins_RejectsAFileToolWhoseDigestIsNotPinned`) and the rendered checksum bytes are pinned (`TestRenderMiseConfig_PinsAFileToolByUrlAndDigest`); the `versions-conf.bats` order guard is itself mutation-backed per the spec (premature mark fails). No test deletions in DX-005's footprint; `tests/verify-setup.bats` awk was strengthened, not removed.

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | A | All 6 ACs verified with named tests and reproduced live, including negative paths (skip by name, empty/failing output, drift rewrite, digest gate, dangling `latest`, Windows skip). |
| Verification       | A | Every criterion carries a reproducible command; build, vet, go tests, bats suites, lint, live converge and two mutations were re-run fresh in this review and confirmed. |
| Scope              | B | DX-005's own diff matches the proposal with no creep; the launcher-resolved range additionally carries other specs' merged work, noted as a question above. |
| Reliability        | A | Idempotent (`changed=0` reproduced), atomic writes, repair path for a deleted file and a dangling `latest`, loud errors for a present-but-failing tool. |
| Maintainability    | A | Small functions, table/fake-based tests, comments explain why (stderr exclusion, `-u` rationale, `latest` check). |
| Handoff-readiness  | B | `features.json` complete, decisions and promotion candidates recorded, archive checklist present; the stderr-exclusion claim outruns its test coverage (finding 1). |

### Verdict

PASS WITH GAPS — no Blocker, no REAL Major, no rubric D. The gaps are minors, each tracked above with a disposition path: the UNTESTED `ExecStdout` stderr-exclusion claim (finding 1) should get a named test and a `verification.md` disposition; findings 2–3 are documentation dispositions. None of them blocks archiving.

`dotf spec archive` is **advisable** in the current state once finding 1 is dispositioned (applied as a test, or ticketed with its root cause) — the minors do not require a re-review.

### Recommended next steps

- **tests** (outside the contract set, safe to apply now): add a converge test whose fake `completion zsh` writes to stderr and asserts the file contains stdout only — closes finding 1's UNTESTED claim in `verification.md`.
- **spec artifacts** (`verification.md`, excluded from the staleness check): disposition findings 1–3 there — applied, ticketed, or declined with a reason. Do not edit `proposal.md`, `tasks.md` or `features.json`: that would stale this review.
- **follow-up ticket (optional)**: if any machine runs oh-my-zsh, decide whether the dump rule should also govern the omz branch or the omz policy is accepted as-is (finding 2).
