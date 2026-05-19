---
id: "BUG-004-claude-mem-truncate-guard"
type: spec
status: archived
created: "2026-05-19"
archived: "2026-05-19"
merged_pr: 57
tags: [spec, proposal, bug, defense-in-depth, claude-cli, cross-os]
template_version: "1.0"
---

# BUG-004-claude-mem-truncate-guard

> **Naming**: file lives at `<repo>/specs/BUG-004-claude-mem-truncate-guard/proposal.md`.

## Why

<!-- from 11-tasks.md: BUG-004-claude-mem-truncate-guard *(P0, opens 2026-05-19)* — Trigger fix residual after SDD-021 monitor; `claude-mem@thedotmack` triggers `claude plugin install` truncation of `.claude.json` on every setup run. -->

Every run of `setup-windows.ps1` or `setup-linux.sh` silently truncates `~/.claude/.claude.json` from ~75 KB to ~1.5 KB, dropping `organizationType` / `organizationRateLimitTier` / onboarding flags, which forces re-authentication in every project on the next session. Root cause: the plugin-install idempotence guard checks the literal `claude-mem@thedotmack` against the output of `claude plugin list`, but that output only enumerates the `@claude-plugins-official` marketplace — `claude-mem` (from `@thedotmack` marketplace) **never matches**, so the guard yields a false negative on every run, triggering one real `claude plugin install claude-mem@thedotmack` call, which hits upstream `anthropics/claude-code#59870` (CLI's deserialize-modify-serialize cycle drops fields outside its internal struct). SDD-021 (✓ 2026-05-18) added a size monitor to `claude-session-start.{sh,ps1}` as a canary; the trigger fix originally claimed in dotfiles#33 is in practice incomplete — claude-mem is the residual trigger. Empirically reproduced 2026-05-19: setup output `Claude Code plugins ready (1 added, 11 already present)` → `.claude.json` 75k→3444 bytes → re-login prompt in every Claude Code session until restored from `~/.claude/backups/.claude.json.backup.1779138775569` (51980 bytes, 18 May 15:09).

## What

Both `setup-windows.ps1` and `setup-linux.sh` gain a defense-in-depth wrapper around the `claude plugin install` call inside the plugin-install loop. Before each install: snapshot `~/.claude/.claude.json` to a tempfile (Copy-Item / `cp`). After each install: read the new file size; if the **pre-install size was ≥ 10 KB** (sanity gate to avoid acting on fresh-machine tiny files) AND the **post-install size is < 50 % of the pre-install size**, restore the snapshot atomically. The wrapper always cleans the tempfile, success or failure. Restoration logs a `[WARNING]` line citing `anthropics/claude-code#59870`. The existing idempotence check on `claude plugin list` is preserved (still catches the common case for `@claude-plugins-official` entries); the wrapper is the second layer that catches false negatives like `claude-mem`.

Observable post-PR behaviour:

1. Running `setup-windows.ps1` or `setup-linux.sh` twice in a row on an authenticated machine leaves `.claude.json` size unchanged across runs (no re-login required).
2. If the upstream CLI bug fires anyway (e.g. another future plugin shows the same idempotence false-negative class), the wrapper restores `.claude.json` before the next call and emits a visible warning line in the setup log naming the file and the upstream issue.
3. The setup log gains exactly **one** new line per truncation event (no spam on healthy runs).

## Out of scope

Things this PR explicitly does NOT include. Forces a sharp boundary and prevents scope creep.

- Fixing the underlying upstream bug `anthropics/claude-code#59870` — that lives in `@anthropic-ai/claude-code` CLI source, not in this repo. Defence in depth is the only handle we have.
- Removing `claude-mem@thedotmack` from the install array — it remains a legitimate dependency (active MCP for conversation memory per `CLAUDE.md`). The goal is to make its install idempotent under the upstream bug, not to drop it.
- Switching the idempotence regex to also catch claude-mem — the literal `claude-mem@thedotmack` genuinely does not appear in `claude plugin list` output, so no regex fixes the root false negative. The wrapper is the correct layer.
- Replacing SDD-021's session-start size monitor — that canary stays as a complementary alarm (catches truncations from sources other than this setup script). BUG-004 is the trigger fix; SDD-021 remains the detector.
- BUG-005 (PowerShell 5.1 `-AsHashtable` re-exec) — separate atomic PR with its own spec folder.
- Re-running `claude` commands in CI to detect the issue automatically — out of scope; the bats tests assert on the helper's shape and integration, not on a live `claude` invocation.

## Risks / open questions

Failure modes, dependencies, and unknowns to clarify before implementation. If any item here is unresolved, do not move to `tasks.md` yet.

- **Threshold heuristic** (10 KB floor + 50 % shrink): chosen to match SDD-021's existing canary threshold (10 KB) plus a relative drop to tolerate small absolute changes (e.g. legitimate addition of a single plugin entry growing the file by ~200 bytes). False positive: a legitimate operation that genuinely shrinks `.claude.json` to <50 % of its size. None known. False negative: a future bug variant that shrinks by exactly 49 % — defended by SDD-021's absolute-size canary at session start.
- **Subscription state vs. plugin state coupling**: restoring the snapshot reverts any **legitimate** writes the install would have made to `.claude.json` (e.g. registering the new plugin in the manifest section). Mitigation: the subsequent re-run of setup will retry the install; the upstream CLI is the one losing state on each call, so dropping that single write is preferable to losing subscription state. Documented in inline comments.
- **macOS / BSD `stat` flag differences**: `setup-linux.sh` already targets Linux only (Docker integration test is Ubuntu 24.04 per `tests/Dockerfile.integration`). Cross-platform stat (`-c %s` vs `-f %z`) is out of scope; the helper uses GNU `stat -c %s`. If macOS support comes later (separate spec), the helper takes a one-line conditional.
- **PowerShell scope of counter increment**: the original loop increments `$pluginsAdded++` after a successful install. The new wrapper must preserve this without forcing `$script:` scope quirks. Resolved by returning a boolean from the helper and incrementing in the loop body.
- **Concurrency**: not a real risk — setup runs single-threaded; no other process is expected to write to `.claude.json` between snapshot and restore. Documented for completeness.

## Acceptance criteria

Observable outcomes. Each must be testable.

- [ ] After running `setup-windows.ps1` twice in a row on a Windows machine with `pwsh` and `~/.claude/.claude.json` > 50 KB, the file size is unchanged across the two runs (within ±1 byte) and no re-authentication prompt appears in a subsequent Claude Code session.
- [ ] Same as above on Linux with `setup-linux.sh`.
- [ ] When a synthetic truncation is simulated (replace `claude plugin install` with a stub that overwrites `.claude.json` with `{}`), the helper restores the pre-call snapshot and emits exactly one `[WARNING] .claude.json shrunk from <X> to <Y> bytes after install (upstream #59870); restored from backup` line to stdout/stderr.
- [ ] On a fresh machine where `~/.claude/.claude.json` does not exist before the install loop, the helper is a no-op (no temp file lingers, no warning fires) — verified by bats stubbed-env test.
- [ ] On a fresh machine where `~/.claude/.claude.json` exists but is < 10 KB (e.g. 2 KB), a shrink to 1 KB does NOT trigger restoration (the 10 KB floor gates it) — verified by bats.
- [ ] PSScriptAnalyzer (Error+Warning) clean on `setup-windows.ps1`; `bash -n setup-linux.sh` clean; `bats tests/setup-windows.bats tests/verify-setup.bats` green.
- [ ] CI 5/5 green on the PR (GitGuardian, integration, lint, lint-powershell, test).

## References

- Vault: `10_projects/dotfiles/11-tasks.md` (BUG-004 backlog entry)
- Upstream issue: [`anthropics/claude-code#59870`](https://github.com/anthropics/claude-code/issues/59870) (CLI deserialize-modify-serialize cycle drops fields)
- Related: dotfiles#33 (original trigger fix, now identified as incomplete for claude-mem)
- Related: SDD-021 (vault `11-tasks.md`, completed 2026-05-18) — session-start size monitor (canary, complementary)
- Related ADR: `30-architecture/adr-007-mcp-persistence-and-auto-memory.md` (rationale for `claude-mem` being deployed via setup)
- Sibling spec: `specs/BUG-005-setup-ps7-reexec/` (PR after this one)
- Pattern: `00_meta/patterns/pattern-setup-script-idempotence.md` (existing pattern; this spec extends it with the snapshot/restore layer)
