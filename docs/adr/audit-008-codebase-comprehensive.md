---
id: audit-008-codebase-comprehensive
type: audit
status: active
date: "2026-07-06"
related: [audit-009-documentation, audit-010-process-workflows, audit-001-repo-structure, audit-002-cross-os-duplication, audit-005-scripts-classification, audit-007-cli-convergence-state]
tags: [audit, adversarial-review, dotfiles, cross-platform, codebase]
---
# Codebase Audit — dotfiles — 2026-07-06

> Exhaustive, adversarial audit. Every file under `git ls-files` (954 tracked files) was read
> or covered by a dedicated read-only agent: the two setup scripts in full, the entire `cli/`
> Go tree's load-bearing packages, `scripts/` (35 files), `tests/` (66) + `.github/` (10),
> `secrets/`/`git-hooks/`/`systemd/`/`ssh/`, `harness/` + `ai/` + the shell rc payload, all
> ADRs/runbooks, and every root config. Findings are marked **CONFIRMED** (both sides traced)
> or **PLAUSIBLE** (suspected, not fully traced). Where a subsystem is sound it is said once
> and left alone — effort is spent where it isn't.
>
> **Headline:** the Go `dotf` CLI is the best-engineered layer (atomic writes, fail-loud
> secrets, validated registry, Open/Closed resolvers, CI-tested on both OSes). The **shell +
> PowerShell layer around it is where the real defects live** — silent-failure idioms, twin
> drift, dead configuration that lies about being read, and a class of Windows/Linux encoding
> and path divergences that the Go layer already fixed but the shells did not.
>
> **Companion audits:** `audit-009-documentation.md` (docs), `audit-010-process-workflows.md` (process) — same series; see `related:` frontmatter. Vault decide/position layer: `10_projects/dotfiles/research/2026-07-02-project-coherence-audit.md` (external benchmarking + methodology + backlog).

---

## 1. Summary table (severity order)

| ID | Sev | Area | One-line issue | file:line | Status |
|----|-----|------|----------------|-----------|--------|
| C1 | Critical | scripts/health | `grep -c \|\| echo 0` double-emits "0\n0" → arithmetic crash under `set -e`; a *healthy* vault exits FAILED | `scripts/vault-health.sh:132,135,175,201,225` | CONFIRMED |
| C2 | High | rc/bash | All bash aliases are dead: setup writes `~/.bash/bash_aliases`, `.bashrc` sources `~/.bash_aliases` | `setup-linux.sh:148` vs `.bashrc:129` | CONFIRMED |
| C3 | High | scripts/gate | SDD spec-gate fails **open**: an unresolvable base ref → `TOTAL_LOC=0` → "0 LOC < threshold", exit 0 | `scripts/check-spec-gate.sh:181` | CONFIRMED |
| C4 | High | config/mem | The 11 `injectors.*.enabled` flags are dead config — `ClaudeContext` never calls `injectorEnabled()` | `cli/internal/mem/session_start_adapter.go:44-80` | CONFIRMED |
| C5 | High | scripts/win | Windows weekly maintenance task has never run: `--all` can't bind to a `[switch]$All` param | `scripts/vault-maintenance-weekly.ps1:23` | CONFIRMED |
| C6 | High | config/win | `SCRIPTS_DIR` self-contradiction: contract default `.dotfiles\scripts`, PATH+deploy use `~\scripts` | `env-contract.json:40` vs `:129` / `setup-windows.ps1:79` | CONFIRMED |
| C7 | High | scripts/sync | `dotfiles-sync.sh` secrets "newest-wins" ping-pongs forever (`cat >` resets mtime) | `scripts/dotfiles-sync.sh:76` | CONFIRMED |
| C8 | High | scripts/policy | `bitacora-rollout.sh` runs `gh pr merge --auto` — the one thing every AGENTS.md forbids | `scripts/bitacora-rollout.sh:124` | CONFIRMED |
| C9 | High | tests/CI | 15 agy regression guards never run in CI (blanket `skip`, no job installs agy) | `tests/antigravity.bats:17` | CONFIRMED |
| C10 | High | docs | README documents a retired command surface (`secrets_add/rotate/show/list/check`, `dotfiles-sync`) | `README.md:100-107,152` | CONFIRMED |
| C11 | High | scripts/win | `knowledge-crystallize.ps1` single-project mode always misses on Windows (drops a dash in the key) | `scripts/knowledge-crystallize.ps1:57` | CONFIRMED |
| C12 | Med-High | git-hooks | commit-msg hook rejects the repo's own scoped-Conventional-Commit convention | `.github/hooks/validate-commit-msg.sh:5` | CONFIRMED |
| C13 | Med | scripts/sync | `dotfiles-sync.ps1` dirty-check tests only the *last* git exit code → pushes over unstaged changes | `scripts/dotfiles-sync.ps1:133` | CONFIRMED |
| C14 | Med | scripts/sync | Twin drift: `.sh` rsync-copies (ADR-005), `.ps1` `git pull`s `~/.dotfiles`; PS exits 0 on failure | `scripts/dotfiles-sync.{sh,ps1}` | CONFIRMED |
| C15 | Med | rc/bash | `.bashrc` hard-resets `PATH` to system dirs; `.zshrc` deliberately does not | `.bashrc:105` vs `.zshrc:83` | CONFIRMED |
| C16 | Med | setup/win | Auto-memory junction built at the wrong key on Windows (`:`→'' single-dash, not `:`→'-') | `setup-windows.ps1:867` | CONFIRMED |
| C17 | Med | config | Dead version pins: `OBSIDIAN_/EZA_/ZOXIDE_VERSION` have zero consumers; the manifest lies | `versions.conf:16-18` | CONFIRMED |
| C18 | Med | setup | `eza` installed from a `latest/download/` URL — the exact rot pattern the repo's own lesson bans | `setup-linux.sh:214` | CONFIRMED |
| C19 | Med | scripts/guard | `check-backlog-merged.sh` is structurally inert on the `~/Projects/Workspace/` layout | `scripts/check-backlog-merged.sh:86` | CONFIRMED |
| C20 | Med | doctor/win | `checkAntigravity` uses `HasPrefix(path,"/")` for "absolute" → false-FAIL on any Windows abs path | `cli/internal/doctor/checks_deploy.go:350` | CONFIRMED |
| C21 | Med | harness | `skill_targets_agent` substring match: `pi` matches `targets:[copilot]`; Windows twin uses `\b` | `scripts/compile-harness.sh:208` | CONFIRMED |
| C22 | Med | harness | Marker-injection has no "exactly one marker" guard → a lost END marker duplicates + eats user text | `scripts/compile-harness.sh:606`; `setup-windows.ps1:1856` | CONFIRMED |
| C23 | Med | harness/win | The whole `manifest.agents` (curator + presence) stage is not ported to Windows | `setup-windows.ps1:1733` (Deploy-SkillRecord) | CONFIRMED |
| C24 | Med | tests/CI | CI PSScriptAnalyzer covers 4 of ~12 production `.ps1`; `utils.ps1` (the shared lib) is unlinted | `.github/workflows/ci.yml:57` | CONFIRMED |
| C25 | Med | tests/CI | spec-gate has three bypass routes (label, any-spec-touch, `*generated*` glob) | `scripts/check-spec-gate.sh:98,139,207` | CONFIRMED |
| C26 | Med | scripts/secrets | Secret material passed on `argv` (visible in `/proc/*/cmdline`): rollout PAT + `nan-*` API keys | `scripts/bitacora-rollout.sh:148`; `scripts/nan-bench.sh:39` | CONFIRMED |
| C27 | Med | scripts/secrets | `age-encrypt-decrypt.sh` leaves plaintext `.dec` at mode 0644, no cleanup on failure | `scripts/age-encrypt-decrypt.sh:73` | CONFIRMED |
| C28 | Med | setup | Linux vault-maintenance cron keys on script name; a relocated repo leaves a stale dead-path cron | `setup-linux.sh:1405` | CONFIRMED |
| C29 | Med | rc/bash | bash never gets nvm (only `.zshrc` loads it); setup re-deploys `.bashrc` wiping installer init | `.bashrc` (no nvm block); `setup-linux.sh:174` | CONFIRMED |
| C30 | Med | config | env-contract.json `_comment`s cite deleted `doctor.{sh,ps1}`/`healthcheck.*` (also `cli/README.md:43`) | `env-contract.json:2,7` | CONFIRMED |
| C31 | Med | ci-coupling | Go CLI CI is path-filtered to `cli/**`; a breaking edit to its config inputs never runs `go test` | `.github/workflows/cli.yml:5` | CONFIRMED |
| C32 | Med | tests/CI | Supply chain: every GitHub Action pinned by moving tag, none by SHA (runs with write-scope PATs) | `.github/workflows/*` | CONFIRMED |
| C33 | Med | docs/process | PR template still requires a vault `11-tasks.md` entry abolished by ADR-018 | `.github/pull_request_template.md:11` | CONFIRMED |
| C34 | Med | specs | 35 of 57 active specs are shipped-but-unarchived; 2 ID collisions; 1 missing `tasks.md` | `specs/` | CONFIRMED |
| C35 | Med | secrets | `sensitive/env-mapping.conf` re-added with zero code consumers — split-brain vs registry SSOT | `sensitive/env-mapping.conf` | CONFIRMED |
| C36 | Med | harness/setup | Vault-dependent harness steps run before the ADR-025 path cascade loads → wrong `VAULT_PATH` on run 1 | `setup-linux.sh:411,508` vs `:1446` | CONFIRMED |
| C37 | Med-High | docs | Committed overlays hardcode `~/Projects/Workspace/…` as the fallback; contract says `~/Projects/…` | `ai/claude/CLAUDE.md:5`; `env-contract.json` | CONFIRMED |
| C38 | Low-Med | git/local | 12 tracked shell files sit CRLF-in-worktree despite `eol=lf`; break under bash, git stays clean | `git ls-files --eol` | CONFIRMED |
| C39 | Low-Med | rc | `.zshrc` sources oh-my-zsh unguarded and unconditionally; no repo step installs it | `.zshrc:13` | CONFIRMED |
| C40 | Low-Med | setup | `OPENCODE_VERSION` used bare under `set -u`; a checkout missing `versions.conf` aborts setup | `setup-linux.sh:654` | CONFIRMED |
| C41 | Low | tests | Tests that cannot fail (literal `true`; whitespace-eating `$(...)`; one-sided OR asserts) | `tests/verify-setup.bats:308`; `tests/utils.bats:243` | CONFIRMED |
| C42 | Low | .gitconfig/win | `helper = !/usr/bin/gh …` deployed verbatim to Windows where that path doesn't exist | `.gitconfig` | CONFIRMED |
| C43 | Low | docs | README metrics stale: "316 BATS tests" (actual ~949), "~50 scripts" (actual 35) | `README.md:44,56` | CONFIRMED |
| C44 | Low | secrets | `render` substitutes a secret verbatim into JSON; a `"`/`\` in the value corrupts the config | `cli/internal/secrets/render.go:90` | PLAUSIBLE |
| C45 | Low | harness | `--refresh` silently reverts in-marker manual edits with no diff (stderr sent to `/dev/null`) | `setup-linux.sh:509` | CONFIRMED |
| C46 | Low | ssh | Committed `ssh/config` publishes VPS public IP + usernames + a root-login host in a public repo | `ssh/config:49,91,144` | CONFIRMED |
| C47 | Low | model-docs | Model facts (`deepseek` default, ctx sizes) duplicated across 6 files and already drifted | `ai/nan/README.md` vs `ai/opencode/opencode.jsonc:34` | CONFIRMED |
| C48 | Low | harness | Deploy-dir `~/.dotfiles/harness` mirror never prunes; doctor's drift allowlist omits it → zombie records | `setup-linux.sh:543`; `checks_deploy.go:486` | CONFIRMED |

*(Lower-severity confirmed items — duplicate `@test` names, tautological helper tests, dead aliases (`cl=changelog-gen.sh`), `EDITOR` login/non-login split, stale `tmux.conf` symlink comment, non-ASCII in `.ps1` vs the excluded BOM rule — are folded into §3 by category rather than listed individually here.)*

---

## 2. System map (verify my understanding)

**What this repo is.** A cross-platform (Linux + Windows; macOS planned, unimplemented) personal
dev-environment bootstrapper. Two shell entrypoints deploy configs + install tooling; a Go CLI
(`dotf`) is the growing user-facing surface that a strangler-fig (ADR-020) is migrating the
`.sh`/`.ps1` twins into. Knowledge/AI-agent memory lives in an external Obsidian vault, linked in.

**Layers (by language boundary, ADR-020):**
- **Go (`cli/`, own module)** — `dotf {doctor,init,env,spec,vault,tools,mem,secrets,update,review}`.
  ~10k LOC + 67 test files. The user-facing logic destination.
- **Shell bootstrap** — `setup-linux.sh` (1491 lines), `setup-windows.ps1` (2052), and `scripts/`
  (35 files, `.sh`/`.ps1` twins + libraries + CI/hook guards). Provisions the Go binary and wires
  profile/env; per ADR-020 C7 the bootstrap itself stays shell.
- **Payload** — rc files (`.bashrc/.zshrc/.profile/.inputrc/.zsh/`), `powershell/profile.ps1`,
  `ai/` per-agent overlays, `harness/` compiled skill/agent records, `secrets/registry.yaml` +
  `sensitive/*.age`, `ssh/config`, `systemd/` units, `git-hooks/`.

**Real execution paths (traced):**
1. **Bootstrap** — `setup-linux.sh`: source `versions.conf` → copy repo→`~/.dotfiles` →
   `deploy_file` rc files → install tools to `~/.local/bin` → `install-dotf.sh` → `dotf secrets show`
   deploy-time key → per-agent config (agy/claude/opencode/pi/copilot) → MCP register (with the
   `.claude.json` truncation guard) → plugins → auto-memory symlinks → `compile-harness --refresh`
   (vault→committed blocks) then `--deploy` (records→`$HOME`) → cron → `check_deployed` → `dotf env
   generate` → `dotf doctor`. `setup-windows.ps1` mirrors most of this with junctions + Scheduled
   Tasks, but **omits GUARD-001 git-hooks, `--refresh`/`--check`, and the harness `agents` stage.**
2. **Path resolution (ADR-025)** — `env-contract.json` defaults ⊕ `~/.config/dotfiles/machine.json`
   overrides → `dotf env generate` renders `~/.dotfiles/paths.{sh,ps1}` → rc profiles source it.
   The Go `env` package is the one true resolver; rc files carry a bootstrap fallback subset.
3. **Session-start hook** — Claude's `SessionStart` → `dotf mem session-start` → agnostic brief
   (vault detect, `vault-health.sh`, spec counts, lessons) + Claude injectors (doctor-drift,
   hive-project, auto-memory `memlink`, knowledge/memory-temperature, `.claude.json` canary) →
   `additionalContext` JSON envelope. `session-end` archives the `## Session Handoff` block.
4. **Secrets (ADR-028)** — `secrets/registry.yaml` (var→age|bw source SSOT) → `dotf secrets
   {run,show,render,set,migrate,sync,backup}`. `render` bakes `{env:VAR}` into opencode/pi configs
   at deploy; `run` injects into a child env fail-fast; age is the floor, Bitwarden the migration target.
5. **CI** — `ci.yml` (shell lint, PSSA-on-4-files, bats, PR-only Windows e2e, Docker integration),
   `cli.yml` (Go build/test/lint/goreleaser, **path-filtered to `cli/**`**), `spec-gate.yml`,
   `release-please.yml`, `pat-expiry.yml`, board automation.

**Key invariants (where actually enforced):**
- *One var → one secret source* — enforced in Go at parse (`registry.go:validate`), **not** in the
  parallel `env-mapping.conf` (which no longer has code consumers — C35).
- *Deploy = copy, never symlink; drift is asserted* — enforced by `dotf doctor` (`checks_deploy.go`)
  for a fixed allowlist that **omits `harness/`, `AGENTS.md`, `ai/`** (C48).
- *Memory sinks only in the vault* — enforced by the `git-hooks/` pre-commit guard **on Linux only**
  (never installed by `setup-windows.ps1`).
- *Claude project-key encoding* (`/ \ :` → `-`) — correct in Go `memlink` (self-heals), **wrong** in
  the `setup-windows.ps1` inline junction loop (C16) and `knowledge-crystallize.ps1` (C11).

---

## 3. Findings by hunt category (severity order within each)

### 3.1 Correctness — logic errors, swallowed failures, wrong propagation

**C1 — `vault-health.sh` crashes on a healthy vault (`grep -c || echo 0`). CONFIRMED.**
`scripts/vault-health.sh:132` `ORPHAN_COUNT=$(echo "$ORPHAN_OUTPUT" | grep -c '[^[:space:]]' 2>/dev/null || echo "0")` (and `:135,:175,:201,:225`). `grep -c` on no match prints `0` **and** exits 1, so `|| echo "0"` appends a *second* `0` → the variable holds `"0\n0"`. At `:138` `ORPHAN_PCT=$((ORPHAN_COUNT * 100 / TOTAL_FILES))` that is an arithmetic syntax error; `set -euo pipefail` (`:10`) then aborts the whole report. A vault with 0 orphans / dead-ends / unresolved-links — i.e. a *perfect* vault — exits FAILED.
*Scenario:* clean vault → `dotf mem session-start` runs `vault-health.sh` → crash → the session brief shows a garbage "Results: 0 0" / FAIL instead of "ALL CHECKS PASSED"; the Sunday cron logs a truncated error (masked by `|| true` at `vault-maintenance-weekly.sh:26`). The correct idiom is documented one file over (`knowledge-crystallize.sh:115-117`: "use `|| count=0` … NOT `|| echo 0`"). *Fix:* `|| count=0` (assignment, not echo) at all five sites.

**C2 — All bash aliases are dead (path mismatch). CONFIRMED.**
`setup-linux.sh:148` writes the generated alias file to `$HOME/.bash/bash_aliases`, but the deployed repo `.bashrc:129-130` sources `~/.bash_aliases` (a different location). The only `.bashrc` that sources the correct path is the *fallback heredoc* at `setup-linux.sh:162-163`, used solely when no repo `.bashrc` exists — which never happens. So in bash the entire alias payload (`gs/gd/gl/gp/k/kc/dch/tx/oclog/qq/qf…`) silently never loads; zsh is unaffected. The final check (`setup-linux.sh:1430`) only asserts the file *exists*, so it always passes. *Compounding:* had it loaded, `alias qq='noglob _qq_call …'` (zsh `noglob`) would error in bash and shadow the bash `qq()` function. *Fix:* point `.bashrc` at `~/.bash/bash_aliases`, or write the file to `~/.bash_aliases`.

**C3 — SDD spec-gate fails open. CONFIRMED.**
`scripts/check-spec-gate.sh:181` `done < <(git diff --numstat "${BASE_REF}...${HEAD_REF}" 2>/dev/null || true)`. If `BASE_REF` doesn't resolve (shallow clone, typo, detached worktree), `git diff` errors, stderr is discarded, `|| true` swallows the exit → the loop reads nothing → `TOTAL_LOC=0` → `:202` prints "Production diff 0 LOC < threshold" and exits 0. The Tier-4 discipline gate — the mechanism that supposedly forces a spec on every ≥50-LOC PR — silently passes wherever the base ref is unavailable. CI (`spec-gate.yml:26`) fetches the ref first so it's usually fine there, but the opt-in local pre-push hook (`.pre-commit-config.yaml:32`) and any manual run inherit the fail-open. *Fix:* a `git diff` error must be fail-closed (exit 2), not `|| true`.

**C4 — `injectors.*.enabled` is dead config. CONFIRMED.**
`session-start-config.json` declares 11 injector toggles and its `_comment` promises "The Go adapter reads these thresholds + injector flags". `injectorEnabled()` exists at `cli/internal/mem/session_start_config.go:51`, but `ClaudeContext` (`session_start_adapter.go:44-80`) **never calls it** — every injector runs unconditionally (verified: `injectorEnabled` has no non-test caller). Only the 6 `thresholds` keys are honored. *Scenario:* a user sets `"vault_health": {"enabled": false}` to silence a noisy injector; nothing happens. The config file and the SDD-004 spec both describe a contract that stopped being honored at the CLI-025 Go port. *Fix:* gate each injector on `cfg.injectorEnabled(name)`, or delete the block and fix the comment.

**C5 — Windows weekly maintenance task has never worked. CONFIRMED.**
`scripts/vault-maintenance-weekly.ps1:23` `& "$ScriptDir\knowledge-crystallize.ps1" --all`. `knowledge-crystallize.ps1` declares `[switch]$All` (bound as `-All`); the literal string `--all` binds positionally to `$ProjectDir`, `$All` stays `$false`, and `Resolve-Path '--all'` throws under `$ErrorActionPreference='Stop'` → caught by the weekly script → every Sunday the task (`DotfilesVaultMaintenance`, `setup-windows.ps1:1899`) logs "Cannot find path '--all'" and processes zero projects. The bash twin works. *Fix:* pass `-All`.

**C7 — Secrets sync ping-pongs forever. CONFIRMED (logic traced).**
`scripts/dotfiles-sync.sh:76` `cat "$local_file" > "${repo_file}.tmp" && mv "${repo_file}.tmp" "$repo_file"`. `cat >` stamps the destination with *now*. Run N: local newer → copy to repo (repo mtime=now). Run N+1: repo now "newer" → copy back (local mtime=now). Every run flips direction and reports "1 synced"; "All secrets already in sync" (`:97`) is unreachable. Worse, *newest-wins* is corrupted — a genuinely newer edit can be overwritten by a stale file whose mtime the previous sync refreshed. The PowerShell twin uses `Copy-Item` (preserves `LastWriteTime`) and converges. *Fix:* `cp -p` / `touch -r`.

**C11 — `knowledge-crystallize.ps1` single-project mode always misses on Windows. CONFIRMED.**
`scripts/knowledge-crystallize.ps1:57` `return $Path.Replace('\','-').Replace(':','')` yields `C-Users-…` (colon deleted), but Claude's real key is `C--Users-…` (colon→dash). `Get-MemoryFilePath` builds a nonexistent path → the script warns "No MEMORY.md found" and exits 0 for every project on Windows. `-All` mode survives only because a decode heuristic tolerates the doubled backslash. Same encoding bug class as C16.

**C13 — `dotfiles-sync.ps1` dirty-check is dead. CONFIRMED.**
`scripts/dotfiles-sync.ps1:133-135` runs `git diff --quiet` then `git diff --cached --quiet` then tests `if ($LASTEXITCODE -ne 0)`. `$LASTEXITCODE` reflects only the *second* command, so an unstaged-only dirty repo passes the guard and syncs over dirty state. The bash twin correctly requires both (`dotfiles-sync.sh:104`).

**C40 — `OPENCODE_VERSION` unguarded under `set -u`. CONFIRMED.**
`setup-linux.sh:11` sets `set -euo pipefail`; `:31` sources `versions.conf` only `[ -f ... ] &&`; `:654/666/674` reference `$OPENCODE_VERSION` bare. A checkout missing `versions.conf` (partial/corrupt clone) → unbound-variable abort, contradicting the adjacent comment ("OPENCODE_VERSION is empty — fall back to latest"). Sibling vars are guarded (`${YARN_VERSION:-}`, `${PI_VERSION:+…}`, `${SHELLCHECK_VERSION:-0.11.0}`); this one isn't. *Fix:* `${OPENCODE_VERSION:-}`.

**C44 — `render` can corrupt a JSON config. PLAUSIBLE.**
`cli/internal/secrets/render.go:90` `strings.ReplaceAll(out, "{env:"+name+"}", value)` substitutes the decrypted secret verbatim into `opencode.jsonc`/`models.json`. `stripNewlines` removes CR/LF but not `"` or `\`; such a value would produce invalid JSON. API keys are almost always `[A-Za-z0-9._-]`, so low-probability, but nothing enforces it. *Fix:* JSON-encode the value for JSON targets, or validate the rendered file.

### 3.2 Alternative / unintended paths

**C36 — Vault-dependent steps run before the path cascade loads. CONFIRMED.**
`setup-linux.sh:411` and `:508` read `${VAULT_PATH:-$HOME/Projects/knowledge}` to render agy's MCP config and gate the harness `--refresh`, but `dotf env generate` + `. paths.sh` don't run until `:1446-1449`. On a machine whose `machine.json` relocates the vault (this machine: `~/Projects/Workspace/knowledge`), a *first* run from a shell that never sourced a prior `paths.sh` silently takes "Vault absent; deploying committed blocks" and bakes a wrong `VAULT_PATH` into agy's config; it self-heals only on a later run after the profile is re-sourced. The engine repeats the stale literal (`compile-harness.sh:72`).

**C16 — Windows auto-memory junction built at the wrong key. CONFIRMED.**
`setup-windows.ps1:867` `$encodedPath = $cwdPath.Replace('\','-').Replace(':','')` produces `C-Users-…` (colon deleted). Claude Code and the Go `memlink.ClaudeProjectKey` (`:106`, `NewReplacer("/","-","\\","-",":","-")`) produce `C--Users-…` — confirmed against this machine's `~/.claude/projects/`, which contains *both* keys. setup creates a junction Claude never reads, prints a misleading "Linked auto-memory", and the real link is created later by `dotf mem session-start` (correct key). This is exactly the #551/HARNESS-040 encoding bug the Go layer fixed — the setup twin was not updated. Non-fatal (Go self-heals) but deposits junk junctions.

**C22 — Marker injection has no "exactly one marker" guard. CONFIRMED.**
`scripts/compile-harness.sh:606-617` (`inject_agent_presence`) guards only with `grep -q BEGIN && grep -qF END`. If the END marker is lost (merge/manual edit), the next run appends a second BEGIN; the run after that takes the replace branch whose awk fires on *both* BEGIN lines, emitting the block twice and discarding everything between the orphan and appended markers. The patterns region avoids this via `validate_markers` (exactly 1/1); presence never calls an equivalent. The Windows catalog injector (`setup-windows.ps1:1856`) has the identical flaw and *silently truncates* `~/.copilot/copilot-instructions.md` to head+bullets when the END marker is missing.

**C39 — `.zshrc` errors on a fresh box. CONFIRMED.**
`.zshrc:13` `source $ZSH/oh-my-zsh.sh` — unguarded, and no repo step installs oh-my-zsh. setup deploys `.zshrc` unconditionally, so a clean machine prints an error on every zsh start until OMZ is installed by hand.

### 3.3 Incoherences — names/config that lie, duplicated SSOTs, twin drift

**C6 — `SCRIPTS_DIR` self-contradiction (Windows). CONFIRMED.** `env-contract.json:40` declares the Windows default `$env:USERPROFILE\.dotfiles\scripts`, but the same file's `required_path_entries.windows` (`:129`) puts `$env:USERPROFILE\scripts` on PATH, and `setup-windows.ps1:79` deploys to `~\scripts`. Two divergent script trees result: on this machine `~/.dotfiles/scripts` holds only `load-secrets.ps1` while `~/scripts` holds the real scripts *plus* retired orphans (`claude-session-start.ps1`, `doctor.ps1`, `healthcheck.ps1`, `claude-mem-heal.ps1`, `diff-check.ps1`) still on PATH. The doctor pre-export (`setup-windows.ps1:1938`) uses a third value. `REFACTOR-007` has tracked this undecided since 2026-05-27.

**C14 — `dotfiles-sync` twins implement two deploy models. CONFIRMED.** `.sh:118` `rsync -a --delete …` (ADR-005: `~/.dotfiles` is a copy); `.ps1:148` `git -C $DotfilesLocal pull` (treats it as a clone). On a copy-deployed Windows box the pull fails, `Write-Err "Pull failed"` fires, then execution continues to `Write-Ok "Sync complete."` and exits 0 — architectural drift *and* a lost failure signal.

**C17 / C18 — versions.conf lies about being a manifest. CONFIRMED.** `OBSIDIAN_VERSION`, `EZA_VERSION`, `ZOXIDE_VERSION` (`versions.conf:16-18`) have zero consumers repo-wide; the header claims "the Windows CI downloads" eza/zoxide but `.github/` contains no such download. Meanwhile `setup-linux.sh:214` installs eza from `releases/latest/download/…` — unpinned, ignoring the pin and reusing the `latest`-URL rot pattern the repo's own lesson (and #648) condemns.

**C21 — `pi` matches `copilot` (substring). CONFIRMED.** `compile-harness.sh:208` `case "$line" in *"$agent"*)`. `"pi"` ⊂ `"copilot"`, so a `targets:[copilot]` record deploys to `~/.pi/agent/skills/`. Latent today (every copilot record also names pi), but the Windows twin uses a `\b` word boundary (`setup-windows.ps1:1749`) — so the first copilot-only skill makes Linux and Windows deploy different skill sets from identical records.

**C30 / C33 / C35 / C37 / C47 — stale SSOTs & dead references.** env-contract `_comment`s cite deleted `doctor.{sh,ps1}`/`healthcheck.*` (C30, also `cli/README.md:43`, `session-start-config.json:14`, `ai/opencode/opencode.jsonc:38`). The PR template still demands the ADR-018-abolished vault `11-tasks.md` (C33). `sensitive/env-mapping.conf` is tracked again with no code consumers, duplicating the registry (C35, = the #669 split-brain). Six committed overlays hardcode `~/Projects/Workspace/…` as the fallback path while the contract default is `~/Projects/…` (C37). Model facts (`deepseek` "default in opencode" vs `opencode.jsonc:34` `nan/qwen3.6`; ctx "1M"/"500K"/"262144"/"256000") are duplicated across six files and already disagree (C47). Also: `ai/claude/settings.json` enables `feature-dev`/output-style plugins never in either install loop (silent no-ops), and references a nonexistent `Skill(bender-config)`.

**C12 — commit-msg hook rejects the repo's own convention. CONFIRMED.** `.github/hooks/validate-commit-msg.sh:5` `grep -Eq '^[a-z]+: .+'` rejects scoped Conventional Commits — `feat(cli): x` fails on the parenthesized scope, yet the repo's live history is scoped (`feat(cli):`, `chore(spec):`, `test(cli):`) and release-please consumes scopes. Wired via `.pre-commit-config.yaml` (commit-msg stage) + `install-precommit.sh:48`. Any machine that ran `install-precommit.sh` is blocked from the standard scoped message (forced to `--no-verify` or unscoped). `tests/validate-commit-msg.bats:21` tests only the unscoped `feat:` case, hiding the drift. *Fix:* `^[a-z]+(\([a-z0-9-]+\))?: .+`.

**C19 — `check-backlog-merged.sh` is structurally inert on this machine's layout. CONFIRMED.** `scripts/check-backlog-merged.sh:86` infers `REPO="$HOME/Projects/$proj"`, but repos here live under `~/Projects/Workspace/`, so `[ ! -d "$REPO/specs/archive" ]` always trips → `SKIP … exit 0`. `vault-health.sh:265` invokes it *without* `--repo`, so the stale-merged advisory never fires while reporting a clean pass — the silent scope-drop the fix-or-ticket rule bans. (ADR-025 seam violation, #463 class.)

**C15 / C29 — bash vs zsh PATH & nvm drift. CONFIRMED.** `.bashrc:105` hard-resets `PATH` to system dirs; `.zshrc:83` has that line commented out. Any nested interactive bash (tmux pane, IDE terminal) wipes inherited PATH (nvm node, `/snap/bin`, `~/bin`). nvm loads only in zsh; `.bashrc` has no nvm block, and `setup-linux.sh:174` re-deploys `.bashrc` each run, deleting installer-appended init lines — while the comment at `:1426` calls that drift "expected". Node-via-nvm therefore exists in zsh but not bash, and intermittently vanishes from bash after a setup.

### 3.4 Affordance mismatches — the API shape promises what the code doesn't deliver

- **C4 (again):** the config shape (`injectors: { X: { enabled: false } }`) *invites* toggling injectors; the code ignores it. The most discoverable knob is a no-op.
- **C10 / README §Secrets:** README teaches `secrets_add VAR file`, `secrets_rotate`, `secrets_show`, `secrets_list`, `secrets_check`, `secrets_add_file` and `dotfiles-sync [--secrets-only]` as commands. None resolve (grep across `.zsh/`, `.bashrc`, `scripts/*.sh` = empty); ADR-028 replaced them with the `dotf secrets` facade. A newcomer following the README's headline "Key Commands" hits command-not-found. README's own "Human entrypoints" note ("scripts are NOT on PATH") also contradicts the bare `dotfiles-sync`/`vault` invocations elsewhere in the same file.
- **C20:** `dotf doctor` on Windows (if agy is on PATH) reports `AGY_APP_DATA is relative or unset` for a valid `C:\Users\…` path, because `checks_deploy.go:350` tests `HasPrefix(path,"/")` for "absolute". The easy read ("doctor says my agy config is broken") is wrong.

### 3.5 Missing functionality

- **GUARD-001 is Linux-only.** `setup-windows.ps1` never wires `core.hooksPath` / installs `git-hooks/` (grep = zero hits). AGENTS.md's "MEMORY SINGLE-SINK" claims a *machine-wide* pre-commit guard; on Windows the guard does not exist. `dotf doctor`'s `checkGuardHooks` still runs on Windows and FAILs "core.hooksPath unset" with a `--fix` that points at a `git-hooks/` dir setup never deployed.
- **C23 — harness `agents` stage un-ported to Windows.** No `~/.claude/agents/curator.md`, no AGENT-PRESENCE region on Windows; `dotf doctor` flags none of it.
- **C48 — no prune + no drift coverage for the harness mirror.** `setup-linux.sh:543` copies `harness/.`→`~/.dotfiles/harness/` without `--delete`, and `isManagedDeployPath` (`checks_deploy.go:486`) omits `harness/`, `AGENTS.md`, `ai/` despite its own comment claiming it mirrors the copy block. A skill deleted from the repo lives on as a zombie record that doctor validates and calls "no drift".
- **C28 — Linux cron never self-heals a moved repo** (name-keyed grep); the Windows task does (arg-diff).
- **DR restore has no end-to-end test** (`dotf secrets backup`/restore — matches open OPS-019).

### 3.6 Boundary & safety

- **C26 — secrets on argv.** `bitacora-rollout.sh:148` `gh secret set BITACORA_PAT --body "$BITACORA_PAT"` and `nan-{bench,debug,quality-bench}.sh` `-H "Authorization: Bearer $NAN_API_KEY"` expose secret material in `/proc/<pid>/cmdline`. `utils.sh:316` already shows the correct stdin idiom.
- **C27 — plaintext at rest.** `age-encrypt-decrypt.sh:73` / `age-standalone.sh:91` write `.dec` files at umask 0644, with no `chmod 600` and no trap-cleanup; a mid-loop failure leaves earlier `.dec` plaintext behind.
- **C8 — auto-merge.** `bitacora-rollout.sh:124` `gh pr merge … --squash --auto`. Against a protected repo with auto-merge enabled, the workflow-deploy PR lands the instant CI is green — the exact human-review bypass every AGENTS.md's "Overrides of Harness Defaults" bans.
- **C46 — infra topology in a public repo.** `ssh/config` commits a VPS public IP (`162.55.57.175`), mesh + LAN IPs, usernames, and a `User root` host (`hermes-nan`). No secrets, but public recon value. **This repo is public** (`gh repo view` → `"visibility":"PUBLIC"`), which also raises the stakes on the deploy-time plaintext OPENROUTER key baked into `~/.gemini/config/mcp_config.json` (by design per agy's no-env-expansion, but two plaintext copies on disk).
- **C42 — `.gitconfig` Linux path on Windows.** `helper = !/usr/bin/gh auth git-credential` deployed verbatim to `%USERPROFILE%\.gitconfig`; `/usr/bin/gh` doesn't exist in Git-for-Windows → HTTPS auth breaks until edited. Portable form: `!gh auth git-credential`.
- **C32 — supply chain.** Every Action is tag-pinned, none SHA-pinned. A compromised moving tag runs with `BITACORA_PAT` (repo+project write) or `RELEASE_TOKEN` (contents+PR write) — full write + release forgery.

### 3.7 Documentation

- **C10 / C43 (README):** retired command surface (C10); "316 BATS tests" (actual ~949 `@test`s), "~50 scripts" (actual 35). The clone URL and step order are fine; the drift is in the command tables.
- **C9 (tests):** `tests/antigravity.bats:17` `skip`s the whole file when agy is absent, and no CI job installs agy, so all 15 tests — including ~8 pure-static regression guards ("--directory must not return", "no hardcoded /home/<user>") that need no agy — never execute. A PR reintroducing `"--directory","/home/manu/Projects/hive"` into `ai/agy/mcp_servers.json` passes CI green while breaking Windows agy exactly as the original bug. The test names claim coverage CI does not provide.
- **Stale ADR/spec pointers:** many shipped specs reference deleted `healthcheck.{sh,ps1}`/`doctor.{sh,ps1}` sections (#520/#533); `docs/adr/dotfiles-architecture-map.md` still lists `env-mapping.conf` + `load-secrets.{sh,ps1}` among the SSOTs. `tmux.conf:2` says "symlink" but deploy is a copy and doctor *fails* on a symlink — a comment that inverts the enforced invariant.

### 3.8 Developer experience

- **C24 — PSSA covers 4 of ~12 `.ps1`.** `utils.ps1` (the shared `Set-StrictMode` library), `dotfiles-sync.ps1`, `install-dotf.ps1`, `vault-maintenance-weekly.ps1`, `windows/*.ps1`, `paths.ps1` get no analyzer anywhere. The vault's "ps1 must be ASCII / PSSA fails CI on non-ASCII" rule is thus unenforced — em dashes already sit in `setup-windows.ps1`/`install-dotf.ps1` comments — and `.PSScriptAnalyzerSettings.psd1` excludes the BOM rule.
- **C31 — Go config inputs escape Go CI.** `cli.yml` is path-filtered to `cli/**`. A breaking edit to `env-contract.json`, `secrets/registry.yaml`, `session-start-config.json`, `harness/manifest.json`, or `versions.conf` — all parsed by Go — does not trigger `go test`; only a post-merge runtime failure surfaces it.
- **C38 — CRLF-in-worktree.** 12 tracked `*.sh`/`.gitconfig` files show `w/crlf` under `git ls-files --eol` despite `eol=lf`; they fail under bash on the `\r` while `git status` stays clean (the `Set-Content`-CRLF hazard the project memory records). A `dotf doctor` guard would catch it (incident→guard rule).
- **C41 / C25 — tests that can't fail & gate bypasses.** `verify-setup.bats:308` body `true`; `utils.bats:243/258` whitespace-eating `$(...)`; `shell-profile.bats:29` title/body mismatch; `dotfiles-sync.bats:20/33` OR-form asserts; `setup-windows.bats:109` grep that passes even if the source flips to fatal. spec-gate's three bypass routes (`check-spec-gate.sh:98` `*generated*` glob, `:139` `dependencies` label with no actor/rationale check, `:207` any-active-spec-touch — trivially satisfied given the 35 unarchived specs). These erode trust that green means correct.
- **Positive DX:** the Go smoke tests, the deterministic-age Windows CI (BUG-025), the `.claude.json` truncation guard, the fixture-driven `compile-harness`/`check-spec-gate`/`guard-memory-sink` suites, and `pat-expiry.yml`'s PR-capability probe are genuinely falsifiable and well-built.

---

## 4. Design tensions (the approach, not a line)

**T1 — The strangler-fig is stuck in the middle, so every feature exists 2–3 times and the copies drift.**
ADR-020 says port a twin to Go "on contact" and delete the pair. In practice the repo is a permanent
three-body problem: `dotfiles-sync.{sh,ps1}` coexist with `dotf update` (#667 shipped, twins still deployed
and README-documented); `age-encrypt-decrypt.sh` + `backup-secrets-to-usb.sh` coexist with `dotf secrets
backup`; `compile-harness.sh` is the engine while `dotf harness` (CLI-026) is designed-not-built. Where the
Go side is authoritative and the shell side lingers, they *diverge silently* — C7/C13/C14 (sync), C11/C16
(encoding), C21 (targets matching), C5 (param binding). **The alternative to weigh:** make "on contact"
enforceable — a CI guard that fails when a `dotf` subcommand and its named `.sh`/`.ps1` twin both exist, so
the migration can't stall in the drift-prone middle. The Go layer is good enough that finishing the port is
lower-risk than maintaining the twins.

**T2 — Two setup scripts, one contract, and Windows is a second-class re-implementation.**
`setup-windows.ps1` is not a port of `setup-linux.sh`; it's a parallel implementation that reproduces some
stages, silently omits others (GUARD-001 hooks, harness `--refresh`/`--check`, the whole `agents`/presence
stage — C23), and encodes things differently (junction key C16, catalog dash D2, absolute-path test C20).
There is no executable contract that says "these two must deploy the same end-state," so drift is invisible
until a Windows box misbehaves. **The alternative:** push more of the deploy *logic* into `dotf` (which is
already cross-compiled and CI-tested on both OSes) and shrink both setup scripts toward the "detect-OS,
fetch-binary, wire-PATH" bootstrap that ADR-020 C7 actually reserves for shell. Every stage that lives in
Go instead of twinned shell is a class of Linux/Windows divergence that stops being possible.

**T3 — `set -euo pipefail` + a house style of `|| true` / `2>/dev/null` = failures that vanish.**
The shell layer pairs strict mode with an idiom that defeats it: `|| true`, `|| echo 0`, `2>/dev/null`,
`>/dev/null 2>&1`. The result is the worst of both — `set -e` aborts on the *unexpected* (C1, C40) while the
deliberate guards swallow the *expected* failures that should be surfaced (C3 gate fail-open, C14 sync "OK"
after a failed pull, A7/A8 harness refresh/deploy errors sent to `/dev/null`). The `grep -c || echo 0`
foot-gun (C1) is documented as *wrong* in one script and copied into another. **The alternative:** a small
sourced helper library of safe idioms (`count_lines`, `try_or_warn`, atomic-write) plus a shellcheck/CI lint
that flags `|| echo` after `grep -c` and bare `2>/dev/null` on state-changing commands. The Go layer already
models the discipline (fail-loud on empty secrets, atomic writes, typed error classification) — the shell
layer should inherit it, not reinvent silent-failure per script.

**T4 — Config files present a contract the code doesn't honor; the SSOT boundary is porous.**
`session-start-config.json`'s injector toggles are dead (C4); `versions.conf`'s eza/zoxide/obsidian pins are
dead (C17); `env-mapping.conf` is a second secrets SSOT with no readers (C35); `env-contract.json` contradicts
itself on Windows `SCRIPTS_DIR` (C6) and describes consumers that were deleted (C30). Each is a file that
*looks* load-bearing and isn't. The deeper issue: several of these are inputs the Go CLI parses, yet they live
outside `cli/` so a breaking edit never runs `go test` (C31). **The alternative:** treat each declarative
config as code with a test — a schema/consumer test that fails when a declared key has no reader (or a reader
references an undeclared key), and widen `cli.yml`'s path filter to the files Go actually parses. A config that
can't drift from its consumers is worth more than one that documents good intentions.

**T5 — The knowledge/provenance machinery is heavier than the thing it governs, and its own guards have gaps.**
Between `harness/` (compile → refresh → check → deploy across four agents), the marker-injection SSOT dance,
the spec lifecycle (57 folders, 35 shipped-but-unarchived), and the session-brief injectors, a large fraction
of the repo exists to keep *other* text in sync. Yet the guards protecting that machinery have holes: marker
injection can duplicate/truncate on a lost marker (C22), the deploy mirror never prunes and isn't drift-checked
(C48), spec IDs can collide with no CI guard (C34), and the spec-gate that enforces the whole SDD ritual has
three bypasses and a fail-open (C3/C25). **The alternative to weigh:** is the marginal governance surface
earning its keep? A smaller, ruthlessly-guarded core (one injection primitive with an exactly-one-marker
invariant, an archive-on-merge CI gate, a spec-ID uniqueness check) would likely deliver more actual
consistency than the current broad-but-porous system.

---

## 5. Expectation gaps (expected X, found Y)

| I expected… | I found… | Ref |
|---|---|---|
| `dotf secrets` docs, since ADR-028 | README still teaches `secrets_add/rotate/show/list/check` (none exist) | C10 |
| `dotfiles-sync` to be a command | It's `scripts/dotfiles-sync.sh`, not on PATH, not aliased; README implies a bare command | C10 |
| Setting `"vault_health": {enabled:false}` to silence it | The flag is never read; the injector always runs | C4 |
| A healthy vault → "ALL CHECKS PASSED" | A healthy vault → arithmetic crash → FAILED | C1 |
| bash aliases to work after setup | They never load (setup writes one path, `.bashrc` sources another) | C2 |
| The spec-gate to block a big PR with no spec | It passes if the base ref doesn't resolve, or a stale spec is touched, or a `dependencies` label is added | C3, C25 |
| The memory-sink guard to protect every repo "machine-wide" | It's never installed on Windows | §3.5 |
| `dotf doctor` green on a correctly-configured Windows agy | `AGY_APP_DATA is relative or unset` FAIL for a valid `C:\…` path | C20 |
| versions.conf to be the version SSOT | eza/zoxide/obsidian pins have no consumers; eza installs from `latest/` | C17, C18 |
| Editing a `.ps1` library to be lint-gated | `utils.ps1` (and 7 others) are analyzed by nothing in CI | C24 |
| A newcomer's README "Key Commands" to run | Several are retired-command or not-on-PATH | C10, C43 |
| The PR template to match ADR-018 | It still requires a vault `11-tasks.md` entry that was abolished | C33 |
| One deploy model for `~/.dotfiles` | `.sh` rsync-copies, `.ps1` `git pull`s the same dir | C14 |

---

## 6. Open questions (code alone can't resolve — maintainer answers)

1. **`env-mapping.conf` (C35):** it was deleted by #601 then re-added by #659 with no code consumers. Is any
   external tool (a script outside this repo, Hermes, a machine not audited here) still reading it, or is this a
   clean delete (the #669/GUARD-002 outcome)?
2. **`SCRIPTS_DIR` on Windows (C6 / REFACTOR-007):** is the intended deploy target `~\scripts` or
   `~\.dotfiles\scripts`? The contract, PATH entry, and setup disagree; the fix is one decision + a small diff,
   but it's a taste/ownership call the audit can't make.
3. **`dch` semantics (C43):** README's Diagnostics section describes `dch` as "repo vs `~/.dotfiles` drift" but
   the alias is now `dotf doctor` (the full sweep). Intended, or should `dch` map to a drift-only subcommand?
4. **agy MCP forks (F3):** agy still uses `uvx hive-vault` (per-session) while claude/opencode moved to the
   `hive client` daemon, and agy gets no `context7` — contradicting `mcp-servers.json:3`'s "keep hive+context7+
   sequential-thinking across all environments". Deliberate (agy limitation) or drift?
5. **Spec archive policy (C34):** 35 shipped specs sit unarchived with several `status:` fields that lie
   (`draft`/`implementing` on merged work). Is the plan a one-time #670 sweep + an archive-on-merge CI gate, or
   is "active folder = in-tree regardless of status" an accepted convention?
6. **Public-repo posture (C46, deploy-time plaintext key):** given the repo is public, is committing the full
   `ssh/config` host inventory (public VPS IP, usernames, a root-login host) an accepted trade-off, or should the
   inventory move to `sensitive/`?
7. **Windows parity scope (T2, C23, GUARD-001):** is Windows intended to reach full parity (memory-sink guard,
   harness agents stage), or is it a deliberately reduced surface? That decision changes whether the omissions
   are bugs or by-design.

---

*Audit method: adversarial read-first (state expected behavior, then confirm/refute against source), with
every finding tied to a concrete inputs→wrong-result scenario and marked CONFIRMED/PLAUSIBLE. Findings that
did not survive scrutiny were discarded (e.g. an initial "the Go CLI has no CI" hypothesis — refuted by
`cli.yml`; "the pre-commit dispatcher orphans repo-local hooks" — refuted by the `chain-local-hook.sh` exec).
Left uncommitted for the maintainer to triage; each finding carries a stable ID (C1…C48) a fixing agent can cite.*
