---
id: audit-010-process-workflows
type: audit
status: active
date: "2026-07-07"
related: [audit-008-codebase-comprehensive, audit-009-documentation]
tags: [audit, process, workflow, dotfiles, end-to-end]
---
# Process Audit — end-to-end workflow walk (2026-07-07)

**Method.** Every documented journey was walked twice — as a first-time human following only the docs, and as an agent chaining exit codes — empirically, in throwaway environments: four Ubuntu 24.04 containers (E1–E4: README-requirements-only newcomer; full README Quick Start; copy-branch + self-deploy precision pass; checkout-dirt identification), the CI integration image built from this working tree, and scrubbed-environment sandbox runs of a locally-built `dotf` on Windows. No repo state was modified beyond this report. Findings marked **CONFIRMED** were reproduced with the command sequences given; **PLAUSIBLE** were traced in source only.

**Scope note.** The audit prompt is a generic template (distributed via the maintainer's gist, run from whatever repo it targets); its Context/Scope sections name subsystems of a different codebase (ingest preview, review sidecars, publication.status, vault manifests, `somostodos-init`). An exhaustive search of this machine (all Workspace repos, `~/Projects`, home, other drives, `gh repo list`) found no codebase containing them; the maintainer directed this run at the dotfiles repo explicitly. Each scoped process was therefore mapped to its real counterpart here; §3 verifies this repo's own recently shipped fixes (#667, #663, #661, #664, HARNESS-040) through full workflows, and the template's named prior-audit defects are N/A (no such subsystems exist to regress).

**Spec coverage matrix** (template scope item → what was walked here → where):

| Spec scope item | Mapping in this codebase | Walked | Report |
|---|---|---|---|
| 1. Onboarding: empty install → health → init → manual config → health green; newcomer stranding; fix_suggestions executable verbatim | `git clone` → `setup-linux.sh`/`setup-windows.ps1` → `dotf doctor` → `machine.json` + `dotf env generate` → doctor green | E1 (README-requirements-only), E2 S1-S5 (Quick Start verbatim), real-machine doctor; every fix_suggestion re-executed | P2, P3, P6, P8; §2.1 |
| 2. Ingest lifecycle: first run → re-run same source → changed inputs → duplicates; durable registry | Deploy lifecycle: first setup → re-run setup → repo-vs-deploy drift → registry/store integrity (`secrets/registry.yaml` + `sensitive/`) | E2 S5 (second call), E4 (what a run leaves behind), real-machine drift FAILs, orphan store file | P1, P2, P13; §2.4 |
| 3. Review loop: state consistency, recovery/reopen path | Spec lifecycle (nearest per-record review flow): init → gate → fill seam → archive → abandoned → reopen path | E2 S10-S12, W1/W1c gate probes; no unarchive command found | P14, §2.3 |
| 4. Filing: apply → re-apply idempotency AND exit code → partial failure → does config control destination | `dotf env generate`/`tools install`/`vault project`: re-run idempotency, exit codes, half-completed prior run (post-abort setup state), machine.json controlling destinations | E2 S16-S17, S19-S20, E3 T5; P2 (half-completed setup), P3 (config not consulted — phantom default) | P3, P9, P10 |
| 5. Publication: can a record reach the terminal state purely via CLI; which command owns the status write | Self-deploy: can a machine reach "updated" purely via `dotf update`; which process owns deploy-state freshness | E3 T3-T4 (full behind→ff→setup walk), E2 S7-S9 | P1, P3; §2.1 |
| 6. Side processes: memory lifecycle, artifact writers vs validators, health/doctor as preflight | `dotf mem session-start/end`, harness/skills compile-deploy seam, doctor as preflight for every flow | E2 S18, doctor before/after each walk, generate-vs-doctor validator disagreement | P4, P6, P8 |
| 7. Cross-process coherence: enumerate every state, command that moves each forward; real state machine; unreachable states + absorbing dead ends | Machine install state, secret state, spec state, paths/config state | §2.1–2.4 with owning commands, 2 absorbing dead ends, 1 unreachable state marked | §2 |
| Hunt-for: dead ends / missing processes / re-run semantics / agent ergonomics (exit codes, JSON, help vs flags) / docs drift / concurrency | All six bullets walked against the real CLI surface (12 nouns, every subcommand help + probes) | E2 S6/S13-S15/S19-S20, E3 T5-T7, W5-W6b, docs walk | P5, P7, P9, P11, P12 |

**Audit-harness disclosure.** One sandbox invocation of `dotf env generate` early in the audit leaked through ambient `DOTFILES_DIR` and rewrote the real `~/.dotfiles/paths.ps1` with sandbox paths; it was restored by re-running `dotf env generate` from a correct environment. The incident is itself evidence for P10 (generate trusts ambient env, no confirmation, no backup) and demonstrated the recovery path works.

**Companion audits:** `audit-008-codebase-comprehensive.md` (code), `audit-009-documentation.md` (docs) — same series; see `related:` frontmatter. Vault decide/position layer: `10_projects/dotfiles/research/2026-07-02-project-coherence-audit.md` (external benchmarking + methodology + backlog).

---

## 1. Summary

| ID | Severity | Process | Issue | Status |
|----|----------|---------|-------|--------|
| P1 | critical | self-deploy | Setup rewrites a tracked file inside the checkout (`.github/copilot-instructions.md`); every subsequent `dotf update` skips on "dirty worktree" with exit 0 — self-deploy disables itself after the first run, silently | CONFIRMED |
| P2 | critical | onboarding (Linux) | README's documented layout (clone into `~/.dotfiles`) kills setup mid-run (`cp` same-file + `set -e`) and self-destructs the git-hooks dispatcher with a false `[SUCCESS]`; second run refuses; doctor's advice is circular | CONFIRMED |
| P3 | high | all `dotf` flows | Contract default `DOTFILES_REPO_DIR=~/Projects/dotfiles` matches no documented install path and no journey creates `machine.json` → `dotf update` is a permanent no-op (exit 0), `mem session-start` probes phantom paths, `doctor --fix` tells users to export the phantom | CONFIRMED |
| P4 | high | diagnostics | `dotf doctor` resolves `env-contract.json`/`versions.conf` deployed-copy-first while `dotf env generate` resolves repo-first → contradictory verdicts on the same machine ("paths.ps1 is stale" vs "ok: up to date") and nonsense version-drift directions | CONFIRMED |
| P5 | high | secrets / AI deploy | Both setups fetch a deploy-time secret as `dotf secrets show openrouter-api-key` — an id that does not exist in the registry — swallowed by `\|\| true` → agy MCP config is baked with an empty API key on every deploy, both OSes | CONFIRMED |
| P6 | high | onboarding → doctor green | Doctor-green is unreachable on a docs-faithful fresh machine: `bw` is npm-sourced with npm undeclared/uninstalled, doctor's fix commands fail identically, one fix references the retired `secrets_refresh`; 31 FAILs right after documented onboarding | CONFIRMED |
| P7 | medium | docs | README promises retired commands (`secrets_add`/`secrets_show`/…, `. scripts/load-secrets.sh`) and three human entrypoints (`vault`, `obs`, `dotfiles-sync`) that exist in no shell; `.zshrc` hard-requires oh-my-zsh that nothing installs or documents; Requirements omit `curl` (E1 dies on it); Windows setup prints retired `project-init` as a next step | CONFIRMED |
| P8 | medium | diagnostics | `doctor --fix` prints "Applied 11 fix action(s)" when it only printed suggestions; the suggestions (hand-added profile exports) permanently shadow `machine.json` via cascade rule #1 — the exact corruption class this audit accidentally demonstrated | CONFIRMED |
| P9 | medium | agent ergonomics | No JSON output anywhere in the CLI; `dotf env path UNKNOWN_KEY` → empty line + exit 0; `spec archive` re-run errors instead of no-op; `secrets set` without bw installed reports misleading "item not found"; `secrets render` is default-lax (exit 0 leaving placeholders — setup depends on it); `migrate` guard recommends `--split`, which does not exist | CONFIRMED |
| P10 | medium | env generation | `dotf env generate` trusts ambient `DOTFILES_DIR`/`HOME` for target and rendering with no confirmation, no atomic write, no backup — a mis-set environment silently rewrites the live paths file | CONFIRMED |
| P11 | medium | cross-platform | `.gitattributes` misses extensionless rc files (`.bashrc`, `.zshrc`, `.profile`, `*.local.example`) → CRLF on Windows checkouts → any Windows→Linux copy deploys a syntactically broken `.bashrc`; the existing bats syntax guard can only run where the bug cannot occur | CONFIRMED |
| P12 | low | concurrency | Paths-file writes are non-atomic (`os.WriteFile`); concurrent `env generate` observed benign but a sourcing shell can read a torn file; `update`'s dirty check counts untracked files, so any stray file blocks self-deploy (compounds P1) | PLAUSIBLE |
| P13 | low | state hygiene | States with no owning command on the real machine: orphan `sensitive/kubelab-dispatch-token.secret.age` (doctor FAILs it; only hand-edit fixes it); deployed-copy staleness accumulates silently when the opt-in timer is off (observed: deployed contract missing #663's AGE keys) | CONFIRMED |
| P14 | low | spec lifecycle | Gate errors conflate "issue not found" with "gh failed"; archive of an already-archived spec says "spec not found" instead of acknowledging the state | CONFIRMED |

Counts: **2 critical, 4 high, 5 medium, 3 low** — 13 CONFIRMED, 1 PLAUSIBLE.

---

## 2. Process map — the real state machines

### 2.1 Machine install state (per machine)

```
NOT_INSTALLED
  └─ git clone (README)
       ├─ layout A: clone INTO ~/.dotfiles  (README-documented)
       │    └─ ./setup-linux.sh → ✖ DIES mid-run (P2) → PARTIALLY_SETUP
       │         └─ re-run setup → git-hooks step refuses (P2) → ABSORBING without
       │            undocumented `git checkout -- git-hooks` knowledge
       └─ layout B: clone elsewhere (~/dotfiles-repo; only CI exercises this)
            └─ setup → exit 0 → DEPLOYED_UNCONFIGURED
DEPLOYED_UNCONFIGURED  (no machine.json; contract defaults point at ~/Projects/dotfiles)
  ├─ dotf update            → no-op "not a git repo", exit 0 (P3)  [silent dead end]
  ├─ dotf doctor            → exit 1 forever (P6)                  [green unreachable]
  └─ MANUAL, UNDOCUMENTED: write machine.json + dotf env generate → CONFIGURED
CONFIGURED
  └─ any setup run from the checkout → checkout dirty (P1) → SELF_DEPLOY_DISABLED
       └─ dotf update → "dirty worktree — skipping", exit 0, forever  [ABSORBING]
            recovery: manual `git checkout -- .github/copilot-instructions.md`
                      (undocumented; re-dirtied by the next setup run)
DRIFTED  (repo moved, deploy dir stale — observed live: contract missing AGE keys)
  ├─ owning command: re-run setup (manual memory only; timer opt-in & off)
  └─ doctor flags file drift, but reads the STALE side as truth for its own
     checks (P4) → contradicts env generate; fix texts do not converge
```

**Dead ends:** PARTIALLY_SETUP after a layout-A run (P2); SELF_DEPLOY_DISABLED (P1). Both are absorbing given only documented commands.
**Unreachable state:** doctor-green on a fresh docs-faithful machine (P6).

### 2.2 Secret state (per registry id)

```
registry-mapped ── backend: age ──► resolvable (age key present) / FAILED (absent)
      │                              verify → exit 1 on FAILED  ✓ (good agent contract)
      │
      ├─ backend: bw ──► requires bw CLI: npm-sourced, no npm on fresh box (P6)
      │                   set/migrate/backup fail; `set` error text misattributes (P9)
      ├─ migrate age→bw: parity-gated ✓; shared-source refusal points at
      │                   nonexistent `migrate --split` (P9/C9)  [dead-end pointer]
      ├─ rotate: declared in registry (`rotate: 90d`) — NO command; manual bw edit
      │                   (runbook admits "manual today")         [missing process]
      └─ orphan file in sensitive/ (no registry entry) — doctor FAILs it;
                          no adopt/remove command                 [dead end, P13]
DR escrow: backup ✓ (round-trip verified; clean failure w/o bw). restore: manual
runbook only (age -d + bw import) — no command                    [missing process]
```

### 2.3 Spec state (per feature id)

```
(no spec) ─ spec init [--issue gate ✓ / --force-no-gate] ─► active
active ─ spec archive ─► archived          (re-archive → "spec not found" error, P14)
active ─ spec archive --abandoned ─► abandoned
archived ─ (no reopen/unarchive command — hand-move only)          [dead end, minor]
Vault promotion + backlog tick: deliberately manual/agent seam (printed by the CLI).
```

### 2.4 Paths/config state

```
contract(repo) ─ commit ─► contract(deployed)   [only via setup; ages silently, P13]
contract + machine.json ─ env generate ─► paths.{sh,ps1}
  ├─ --check drift detection ✓ (exit 1 on drift)
  ├─ doctor's own stale-check uses the OTHER contract copy (P4)   [split verdicts]
  └─ ambient env poisons render/target (P10)
machine.json: created by NO journey (P3) — the load-bearing file onboarding never makes.
```

---

## 3. Prior-fix verification (this repo's recent merges, exercised through full flows)

| Fix | Verdict | Evidence |
|---|---|---|
| #667 `dotf update` self-deploy port | **PARTIAL** | Core loop works: with `DOTFILES_REPO_DIR` set to a clean behind checkout, update fast-forwarded (ccc3189 → 70654ea), re-ran setup, exit 0 (E3 T3). But in every documented journey it is inert: phantom repo default → "not a git repo" no-op (P3), and after one setup run → permanent "dirty worktree" skip (P1). The unit shipped; the process it serves does not close. |
| #663 age root-of-trust (AGE_KEY_PATH/SOPS_AGE_KEY_FILE in contract + doctor) | **PARTIAL / AT-RISK** | Works from the repo: generate renders both keys; doctor's secrets-integrity section ran green (34 checks) in-container. Regression vector live: this machine's deployed contract predates #663 — a `dotf env generate` resolving deployed-first (any run without the repo reachable) renders paths WITHOUT the age keys, silently reintroducing the #518 failure. Observed empirically during the audit-harness incident. |
| #661 `dotf secrets backup` DR escrow | **PASS (surface)** | Command present, help accurate; without bw it fails loud and clean (`backup: bw sync: exec: "bw": not found`, exit 1 — E2 S15). Full escrow round-trip not exercisable in sandboxes (no Bitwarden session); runbook RECOVER remains manual (§5). |
| #664 help-text dangling-doc guard | **PASS-lite** | No dangling doc references observed across the full `--help` walk of all 12 nouns. One adjacent gap: help text and error text still reference non-commands (`secrets_refresh` in doctor SKIP text, `migrate --split`) — the guard covers docs refs, not command refs (P6/P9). |
| HARNESS-040 junction repair (`doctor --fix`) | **PASS (observed)** | Real-machine doctor: auto-memory junction checks green among 96 passes, read-only run. Creation path traced (`checks_automemory.go` + unit tests); not re-created empirically on Windows in this audit. |
| Prompt-template prior findings (ingest/review/publication/vault-manifest) | **N/A** | No such subsystems exist in this codebase. |

---

## 4. Gaps and errors by process (severity order)

### P1 — Setup dirties the checkout; self-deploy permanently self-disables (critical, CONFIRMED)

**Where:** `setup-linux.sh:918-935` (SDD-005 parity sync writes `$CURRENT_DIR/.github/copilot-instructions.md`); `cli/internal/update/update.go:51` (dirty = any `git status --porcelain` output, tracked or untracked).

**Repro (container, clean GitHub clone):**
```
git clone --depth 5 https://github.com/mlorentedev/dotfiles.git ~/dotfiles-repo
cd ~/dotfiles-repo && bash setup-linux.sh        # exit 0
git status --porcelain                            # " M .github/copilot-instructions.md"
DOTFILES_REPO_DIR=$HOME/dotfiles-repo dotf update # "dirty worktree — skipping", exit 0
```
E3 showed the full arc: reset-behind → update works ONCE (its own setup re-run re-dirties) → every later update skips. A machine with the opt-in timer enabled reports green (exit 0) daily while never updating again.

**Stranded user:** enables the self-deploy timer per `guide-self-deploy-timer.md`, sees `systemctl status` green forever, machine quietly ages. Nothing in doctor points at the checkout's dirt as the cause.

**Also implied:** the committed `.github/copilot-instructions.md` is out of parity with `ai/copilot/copilot-instructions.md` in HEAD right now (the sync produced a 2+/4− diff on a fresh clone) and no CI check enforces the parity rule.

**Direction:** setup must never write into the checkout (deploy-dir writes only), or the parity sync must move to CI/pre-commit; `dotf update` could additionally diagnose *which* files dirty the tree and say so.

### P2 — README's Linux Quick Start layout kills setup and destroys the git-hooks dispatcher (critical, CONFIRMED)

**Where:** README.md:16-19 documents `git clone … ~/.dotfiles && cd ~/.dotfiles && ./setup-linux.sh`; `setup-linux.sh:11` (`set -euo pipefail`); the env-contract deploy `cp` (~line 1437) errors "same file" when `CURRENT_DIR == DOTFILES_DIR` and set -e aborts the run there — everything after (env generate, final doctor) never runs. `scripts/install-git-hooks.sh:60-63`: `rm -rf dest` + unchecked `cp src/. dest` with src==dest empties the dispatcher and still logs `[SUCCESS]` (E3 T8); the second setup run then refuses (`has no pre-commit dispatcher`), and doctor's fix text ("run dotfiles setup to deploy git-hooks/") is circular.

**Repro:** E2 S1 (SETUP1_EXIT=1, log ends at the same-file cp), E2 S5 (second-run git-hooks refusal), E3 T8 (targeted same-dir destruction, `DEPLOY_SAMEDIR_2ND_EXIT=1`).

**Stranded user:** follows the four documented commands; ends with exit 1, no `paths.sh`, no final doctor, an inert GUARD-001, and a re-run that fails differently. Recovery (`git checkout -- git-hooks`) is documented nowhere.

**Note:** CI's integration image deliberately clones to `~/dotfiles-repo` "to trigger the copy branch" (tests/Dockerfile.integration:48) — the CI-tested layout and the README-taught layout are disjoint; E1 additionally dies earlier (no `curl`, `setup-linux.sh:238` assignment under set -e) because README Requirements omit curl.

**Direction:** either support repo==`~/.dotfiles` (guard every same-file copy; make install-git-hooks a no-op when src==dest) and add a CI variant for it, or change the README to the copy layout and have setup hard-refuse the in-place layout with a clear message.

### P3 — The phantom repo default: no journey creates `machine.json` (high, CONFIRMED)

**Where:** `env-contract.json` default `DOTFILES_REPO_DIR` = `~/Projects/dotfiles` (Linux) — matching neither README location; `setup-linux.sh:1473` exports the same phantom fallback; README.md:117-130 explains machine.json only as a *relocation* tool, not an onboarding step.

**Repro (E2, right after documented onboarding):** `dotf update` → `not a git repo: /home/u/Projects/dotfiles — nothing to self-update` exit 0 (S7-S9); `dotf mem session-start` → `vault-health.sh not found at /home/u/Projects/dotfiles/... — run dotfiles setup to install` (S18 — setup HAD run); `doctor --fix` → `export DOTFILES_REPO_DIR="/home/u/Projects/dotfiles"` (S4 — advice to export a path that doesn't exist).

**Stranded user/agent:** every repo-anchored flow (update, registry writes, session brief) silently targets a nonexistent directory; exit codes say success.

**Direction:** first-run setup should write `machine.json` with `DOTFILES_REPO_DIR=$CURRENT_DIR` (it knows the answer) and run generate; alternatively `update`/`mem` should fall back to the cwd walk-up resolver (`env.RepoDir()`) the secrets/spec commands already use — today two different "where is the repo" answers coexist.

### P4 — doctor and env generate read opposite sources of truth (high, CONFIRMED)

**Where:** `cli/internal/doctor/config.go:34-43` (deployed-first, repo fallback) vs `cli/internal/env/env.go:116-138` (repo-first via DOTFILES_REPO_DIR, deployed fallback).

**Repro (real machine, same shell, same binary):** `dotf doctor` → `[FAIL] paths.ps1 is stale — run dotf env generate`; `dotf env generate --check` → `ok: … up to date`, exit 0. Doctor also reported `dotf version drift: installed=0.28.0 pinned=0.23.0 (run ./scripts/install-dotf.sh)` while the repo pins `DOTF_VERSION=0.30.0` — the "fix" would act on a stale pin.

**Stranded user:** obeys doctor, runs generate, nothing changes, doctor still red — an infinite loop between the two commands' verdicts. An agent keying on doctor's FAIL count can never converge.

**Direction:** one shared resolver (repo-first with explicit provenance in output: "contract: <path>"). Doctor printing *which* copy it read would have made this self-diagnosing.

### P5 — Deploy-time secret fetched by a nonexistent id, silently (high, CONFIRMED)

**Where:** `setup-linux.sh:272` (`OPENROUTER_API_KEY="$(dotf secrets show openrouter-api-key 2>/dev/null || true)"`, again at :404) and `setup-windows.ps1:606`; registry id is `OPENROUTER_API_KEY` and `Lookup` matches id/exposed-var only.

**Repro:** `dotf secrets show openrouter-api-key` → `Error: unknown secret "openrouter-api-key" (try dotf secrets ls)`, exit 1 (E2 S14). In setup the `|| true` turns that into an empty exported key.

**Stranded user:** every fresh deploy bakes an empty `OPENROUTER_API_KEY` into agy's MCP config; the agy MCP silently fails at runtime with no deploy-time signal. This is the registry-rename split-brain class (#635/#659) recurring in the setup seam.

**Direction:** fix the id at both call sites; drop the blanket `|| true` in favor of a WARN with the actual error; add a bats/CI grep asserting every `dotf secrets show <id>` call site names an id present in `registry.yaml`.

### P6 — Doctor-green is unreachable on a fresh, docs-faithful machine (high, CONFIRMED)

**Where/Repro (E2 S3-S5, E3 T7):** after documented onboarding, doctor = 83 pass / **31 FAIL** / exit 1. The FAIL set includes: `bw not in PATH — run 'dotf tools install'` → that command fails (`npm: executable file not found`; `packages.json` sources bw as `npm:@bitwarden/cli`; npm is only an "optional dependency" warning); `[SKIP] … run secrets_refresh` → function retired (#587); `opencode … not in PATH (reload shell)` / `pi … re-run setup` → both already done per docs. `doctor --fix` then `doctor` again: still exit 1.

**Stranded user:** cannot tell real breakage from fresh-box noise; an agent using doctor as a preflight gate is permanently blocked.

**Direction:** every fix_suggestion must be executable verbatim and converging (test: run the suggestion, re-run doctor, the item flips); bw needs a non-npm acquisition path (native binary in packages.json) or npm must be a declared dependency; retire the `secrets_refresh` text.

### P7 — README/docs promise processes that no longer exist (medium, CONFIRMED)

- README.md:100-107: `secrets_add`, `secrets_add_file`, `secrets_rotate`, `secrets_show`, `secrets_list`, `secrets_check` — zero definitions anywhere in `.zsh/`, `.bashrc`, `.profile` (retired with load-secrets, #587). README's primary "Secrets" section teaches an interface that is gone; the ADR-028 `dotf secrets` facade is not mentioned there.
- README.md:94 + :54: `. scripts/load-secrets.sh` — file does not exist.
- README.md:91-93 (Human entrypoints): `vault`, `obs`, `dotfiles-sync` resolve in **no** shell (E3 T6: MISSING even in interactive zsh; only `dch`, `profile-shell`, `tx` exist, zsh-only — bash gets none).
- `.zshrc:13` hard-`source`s oh-my-zsh; no installer, no README mention → documented `source ~/.zshrc` errors on every fresh machine (E1+E2 S2).
- README Requirements omit `curl` (E1: setup dies at `setup-linux.sh:238`), and `xz-utils`/`ca-certificates` that the integration Dockerfile had to add.
- `setup-windows.ps1:2050-2051` prints `3. Initialize a project: project-init test-project python` — retired command (now `dotf init`).
- `docs/runbooks/secrets-management.md` is banner-flagged out-of-date but README repeats its retired content without a flag.

**Direction:** README secrets section rewrite to the facade (already tracked as #600 — extend to README explicitly); alias or de-document `vault`/`obs`/`dotfiles-sync`; declare or auto-install oh-my-zsh (or make `.zshrc` degrade); complete Requirements; fix the Windows next-steps text.

### P8 — `doctor --fix` misreports and its advice fights the cascade (medium, CONFIRMED)

**Repro (E2 S4):** `--fix` printed 11 `export …` suggestions and then `Applied 11 fix action(s)` — nothing was applied (by design a subprocess cannot export; the label lies). The suggestions duplicate exactly what the deployed-but-unsourced `paths.sh` already contains, and a user following them plants permanent profile exports that from then on win cascade rule #1 over `machine.json` — the very shadowing class behind this audit's harness incident (see disclosure).

**Direction:** count only real wiring actions as "applied"; phrase env suggestions as "ensure your profile sources paths.sh" instead of raw exports; suggest machine.json for path relocations.

### P9 — Agent ergonomics: contract gaps across the CLI (medium, CONFIRMED)

- No command offers `--json`; agents parse prose (inventory: zero occurrences of a JSON flag CLI-wide).
- `dotf env path NO_SUCH_KEY` → empty stdout, **exit 0** (E3 T5) — unknown key indistinguishable from unset value.
- `dotf spec archive` re-run → `Error: spec not found`, exit 1 (E2 S12) — not idempotent, message doesn't acknowledge "already archived".
- `dotf secrets set` with bw missing → `Error: item "nan-api-key" not found; re-run with --yes…` (E2 S15) — misattributes a missing binary as a missing item; `secrets backup` gets it right.
- `dotf secrets render` default exits 0 leaving `{env:VAR}` unresolved (warning only); both setups call it non-strict, so half-baked configs deploy silently (pi models.json FAIL in doctor is downstream of exactly this).
- `dotf secrets migrate` shared-source guard (good) recommends `migrate --split` → `Error: unknown flag: --split` (W6b).
- Good contracts worth keeping: `secrets verify` exit 1 on FAILED; `render --strict` exit 1; `secrets run --only UNKNOWN` exit 1; `update`'s skip-vs-fail split; `env generate --check` exit 1 on drift.

### P10 — `env generate` trusts ambient env for target and content (medium, CONFIRMED)

`DotfilesDir()` honors ambient `DOTFILES_DIR` and `Home()` prefers `HOME` (Git Bash sets a POSIX-style one); render inputs and output path follow them with no confirmation, no `-w`-style guard, no backup, non-atomic write (`generate.go:110-119`). Demonstrated: a single overridden-HOME invocation rewrote the live `~/.dotfiles/paths.ps1` with Temp-dir paths (restored by re-running generate in a sane env). A `dotf env generate` run from Git Bash on Windows (POSIX HOME) is one plausible real-world trigger of the same corruption.
**Direction:** atomic write + print old→new diff summary; refuse (or require `--force`) when the resolved home/target disagree with the OS convention or with the file being replaced.

### P11 — CRLF hole for extensionless shell rc files (medium, CONFIRMED)

`.gitattributes` normalizes `*.sh/*.bash/*.zsh/*.bats` but `.bashrc`/`.zshrc`/`.profile`/`.{bash,zsh}rc.local.example` fall to `text=auto` → CRLF working tree on Windows (`git ls-files --eol`: `i/lf w/crlf`). Effect reproduced: integration image built from this Windows checkout → `verify-setup.bats` test 42 fails (`syntax error near unexpected token 'in\r'` in deployed `.bashrc`). The guard test only ever runs in CI on LF checkouts — it can never catch the case it guards against. Any Windows→Linux copy flow (docker build, WSL, rsync) ships a broken `.bashrc`.
**Direction:** add explicit `eol=lf` lines for the rc files; renormalize; optionally a CI job that greps the tree for CR in shell-sourced files.

### P12 — Concurrency and torn state (low, PLAUSIBLE)

Two concurrent `dotf env generate` resolved benignly ("wrote"/"unchanged", E2 S19) but the write is non-atomic — a shell sourcing `paths.sh` mid-write can read a torn file. `update.go`'s untracked-counts-as-dirty is fail-safe alone, absorbing when combined with P1. No file locking exists anywhere in the CLI; low practical risk at current usage.

### P13 — States with no owning command (low, CONFIRMED)

- `sensitive/kubelab-dispatch-token.secret.age` exists with no registry entry → real doctor FAILs it (`orphan: … no registry entry`); no `dotf secrets adopt/rm` — only hand-editing files fixes the FAIL.
- Deployed-copy staleness (observed live: contract missing #663 keys, versions.conf pin at 0.23.0 vs repo 0.30.0, github.token drift) accumulates invisibly when the opt-in timer is absent — nothing nags "your deploy is N commits behind".
- Windows: GUARD-001 dispatcher missing at `~\.dotfiles\git-hooks` and doctor's fix says "run dotfiles setup" — but `setup-windows.ps1` has no git-hooks deploy step (the Windows twin is a "tracked follow-up" in install-git-hooks.sh's header) → non-converging FAIL on every Windows box.

### P14 — Spec-gate error attribution (low, CONFIRMED)

`dotf spec init X --issue 999999 --bitacora-repo mlorentedev/dotfiles` with unauthenticated gh → `work-gate issue #999999 not found (or gh failed): To get started with GitHub CLI…` — a real "issue doesn't exist" and "gh broken" produce the same exit and prefix. Repo-detection error without origin is good (offers `--bitacora-repo`/`DOTF_BITACORA_REPO` verbatim). Archive/re-archive asymmetry noted in P9.

---

## 5. Missing-process backlog (by unblocking value)

> **Filed 2026-07-07:** items 1→#694 (P1), 3→#695 (P2), 2→#696 (P3), 4→#697 (P4), 5→#698 (P5), 6→#699 (P6/P8/P13-orphan), 7→#700 (P9). Mapped onto existing tickets by comment instead of duplicating: README truth pass → #677 (P7), CRLF/.gitattributes → #693 (P11), Windows GUARD-001 twin → #691 (P13), bw non-npm source → #649. Items 8 (`secrets rotate/restore`), 11 (atomic generate) and 12 (staleness nag) remain unfiled — lower unblocking value; revisit at triage.

1. **Repo-write-free setup + parity CI** (unblocks P1, the self-deploy promise): move the `.github/copilot-instructions.md` sync out of setup into CI/pre-commit; assert `git status --porcelain` clean at end of the integration test.
2. **machine.json bootstrap at first setup** (unblocks P3 and every repo-anchored flow): setup writes `DOTFILES_REPO_DIR=$CURRENT_DIR` into `~/.config/dotfiles/machine.json` when absent, then generates. Alternative: make `update`/`mem` use `env.RepoDir()` cwd walk-up like secrets/spec do.
3. **README-layout decision + guard** (unblocks P2): either support clone-into-`~/.dotfiles` (same-file-safe copies, same-dir-safe git-hooks, CI variant) or teach the copy layout and refuse in-place with a message. Include curl/oh-my-zsh/Requirements truth-up (P7).
4. **Single contract/versions resolver shared by doctor and env** (+ provenance line in output) (unblocks P4, protects #663 from the stale-copy regression).
5. **Secrets deploy-seam integrity**: fix `openrouter-api-key` call sites (P5); CI grep gating `dotf secrets show` ids against `registry.yaml`; make setup's render/show failures visible (WARN with cause, not `\|\| true`).
6. **Converging doctor fixes** (P6/P8): non-npm bw source in packages.json; retire `secrets_refresh` text; "applied" only for applied actions; a meta-test that executes each fix_suggestion and re-runs the check.
7. **Agent-mode contract** (P9): repo-wide exit-code doc + `--json` on doctor/verify/ls/update at minimum; `env path` unknown-key exit 1; idempotent `spec archive`; accurate bw-absent errors; implement or un-advertise `migrate --split` (C9).
8. **`dotf secrets rotate`/`restore`** — registry already declares cadence; runbook admits manual; the two missing verbs of ADR-028's lifecycle.
9. **Windows GUARD-001 deploy step** (P13) — the tracked follow-up, plus doctor fix text that matches reality until then.
10. **`.gitattributes` rc-file normalization + CR guard in CI** (P11).
11. **Atomic, guarded `env generate`** (P10, P12).
12. **Deploy-staleness nag**: doctor INFO "deploy dir is behind the checkout (N files differ)" exists per-file — add an aggregate hint that maps to "re-run setup or enable the timer" (P13).

## 6. Open questions (maintainer-only)

1. Is clone-into-`~/.dotfiles` a layout you intend to support (README teaches it) or should the README move to the copy layout the CI tests? P2's fix differs completely by answer.
2. `DOTF_VERSION` pin semantics: minimum (then doctor's "installed 0.28.0 > pinned 0.23.0" WARN is noise) or exact (then the fix text should say "downgrade")? Related: should the pin track releases automatically (release-please touchpoint)?
3. Should `terraform` be a hard-FAIL required binary on every machine, including boxes that never run IaC? (It is the only tool FAIL on the otherwise-green real machine.)
4. `ai/copilot/copilot-instructions.md` vs `.github/copilot-instructions.md`: which is the SSOT, and is the banner transform intentional? They are out of parity in HEAD today.
5. The orphan `kubelab-dispatch-token.secret.age`: adopt into the registry or delete? (Doctor FAILs it on every run; no command can resolve it.)
6. `setup-windows.ps1` was NOT executed end-to-end in this audit (no throwaway Windows environment; a real run mutates `$PROFILE`, junctions, Scheduled Tasks). Its P5 call site and P13 gap were verified by trace + real-machine doctor. Worth a dedicated Windows-sandbox (VM) pass?
