---
id: "BUG-168-converge-step-dependencies"
type: spec
status: verifying # draft | implementing | verifying | archived
created: "2026-11-16"
issue: "mlorentedev/dotfiles#3554"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
owner: manu
wip_override: "owner-requested fix: a failed leaf step keeps every later converge step from being reached (#3554) (10 active, limit 10, 2026-11-16)"
---

# BUG-168-converge-step-dependencies

## Why

`dotf converge` stopped at the first failed step and reported every later step as `not reached`. On 2026-11-16 `system-upgrade` (step 6 of 24) failed on a Windows workstation (#3553), and git config, git hooks, lanes, the SSH target, the agent extensions and the setup script did not run, although none of them reads anything `system-upgrade` writes. The nightly `dotf update` repeated it every night. The owner chose on #3554 to let a failure hold back only the steps that depend on it.

## What

- Each step in the registry declares the earlier steps it needs.
- A failed step keeps from running only the steps that need it, directly or through another. They are reported `[SKIP] not reached: <step> failed` (or `not reached: needs <step>, <step> failed`). Every other step runs, in the same order as before.
- The run still fails when any step failed: non-zero exit, and the error names every failure.
- A step that declares nothing needs every step before it, which is the old behaviour.

## Out of scope

- Re-ordering the registry, or running steps in parallel.
- Narrowing what `tools` holds back. Most steps use what it installs; each edge needs its own evidence.
- Giving `legacy-setup` a list. It needs every native step (ADR-045 decision 5).
- The plan gate, which still skips every step after it.
- The winget elevation failure itself: #3555.

## Step needs and their evidence

Paths are under `cli/internal/`. "Policy" marks an edge a decision requires although no code reads the earlier step's output. An uncertain edge is kept (ADR-064 decision 7).

| Step | Needs | Evidence |
|---|---|---|
| `checkout` | none | root; installs git itself (`converge/checkout.go:87-101`) |
| `records-mirror` | checkout | `harness.Mirror(env.RepoRoot, …)` `converge/records.go:171` |
| `env-generate` | checkout, records-mirror | reads `env-contract.json` `converge/envgenerate.go:23`; mirror is policy (`envgenerate.go:13-15`), kept |
| `records-harness` | checkout, records-mirror | sources from the checkout `converge/harness.go:90,99`; mirror is policy (`records.go:159-160`, ADR-045 D4) |
| `tools` | checkout, records-mirror, env-generate, records-harness | `packages.json` `converge/catalog.go:58`; no agent before its instructions `harness.go:14-18` (#2016); paths before agents `envgenerate.go:14-15` |
| `system-upgrade` | checkout, tools | catalog `converge/sysupgrade.go:264`; "after tools" `sysupgrade.go:21-22`, kept |
| `cache-gc` | checkout, tools | `mise prune` after a half-failed tools run removes versions still in use `converge/cachegc.go:60-63,316-335` |
| `pwsh-modules` | checkout, tools | `versions.conf` `converge/pwshmodules.go:67`; pwsh comes from the catalog |
| `pi-settings` | checkout, tools | `ai/pi/settings.json` `converge/pisettings.go:30`; pi comes from tools |
| `configs-deploy` | checkout, records-mirror, tools, pi-settings | `ai/deploy.json` `converge/configs.go:36`; merges into the file pi-settings seeds (`pisettings.go:16-21`, #1948); `{VAR}` paths can resolve from the mirrored contract `env/env.go:205-207` |
| `orca-hooks` | checkout | reads only Orca's own files `orca/hooks.go:30-33`; checkout is the floor |
| `shell-completions` | checkout, tools | zsh and herdr from tools `converge/shellcompletions.go:56-63` |
| `records-bind` | checkout, records-mirror, records-harness, tools, configs-deploy | bind targets `converge/bind.go:34`; settings seeded by configs-deploy `bind.go:17-19`; `{VAR}` `harness/bind_apply.go:164-174`; binds the agents records-harness wrote |
| `git-config` | checkout, tools, configs-deploy | its probe fails without the deployed gitconfig `converge/gitconfig.go:66-68` |
| `ts-bridge-client` | checkout | reads machine roles and its own drop-in `converge/tsbridge.go:66-76`; floor |
| `git-lfs` | checkout, tools | git and git-lfs from the catalog `converge/gitlfs.go:12-13` |
| `git-hooks` | checkout, tools | `RepoRoot/git-hooks` `converge/githooks.go:27`; git from tools |
| `lanes` | checkout, records-mirror, tools, git-hooks | `converge/lanes.go:33,36,60`; creating a worktree fires the hooks (`lane/activate.go:79`) |
| `env-persist` | checkout, tools | contract `converge/envpersist.go:81`; writes the tool dirs into the user Path `envpersist_user.go:90,113` |
| `self-deploy` | checkout | roles and scheduler only `converge/selfdeploy.go:57,75`; floor |
| `hive-upgrade-retire` | checkout | scheduler and home only `converge/hiveupgrade.go:45-74`; floor |
| `workstation-target` | checkout | `ssh/` in the checkout `converge/workstation.go:41`; not configs-deploy (`sshtarget/target.go:89,559,566`) |
| `agent-extensions` | checkout, records-mirror, records-harness, tools, configs-deploy, records-bind | `converge/extensions.go:31-32,86-87`; writes beside the hooks bind writes `extensions.go:26-29` |
| `legacy-setup` | every earlier step | declares nothing; ADR-045 D5, `converge/legacy.go:18-21` |

Edges removed, compared with "needs every earlier step": every pair not in the table. The ones that matter for #3554 are the eleven steps after `system-upgrade` and `cache-gc` that needed them only by position. No code in those steps reads the package manager's state or the caches.

## Risks / open questions

- **A missing edge runs a step against a failed prerequisite.** Mitigated by keeping uncertain edges and by tests that fix what `checkout`, `records-mirror`, `records-harness`, `pi-settings`, `configs-deploy`, `git-hooks` and `records-bind` hold back.
- **`checkout` and `tools` extend the process PATH before they can fail** (`checkout.go:64`, `tools.go:37`), so a later `has()` can see a shim whose install failed. Every step that execs a tool from `tools` needs `tools`.
- **A failed revisit of `records-harness` happens after `tools` ran.** Steps that need only `tools` still run; `records-bind` and `agent-extensions` need `records-harness` and are held back.
- **The quiet-day update reads a skipped `system-upgrade` as the upgrade's own skip** (`cmd/update.go`). A test keeps `system-upgrade` and `cache-gc` from needing each other.
- **Several failures make the recorded error multi-line.** `dotf doctor` now prints its first line and counts the rest.
- No open questions.

## Acceptance criteria

- [ ] AC1: a failed step keeps from running only the steps that need it, directly or transitively; the others run and are probed.
- [ ] AC2: the run returns an error naming every failed step, and a later step that changed the machine does not erase it.
- [ ] AC3: a step held back is reported `not reached: <cause>`, in its place in the report, and the observer hears it.
- [ ] AC4: a step that declares nothing needs every step before it.
- [ ] AC5: a need that is not in the run, skipped, or not supported on this OS holds nothing back.
- [ ] AC6: a failed step is not revisited; a failed revisit holds back that step's dependents and nothing else.
- [ ] AC7: every step of `Registry` declares its needs except `legacy-setup`; each need is an earlier step.
- [ ] AC8: in `Registry`, a failed `system-upgrade` holds back only `legacy-setup`; a failed `checkout` holds back every step; a failed `records-mirror` or `records-harness` holds back `tools`.
- [ ] AC9: `dotf doctor` shows a multi-line recorded error on one line.

## References

- Bitácora board: mlorentedev/dotfiles#3554
- Related ADR: `docs/adr/adr-064-a-failed-converge-step-holds-back-only-its-dependents.md`, amending ADR-041 decision 4 and ADR-045 decisions 4, 5, 11
- Related: #3553, #3555
