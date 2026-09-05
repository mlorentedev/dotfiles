---
id: "CLI-021-dotf-vault-build-knowledge"
type: spec
status: implementing
created: "2026-08-08"
issue: "mlorentedev/dotfiles#490"
tags: [spec, tasks]
template_version: "1.0"
---

# Tasks — CLI-021-dotf-vault-build-knowledge

TDD order. Golden characterization tests come **before** the Go implementation: the shell is the
oracle, so its output must be captured while it is still the only implementation.

## 0. Resolve the open question first

- [x] Decide what `vault health` means — the shell's local checks, or the Hive `vault_health` MCP
      tool. Blocks increment 2; does not block increment 1. → **the shell's local checks**, resolved
      in `proposal.md` (§Risks, struck through) and landed with the golden corpus in `eda3430`
      (#876). This is a twin port, so the oracle has to be the script: #672's golden
      characterization tests cannot run against an MCP surface. Aligning the two notions of
      "health" is separate work, if wanted at all.
      *The decision was recorded in the proposal but not ticked here, so increment 2 read as
      blocked for a session longer than it was.*

## 1. Golden corpus (before any Go)

- [x] Build a fixture set of `MEMORY.md` inputs covering: fresh file (no markers); file with
      `# currentDate` only; with `## Last Crystallized:` only; with both; with duplicates; with a
      `## Session Handoff` block (the HARNESS-029 case from BUG-060); without one; over the
      150-line limit. **14 cases** in `tests/golden/crystallize/cases/`; the listed set plus
      `marker-without-dateline`, `handoff-with-duplicates`, `yaml-wrapped`, `no-memory-file`,
      `help` and `all-mixed`.
- [x] Capture the **shell's** output for each fixture as the golden file. Record the exact script
      revision used as the oracle. → `tests/golden/crystallize/{capture.sh,ORACLE}`; oracle pinned
      **per file** at `9caedc1` (not a branch tip), and a test fails the suite if the tree moves
      away from it, so a recapture must be deliberate.
- [x] Port the two BUG-060 behavioural BATS cases into the corpus (handoff stays last; idempotent
      across two runs) — they are the seed, not the whole set. → `handoff-no-markers`,
      `idempotent-twice` (byte-identical to the single-run golden, which is the assertion).
- [x] Enumerate every behaviour where `.sh` and `.ps1` disagree today. Linux is the reference;
      anything `.ps1`-only must be listed before it is dropped. → `divergences.md`: 5 divergences.
      **`pwsh` is absent on this machine, so the `.ps1` column is read, not measured** — empirical
      `.ps1` capture is deferred to a Windows session (stated deferral, not an omission).

### Captured while porting — oracle defects, NOT fixed here

Reproduced faithfully as fixtures and ticketed, per the proposal's "Out of scope":

- **#873 (BUG-064)** — `marker-without-dateline`: logs `Updated currentDate` and prints
  `[x] currentDate updated` having written no date at all. The "prints success while doing nothing"
  class this script's own BUG-062 comment argues against.
- **#874 (BUG-065)** — the standalone fallback defines `log_info`/`log_success`/`log_warning` but
  **not `log_error`**, which the BUG-062 refusal path calls; without `utils.sh` that guard dies on
  127 instead of refusing.

Fixing either in the port requires a **deliberate recapture** with the reason in the commit — never
a silent regeneration to turn a red golden green.

## 2. Increment 1 — `dotf vault crystallize`

- [x] Wire the subcommand under the existing `vault` noun (NOT top-level — that is the collision).
- [x] Path encode/decode: reuse the existing `dotf mem project-key` code path, do not
      reimplement (the #689 drive-colon regression lives here). → `memlink.ClaudeProjectKey` /
      `ClaudeMemoryTarget`. Note it also maps `\` and `:` where the shell maps only `/`; on the
      POSIX paths the shell handles the two encodings are identical.
- [x] Port: dedup currentDate, update currentDate, stamp Last Crystallized, line-count warning,
      checklist output, `--all` discovery. → `cli/internal/vault/crystallize.go`.
- [x] **HARNESS-029:** insert before `## Session Handoff` when present. Add a test that fails
      against a naive append, mirroring BUG-060. → verified by mutation: replacing the guard with
      an unconditional append turns 4 cases red.
- [x] Golden tests green: Go output byte-identical to shell for every fixture. →
      `tests/knowledge-crystallize-go-parity.bats`, **13/13 byte-identical**, driven by the SAME
      goldens and the SAME runner as the shell suite. `help` is excluded and why is documented
      (cobra generates its usage; the shell hand-rolls it).
- [x] Table-driven unit tests for encode/decode and section insertion. →
      `cli/internal/vault/crystallize_test.go`.

**Deliberate divergence, stated because the goldens structurally cannot catch it:** the shell
colours its log tags via `utils.sh`; no Go command in this CLI emits ANSI, so this one does not
either. Normalisation strips ANSI before comparing, so parity is green either way — which is
exactly why it is written down rather than left to be discovered.

## 3. Increment 2 — `dotf vault health`

**Unblocked:** task 0 is resolved (the shell's local checks).

**The proposal's "smallest, no writes, safest" is wrong, and the correction matters for
sequencing.** Increment 2 is read-only, but it is not the small one. `vault-health.sh` is 284
lines against crystallize's ~200, and it carries two seams crystallize did not have:

1. **An external binary contract.** Four of its seven sections shell out to `obsidian`, which talks
   over IPC to a running GUI. Characterizing it means stubbing that binary on `PATH` (the precedent
   is the `gh` stub in `bitacora-reconcile.bats`) — and the stub must **log its argv into the
   compared artefacts**, or the port could drift in *how* it invokes obsidian while stdout stays
   byte-identical.
2. **A subprocess seam.** Section 7 execs `check-backlog-integrity.sh` and
   `check-backlog-merged.sh`, resolved through the shell's own `$SCRIPT_DIR`. A Go binary has no
   script dir, so it needs an explicit location seam (ADR-025 / `dotf env path`), failing loud when
   unresolvable rather than silently skipping the section.

- [x] Resolve task 0.
- [x] Golden corpus at `tests/golden/vault-health/`, obsidian stubbed on `PATH`, argv logged into
      the compared artefacts. One case per branch, not per combination. → **16 cases**, 19 tests in
      `tests/vault-health-golden.bats`, exits 0/1/2 all covered.
      - `PATH` is **replaced, not extended**, and the replacement is asserted: a real `obsidian`
        binary exists on this machine (`~/.local/bin` → an AppImage), so a leaked `PATH` could have
        run the real GUI against the real vault. The runner refuses to proceed unless `obsidian`
        resolves into the sandbox — or nowhere, for the absent case.
      - **Oracle defect captured:** `obsidian_cmd()` appends `--vault "$VAULT_NAME"` and four of
        its five callers pass it again, so those invocations carry the flag twice. stdout is
        identical either way — **only the argv artefact sees it**, which is the whole argument for
        capturing argv. Reproduced faithfully, ticketed, not fixed here.
      - Boundary cases (`orphans-boundary`, `unresolved-boundary`, `frontmatter-boundary`) exist
        because mutation testing demanded them: moving the orphan threshold 30→25 turned only one
        unrelated case red, since every fixture sat comfortably inside a band. A boundary no
        fixture lands on is a boundary no test defends. With them, the same mutation goes red on
        the case named for it.
- [x] Port. **Exec the two backlog scripts, do not port them** — they are separate `vault`
      dispatcher subcommands (`check-tasks`, `check-merged`), outside #490's three increments, and
      they survive the CLI-023 cutover, so the exec dependency stays valid afterwards. Porting them
      is ~240 lines of scope creep into another ticket's territory. → `cli/internal/vault/health.go`,
      wired as `dotf vault health` in `cli/internal/cmd/vault_health.go`. Execed through the
      resolved bash interpreter (`mem.ResolveBash()`, exported for this second caller) rather than
      relying on the shebang + executable bit the shell uses directly — there is no `.ps1` twin for
      either script to fall back to. `ScriptsDir` unresolved (or either script missing) FAILS
      section 7 loudly rather than skipping it — but only once a `10_projects/*/11-tasks.md` file
      has actually been discovered; with none found, section 7 still SKIPS exactly as the shell
      does, ScriptsDir notwithstanding. Per this section's own seam #2 above; no golden exercises
      the fail-loud path (a shell always knows its own `$SCRIPT_DIR`), so it is unit-tested instead
      (`cli/internal/vault/health_test.go`).
      - **Oracle defect found while porting, NOT fixed here (#1314):** the shell re-execs
        `check-backlog-integrity.sh` a SECOND time, piped straight into `sed`, to print the failure
        it already detected — unlike the sibling merged-check loop, which captures to a variable
        first. Under `set -euo pipefail` that unnegated pipeline's non-zero exit aborts the WHOLE
        SCRIPT on the FIRST drifted file: no later files in the same loop, no merged-check pass, no
        closing footer. Reproduced faithfully (the `backlog-drift` golden's `expected/stdout` simply
        stops mid-section) rather than "fixed while translating."
- [x] Read-only; no writes under any flag. → `RunHealth` never calls `os.WriteFile` or exec's the
      scripts with anything beyond the fixed `check-backlog-{integrity,merged}.sh <tasks-file>` argv
      shape; `--verbose`/`--vault` are the only flags and both are read-only.
- [x] Go/shell byte-parity proven on all 16 golden cases →
      `tests/vault-health-go-parity.bats`, driven by the SAME goldens and the SAME runner
      (`gvh_run_case`, `GVH_IMPL_MODE=go`) as the shell suite — mirrors
      `tests/knowledge-crystallize-go-parity.bats`'s pattern from increment 1.

Two twin divergences surfaced while scoping this increment — no `vault-health.ps1` exists at all,
and the PowerShell weekly runs no health step despite its header. Both recorded in
`divergences.md`, §Increment 2.

## 4. Increment 3 — `dotf vault maintain`

**No golden corpus here, deliberately.** The twin is a 52-line wrapper whose output is a
timestamped log around two subcommands whose byte-parity is *already* proven by increments 1 and 2.
What is left to characterize is the wrapper — log path, section framing, the issue-count regex, the
notification threshold, the exit code — and those are behaviours, not bytes. Building a third
fixture scheme to re-prove the inner two would measure the same thing a third time.

**Composition is in-process, not `exec dotf`.** The shell had no choice; a Go binary that already
owns both steps does. It deletes the twin's own documented failure mode: cron's minimal PATH
excludes `~/.local/bin`, so a bare `dotf` there silently no-ops under `|| true` every Sunday
(`vault-maintenance-weekly.sh:12-16`, guarded by that file's bats case at line 147). There is no
PATH to harden when there is no subprocess. This is not "improving while translating" — the defect
being removed is an artifact of subprocess composition, not a behaviour of the maintenance run.

**Linux is the reference, and it was already chosen.** The `.ps1` runs no health step despite its
own header claiming it does — `divergences.md` §Increment 2. `maintain` runs both steps on every
OS, per the ADR-020 precedent CLI-024 set (reconstruct the Linux superset, not the `.ps1` subset).

### The open decision: what does `dotf vault maintain` exit with?

The twin always exits 0 — every step is `|| true`, and `vault-maintenance-weekly.bats:171` asserts
that the issue count "only drives the notification urgency, **never the exit code**". That settles
the *issue count*. It does not settle `RunHealth`'s return, which the shell swallows with `|| true`
and never had a way to surface.

| | Always 0 (faithful) | Propagate health's code |
|---|---|---|
| Cron / Task Scheduler | silent, as today | cron mails the owner on every non-zero Sunday |
| Human running it directly | must read the log to learn anything | exit status carries the verdict |
| Precedent | the oracle's literal behaviour | `vault_health.go:23-33` — this spec already diverged from the oracle once, on vault resolution, because `dotf vault health` is a **new directly-invokable entry point** and the oracle's gap existed only because nothing called it standalone |

- [x] **Decided: exit 0 whenever the run did its job; health's verdict goes to stdout, never to the
      status.** *A finding is not a failure.* `RunHealth`'s 0/1/2 answers "what did the report
      find"; maintain's status answers "did the run do its job". Propagating health's code would
      make the status depend on whether a desktop GUI happened to be running — nondeterministic
      from cron's view, and code 2 would mail the owner a failed-job notice every Sunday the laptop
      was shut. False alarms on a weekly channel train the owner to ignore it, taking the desktop
      notification (the signal this command was built around) down with it. The finding is routed
      instead: report body to the log, count to the notification urgency, verdict to stdout for
      whoever ran it by hand. The only error is being unable to write the log.
      **`maintainExitCode` was deleted rather than made to `return 0`** — a function ignoring both
      its parameters is a trap for the next reader who "fixes" it by wiring it to `os.Exit`; with
      it went the `int` from `RunMaintain`'s signature.
      *Not* the same fork `vault_health.go:23-33` took: there the oracle had a GAP (nothing called
      it standalone), here the oracle made a DECISION, and it made it as a cron job.
- [x] Compose crystallize + health, mirroring `vault-maintenance-weekly.{sh,ps1}`. →
      `cli/internal/vault/maintain.go`, wired as `dotf vault maintain` in
      `cli/internal/cmd/vault_maintain.go`. Zero flags, matching the twins.
      `healthOptions()` was EXTRACTED from `vault_health.go` rather than copied — the
      `$VAULT_DIR`-then-ADR-025 cascade is a contract, and two copies drift.
- [x] Table tests for `CountIssues`, `notificationFor`, `logFileFor` (both GOOS), and the
      composition against injected steps — no real vault, no real log location, no desktop bus.
      → `cli/internal/vault/maintain_test.go`, 22 cases.
      - `DefaultLogFile` was split into a thin `runtime.GOOS` wrapper over `logFileFor(goos, home,
        localAppData)`, so BOTH OS branches are tested from either host. Reading `runtime.GOOS`
        inline leaves the branch you are not running on covered only by the compiler.
      - The two steps are injectable as **unexported** struct fields: package-level vars would let
        parallel tests race the seam, and exported ones would let a caller substitute a step and
        still call the result a maintenance run.
      - **Mutation-verified, and the harness proves each mutation landed before believing the red**
        (lesson 267): threshold `>0`→`>1` killed by `TestNotificationFor/boundary_at_one`; the
        code-1 verdict line silenced, killed by `.../failed_checks`; the crystallize error swallowed
        instead of logged, killed by `TestRunMaintainCrystallizeFailureDoesNotStopHealth`;
        `LOCALAPPDATA` ignored, killed by `TestLogFileFor/windows_prefers_LOCALAPPDATA`.
- [x] Extend `tests/vault-maintenance-weekly.bats` with Go-path behavioural cases (log location,
      section order, exit status under health findings). 13 shell cases + 4 Go cases, 17 total.
      **No golden corpus and no `GVH_IMPL_MODE`-style parity runner here, deliberately** — see the
      head of this section. The Go cases assert the wrapper's behaviour directly; the shell-only
      cases (bash/zsh syntax, `set -euo`, `BASH_SOURCE`) stay shell-only because they characterize
      a shell, not the command.
      - The fourth case is the one that earns its place: **the Go path needs no PATH hardening
        under a cron-minimal PATH.** The `.sh` must `export PATH="$HOME/.local/bin:$PATH"` or its
        bare `dotf` silently no-ops under `|| true` every Sunday, and carries a regression guard for
        exactly that (line 147). In-process composition has no subprocess to fail to resolve, and
        the case asserts that structurally rather than by comment.

### Recorded, not fixed — divergences this port accepts

- **Header timestamp.** `time.UnixDate` matches GNU `date(1)`'s default, which is what `$(date)`
  emits in the `.sh`. `Get-Date`'s default in the `.ps1` is a different, locale-dependent shape;
  Linux is the reference, so the `.ps1` shape is dropped rather than reproduced.
- **Log encoding.** `Out-File -Encoding UTF8` writes a BOM under Windows PowerShell 5.1. Go writes
  none. The log is read by humans and by `grep`, both of which are better off without it.
- **The issue regex over-counts.** `warning|fail|action|stale` is a substring match, so "failed"
  and "stalemate" score, and the section headers the wrapper itself prints are counted alongside
  the tools' output. Reproduced faithfully — it is the oracle's behaviour, and the count only
  picks a notification urgency.

## 5. Close out

- [ ] `git diff` touches only `cli/` and `specs/` — proving no twin deleted, no caller repointed.
- [ ] PR merged, CI green (including `test-windows`).
- [ ] `dotf spec archive CLI-021-dotf-vault-build-knowledge`, #490 closed.

---

## Flip checklist — NOT this PR, hand to CLI-023 (PR7)

Collected while porting so it is not rediscovered later. **Every one of these still points at the
shell and must keep doing so until the cutover PR.** Verified present 2026-08-08.

### Callers (executable — break if the script moves)

**Re-verified 2026-09-04 against `origin/main` @ `9c2758a`, by grep, not by reading this table.**
The four rows that were here are gone: all of them named `knowledge-crystallize.{sh,ps1}`, which
CLI-050 (#1269) deleted and cut over. Only comments referencing it survive
(`vault-maintenance-weekly.sh:13`, `.ps1:28`) and those are historical notes, not callers. The rows
below replace them and are what CLI-023 actually has to repoint.

| Location | What it does |
|---|---|
| `setup-linux.sh:1605` | **crontab entry** — `7 10 * * 0 $CURRENT_DIR/scripts/vault-maintenance-weekly.sh`, plus the `grep`/`crontab -` at 1606/1613 |
| `setup-windows.ps1:2185` | **Task Scheduler** — `$expectedTaskScript = "$DotfilesDir\scripts\vault-maintenance-weekly.ps1"` |
| `scripts/vault-maintenance-weekly.sh:32` | invokes `$SCRIPT_DIR/vault-health.sh` (the `.ps1` twin invokes no health step at all — `divergences.md` §Increment 2) |
| `setup-linux.sh:116` | `chmod +x` on `vault-health.sh` |
| `scripts/vault.sh:63` | dispatcher `exec`s `vault-health.sh` |
| **`cli/internal/mem/session_start.go:135`** | **the Go CLI shells out to the shell twin.** `vaultHealth()` builds `<ScriptsDir>/vault-health.sh` and runs it for the SessionStart brief, with `ansiSeq` at :57 stripping the colours it emits. The least obvious caller by far — `dotf` calling the script `dotf` is replacing — and it is the one that makes deleting `vault-health.sh` a Go change, not a shell change |

### Not this ticket, found while re-verifying — worth a ticket of its own

`harness/skills/crystallize/SKILL.md:70` reads *"Run: `knowledge-crystallize.sh` (or `dotf vault
crystallize`)"*. That script was **deleted** by CLI-050; the skill names a nonexistent command
first and the working one as the parenthetical. Out of scope here (this PR's acceptance forbids
touching anything but `cli/`, `tests/` and `specs/`, and `harness/` belongs to the parallel
persona/gate work), but it is a live instruction pointing at nothing.

### Docs and comments (stale text — no runtime break)

| Location | What it says |
|---|---|
| `scripts/vault.sh:25` | "Weekly automated maintenance: knowledge-crystallize + health + desktop" |
| `scripts/utils.ps1:113` | references the `.ps1` decoder in a comment |
| `specs/SDD-004-session-start-config/proposal.md:26` | historic — names crystallize as a deferred SSOT candidate. Leave as an audit trail. |

### Vault (the harness side — needs a `--refresh` + `--deploy` after editing)

| Location | What it says |
|---|---|
| `00_meta/skills/crystallize/SKILL.md:55` | *"Run: `knowledge-crystallize.sh` (current canonical invocation — **not yet converged to a `dotf` noun**)"* — written in anticipation of exactly this ticket |
| `00_meta/patterns/pattern-ai-memory.md` | references the script |
| `00_meta/patterns/pattern-ai-protocol.md` | references the script |
| `00_meta/skills/CURRENT-STATE.md` | references the script |

### Harness / hooks

- [ ] The SessionStart hook text that emits *"Knowledge crystallization never run — run:
      `./scripts/knowledge-crystallize.sh`"*. Locate its source and repoint at cutover.

> **Sequencing note.** Editing the vault skill *now* would point agents at a command that is not
> canonical yet — #490's acceptance is explicit that the shell stays canonical through this PR.
> The skill's own parenthetical already flags the pending convergence, which is the correct state
> until CLI-023 flips it.
