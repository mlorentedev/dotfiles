---
id: "AI-028-hive-install-model-migration"
type: spec
status: draft # draft | implementing | verifying | archived
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
- **Linux path.** A3 is a Windows mechanism (in-use-file locking does not apply on POSIX). `ai/hermes/setup.sh:83` and `setup-linux.sh` plausibly keep `uv tool`. [AGENT-DRAFT — review before archive] Proposed resolution: Linux keeps `uv tool install --upgrade`, and the decision is recorded explicitly in this spec and in the amended `AI-022`, so it is a choice rather than an omission.
- **Blast radius.** This changes how the maintainer's daily hive install resolves on Windows. Mitigated by the stdio fallback remaining available throughout, and by PR1 landing independently of the model change.

## Acceptance criteria

- [ ] **AC1** — `windows/hive-upgrade.ps1` emits a distinguishable, non-empty message when no hive install is found, and stays silent when the install is present and already current. Asserted by `tests/hive-upgrade-timer.bats`, not by convention.
- [ ] **AC2** — On a machine with no hive install and no `hive` on PATH, setup bootstraps a working install; `hive --version` succeeds from a fresh shell afterwards.
- [ ] **AC3** — The steady-state upgrade trigger is a bare `hive self-upgrade` with no daemon stop/start orchestration, and re-running it when already current is a no-op.
- [ ] **AC4** — No mechanism in this repo infers "hive is installed" from `uv tool list` alone on the Windows path.
- [ ] **AC5** — `AI-022`'s A1 decision is reconciled with hive's shipped A3, and archived `AI-023` carries a forward pointer; ADR-015 is referenced consistently.
- [ ] **AC6** — The Linux path is explicitly decided and documented, not left implicit.
- [ ] **AC7** — On the maintainer's currently-broken Windows box, all four before-state symptoms flip (see `verification.md`), and Claude Code's `hive` MCP entry resolves to a working binary.

## References

- Bitácora board: [dotfiles#791](https://github.com/mlorentedev/dotfiles/issues/791)
- Sibling spec: AI-029 / [dotfiles#792](https://github.com/mlorentedev/dotfiles/issues/792) — MCP registration SSOT
- Upstream dependency: [hive#328](https://github.com/mlorentedev/hive/issues/328) — PATH launcher + `_resolve_exec()` hardening
- Safety net: [dotfiles#574](https://github.com/mlorentedev/dotfiles/issues/574) (HARNESS-049) — `dotf doctor --fix`
- Predecessors: `specs/AI-022-hive-daemon-activation/`, `specs/archive/AI-023-hive-auto-upgrade-timer/`
- hive: `specs/HIVE-267-upgrade-swap/` (A3 mechanism), `docs/adr/adr-015-windows-daemon-supervision-upgrade.md` (`proposed`), [hive#292](https://github.com/mlorentedev/hive/issues/292), [hive#176](https://github.com/mlorentedev/hive/issues/176)
- Troubleshooting: `docs/troubleshooting/hive-mcp-orphaned-trampoline.md`
