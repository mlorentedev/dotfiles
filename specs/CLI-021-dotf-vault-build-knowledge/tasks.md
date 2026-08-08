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

- [ ] Decide what `vault health` means — the shell's local checks, or the Hive `vault_health` MCP
      tool. Blocks increment 2; does not block increment 1.

## 1. Golden corpus (before any Go)

- [ ] Build a fixture set of `MEMORY.md` inputs covering: fresh file (no markers); file with
      `# currentDate` only; with `## Last Crystallized:` only; with both; with duplicates; with a
      `## Session Handoff` block (the HARNESS-029 case from BUG-060); without one; over the
      150-line limit.
- [ ] Capture the **shell's** output for each fixture as the golden file. Record the exact script
      revision used as the oracle.
- [ ] Port the two BUG-060 behavioural BATS cases into the corpus (handoff stays last; idempotent
      across two runs) — they are the seed, not the whole set.
- [ ] Enumerate every behaviour where `.sh` and `.ps1` disagree today. Linux is the reference;
      anything `.ps1`-only must be listed before it is dropped.

## 2. Increment 1 — `dotf vault crystallize`

- [ ] Wire the subcommand under the existing `vault` noun (NOT top-level — that is the collision).
- [ ] Path encode/decode: reuse the existing `dotf mem project-key` code path, do not
      reimplement (the #689 drive-colon regression lives here).
- [ ] Port: dedup currentDate, update currentDate, stamp Last Crystallized, line-count warning,
      checklist output, `--all` discovery.
- [ ] **HARNESS-029:** insert before `## Session Handoff` when present. Add a test that fails
      against a naive append, mirroring BUG-060.
- [ ] Golden tests green: Go output byte-identical to shell for every fixture.
- [ ] Table-driven unit tests for encode/decode and section insertion.

## 3. Increment 2 — `dotf vault health`

- [ ] Blocked on task 0.
- [ ] Read-only; no writes under any flag.

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
