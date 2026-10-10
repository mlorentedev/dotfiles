---
id: "PLAT-001b-one-entrypoint"
type: spec
status: draft # draft | implementing | verifying | archived
created: "2026-10-06"
issue: "mlorentedev/dotfiles#2013"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
wip_override: "PLAT-001 track P: the macOS bring-up needs one cross-OS entrypoint (install.{sh,ps1} -> dotf converge) before any further setup runs on the Mac. DX-007 is abandoned in #2017 and frees one slot; the other stalled specs are outside this session's scope (17 active, limit 10, 2026-10-06)"
---

# PLAT-001b: One entrypoint on every OS (`install.{sh,ps1}` → `dotf converge`)

## Why

<!-- from issue #2013: PLAT-001: [EPIC] OS-agnostic from-zero bootstrap — macOS bring-up as the forcing function -->

A new machine reaches its working state through three different front doors, and none works on macOS:

- **Linux:** the root `install.sh` clones the repo and runs `setup-linux.sh` (1,658 lines).
- **Windows:** the owner clones by hand and runs `setup-windows.ps1` (2,249 lines), which reimplements the same steps.
- **macOS:** there is no front door. `setup-linux.sh` is barred on the Mac because it installs Linux binaries and a Linux toolchain layout (#2013).

The setup scripts are twins: every behaviour is written twice and drifts on one OS at a time. The engine that should replace them is already planned: `dotf converge`, rows B6 and B7 of #1843, ordered by ADR-041 decision 4. It has no implementation. The macOS machine is factory-fresh, which makes it the one place a from-zero run can be built and measured instead of assumed.

On that machine the agent ran before its instruction files existed, and its harness defaults leaked AI attribution into commits (F-060, #2016). Instruction files are records, and records have to come first.

The owner's goals (2026-10-06):

- Nothing is done by hand.
- One engine for every OS, with no duplicated code.
- The result is SRE-grade and distributable as open source.

## What

The design is ADR-045 (`docs/adr/adr-045-one-entrypoint-converges-every-os.md`); this spec delivers it.

- **ADR-045** records the two-tier entrypoint and the reconciler model. It is `proposed`, so the review decides #2013 D4 and the reach of D2.
- **`dotf converge --plan`** (#1843 B6):
  - It runs an ordered registry of reconcilers. Each one has a name, the platforms it applies to, a read-only plan, an apply step, and a post-condition probe.
  - The first native reconciler is **records** (#2016, P9): the harness mirror, the agent instruction files and skills, and the hook bindings.
- **`dotf converge` apply mode** with a persisted report (#1843 B7).
  - A second run with nothing to do reports zero changes.
  - A reconciler that fails its probe fails the run and names itself.
- **Root `install.sh` and `install.ps1`** replace `scripts/install-dotf.{sh,ps1}` and the current clone-and-setup `install.sh`.
  - They download `dotf` at the stable pin, verify its checksum, run it, and `exec dotf converge`.
  - The clone becomes the **checkout** reconciler.
  - Every live reference to `install-dotf` moves. A test fails on the old name outside the historical records (ADRs, audits, lessons, archived specs).
- **Legacy reconcilers.** `setup-linux.sh` runs as one reconciler on Linux and `setup-windows.ps1` on Windows, after every native reconciler. On darwin it does not run; each step the twins perform that no native reconciler covers yet is reported `skipped`.
- **`dotf update`** runs `dotf converge` instead of the setup script. Its name, its exit semantics and its fast-forward-only rule do not change.
- **CI from zero (#2013 X1, non-required at first):** on a clean `macos-latest` and `ubuntu-latest` runner, `install.sh`, then `dotf doctor`, then a second run that reports zero changes.

## Out of scope

- **Porting the setup scripts' blocks into reconcilers.** That is tracks W, T, P and S of #2013 and B of #1843; each port is its own PR. This spec delivers the engine, the entrypoint and the first native reconciler.
- **Packaging** (a Homebrew tap, Scoop, deb and rpm) and the repo split. #1843 defers both until the repo's final shape is settled.
- **The `platforms` selector in `ai/deploy.json`, `env-contract.json` and doctor checks** (#1843 B1, #2013 P2 and P3). ADR-045 proposes the vocabulary; those rows own the schema changes.
- **Scheduled convergence** (#1843 A5, B8).

## Risks / open questions

- **The legacy reconciler hides drift.** `setup-linux.sh` cannot plan, so `--plan` reports it as `opaque`, never as "no change". *Resolved:* the report states it, and the from-zero CI asserts idempotence only on native reconcilers until the twins are gone.
- **The rename touches about 118 references, including the sourced `install_dotf` function and doctor's remediation text.** *Resolved:* one PR does the move mechanically, keeps the function name, and adds the old-name guard. `GOOS=windows go vet` and the Pester suite cover the PowerShell side.
- **The README one-liner changes meaning:** it used to clone and run setup, and now it installs and converges. A machine that used `DOTFILES_SKIP_SETUP=1` uses `dotf converge --plan` instead; the README and the release runbook say so in the same PR.
- **A from-zero macOS runner costs 10× Linux minutes** (#2013 owner actions). X1 runs on pull requests to `main` and on push to `main` (the run the verification reads), never on feature-branch pushes, and is non-required until green.
- **Windows proof needs the Windows box** (owner rule: batch Windows-empirical work). CI `test-windows` and Pester cover the scripts; the real-machine run is #2013 X3.
- **Open for the ADR review:** #2013 D4 (what Homebrew installs on macOS) and the reach of D2. See ADR-045, *Open questions*.

## Acceptance criteria

- [ ] AC1: `dotf converge --plan` lists every reconciler that applies to this OS with its planned action, and changes nothing on disk.
- [ ] AC2: on a machine with no `~/.claude/CLAUDE.md`, `dotf converge` deploys the agent instruction files and skills before any later reconciler runs, and the records reconciler fails if they are absent afterwards (#2016).
- [ ] AC3: a second `dotf converge` with nothing to do reports zero changes for every native reconciler, and writes a report that says so.
- [ ] AC4: a reconciler whose post-condition probe fails makes `dotf converge` exit non-zero and names that reconciler; it never reports success.
- [ ] AC5: a reconciler that does not list the current OS is reported as `skipped` with the OS named, not as passed.
- [ ] AC6: `install.sh` (bash 3.2 and zsh) and `install.ps1` install `dotf` at the stable pin with checksum and exec probe, then hand off to `dotf converge`; with no network or a bad checksum they fail and place nothing.
- [ ] AC7: no live file outside the historical records (ADRs and audits, lessons, specs, the changelog) references `install-dotf.sh`, `install-dotf.ps1` or the old clone-and-setup flow, enforced by a test.
- [ ] AC8: `dotf update` runs `dotf converge`, keeps its exit semantics, and its existing tests pass.
- [ ] AC10: `dotf converge` deploys every `ai/deploy.json` config that applies to the machine, so a template change merged to main lands without a manual `dotf deploy`; a config whose secrets the store cannot resolve during the run keeps its installed file and is named in the report (#1843 B15).
- [ ] AC9: a from-zero CI job on `macos-latest` runs `install.sh`, then `dotf doctor`, then a second converge with zero native changes.

## References

- ADR-045 (this spec's design); ADR-041 decisions 4 and 7; ADR-020 C7 and §5; ADR-019 D2; ADR-036; ADR-044; ADR-003; lesson 337.
- Epic #2013 (rows P0, P4, P9, X1; decisions D1, D2, D4); epic #1843 (rows B6 and B7 delivered here, see the comment on #1843; B1, B11); #2016 (P9); CLI-090 (#1803).
