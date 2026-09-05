---
spec: "CLI-021-dotf-vault-build-knowledge"
verdict: "PASS"
reviewed_sha: "52035f1a174e82f3176bc458aa128ebc77909f9e"
reviewer: "nan/deepseek-v4-flash"
date: "2026-09-04"
---

## Adversarial review

**Scope**: CLI-021-dotf-vault-build-knowledge (#490, AUDIT-007 PR5) at HEAD `52035f1`. The spec is a
three-increment build-beside port of the knowledge half of `dotf vault`. Increment 1 (`crystallize`)
and increment 2 (`health`) landed earlier and were reviewed through their parity suites here;
increment 3 (`dotf vault maintain`) is the substantive new diff in this HEAD and is the focus of the
adversarial pass.

**Sources**: `specs/CLI-021-dotf-vault-build-knowledge/{proposal,tasks,verification,divergences}.md`,
`cli/internal/vault/{maintain,health,crystallize}.go`, `cli/internal/cmd/{vault_maintain,vault_health,vault}.go`,
`cli/internal/vault/{maintain_test,crystallize_test}.go`, `tests/vault-maintenance-weekly.bats`,
`tests/knowledge-crystallize-go-parity.bats`, `tests/vault-health-go-parity.bats`,
`scripts/vault-maintenance-weekly.sh`, `scripts/vault-health.sh`, `harness/reviewer-pool.json`.

### Spec and task alignment

- Acceptance criteria (proposal §Acceptance) are all met for the reviewed state. Criterion 1
  (`--help` + `--all` + positional dir) verified by running `/tmp/dotf-review vault {crystallize,maintain,health} --help`;
  all three exist with the described interface, and `crystallize` supports `--all` and a positional path.
- Criterion 2 (golden characterization, byte-identical) holds for crystallize (14/14 parity bats) and
  health (17/17 parity bats). **Maintain is deliberately NOT golden-characterized** (tasks.md §4) — a
  recorded decision, not an omission: its inner two subcommands are already byte-pinned, and the
  wrapper's behaviour (log path, framing, exit code) is asserted by table tests + 4 bats cases.
- Criterion 3 (HARNESS-029) verified by mutation: replacing `AppendBeforeHandoff`'s guard with a naive
  append turned `TestAppendBeforeHandoffKeepsHandoffLast` red on 3 cases (and the parity case 6).
- Criterion 4 (table-driven unit tests) satisfied: `crystallize_test.go`, `health_test.go`,
  `maintain_test.go` (22 cases).
- Criterion 5/6 (negative: no twin deleted / no caller repointed, shell stays canonical) hold for the
  increments' own diffs — nothing under `scripts/`, `setup-*.{sh,ps1}`, or the vault is touched by the
  increment-3 diff. **However, the crystallize twin was deleted by CLI-050 (#1269)** (a sibling spec),
  not by CLI-023 as the proposal's plan originally described; see finding Q1.
- `tasks.md` §5 close-out boxes (`git diff` touches only `cli/`/`specs/`, PR merged, archive) are still
  `[ ]`, which is the correct pre-archive state.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Minor | REAL | log content | The Go `maintain` log omits the ANSI SGR colour codes the shell twin's log carries (from `vault-health.sh` via `utils.sh`). This is the documented deliberate divergence for the whole Go CLI, but it is the one increment where it is NOT normalised away (no golden strips it), so it is the first place a human comparing the two logs sees it. | Ran `bash scripts/vault-maintenance-weekly.sh` and `/tmp/dotf-review vault maintain` on identical empty HOME/VAULT: shell log has `[0;34m[1/7][0m`, `[1;33mSKIP[0m`, `[0;31m[ERROR][0m`; Go log has none. Content is otherwise byte-identical (diff after stripping ANSI + timestamps is empty). | UNTESTED (no test asserts the maintain log's ANSI absence; bats cases assert content, not colour) | spec (record as a maintain divergence in verification.md, mirroring the crystallize note) |
| Minor | REAL | spec artifact (verification.md) | verification.md's "load-bearing check" says `git diff --stat` must touch only `cli/` and `specs/`, but the proposal's corrected acceptance allows `tests/` (the negative that matters is "nothing under `scripts/`, `setup-*.{sh,ps1}`, or the vault"), and the actual increment-3 diff touches `tests/vault-maintenance-weekly.bats`. The sentence is stale and contradicts both the corrected acceptance and the diff. | Read proposal.md ("Corrected to what `verification.md` has actually been enforcing… nothing under `scripts/`, `setup-*.{sh,ps1}`, or the vault") vs verification.md "must touch only `cli/` and `specs/`"; `git show --name-only 52035f1` lists `tests/vault-maintenance-weekly.bats`. | N/A (documentation) | spec (verification.md — align the sentence with the corrected acceptance) |
| Minor | REAL | spec artifact (proposal.md) | All six acceptance criteria in proposal.md are `[ ]` (unticked) although verification.md asserts all three increments landed. The archived CLI-072 proposal carries `[x]` for its criteria. Since this spec is still `implementing`, this is a staleness/close-out question rather than a contradiction. | Read proposal.md §Acceptance (all `[ ]`); compare `specs/archive/CLI-072-dotf-hooks-install/proposal.md` (all `[x]`). | N/A | spec (proposal.md — tick the criteria verification.md proves, or note they are ticked at archive) |
| Minor | THEORETICAL | Windows toast | `NotifyDesktop`'s Windows branch is unexercised anywhere (verification.md states this). The PowerShell script is assembled with Go `%q` escaping; the current fixed title/body strings contain no chars `%q` would escape, so it works today, but a title/body containing `"` or `\` would be escaped in Go style and PowerShell would not interpret it identically — a future drift risk. Fire-and-forget, so a failure is silent by design. | Code read of `NotifyDesktop` (`exec.Command("powershell", "-NoProfile", "-Command", script)` with `%q`); verification.md "the PowerShell toast in `NotifyDesktop` is unexercised anywhere… Stated as a gap, not covered." | UNTESTED | tests (Windows-only case) or code (PowerShell-native escaping) |
| Minor | REAL | maintainability | `RunMaintain` is 66 lines, over the repo's <40-line function rule (AGENTS.md). Cyclomatic complexity is low (~9, sequential orchestration), structure is clear with extensive WHY comments, and there are table tests, so this is a smell rather than a defect. | `awk '/^func RunMaintain/,/^}$/' internal/vault/maintain.go | wc -l` → 66; `golangci-lint run ./...` reports 0 issues (no `funlen` gate configured). | N/A | code (optional helper extraction) |
| Minor | THEORETICAL | duplication risk | The `[INFO] Date: %s` line is emitted both by the `vault crystallize --all` cmd layer (`vault.go`) and manually by the `maintain` wrapper (`maintain.go`), because maintain calls `CrystallizeAll` directly and must reproduce the cmd's framing. Currently consistent (verified byte-identical end-to-end), but the literal is duplicated across two call sites and, because maintain is not golden-characterized, a drift would not be caught. | `vault.go` `if all { … "[INFO] Date: %s\n" }` vs `maintain.go` `fmt.Fprintf(&log, "[INFO] Date: %s\n", opts.Today)`; end-to-end diff showed exactly one such line in each log. | UNTESTED | code (share a constant/helper) or spec (document) |
| Question / assumption | REAL | spec coherence | CLI-021's acceptance criteria 5 and 6 ("No twin deleted, no caller repointed", "The shell scripts remain the canonical invocation") are now violated for `crystallize`: its shell/PowerShell twin was deleted and every caller repointed by CLI-050 (#1269), not by CLI-023 as the proposal's plan describes. verification.md acknowledges this ("Cut over separately in #1276 (CLI-050)"). Whether the crystallize cutover deliberately jumped ahead of the CLI-023 plan needs human confirmation. | `scripts/knowledge-crystallize.{sh,ps1}` absent; `scripts/vault-health.sh` and `scripts/vault-maintenance-weekly.sh` present; crontab (`setup-linux.sh:1642`) and Task Scheduler (`setup-windows.ps1:2240`) still point at shell twins; `cli/internal/mem/session_start.go:135` still execs `vault-health.sh`. | N/A | spec (proposal/verification) or vault (record the sequencing decision) |

No Blocker and no Major findings. The only REAL code-level differences observed are the documented ANSI
divergence and the maintain wrapper's added stdout lines — both deliberate, both recorded.

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | B | Acceptance criteria met; maintain wrapper is byte-identical to the shell twin (modulo documented ANSI + timestamp); negative paths (exit code, error handling, nil notifier, unwritable log) covered; minor gaps are the unexercised Windows toast and the accepted ANSI divergence. |
| Verification       | B | Evidence is reproducible and command-backed (parity suites, mutation-verified decisions, end-to-end byte diff); the "proof that matters most" sentence is stale/inconsistent with the corrected acceptance and the actual diff. |
| Scope              | B | Diff matches the corrected proposal acceptance (nothing under `scripts/`, `setup-*.{sh,ps1}`, or the vault); `tests/` and spec-artifact edits are in-scope and documented; no twin deleted, no caller repointed. |
| Reliability        | B | Error paths handled (log write is the sole error, tested); best-effort steps; health findings routed to log/stdout rather than failing the run; Windows toast unexercised but fire-and-forget. |
| Maintainability    | B | Clear naming, low cyclomatic complexity, extensive WHY comments, strong table tests; `RunMaintain` at 66 lines exceeds the <40-line convention (minor smell). |
| Handoff-readiness  | B | Spec updated in the same PR; flip checklist handed to CLI-023; mutation/`|| true` lessons already exist (lesson 224, lesson 267); BUG-060 lesson capture deferred to that spec's archive. |

### Verdict
PASS

### Recommended next steps (before archive)

1. **Align verification.md's "load-bearing check" sentence** with the proposal's corrected acceptance
   (allow `tests/`; the negative is "nothing under `scripts/`, `setup-*.{sh,ps1}`, or the vault") so the
   artifact and the diff agree. Minor, but it is the document's own claimed proof.
2. **Tick the proposal acceptance criteria** that verification.md proves, or state in `verification.md`
   that they are ticked at close-out, so the spec's own "done" contract reads consistently.
3. **Resolve the CLI-050-vs-CLI-023 sequencing question** (Q1): confirm the crystallize cutover ahead of
   the CLI-023 plan was deliberate, and record it, so CLI-021's "no twin deleted / shell canonical"
   acceptance is not read as silently superseded.
4. **Record the maintain ANSI divergence** in verification.md (mirroring the crystallize note) so the
   maintain log's colour-free output is written down rather than discoverable only by running both.
5. Optional, before the CLI-023 cutover: add a Windows case (or at least a named note) for the
   `NotifyDesktop` toast path, and consider whether the `[INFO] Date:` framing literal should be shared
   between `vault.go` and `maintain.go`.
