---
id: "OPS-040-dead-migration-purge"
type: spec
status: verifying # draft | implementing | verifying | archived
created: "2026-09-01"
issue: "mlorentedev/dotfiles#1333"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# OPS-040-dead-migration-purge

> **Naming**: file lives at `<repo>/specs/OPS-040-dead-migration-purge/proposal.md`. `OPS-040-dead-migration-purge` is `AREA-NNN-slug`.

## Why

<!-- from issue #1333: OPS-040: dead-migration purge in both setups - converge-old-machines code with no expiry, plus a secret decrypted for nothing on every run -->

Both setup scripts carry one-time migration blocks written to converge machines off a retired arrangement, none of which records when it may be removed. They accumulate: every reader must work out whether a block is still load-bearing, and the answer is not in the code. One of them decrypts a credential on every run for a consumer that no longer exists.

The ticket proposes deleting roughly 200 lines as finished work. **Measurement does not support that framing, and this spec exists to replace it with a per-block one.** A block is safe to delete only when skipping it costs nothing on an unconverged machine, so each was classified by what breaks if it never runs, and the conditions the correcting ones test were probed on this box. Three of the four correcting blocks are converged on Linux. The fourth has never worked at all.

## What

Nine blocks leave both setup scripts. Two stay, each for a stated reason. After this change, `setup-linux.sh` and `setup-windows.ps1` no longer decrypt a credential nobody reads, no longer install a runtime for a retired consumer, and no longer carry cleanup for paths that stopped existing.

**Deleted — dead code (skipping costs nothing on any machine, converged or not):**

| # | Block | Location | Why it is dead |
|---|---|---|---|
| 1 | `OPENROUTER_API_KEY` deploy-time export | `setup-linux.sh:282-296`, `setup-windows.ps1:690-701` | Fetched for the agy `mcp_config` cascade that CLI-042 AC8 removed. Zero readers remain: the only other repo mentions are two comments recording that the cascade is gone, and the interactive `opencode`/`pi` wrappers in `.bashrc:113-114` / `powershell/profile.ps1:297-298` resolve it independently through `dotf secrets run`. Removing it stops a decryption on every setup run. |
| 2 | `ANTIGRAVITY_ENDPOINT` / `CLOUDCODE_URL` exports | `setup-linux.sh:390-397`, `setup-windows.ps1:1413-1418` | Dead **within setup**, which is all these lines reached. See the note below — they are not dead repo-wide, and nothing about the user's environment changes. |
| 3 | SDD-007 orphan `mcp_config.json` removal | `setup-linux.sh:482-489` | Deletes a file at the pre-SDD-007 path. agy reads the canonical path; the orphan is inert. |
| 4 | SDD-007 legacy `GEMINI.md` removal | `setup-linux.sh:500-505`, `setup-windows.ps1:1490-1497` | **Not dead — actively wrong. See below.** |
| 5 | CLI-014 orphan `init-project.sh` removal | `setup-linux.sh:592-594` | Deletes a script retired in favour of `dotf init`. Nothing invokes it. |
| 6 | `python` → `python3` convenience symlink | `setup-linux.sh:642-648` | Not a migration — a symlink inside `~/Applications/python-*/bin`. Nothing in either script installs to `~/Applications` any more, so the glob matches nothing and the block has been a no-op since that install path was dropped. |
| 7 | Bun installer | `setup-windows.ps1:1064-1086` | Installed for *"some Claude Code plugin workers for bun:sqlite"* — the claude-mem worker, retired 2026-06-23. No repo code invokes `bun`, `bunx` or `bun run`. |

**Deleted — correcting, and probed converged on the only OS that runs them:**

| # | Block | Location | Probe result |
|---|---|---|---|
| 8 | gh-copilot stale `eval` line strip | `setup-linux.sh:977-984` | Strips `eval "$(gh copilot alias -- bash)"`, which errors on every shell startup when present. Absent from both `~/.zshrc` and `~/.bashrc` on msi. Linux-only; no Windows twin. |
| 9 | Legacy `uv tool upgrade hive-vault` cron removal | `setup-linux.sh:1212-1219` | Retires the manual cron now `hive-upgrade.timer` owns upgrade policy. Absent from `crontab -l` on msi. Linux-only by construction (crontab). |

**Kept, with the reason stated in the code:**

| # | Block | Location | Why it stays |
|---|---|---|---|
| 10 | HIVE-118 stale `uvx hive-vault` MCP entry | `setup-linux.sh:1073-1082`, `setup-windows.ps1:563` | Correcting, and it has a Windows twin. Skipping it leaves the `hive` MCP entry pinned to the retired `uvx` definition, and because the surrounding loop is skip-if-present, the current `hive client` definition is never re-added. Probed absent on msi; **the Windows box cannot be probed from here**, so it is deferred to the batched Windows session rather than deleted on one OS's evidence. |
| 11 | MEM-002 claude-mem retirement | `setup-linux.sh:1305-1337`, `setup-windows.ps1:894-935` | **It has never worked.** Filed as #1431. Untouched here — the whole block, not part of it. |

### Block 2 is dead inside setup only, and `.zshrc` is where the value actually comes from

Stated carefully, because a first pass got it wrong by grepping only the two setup scripts. `ANTIGRAVITY_ENDPOINT` / `CLOUDCODE_URL` have three homes, and only one of them is deleted here:

- **`setup-linux.sh` / `setup-windows.ps1`** — an `export` into the setup process. Nothing later in either script reads it; on Linux it cannot outlive the process. **Deleted.** On Windows the same block also wrote them to the user environment, which is the part that leaves values behind (see Out of scope).
- **`.zshrc:42-43`** — the live source. Every interactive zsh exports them, which is why they are set in a running shell despite setup having no lasting effect. **Untouched**, and `tests/antigravity.bats:84` still pins it. (`.bashrc` does not export them — a parity gap that predates this change and is not in scope.)
- **`cli/internal/doctor/checks_deploy.go:801`** — `dotf doctor` reads the variable, but as `sys.env("ANTIGRAVITY_ENDPOINT", "https://cloudcode-pa.googleapis.com")`, i.e. **defaulting to production when unset**, inside a section that SKIPs entirely when `agy` is not on PATH. So the check passes identically before and after.

The block's own comment — *"Setting these has no observed effect on routing"* — is a statement about **agy's behaviour**, not about whether anything reads the variables. Something does. The deletion is still right, and it is a no-op for both the user's shell and `dotf doctor`; the reason is narrower than "these are inert".

### Block 4 is a defect, and removing it is the fix

Probing the deleted blocks' targets on msi turned up `~/.gemini/GEMINI.md` **present, 12029 bytes, generated doctrine with a harness `sha256:` marker**. It is not an orphan of the retired `gemini-cli`. `harness/manifest.json:151` declares it as agy's live doctrine deploy target, and records why: *"Antigravity reads global rules from `~/.gemini/GEMINI.md`… The file is shared with Gemini CLI, so injection is append-and-replace-in-place, never an overwrite."* `tests/compile-harness.bats:808-814` asserts precisely that a user's own rules in that file survive a deploy.

Both setup scripts `rm -f`'d it on every run, describing it as a legacy orphan. A file cannot be both a deploy target with an append-in-place contract and something setup deletes. Only ordering hid the conflict — setup removed it early, `compile-harness.sh --deploy` rewrote it near the end — so the destruction was invisible unless a run stopped in between, the deploy was skipped, or the order changed. The append-in-place contract exists to protect user content in that file, and the `rm -f` defeated it on every single run.

So block 4 is deleted as a **fix**, and it carries a guard: `tests/guard-doctrine-target-not-deleted.bats` reads `.doctrine.deploy[]` from the manifest and refuses any removal naming a declared target in either setup script. It reads the manifest rather than hardcoding `GEMINI.md` so a target declared later is covered the day it is declared. Verified fail-first: red against `main`'s scripts at `setup-linux.sh:503`, green on this tree.

**Shell rc `BUN_INSTALL` exports are also kept** (`.zshrc:130-135`, `.bashrc:152-154`), which the ticket lists for deletion. `bun` is installed on msi at `~/.bun/bin/bun`; deleting the `PATH` export removes a working binary from the user's interactive shell for no benefit. "Zero consumers in the repo" is not "zero consumers on the machine". Setup stops *installing* bun; the shell keeps *finding* the one that is there.

## Out of scope

- **MEM-002 (#1431).** Measured during this spec: `~/.claude/plugins/marketplaces/thedotmack` is a live clone of `thedotmack/claude-mem` re-fetched an hour before measurement, because Claude Code moved the marketplace registry to `~/.claude/plugins/known_marketplaces.json` (`autoUpdate: true`) while the block still strips `settings.json`'s `extraKnownMarketplaces` key — which no longer exists. The `rm -rf` runs, Claude re-clones, forever. Fixing it means choosing between a `claude plugin marketplace remove` subcommand that may or may not exist and hand-editing a state file this repo has never touched, plus a Windows leg. That is new behaviour, unverified on either OS, and it does not belong in a PR whose thesis is "delete code proven dead".
- **HIVE-118**, deferred to the Windows session for the reason in the table above.
- **A `# EXPIRES:` convention or a CI check enforcing it.** The lesson is written; the machinery is not built. Proposed to the owner in the PR body as a decision, not landed unasked — and adding an expiry-less migration inside the PR whose lesson is "migrations carry an expiry" would refute itself.
- **Scrubbing the Windows registry values** that `ANTIGRAVITY_ENDPOINT` / `CLOUDCODE_URL` persisted. Removing them needs a new one-time migration, which is the thing this spec is deleting. They linger inert — "no observed effect" is the block's own measurement. Same owner decision as above.
- **#1337** (OPS-043, the duplicated doctor checks). Separate concern, separate PR.

## Risks / open questions

- **Convergence is probed on one machine, not proven on both.** msi is clean for every correcting block deleted here, and both are Linux-only surfaces (`~/.zshrc`/`~/.bashrc`, `crontab`), so msi is the only machine they could ever have run on. Every cross-platform block deleted is dead code, whose deletion is safe on an unconverged machine by definition. RESOLVED by that split: nothing deleted here depends on the Windows box having converged.
- **`secrets-show-callsites.bats` goes vacuous.** Blocks 1 removes the only two real `dotf secrets show` call sites (`setup-linux.sh:291`, `setup-windows.ps1:697`); the four remaining grep hits are log strings and prose. Its loop would then iterate an empty list and report a pass having checked nothing. RESOLVED: the guard learns to SKIP with its reason when it finds no call sites, per C15 — a check that cannot answer must not pass.
- **`tests/setup-windows.bats:993` pins the Bun installer** alongside the uv installer (both must run in a child pwsh, TEST-003). RESOLVED: narrow the assertion to uv; the invariant it protects is about child-pwsh execution, not about Bun specifically.
- **Unknown breakage in the other marker-referencing suites** (`setup-linux.bats`, `iac-deploy.bats`, `antigravity.bats`). Not resolvable by reading; resolved empirically by running bats after the deletion and fixing only what breaks.

## Acceptance criteria

- [ ] AC1 — Neither setup script contains a `dotf secrets show OPENROUTER_API_KEY` call site, and no repository code exports `OPENROUTER_API_KEY` into the setup process.
- [ ] AC2 — Blocks 2 through 7 are absent from both setup scripts: no `ANTIGRAVITY_ENDPOINT`/`CLOUDCODE_URL` export, no pre-SDD-007 `mcp_config.json` or legacy `GEMINI.md` removal, no `init-project.sh` orphan removal, no `~/Applications` python symlink, no Bun installer.
- [ ] AC3 — Blocks 8 and 9 are absent from `setup-linux.sh`: no `gh copilot alias` strip loop, no `uv tool upgrade hive-vault` crontab removal.
- [ ] AC4 — The HIVE-118 and MEM-002 blocks are byte-identical to `main` in both scripts, and `.zshrc`/`.bashrc` retain their `BUN_INSTALL` exports.
- [ ] AC5 — `secrets-show-callsites.bats` reports a SKIP naming the reason when no `dotf secrets show` call site exists, rather than a pass, and still fails on a call site referencing an unknown registry id.
- [ ] AC6 — `bats tests/*.bats` passes under both bash and zsh, and `bash -n` / `shellcheck` are clean on `setup-linux.sh`; PSScriptAnalyzer is clean on `setup-windows.ps1`.
- [ ] AC7 — `tests/guard-doctrine-target-not-deleted.bats` fails when either setup script contains a removal naming a file declared in `harness/manifest.json` `.doctrine.deploy[]`, resolves its target list from that manifest rather than a hardcoded name, and SKIPs with a reason when the manifest yields no targets.
- [ ] AC8 — `docs/lessons/` records the lessons this turned up: that a one-time migration without a removal date is indistinguishable from live code; that a cleanup block's own description of what it deletes is not evidence (block 4 called a live deploy target a legacy orphan); and that MEM-002 asserted against a proxy rather than the end-state — the same class archived spec `BUG-014-claude-mem-marketplace-register` had already recorded one cycle earlier.

## References

- Bitácora board: mlorentedev/dotfiles#1333.
- Spun out of this spec: #1431 (MEM-002 never converged), and the deferred HIVE-118 Windows leg.
- Sibling debt ticket, deliberately not folded in: #1337 (OPS-043).
- Source audit: scripts-parity audit F3, `10_projects/dotfiles/research/2026-08-27-windows-drive-audit-plan.md` (knowledge vault).
- Prior art on the proxy-vs-end-state failure: `specs/archive/BUG-014-claude-mem-marketplace-register/`.
