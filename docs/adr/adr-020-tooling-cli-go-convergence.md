---
id: "ADR-020-tooling-cli-go-convergence"
type: adr
status: accepted
owner: manu
date: "2026-06-10"
relates: [adr-003-dual-shell-bash-zsh, audit-002-cross-os-duplication, adr-004-bats-testing, adr-011-model-tier-policy]
tags: [architecture, decision, tooling, go, cli, cross-platform, strangler-fig, cross-model-review]
created: "2026-06-10"
---

# ADR-020: Converge cross-platform tooling into a single Go CLI (strangler-fig)

> The ~18 logic scripts that exist as `.sh` + `.ps1` twins (each fix applied twice, tested twice
> in bats *and* Pester) converge into one statically-linked Go CLI binary (`dot`). Migration is
> strangler-fig (port on contact, delete the shell pair in the same PR), not a big-bang rewrite.
> The language boundary is **two** layers, not three: Go owns user-facing tooling; shell owns the
> thin bootstrap + profile/env. **Python is explicitly not a layer here** — there is none in the
> repo and none is introduced. First PR: `dot review`, a cross-model code-review subcommand.

## Status

Accepted. Architectural direction; implementation follows via per-feature specs and PRs.

## Context

Roughly 18 scripts under `scripts/` are maintained as `.sh` + `.ps1` twins (init-spec, archive-spec,
doctor, healthcheck, claude-mem-heal, knowledge-crystallize, dotfiles-sync, session-handoff,
init-project, init-repo-*, load-secrets, diff-check, obs-cli, vault-maintenance, dotfiles-selfupdate,
utils). Every behavioural fix is applied twice and tested twice (bats for POSIX, Pester for Windows).
This duplication is the problem flagged in `audit-002-cross-os-duplication`; its cost is on the record
— the consumer-EPIPE fix (PR #242) and the `heal_mcp_json` sibling (PR #259) both had to be ported to
both shells with parity tests on each side.

The trigger for revisiting it now: a desire for a portable **cross-model review** tool (send a diff to a
non-Claude model — NaN deepseek/qwen or OpenRouter — for an independent second opinion). That tool must
not be Claude-Code-specific (hooks do not port across agents), which forced the broader question: what is
the clean, portable home for *all* this tooling?

Verified state (Phase A, evidence not memory): repo on `main`, clean, latest ADR `adr-019`, no vault
project area for `dotfiles` (fully on the repo-docs placement model). **Zero `.py` files in the repo** —
the AI/MCP servers in use (claude-mem, context7, sequential-thinking) are third-party Node packages that
are *consumed*, not Python that is *maintained*. The Python standards in `AGENTS.md` govern other repos
(e.g. the sensor SDK platform), not this one.

## Constraints

- **C1 — Cross-platform parity without dual maintenance.** One source of truth per script; a fix lands once.
- **C2 — One test suite.** Replace the bats + Pester pair for migrated logic.
- **C3 — No runtime provisioning burden.** A fresh machine should run the tooling with minimal prerequisites.
- **C4 — Portable across agents.** Tooling invoked the same way regardless of which AI agent (or none) drives it; no harness-specific (Claude hook) lock-in.
- **C5 — No language fragmentation by accident.** Any new language must have an explicit, defended boundary.
- **C6 — Incremental, reversible migration.** No big-bang; stable untouched scripts keep working until ported on contact.
- **C7 — Irreducible shell bootstrap.** The step that provisions the tooling itself stays shell (chicken-and-egg).

## Options Considered

| Option | C1 | C2 | C3 | C4 | C5 | Verdict |
|---|---|---|---|---|---|---|
| A. Keep dual shell (status quo) | gap | gap | ok | ok | ok | Rejected — the recurring tax |
| B. Converge to a Python CLI | ok | ok | **gap** | ok | gap | Rejected — runtime dependency; and no existing Python to reuse |
| C. **Converge to a Go CLI + thin shell bootstrap** | ok | ok | **ok** | ok | ok | **Selected** |
| D. Go + Python + shell trichotomy | ok | ok | gap | ok | **gap** | Rejected — no Python in repo; fragments without cause |

## Decision

1. **One Go CLI, `dot`**, distributed as a statically-linked binary (one per OS/arch), built with
   Cobra (subcommands) + Viper (config) — the prior art of gh / kubectl / chezmoi. Same command on every
   OS: `dot spec init`, `dot vault health`, `dot heal claude-mem`, `dot review`.
2. **Two-language boundary, declared (closes C5):**
   - **Go** → user-facing tooling CLI (the logic of the `.sh`/`.ps1` twins).
   - **Shell** → thin bootstrap (detect OS/arch, fetch the right binary, put on PATH) + profile/env.
   - **Python is not a layer here.** It only reopens this ADR if Go-native tooling Python is ever written — and the default answer would still be Go.
3. **Single binary, no runtime (C3).** Go compiles to a dependency-free static binary; cross-compiled
   for linux/macOS/windows from one host via `goreleaser`. The bootstrap shrinks to "download a binary"
   — no Python/venv/uv provisioning, no version skew.
4. **One test suite (C2).** `go test` (table-driven) replaces bats + Pester for every migrated script.
5. **Strangler-fig migration (C6).** Port a script into `dot` only when it is next touched; in the same
   PR, delete the `.sh` + `.ps1` pair and their bats/Pester tests. Never leave the three coexisting
   (no triple-maintenance). Stable untouched scripts stay shell until their turn.
6. **Home: own module with its own release pipeline.** A Go module (own repo, or a `cli/` subtree with
   its own `go.mod`) so binary tags = releases, kept out of the dotfiles root.
7. **First PR: `dot review`** — greenfield, zero migration risk. Reads a git diff (or stdin), posts to an
   OpenAI-compatible endpoint (NaN via `NAN_BASE_URL`, or OpenRouter), default reviewer a non-Claude
   family for decorrelated second opinions. Proves the pattern before any twin is migrated.

## Consequences

**Positive**
- A behavioural fix lands once and ships to both OSes; the EPIPE/heal "port to both shells" class of work disappears for migrated scripts.
- Fresh-machine setup needs no language runtime — drop one binary.
- `dot --help` is a discoverable, typed, testable surface; refactors are compiler-checked.
- Aligns with the Go standards already in `AGENTS.md` (1.26+, table-driven tests, context, generics).

**Negative**
- A new toolchain (Go build + `goreleaser` + release flow) is upfront infra the shell scripts did not need.
- Orchestration-heavy glue (shelling out to `gh`/`git`/`obsidian`/`claude-mem`, text/JSON munging) is more verbose in `os/exec` than in a bash pipe — traded for testability and error handling.
- During migration the repo is mixed (Go + remaining shell) until the strangler completes.

**Neutral**
- `AGENTS.md` must declare the two-language boundary so all agents follow it (follow-up).
- `audit-002-cross-os-duplication` is resolved by this ADR; `adr-004-bats-testing` and `adr-003-dual-shell-bash-zsh` remain valid for the shell that stays (bootstrap/profile) but no longer govern migrated logic.

## Alternatives rejected

- **Keep dual shell (status quo):** the maintenance tax this ADR exists to remove.
- **Python CLI:** reintroduces the runtime dependency chezmoi deliberately avoided; and there is no existing Python in the repo to consolidate, so it would add a language rather than converge one.
- **Go + Python + shell:** three languages where two suffice; fragments the repo with no Python workload to justify the third.
- **Big-bang rewrite of all 18 scripts:** large regression surface for zero functional gain on scripts that already work; rejected per the incremental wisdom of Capital One "Bashing the Bash" / ninjaaron.

## Follow-ups

- Open a tracking issue in the bitácora Project and `/spec init` the first feature (`dot review`).
- Patch `AGENTS.md` to declare the Go/shell boundary (Standing Order #2, SSOT) once the first PR lands.

## References

- chezmoi — single statically-linked Go binary, cross-platform without per-OS shell scripts: https://www.chezmoi.io/what-does-chezmoi-do/
- gh / kubectl — Cobra-based CLI prior art.
- Capital One, "Bashing the Bash — Replacing Shell Scripts with Python": https://www.capitalone.com/tech/software-engineering/bashing-the-bash/
- ninjaaron, "Replacing Bash Scripting with Python" (incremental, move-the-logic): https://github.com/ninjaaron/replacing-bash-scripting-with-python
- Related: `adr-003-dual-shell-bash-zsh`, `audit-002-cross-os-duplication`, `adr-004-bats-testing`, `adr-011-model-tier-policy`.
