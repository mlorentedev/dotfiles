---
id: "ADR-057-project-workspace-boundary-and-worktree-roots"
type: adr
status: accepted
owner: manu
date: "2026-11-05"
issue: "mlorentedev/dotfiles#3191"
extends: [adr-025-cross-machine-path-resolution, adr-050-one-lane-contract-for-every-repo-agent-and-forge, adr-051-a-merge-entry-records-what-it-manages]
tags: [architecture, decision, paths, worktrees, cross-platform, agents]
created: "2026-11-05"
---

# ADR-057: One project workspace boundary, with managed worktree roots below it

## Status

Accepted 2026-11-05.

## Context

The repositories on the Windows workstation moved from `~/Projects/<repo>` to
`~/Projects/Workspace/<repo>`. ADR-025 already provides the correct per-machine path
cascade, but each path is still configured independently:

- `DOTFILES_REPO_DIR` and `VAULT_PATH` have machine overrides under `Projects/Workspace`.
- `WORKTREE_ROOT` is persisted in the Windows user environment as
  `Projects/worktrees`, a directory that no longer exists.
- herdr and `dotf worktree` consume `WORKTREE_ROOT`.
- GitHub Copilot Desktop stores its parent location as `Projects/Workspace` and creates
  `copilot-worktrees/<repo>/<branch>` below it.
- Copilot CLI exposes native worktree commands but no documented worktree-root setting.
- Copilot and agy trust files repeat literal `Projects/Workspace` paths.

Several linked worktree directories were moved manually from the old root to
`Projects/Workspace/.worktrees`. Their checkout `.git` files and the main repositories'
administrative records now disagree:

- Git still lists old `Projects/worktrees/...` paths as prunable.
- the moved dotfiles, fae-brain and ts-bridge checkouts still contain recoverable work;
- the moved kubelab directory points at administrative metadata that no longer exists;
- `.worktrees` is matched by agy's `**/.worktrees/**` exclusion, so those checkouts are
  invisible to that agent.

Pruning the old registrations would therefore discard the only Git identity for
recoverable work. Moving the directories again with the filesystem would repeat the
same defect. The recovery and the future placement contract must be one design.

Windows path depth is also a real constraint. On this machine, the current repositories'
deepest tracked paths project to 243 characters under
`Projects/Workspace/worktrees` and 251 under
`Projects/Workspace/copilot-worktrees`. A realistic 60-character branch slug can exceed
260 in either layout. `LongPathsEnabled=1` on this host does not make every Windows
consumer long-path aware.

The GitHub Copilot session-start incident investigated in the same session had a separate
root cause: the unsupported local setting `telemetry.disabled`. It does not motivate this
path decision and is tracked separately.

## Reference audit

The references converge on four facts:

1. A machine needs one authoritative workspace boundary, not repeated absolute paths.
2. Git administrative metadata is the authority for linked worktrees.
3. Relocation is a lifecycle operation because several agents key sessions by path.
4. Third-party tools do not expose one uniform exact-root contract.

The last fact makes "one exact directory for every tool" an unsupported promise. The
portable invariant is one workspace boundary, one exact root for managed creators, and
explicitly declared third-party roots below the same boundary.

## Constraints

- The design works on Windows, Linux and macOS and is provisioned from zero.
- An unreadable or ambiguous inventory fails loud; it never becomes an empty success.
- Every new mechanism ships with a guard.
- Existing explicit per-path overrides remain valid.
- No writes to third-party internal databases.
- No junction or symlink aliases that give one checkout multiple path identities.
- Windows placement is checked against the repository's real projected path length.
- A second convergence pass reports `changed=0`.

## Options considered

### A. Change only `WORKTREE_ROOT` on each machine

This fixes the immediate machine with the smallest diff. It leaves every repository,
vault and trust path independent, so the next workspace move repeats the same incident.
Rejected because it does not create a reusable machine contract.

### B. Use `copilot-worktrees` as the one exact root

This aligns `dotf`, herdr and Copilot Desktop on the current workstation. It makes a
third-party application's private naming and lifecycle the neutral platform contract,
does not control Copilot CLI's native command, and has the worst measured Windows path
margin. Rejected.

### C. One workspace boundary plus a derived managed root

`PROJECTS_PATH` is the per-machine boundary. Managed repositories and worktrees derive
their defaults from it. `WORKTREE_ROOT` remains the exact root for `dotf` and herdr.
Third-party roots without an exact-root setting remain explicit children of
`PROJECTS_PATH` and are inventoried honestly. Selected.

### D. Filesystem aliases for third-party root names

Junctions or symlinks would make `.worktrees`, `worktrees` and `copilot-worktrees` appear
to share storage. Git and session stores would still see different path identities,
cleanup could target an alias rather than the registered path, and `.worktrees` would
still match agy's exclusion. Rejected.

### E. Store every absolute path independently in `machine.json`

This is explicit but creates one edit per consumer on every relocation, with no common
boundary against which doctor can validate placement. Rejected.

## Decision

### 1. `PROJECTS_PATH` is the per-machine workspace boundary

Add `PROJECTS_PATH` to `env-contract.json`.

- Default: `$HOME/Projects` on Linux/macOS and
  `$env:USERPROFILE\Projects` on Windows, preserving the existing default layout.
- A machine whose repositories live elsewhere declares one override in `machine.json`.
  This workstation declares `C:\Users\mlorente\Projects\Workspace`.
- The path may be absent on a from-zero machine before its first clone; existence is not
  a hard bootstrap precondition.

### 2. Dependent defaults are explicit and backward-compatible

Contract entries may declare an additive `derived_from` relationship:

```json
{
  "name": "WORKTREE_ROOT",
  "derived_from": { "key": "PROJECTS_PATH", "path": "worktrees" },
  "default": {
    "linux": "$HOME/Projects/worktrees",
    "windows": "$env:USERPROFILE\\Projects\\worktrees"
  }
}
```

The absolute `default` remains the compatibility fallback for a released binary that
does not understand `derived_from`. A new resolver uses the derived value when the
dependent key has no explicit environment or machine override.

The resolver:

- rejects an unknown base key;
- rejects a cycle;
- rejects an absolute `path` component;
- rejects any unresolved contract token;
- tests that each compatibility default equals the derived value under the default
  `PROJECTS_PATH`.

This avoids arbitrary string interpolation and prevents an old binary from creating a
literal `{PROJECTS_PATH}` directory.

`DOTFILES_REPO_DIR`, `VAULT_PATH`, `HIVE_VAULT_PATH`, `WORKTREE_ROOT` and
`COPILOT_WORKTREE_ROOT` adopt the relationship where their default is a sibling below
the project workspace. An explicit machine override continues to win, including a vault
or application root intentionally stored elsewhere.

### 3. Generated persistent environment is not an explicit user override

The path cascade remains explicit process environment -> `machine.json` -> contract.
However, a value that dotf wrote is a generated artifact, not an operator override. This
includes Windows user-environment values named by `DOTF_MANAGED_ENV` and values loaded
from generated `paths.sh` or `paths.ps1`. The generated files are the provenance record
for the keys they export.

Generation remains deterministic: it resolves only the contract and `machine.json`, never
the current process environment. Runtime resolvers and persistence ignore stale
dotf-generated values while computing desired state; an unmanaged environment value keeps
normal precedence.

`dotf env persist` computes the desired value while ignoring stale dotf-managed values,
updates the persistent scope, and records the managed set atomically. An unmanaged value
keeps normal environment precedence.

Doctor fails when:

- a dotf-managed persistent value differs from the resolved machine/contract value;
- the generated `paths.sh` or `paths.ps1` differs;
- the current process still carries the superseded value after persistence changed.

The convergence report names that a new process or shell is required. It never reports
the machine converged while newly launched processes and the current process disagree.

### 4. Managed and third-party roots have different guarantees

The managed root is:

```text
<PROJECTS_PATH>/worktrees/<repo>/<branch-slug>
```

`dotf worktree` and herdr use it exactly.

The declared Copilot Desktop root is:

```text
<COPILOT_WORKTREE_ROOT>/<repo>/<branch>
```

Its default is `<PROJECTS_PATH>/copilot-worktrees`, matching the application's current
user-facing storage-parent behavior. The value exists so trust and inventory have a
stable contract; dotfiles does not write the application's database. A machine that
selects another storage parent sets an explicit override.

For Copilot CLI and any future harness with no supported exact-root setting:

- automation created by dotfiles uses `dotf worktree`;
- a native worktree is accepted only when it remains below `PROJECTS_PATH`;
- doctor reports its actual root and does not claim exact-root parity.

The agent trust projections use:

- `{PROJECTS_PATH}` and `{PROJECTS_PATH}/*` for repositories;
- `{WORKTREE_ROOT}/*/*` for managed worktrees;
- `{COPILOT_WORKTREE_ROOT}/*/*` for Copilot Desktop worktrees.

Literal `Projects/Workspace` trust entries are retired through the existing merge-record
mechanism.

### 5. Worktree reconciliation is plan-first and has two checkpoints

A new worktree reconciliation surface inventories every declared root and every Git
administrative record. It classifies each path as:

- healthy;
- moved but repairable;
- prunable with no recoverable directory;
- orphaned (directory exists, administrative metadata does not);
- active-session;
- locked;
- dirty or ahead;
- outside the workspace boundary;
- third-party application residue.

An unanswerable state is an error.

Checkpoint 1 repairs metadata in place only:

1. prove that the moved checkout's `.git` file and one main-repository record identify
   the same worktree;
2. run `git worktree repair` on the current path;
3. verify that the main repository and the checkout report the same path and branch;
4. require a second plan with no repair action.

Ambiguous candidates and orphaned directories are never repaired automatically.

Checkpoint 2 relocates only sound, clean and idle linked worktrees:

1. skip active, locked, dirty, ahead or path-keyed sessions with a reason;
2. use `git worktree move` to the managed root;
3. verify content and registration at the new path;
4. adopt or refresh the `.dotf-worktree.json` lease;
5. prune an obsolete registration only after proving no recoverable checkout points at
   it.

An orphan has a terminal quarantine/rescue disposition. It does not enter an infinite
retry loop and is never deleted as "prunable".

The repair checkpoint and the move checkpoint ship separately. A failure during repair
cannot leave half of the fleet moved.

### 6. Path depth is a creation-time invariant

On Windows, `dotf worktree add` computes:

```text
len(target root + repo + branch slug + repository's longest tracked relative path)
```

It refuses a target at or above the effective limit and reports the exact margin. It
warns when the remaining margin is below a safety threshold for generated files.
`LongPathsEnabled=1` does not disable the guard because not every consumer is
long-path-aware.

Doctor applies the same calculation to existing worktrees. This closes #2637.

### 7. A linked checkout cannot be hidden from an agent

Doctor rejects a managed worktree root with a dot-prefixed path component or one matched
by a deployed agent exclusion glob.

The reconciliation inventory also scans below `PROJECTS_PATH` for linked-checkout `.git`
files under dot-prefixed components, then cross-checks them against each repository's Git
administrative records. A checkout found under an excluded component is a hidden-root
error even when the configured `WORKTREE_ROOT` itself is valid. The configured-root guard
and the boundary inventory together prevent the current `Workspace/.worktrees` regression
from recurring.

### 8. Rollout order

1. Ship resolver, persistence and doctor support while keeping current defaults.
2. Release and pin a `dotf` version that understands `derived_from`.
3. Add `PROJECTS_PATH` relationships and deploy trust projections.
4. Set this machine's `PROJECTS_PATH`; persist and start a fresh process.
5. Run reconciliation checkpoint 1 until metadata is sound.
6. Resolve or quarantine orphaned directories.
7. Run checkpoint 2 for safe worktrees.
8. Verify a second converge and reconciliation plan report `changed=0`.

## Consequences

### Positive

- Moving the project workspace becomes one machine override instead of several unrelated
  path edits.
- Existing explicit overrides remain valid.
- Git metadata and session-path safety become part of the migration contract.
- Every managed agent reads the same trust boundary and worktree root.
- Third-party capability gaps remain visible instead of being hidden by filesystem hacks.
- Windows path failures are rejected before creating a checkout.

### Negative

- The environment resolver gains dependency semantics and provenance-aware persistence.
- Worktree recovery requires a new plan/apply surface and more state classification.
- Copilot Desktop cannot be fully configured from dotfiles until GitHub exposes a
  supported non-GUI setting for its storage parent.
- Orphan rescue needs an explicit operator decision because the branch and base may no
  longer be provable.

### Neutral

- The exact managed child name remains `worktrees`; a machine may explicitly override it
  to a shorter root when its path budget requires that.
- Third-party roots can coexist below `PROJECTS_PATH`; "one workspace" does not mean
  pretending every tool supports one exact root.

## Reopen triggers

- A supported GitHub Copilot setting exposes the exact worktree root rather than only its
  storage parent.
- A second third-party tool requires a fixed child name outside `PROJECTS_PATH`.
- The path-budget guard shows the default child name cannot support a representative
  repository even with a normal branch slug.
- A real machine needs a repository root outside `PROJECTS_PATH` that cannot be expressed
  as an explicit override.

## References

- #3191
- #2637
- [ADR-025](adr-025-cross-machine-path-resolution.md)
- [ADR-050](adr-050-one-lane-contract-for-every-repo-agent-and-forge.md)
- [ADR-051](adr-051-a-merge-entry-records-what-it-manages.md)
- Vault pattern `pattern-contract-defaults-per-machine-override`
- Vault pattern `pattern-setup-script-idempotence`
