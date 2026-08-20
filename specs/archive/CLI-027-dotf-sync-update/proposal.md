---
id: "CLI-027-dotf-sync-update"
type: spec
status: archived # draft | implementing | verifying | archived
created: "2026-07-01"
issue: "mlorentedev/dotfiles#496"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal, cli, go, convergence, sync, update, strangler-fig]
template_version: "1.0"
review: waived
review_waived_reason: "Work shipped and merged under #496; the issue was then closed by hand, so the archive-on-merge gate (keyed on a PR closing keyword) never saw it and the spec was left active. A retroactive adversarial review cannot gate code already on main, so the waiver is recorded instead of manufacturing one. Backlog reconciliation 2026-08-19."
---

# CLI-027-dotf-sync-update

> **Naming**: `specs/CLI-027-dotf-sync-update/`. AUDIT-007 **Phase B / PR11** of the ADR-020/021
> CLI convergence. Delivered as **two PRs** (see `tasks.md`) to stay under the atomic-PR cap and
> de-risk (update touches no secrets; sync does).

## Why

AUDIT-007 (the CLI-convergence execution map) targets four live twins for deletion here —
`dotfiles-selfupdate.{sh,ps1}` and `dotfiles-sync.{sh,ps1}` — by building `dotf update` and
`dotf sync` in Go. They are maintained-twice logic (each fix applied in bash *and* PowerShell,
tested in bats *and* Pester), the exact duplication ADR-020 exists to kill. Porting them to the
single cross-compiled `dotf` binary removes the twin tax, unifies the divergent per-OS models
into one tested code path, and drops the `.ps1` count toward the ADR-021 end state (~3). The
scheduler that invokes selfupdate stays an OS-native **shim** (systemd `.timer` / Task Scheduler)
that calls `dotf` directly — the AUDIT-007 "keep scheduler shims" floor.

## What

Two observable capabilities, shipped as two PRs:

### PR 1 — `dotf update` (the selfupdate port)

`dotf update` reproduces `dotfiles-selfupdate.{sh,ps1}` exactly: an opt-in, scheduler-invoked
self-deploy that fast-forwards the dotfiles repo and re-runs the idempotent setup — **only** on a
clean fast-forward, **only** when HEAD moved. Every non-actionable condition is a **skip (exit 0)**;
the **only** non-zero exit is a real setup failure (so `systemctl --user status` / the journal
surface it):

- Not a git repo → skip. Dirty worktree → skip. `git fetch` fails (transient/network) → skip.
- No upstream configured → skip. Already current → skip. Diverged (non-fast-forward) → skip.
- Clean fast-forward → `git merge --ff-only @{u}`, then exec the OS setup (`setup-linux.sh` /
  `setup-windows.ps1`, still the shell floor until CLI-028). Setup's exit code is propagated.

The git logic is portable; a `Runner` seam (git + setup exec) keeps it unit-testable with no real
git/setup in CI. **Deletes** `dotfiles-selfupdate.{sh,ps1}` + their bats/Pester tests; **repoints**
the `dotfiles-selfupdate` systemd timer's `ExecStart` and the Windows task action to `dotf update`.

### PR 2 — `dotf sync` (the bidirectional sync port)

`dotf sync` reproduces `dotfiles-sync.{sh,ps1}`: (a) **secrets** — bidirectional **newest-wins**
per-file merge of `sensitive/*.secret.age` + `.secrets-audit.log` between the deployed copy and
the repo (atomic write; **no deletion propagation** — a secret that exists on only one side is
never wiped; per the ratified decision), then (b) **repo → deployed** deploy + `git push`.
`--secrets-only` stops after (a). **Deletes** `dotfiles-sync.{sh,ps1}` + tests.

## Out of scope

- **The setup orchestrator** stays shell (CLI-028, LAST). `dotf update` *invokes* setup; it does
  not absorb it.
- **A `dotf schedule` primitive.** Rejected — scheduler unit files are the AUDIT-007 irreducible
  floor; the escrow-backup schedule (#666) follows the same shim pattern, not a Go primitive.
- **Secrets curation** (#586 folders/token split), the `bw` backend (#585).

## Risks / open questions

- **[PR 2] The repo→deployed deploy model diverges by OS and is unresolved.** Linux rsyncs
  `repo → ~/.dotfiles` with `--delete` (ADR-005: `~/.dotfiles` is a non-git deployed copy,
  `sensitive/` + `.git/` excluded); Windows uses a git-pull model. Reconciling these into one Go
  path is the hard, higher-blast-radius part of `dotf sync` and **must be resolved before PR 2
  is scoped** (confirm the Windows deploy model against `dotfiles-sync.ps1` + `setup-windows.ps1`).
  This is why `update` (deploy-model-agnostic — it just re-runs setup) ships first.
- **[PR 1] skip-vs-fail semantics are load-bearing.** A scheduler run must be quiet on expected
  conditions (exit 0) and loud only on a real setup failure. The port must preserve this exactly,
  or a timer either spams failures or hides a broken deploy. Covered by table tests over every
  branch.
- **[PR 1] the timer/task repoint must land atomically with the deletion.** The unit's `ExecStart`
  must point at `dotf update` before (or in the same PR as) deleting the `.sh` — a guard-grep
  completeness check ensures no caller still references the deleted script (the AUDIT-007 orphan
  rule).
- **Secrets newest-wins is already the cross-OS behavior** (both twins compare mtime); porting it
  is faithful, not a behavior change. Dropping rsync `--delete` applies only to the non-secret
  deploy (PR 2), never to secrets.

## Acceptance criteria

### PR 1 — `dotf update`

- [ ] **AC1** — `dotf update` fast-forwards + re-runs setup only on a clean FF with moved HEAD;
  each non-actionable condition (no repo / dirty / fetch-fail / no-upstream / current / diverged)
  is a skip with exit 0. *Verify:* table tests over every branch via the `Runner` seam.
- [ ] **AC2** — a real setup failure is the **only** non-zero exit, with the child's code
  propagated. *Verify:* seam test with a failing setup runner.
- [ ] **AC3** — `dotfiles-selfupdate.{sh,ps1}` + their tests are deleted; the systemd timer +
  Windows task invoke `dotf update`; a guard-grep finds no lingering reference to the deleted
  scripts. *Verify:* `git grep` guard in CI + the unit/task diff.
- [ ] **AC4** — `go test ./... && go vet && gofmt -l` clean; the production git/setup exec is
  covered by a live smoke, not CI.

### PR 2 — `dotf sync`  *(scoped after the deploy-model open question is resolved)*

- [ ] **AC5** — bidirectional newest-wins secrets merge (no `--delete`); atomic per-file write.
- [ ] **AC6** — repo→deployed deploy + `git push` on the reconciled model; `--secrets-only` stops
  after secrets; `dotfiles-sync.{sh,ps1}` + tests deleted; guard-grep clean.

## References

- Plan: `docs/adr/audit-007-cli-convergence-state.md` (Phase B / PR11 — "keep scheduler shims";
  the 4-script delete target), ADR-020 (convergence), ADR-021 (roadmap), ADR-005 (`~/.dotfiles`
  non-git deployed copy on Linux).
- Issue: `mlorentedev/dotfiles#496` (CLI-027). Sibling: #666 (escrow-backup schedule — same
  scheduler-shim floor).
- Ports: `scripts/dotfiles-selfupdate.{sh,ps1}` (PR 1), `scripts/dotfiles-sync.{sh,ps1}` (PR 2).
- Reuse idiom: the `secrets`/`doctor` seam pattern (inject git/exec/fs; fakes in CI, live smoke
  for real I/O).
