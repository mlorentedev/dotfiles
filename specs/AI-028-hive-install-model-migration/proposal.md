---
id: "AI-028-hive-install-model-migration"
type: spec
status: implementing # draft | implementing | verifying | archived
created: "2026-08-07"
issue: "dotfiles#791"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# AI-028: Hive install model migration

<!-- from issue #791: AI-028: hive bootstrap, daemon activation and auto-upgrade all silently no-op when the uv-tool install is gone -->

## Why

Four mechanisms in this repo — the MCP `prerequisite_command`, the daemon-activation version gate, the 15-minute upgrade timer, and by transitivity hive's A3 versioned layout — all infer *"is hive installed?"* from `uv tool list`. When that install disappeared on the maintainer's Windows box, all four degraded to a **silent no-op**: the scheduled task has reported `LastTaskResult: 0` every 15 minutes for months while doing nothing, no `HiveVaultDaemon` task was ever registered, and the Claude MCP `hive` entry points at an orphaned uv trampoline that cannot start. The vault has been unreachable from Claude Code on that machine and nothing said so. The same failure is already documented in `docs/troubleshooting/hive-mcp-orphaned-trampoline.md` (2026-06-29) with a manual recipe — it has now recurred, which is the evidence that a manual recipe is not a fix.

## What

The install model stops being "a uv tool that dotfiles upgrades in place" and becomes "a versioned layout hive owns, that dotfiles bootstraps and triggers". Observable after this change:

1. `windows/hive-upgrade.ps1` **reports** when it finds no install, instead of exiting 0 in silence. "Already current" stays quiet by design; "no install found" is loud.
2. Setup **bootstraps** hive on a machine that has none, deterministically and without a pre-existing `hive` binary, via `uvx --from hive-vault hive self-upgrade`.
3. The steady-state upgrade trigger is a bare `hive self-upgrade` (A3 junction swap — no daemon stop needed), replacing the A1 stop-upgrade-start orchestration.
4. `AI-022`'s recorded A1 decision no longer contradicts hive's shipped A3.

## Out of scope

- **MCP registration across agent clients** → AI-029 (dotfiles#792). This spec does not touch which clients use the daemon.
- **Detection/repair of an already-broken install** → #574 (HARNESS-049), `dotf doctor --fix`. Complementary safety net; neither subsumes the other.
- **Hive's own launcher + `_resolve_exec()` hardening** → [hive#328](https://github.com/mlorentedev/hive/issues/328). This spec consumes that fix; it does not implement it.
- macOS / launchd activation.

## Risks / open questions

- **Hive builds the layout but installs no PATH launcher — RESOLVED as a dependency, not a blocker for all of it.** `src/hive/_runtime.py` creates `versions/<v>` and flips the `current` junction, then stops; nothing puts `<current>/Scripts/hive.exe` on PATH. Worse, `_service.py:63` `_resolve_exec()` does a bare `shutil.which("hive")`, which on the broken machine returns the dead trampoline — so `hive service install` would register a daemon pointed at a binary that cannot start. Filed as hive#328. **PR2 is gated on it; PR1 and PR3 are not.**
- **Bootstrap chicken-and-egg — RESOLVED, verified empirically.** `pyproject.toml:77-78` ships both a `hive` and a `hive-vault` console script for the same `hive.server:main`, so `uvx --from hive-vault hive --version` works with no prior install (`hive-vault 1.43.0`, rc=0, 2026-08-07). `_runtime.build_version()` is pure `uv venv` + `uv pip install --python <venv> hive-vault==<v>`, so the only prerequisite is `uv` — which `mcp-servers.json` already declares via `prerequisite_binary`.
- **The script under test is a deployed copy.** `~/.claude/scripts/hive-upgrade.ps1` is deployed by setup from the SSOT at `windows/hive-upgrade.ps1`. Verification must exercise the SSOT (or re-deploy first), or a stale copy will read as a pass.
- **Linux path — RESOLVED 2026-08-07.** Linux **keeps `uv tool install --upgrade`**. A3 exists solely to work around Windows' inability to replace an in-use executable; POSIX has no such constraint, so the versioned-dir + junction machinery would buy nothing and would add a second install model to maintain. `ai/hermes/setup.sh:83` and `setup-linux.sh` are unchanged by this spec. Recorded in the amended `AI-022/tasks.md` as well, so the decision is visible from the spec that made the original A1 call rather than only from here.
  *Amended 2026-10-03 (#1993):* Linux and macOS keep `uv tool`, but the version is pinned in `packages.json` and `dotf tools install` installs it. Upgrading on every setup run (`--upgrade`) is no longer the target. The pin is a floor: a version the user upgraded to on purpose is never rolled back. This follows hive ADR-020, under which hive notifies and the user upgrades. The entry lists `platforms: [linux, darwin]`, so the catalog never puts a uv copy on Windows, where hive owns its layout (hive ADR-019). The `prerequisite_command` keeps running from the twins until the cutover deletes their MCP loops.
- **DECIDED (owner, 2026-10-21): a `hive-self` source type in `packages.json`** (`tasks.md` PR6, option A). Installing a version the owner pinned and committed is not an unattended upgrade under hive ADR-020; the new catalog contract carries a repo ADR. The question as it was asked:
  **Who installs hive on Windows, and in what shape.** The catalog skips Windows, `dotf deploy` ignores `prerequisite_command`, and the setup twin still runs `uv tool install --upgrade hive-vault` there, which hive ADR-019 rules out. Two readings decide the design, and neither is settled here:
  1. Does a converge step that runs `hive self-upgrade <pin>` count as acting "on the user's behalf" under hive ADR-020? It upgrades only to a version the repo pinned and the owner committed, not to PyPI's latest.
  2. Is it a new `packages.json` source type (a public contract, and #1993's direction) or a dedicated converge step? The candidate shapes are listed in `tasks.md` (PR6).
- **DECIDED (owner, 2026-10-21): fixed in hive**, which refuses a source build of cryptography and takes the newest version with a wheel (option 1, [hive#515](https://github.com/mlorentedev/hive/issues/515)). The PLAT-003 guest stays the AC2 proof. The question as it was asked:
  **How hive reaches a Windows Arm64 machine.** `hive-vault` needs `cryptography>=42` (hive `pyproject.toml:40`). cryptography shipped `win_arm64` wheels only in 46.0.0–46.0.3; 50.0.2, the latest, has none. So uv builds it from source, and that fails without the MSVC linker (V6). Every install path ends in the same `uv pip install`, `hive self-upgrade` included. Measured options, none chosen:
  1. Forbid the source build: `uv pip compile` for `aarch64-pc-windows-msvc` with `--only-binary cryptography` resolves hive-vault 4.4.1 with cryptography 46.0.3. This is a resolution, not an install. It could live in hive's installer or in an environment marker in hive's own dependency.
  2. Run hive on x64 CPython under emulation (`--python cpython-3.13-windows-x86_64`).
  3. Put build tools in the clean-base. That makes the guest unlike a fresh machine.
  Filed upstream as [hive#515](https://github.com/mlorentedev/hive/issues/515), with the log excerpt and the resolution.
- **DECIDED (owner, 2026-10-21): a `dotf converge` step, `hive-upgrade-retire`** (#2857 option A). The question as it was asked:
  **Retiring the timers on machines that already have them.** Deleting the units from the repo stops new installs; it does not remove `hive-upgrade.timer` or `DotfilesHiveUpgrade` from a machine that has them. That is new behaviour, so it goes in Go, in the shape of the converge retire in #2799. Where the step lives is a design choice (#2857).
- **Blast radius.** This changes how the maintainer's daily hive install resolves on Windows. Mitigated by the stdio fallback remaining available throughout, and by PR1 landing independently of the model change.

## Acceptance criteria

> **Re-scoped 2026-10-21 (owner, #791).** The spec stalled on a Windows box whose broken state no longer exists, and two decisions landed after it was written: hive ADR-019 (Windows owns the A3 layout) and hive ADR-020 (hive notifies, never auto-applies). A clean Windows 11 **Arm64** guest (`dotf vm windows`, PLAT-003) reproduces the failure: converge ends green with no hive on PATH and the daemon skipped in silence (Baseline 2 in `verification.md`). On Arm64 the cause is that no install of hive can succeed there today (V6). x64 is unmeasured: there the same setup most likely installs hive as a uv tool, the copy ADR-019 rules out. AC3 and AC7 are withdrawn, AC2 and AC4 are rewritten so the guest is the proof, and AC9 is added. The original wording stays, struck through, so the change is visible.

- [x] **AC1** — `windows/hive-upgrade.ps1` emits a distinguishable, non-empty message when no hive install is found, and stays silent when the install is present and already current. Asserted by `tests/hive-upgrade-timer.bats`, not by convention.
  *Superseded 2026-10-21:* delivered by #796, and its subject goes away with AC9: the script and its tests are deleted with the timer (#2857). The class it fixed (quiet when broken) is AC4's second half on the setup path.
- [x] **AC2** — ~~On a machine with no hive install and no `hive` on PATH, setup bootstraps a working install; `hive --version` succeeds from a fresh shell afterwards.~~
  *Rewritten 2026-10-21:* on a clean Windows 11 guest with no hive (`dotf vm windows --ref <ref>`), converge installs hive through hive's own layout (ADR-019). In the second converge, a new process, `hive` resolves on PATH and `dotf doctor`'s hive-daemon section no longer reports it missing. The second converge changes nothing.
  **Gated:** the PLAT-003 guest is Arm64, where hive cannot install today (V6). The guest proof waits on a hive release carrying hive#515. Without one, the proof is a clean x64 Windows machine.
  *Met 2026-10-22:* hive 4.4.2 (hive#523) plus the catalog's `no_build: ["cryptography"]` installs on the Arm64 guest. The two-converge run is in `verification.md`, "Guest run on Arm64".
- ~~**AC3** — The steady-state upgrade trigger is a bare `hive self-upgrade` with no daemon stop/start orchestration, and re-running it when already current is a no-op.~~
  *Withdrawn 2026-10-21:* hive ADR-020 forbids any timer, service or hook from running `hive self-upgrade` on the user's behalf. AC9 replaces it.
- [x] **AC4** — ~~No mechanism in this repo infers "hive is installed" from `uv tool list` alone on the Windows path.~~
  *Rewritten 2026-10-21:* on Windows, no setup or converge step creates a `uv tool` install of `hive-vault` or infers hive from `uv tool list`. When `hive` is missing, setup says so instead of skipping the daemon in silence: on main the whole daemon block is skipped with no output (baseline V3).
  *Met in the tree, not yet in the pinned release:* PR6 (#2901) and the guards in `tests/hive-daemon-gate.bats`. Released dotf 0.77.0, which `versions.conf` still pins, does not read `self_install`, so it still runs `uv tool install hive-vault` on Windows (#1675; #2955 is a box where it did). Release 0.78.0 (#2911) closes the gap. A box that already holds the uv copy is #2991.
- [x] **AC5** — `AI-022`'s A1 decision is reconciled with hive's shipped A3, and archived `AI-023` carries a forward pointer; ADR-015 is referenced consistently.
- [x] **AC6** — The Linux path is explicitly decided and documented, not left implicit.
- [x] **AC8** — *Added 2026-10-03 (#1993).* On Linux and macOS, `dotf tools install` installs `hive-vault` at the version pinned in `packages.json`. It upgrades a lower version and never downgrades a higher one. On Windows the tool is reported as unsupported, never installed and never failed.
- ~~**AC7** — On the maintainer's currently-broken Windows box, all four before-state symptoms flip (see `verification.md`), and Claude Code's `hive` MCP entry resolves to a working binary.~~
  *Withdrawn 2026-10-21:* that box no longer holds the before-state (by 2026-08-27 it answered `hive --version` 3.0.0, #1311), so B1–B9 cannot flip. The guest baseline V1–V5 in `verification.md` is the reproducible before-state that AC2 and AC4 flip.
- [ ] **AC9** — *Added 2026-10-21 (#2857).* No timer, task or hook in this repo upgrades hive unattended. A machine that had `hive-upgrade.timer` or `DotfilesHiveUpgrade` has neither after one converge, and a second converge changes nothing.

## References

- Bitácora board: [dotfiles#791](https://github.com/mlorentedev/dotfiles/issues/791)
- Sibling spec: AI-029 / [dotfiles#792](https://github.com/mlorentedev/dotfiles/issues/792) — MCP registration SSOT
- Upstream dependency: [hive#328](https://github.com/mlorentedev/hive/issues/328) — PATH launcher + `_resolve_exec()` hardening
- Safety net: [dotfiles#574](https://github.com/mlorentedev/dotfiles/issues/574) (HARNESS-049) — `dotf doctor --fix`
- Predecessors: `specs/AI-022-hive-daemon-activation/`, `specs/archive/AI-023-hive-auto-upgrade-timer/`
- hive: `specs/HIVE-267-upgrade-swap/` (A3 mechanism), `docs/adr/adr-015-windows-daemon-supervision-upgrade.md` (`proposed`), [hive#292](https://github.com/mlorentedev/hive/issues/292), [hive#176](https://github.com/mlorentedev/hive/issues/176)
- Troubleshooting: `docs/troubleshooting/hive-mcp-orphaned-trampoline.md`
