---
id: "ADR-045-one-entrypoint-converges-every-os"
type: adr
status: proposed
owner: manu
date: "2026-10-06"
supersedes: []
extends: [adr-041-update-channels-and-convergence-order, adr-020-tooling-cli-go-convergence, adr-036-install-channels, adr-044-toolchains-and-pinned-clis-via-mise]
issue: mlorentedev/dotfiles#2013
tags: [architecture, decision, bootstrap, install, converge, cross-os, macos]
created: "2026-10-06"
---

# ADR-045: One entrypoint on every OS: `install.{sh,ps1}` installs `dotf`, and `dotf converge` does the rest

## Context

A machine reaches its working state through three different front doors today, and none of them works on macOS:

| OS | Front door | What it does |
|---|---|---|
| Linux | root `install.sh` (52 lines) | clones the repo, then `exec ./setup-linux.sh` (1,658 lines) |
| Windows | clone by hand | `setup-windows.ps1` (2,249 lines), a second implementation of the same steps |
| macOS | none | `setup-linux.sh` is barred on the Mac until PLAT-001 Wave 1 lands (#2013), because it installs `linux-amd64` binaries and a Linux toolchain layout |

The two setup scripts are twins: each new behaviour is written twice, and each divergence is a bug found on one OS only. ADR-020 §5 already says that logic moves into `dotf`, and that setup scripts do not grow. What was missing is the place it moves to.

That place is already planned. #1843 rows B6 and B7 define `dotf converge`: a reconciler registry with a read-only plan and an apply mode with a persisted report. ADR-041 decision 4 fixes the order of a convergence run (checkout, then records, then binary). B6 and B7 have no implementation and no issue, and #2013 has no entrypoint. Building a separate `dotf bootstrap` for the Mac would create a second engine for the same job. This ADR makes `converge` the one engine, and the installers its only front door. #1843 keeps the rows; spec `PLAT-001b-one-entrypoint` delivers them.

The macOS bring-up also showed what happens when agent instructions arrive late. On a fresh machine, the agent ran before `~/.claude/CLAUDE.md` existed, so its harness defaults (AI attribution) reached commits (F-060, #2016). The instruction files are records, and ADR-041 already puts records first.

## Decision

1. **Two tiers, and only tier 0 is shell (ADR-020 C7).**
   - **Tier 0** is `install.sh` (bash 3.2 and zsh, ADR-003) and `install.ps1` (Windows PowerShell 5.1) at the repository root. They do only what a binary cannot do for itself:
     1. detect the OS and architecture;
     2. download `dotf` at the stable pin, verify its checksum, and run it before placing it (lesson 337);
     3. `exec dotf converge`, passing the arguments through.
   - **Tier 1** is `dotf converge`. Every other step lives there, written once for every OS.
   - Every OS has the same command shape: `curl -fsSL <raw>/install.sh | bash` or `irm <raw>/install.ps1 | iex`. The scripts differ only in interpreter, because a fresh Windows has no bash and a fresh Linux or macOS has no PowerShell. A polyglot script that both can parse is rejected as unmaintainable.
2. **The installers move; the contract of ADR-041 decision 7 does not.** `scripts/install-dotf.sh` and `scripts/install-dotf.ps1` become `install.sh` and `install.ps1` at the root, and the current root `install.sh` (clone and setup) is deleted. Its clone moves into the first reconciler. Three things keep working exactly as decision 7 requires:
   - The installers run with no `dotf` present, and repair a broken one.
   - A raw stream with no channel resolves `stable` through the latest-release API, never an edge build.
   - No installer is ported to Go.
   
   The `install_dotf` function stays sourceable by the setup scripts until the last of them is retired.
3. **`dotf converge` runs an ordered registry of reconcilers (#1843 B6, B7).**
   - Each reconciler:
     - has a name;
     - declares the platforms it applies to;
     - has a read-only `Plan` that reports what it would change;
     - has an `Apply` that changes it and reports `changed`, `ok`, `skipped` or `failed`;
     - has a post-condition probe that `Apply` must pass before it reports success.
   - `dotf converge --plan` touches nothing. `dotf converge` applies and writes a report under the user state directory.
   - A second run with nothing to do reports zero changes. That is the idempotence contract, and the from-zero CI job asserts it (#2013 X1).
   - The registry is code; what each reconciler converges is data (`packages.json`, `ai/deploy.json`, `versions.conf`, `harness/manifest.json`). Nothing personal is written into the engine. This keeps `dotf` distributable (#1843 track E).
4. **The order extends ADR-041 decision 4; it does not change it:**
   1. **checkout** (clone if absent, fast-forward under ADR-019 D2);
   2. **records**: the harness mirror, the agent instruction files and skills, and the hook bindings. This is P9 (#2016): no agent is installed or run before its instructions exist;
   3. **binary**: `dotf` to its channel target, never before step 2 succeeded in the same run;
   4. **tools**: package managers first, then the catalog and mise (W5, ADR-044);
   5. **configs**: `dotf deploy` and the shell rc and env files;
   6. **legacy**: the remaining setup script (see decision 5);
   7. **verify**: `dotf doctor`. Its failures are the run's failures.
   
   On a fresh machine, tier 0 installs `dotf` before step 1, because nothing else can run without it. That is the one bootstrap exception to "records before binary", and the binary it installs is the stable pin that the records on `main` were released against.
5. **The setup scripts become one reconciler each, and shrink to nothing (strangler fig).**
   - `setup-linux.sh` runs as the `legacy` reconciler on Linux, and `setup-windows.ps1` on Windows. It has no plan (it reports `opaque`), and it runs after every native reconciler, so a native step always wins.
   - Each later PR ports one block into a reconciler and deletes it from both twins in the same change (ADR-020 §5).
   - macOS has no legacy reconciler. On darwin, only native reconcilers run, and a step that is not yet native is reported as `skipped: not yet supported on darwin`, not silently passed.
6. **`dotf update` routes through `converge`.** Its name, its exit semantics and its fast-forward-only rule stay, because the systemd unit, the Scheduled Task and the tests call it (ADR-041 decision 7). What it runs changes: `dotf converge` instead of the setup script.
7. **The OS is a declared platform, never inferred** (#2013 D2). Reconcilers use the `platforms` vocabulary that `packages.json` already ships (#2001, ADR-036 amendment 2026-10-03): absent means every OS, and an unlisted OS is skipped, not failed. On macOS, the class-3 package manager that tier 0 may rely on is Homebrew.

## Open questions for the review of this ADR

- **D4: what Homebrew installs on macOS.** #2013 proposes casks, GUI apps and system libraries only, never toolchains (ADR-044 already rules out toolchains). Merging this ADR accepts D4 as stated; a review that disagrees changes this line first.
- **D2's reach.** Decision 7 binds reconcilers and the catalog. The same vocabulary is meant for `ai/deploy.json` entries (#1843 B1), `env-contract.json` (a `darwin` key, #2013 P2) and doctor checks (#2013 P3). Those rows change schemas other code reads. This ADR proposes the vocabulary for them, and each row decides its own schema change.

## Consequences

- A new machine on any OS runs one command, then `dotf doctor` reports its state. On the Mac, everything the legacy scripts did but no reconciler does yet appears as `skipped`, which is the honest backlog for track P.
- The twin setup scripts stop growing by construction: a new step has nowhere to go but a reconciler.
- The root `install.sh` URL in the README keeps working but changes meaning: it installs `dotf` and converges, instead of cloning and running setup. A machine that used `DOTFILES_SKIP_SETUP=1` uses `dotf converge --plan` instead.
- 118 references to `install-dotf` move with the rename (code, tests, docs, the release runbook). ADRs, audits and lessons are historical records and keep the old name. A test fails on the old name anywhere else.
- Packaging (a Homebrew tap, Scoop, deb and rpm) stays out of scope, as #1843 already decided, until the repo's final shape is settled.

## Alternatives rejected

- **A `dotf bootstrap` command for new machines.** It is a second engine next to `converge`, so every step would exist twice, which is the defect this ADR removes from the setup scripts.
- **A `setup-macos.sh`.** A third twin (#2013 D1).
- **Keeping `install.sh` as clone-and-setup.** The clone is a reconcilable step like any other. Leaving it in shell keeps the Linux and Windows front doors different.
- **One polyglot installer for sh and PowerShell.** It is clever, brittle and unreadable; two short scripts with one contract are not duplication of logic, only of interpreter.

## Amendment 2026-10-11 (#2013 D10, D11, D12, owner)

The from-zero job (X1) ran this ADR's consequence, "then `dotf doctor` reports its state", on fresh macOS and Ubuntu runners. It found that the state has two kinds, and that the doctor did not tell them apart.

- **D10: doctor classifies every check as machine or identity.** A machine check covers what converge installs and configures. An identity check covers what only the owner can restore: the age key, the Bitwarden session, the GitHub login and the knowledge vault. `dotf doctor --scope machine` runs the machine checks and prints one SKIP per identity check, naming the step of `docs/runbooks/guide-new-machine.md` that restores it. A plain `dotf doctor` is unchanged. A check mixing both kinds is split: the age binaries (machine) from the age key (identity), and the agent binaries (machine) from the configs rendered from secrets (identity). A check has no default kind, so one added without a decision fails a test. The hive daemon is a machine check: converge is meant to run it.
- **D11: a step that a later step makes stale runs again.** records-harness writes instruction files only for the agents present, and the tools step installs agents after it. When the tools step changed something, converge re-runs records-harness in the same run, so the second converge has nothing left to do.
- **D12: converge guides the identity restore on a terminal.** When converge runs on a TTY and the identity is missing, it ends by walking the person through the same steps (`bw login`, `dotf secrets unlock`, then `dotf secrets verify`; the age key and the vault named with where they come from). Without a TTY (CI, a scheduled run) it asks nothing: it prints the same steps and exits as it would have. Not implemented yet; the scope above is what makes it testable.

The alternative was to fake an identity on the CI runners: a generated age key, an empty vault skeleton and a stub `obsidian`, as `test-windows` does. It makes the doctor green by testing directories, not the restore. Moving `test-windows` to `--scope machine` is a follow-up row on #2013.

## References

- Epic #2013 (rows P0, P4, P9, X1; decisions D1, D2, D4, D10, D11, D12), epic #1843 (rows B1, B6, B7, B11), #2016 (P9), CLI-090 (#1803).
- ADR-003, ADR-019, ADR-020 (C7, §5), ADR-036, ADR-041 (decisions 4 and 7), ADR-044; lesson 337.
