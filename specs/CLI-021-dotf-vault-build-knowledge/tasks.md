---
id: "CLI-021-dotf-vault-build-knowledge"
type: spec
status: draft
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
      section 7 loudly rather than skipping it, per this section's own seam #2 above; no golden
      exercises that path (a shell always knows its own `$SCRIPT_DIR`), so it is unit-tested instead
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

- [ ] Compose crystallize + health, mirroring `vault-maintenance-weekly.{sh,ps1}`.

## 5. Close out

- [ ] `git diff` touches only `cli/` and `specs/` — proving no twin deleted, no caller repointed.
- [ ] PR merged, CI green (including `test-windows`).
- [ ] `dotf spec archive CLI-021-dotf-vault-build-knowledge`, #490 closed.

---

## Flip checklist — NOT this PR, hand to CLI-023 (PR7)

Collected while porting so it is not rediscovered later. **Every one of these still points at the
shell and must keep doing so until the cutover PR.** Verified present 2026-08-08.

### Callers (executable — break if the script moves)

| Location | What it does |
|---|---|
| `scripts/vault-maintenance-weekly.sh:22` | invokes `knowledge-crystallize.sh --all` |
| `scripts/vault-maintenance-weekly.ps1:25` | invokes `knowledge-crystallize.ps1 -All` |
| `setup-linux.sh:117` | `chmod +x` on the `.sh` |
| `setup-windows.ps1:1593-1598` | deploys the `.ps1` into the scripts dir |

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
