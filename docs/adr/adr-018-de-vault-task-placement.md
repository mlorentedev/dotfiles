---
id: "ADR-018-de-vault-task-placement"
type: adr
status: accepted
owner: manu
date: "2026-06-06"
extends: [adr-009-multi-agent-runtime, adr-016-deploy-canonical-agents-to-vault]
tags: [architecture, decision, knowledge-placement, tasks, github-projects, vault, init-project, self-containment]
created: "2026-06-06"
---

# ADR-018: De-vault task placement — 3-layer knowledge model + reusable self-contained flow

> Moves task/work state out of the Obsidian vault into a single cross-repo GitHub Project
> ("bitácora"), makes the per-project flow self-contained so `init-project` works without a
> vault, and defines the 3-layer split that governs knowledge placement from now on.
> dotfiles is the pilot; the model propagates to other repos via a corrected
> `pattern-knowledge-placement`. Epic: [#244](https://github.com/mlorentedev/dotfiles/issues/244).

## Status

Accepted

## Date

2026-06-06

## Context

The prior model stored active task state (backlog, sprint items, portfolio view) in the
Obsidian vault at `10_projects/<repo>/11-tasks.md`. This created three recurring pain points:

1. **Not portable.** A fresh machine, a new contributor, or an AI agent that hasn't cloned the
   vault cannot see the backlog. `init-project` silently degraded: when the vault was absent
   the injected `AGENTS.md` had `$VAULT_PATH` references that 404d off-machine.
2. **Wrong layer for task state.** Trackers (GitHub Issues/Projects, Linear, Jira) are the
   industry-standard home for coordination state — they are searchable, linkable, automatable,
   and shareable without granting filesystem access. The vault's strength is cross-project
   *brain* (patterns, methodology lessons, AI memory) — not task status.
3. **Cold-store pressure.** The vault accumulated 144 archived spec files from this repo alone,
   inflating the "too much content" perception and causing navigation friction unrelated to the
   quality of vault content.

Additionally, the project flow was not self-contained: `init-spec` hard-failed when no vault
entry existed, and `init-project` leaked the private vault path into fresh repos.

**Research gate (Projects v2 mechanics).** Before committing, GitHub Projects v2 limits were
confirmed (see #244 comment-2): user/org-level only, 50 k items, one cross-repo board is the
right shape; issue-closed→Done built-in by default; auto-add is plan-capped (Free=1 repo,
Pro/Team=5) but per-repo `actions/add-to-project` is plan-agnostic and scales to N repos.
PAT with `project` scope required (default `GITHUB_TOKEN` cannot touch Projects v2). No
backfill — existing issues need a scripted one-time migration.

## Decision

### Layer model (3 layers, linked not duplicated)

| Layer | Home | What lives here | What does NOT |
|---|---|---|---|
| **Task state** | GitHub Project "bitácora" (user-level, private, cross-repo) | Backlog items, sprint status, portfolio view, issue-level progress | Durable design, patterns, lessons |
| **Durable design** | This repo: `specs/`, `docs/adr/`, `docs/lessons.md`, `docs/runbooks/` | Architecture decisions, specs, troubleshooting, project lessons | Task status, vault patterns |
| **Cross-project brain** | Vault: `00_meta/`, `90-lessons.md`, AI memory | Universal patterns, methodology lessons, AI session memory | Task state (no `11-tasks.md` equivalent) |

The vault *links out* to repos (via `10_projects/<repo>/` context docs); repos do not
depend on the vault. The bitácora board *aggregates* repo issues; it owns no data of its own.

### bitácora Project spec

- One **user-level private** GitHub Project named "bitácora".
- Fields: Status (single-select: Backlog/In Progress/Done/Blocked), Repo, Priority
  (P0–P3), Type (spec/bug/chore/ideas).
- Views: table (default, group-by Status), board (group-by Status), per-repo filter.
- Built-in workflows (default-on): issue-closed→Done, PR-merged→Done, auto-archive
  Done items > 2 weeks.
- Per-repo auto-add: a `.github/workflows/add-to-project.yml` file (using
  `actions/add-to-project@v1`) wired into `init-project` so every new repo auto-joins.
  Requires a PAT with `project` scope, stored encrypted in the age-secrets system.
- One-time backfill of existing open issues via `gh project item-add` loop (scripted).

### `init-project` self-containment (crown-jewel fix)

`init-project` must work on a machine without a vault. Concretely:
- Add a vault-existence check at startup; print a `[WARN]` if absent but continue.
- Support `--skip-vault` flag to suppress the warning and the vault-dependent steps.
- Vendor a minimal set of SDD rules and templates into the repo itself (at `docs/sdd/`)
  so `init-spec` has a local fallback and never hard-fails when the vault is absent.
- `init-spec`: add `exit 3` fallback — return a repo-local skeleton if vault is absent,
  not a hard error.
- The injected `AGENTS.md` must not contain `$VAULT_PATH` literals that 404 off-machine;
  replace with relative paths or conditional vault-reference blocks.

### Migration path (dotfiles pilot)

1. Set up bitácora per the spec above (`gh project create`, field seed, view config,
   built-in workflows). Bootstrap script committed in `scripts/setup-bitacora.sh`.
2. Backfill: run `scripts/setup-bitacora.sh --backfill-repo mlorentedev/dotfiles` to add
   all open issues to the board.
3. Wire `init-project` to drop `.github/workflows/add-to-project.yml` into new repos.
4. Archive `10_projects/dotfiles/11-tasks.md` in the vault (do not delete; history is
   valuable) with a note pointing to bitácora.
5. Correct `pattern-knowledge-placement` to reflect the 3-layer model.
6. Propagate to other repos as they are next touched (lazy migration; no big-bang).

### AGENTS.md updates (this decision)

Three values change in `AGENTS.md`:
- **Knowledge Placement block:** `tasks:` changes from `repo:issues` to
  `project:bitácora (user-level GitHub Project, cross-repo)`.
- **Neural Hive Phase 1 Context Sync:** remove "Read `11-tasks.md`"; replace with
  "Check the bitácora GitHub Project for the active backlog".
- **Neural Hive Phase 3 Crystallization:** remove the `11-tasks.md` mark-done step;
  closing the GitHub issue moves it to Done automatically (built-in workflow).

## Alternatives considered

**Keep vault 11-tasks.md, improve discoverability.** Rejected: the portability problem
(off-machine 404s, vault-absent hard-fails) remains unsolved; vault markdown is not
filterable by priority, not automatable, and not cross-repo.

**Flat markdown backlog in the repo root.** Rejected: repo-root `TODO.md` / `backlog.md`
is an anti-pattern (already called out in AGENTS.md, closes CHORE-002 spirit) and still
not cross-repo.

**Use Linear or Jira.** Rejected: external paid SaaS adds friction for a solo developer;
GitHub Projects is free, co-located with the issues, and requires no new account. The
teledyne-e2v platform project already validates this model at programme scale.

**Per-repo Projects board.** Rejected: Classic Projects v2 is now user/org-level only
(Classic deprecated and removed 2025-06-03). A single cross-repo board is the correct
and native shape.

## Consequences

**Positive:**
- Task state is visible to any agent, on any machine, without vault access.
- `init-project` works self-contained; fresh repos no longer leak the private vault path.
- `init-spec` degrades gracefully (exit 3 + repo-local skeleton) instead of hard-failing.
- The vault shrinks to its correct role (cross-project brain), eliminating cold-store
  pressure from task accumulation.
- issue-closed→Done is automatic — zero manual bookkeeping for routine completion.
- The model is plan-agnostic: the per-repo `add-to-project` Action scales to N repos on
  any GitHub plan.

**Negative / debt:**
- One-time backfill is a manual scripted step; existing open issues must be added
  explicitly (no native backfill for Projects v2).
- A PAT with `project` scope must be provisioned and kept in the age-secrets store; this
  is a new operational dependency.
- `pattern-knowledge-placement` and any doc/skill referencing `11-tasks.md` must be
  updated — a non-trivial audit pass.
- At ~300–500 open items the single board will need a split strategy (Priority field
  buys time; a per-stream sub-view is the natural first split).
- **Issue homing is constrained by repo visibility (added 2026-06-13).** `gh issue
  transfer` refuses private→public moves (`Old issue cannot be transferred from private
  repository to public repository`). The vault repo (`knowledge`) is private; the existing
  home repos (`kubelab`, `hive`, `dotfiles`) are all public — so the "the bitácora
  aggregates repo issues; it owns no data of its own" target state **cannot** be reached by
  lazy-migrating issues into them without making the private backlog public. Recreating the
  issue in the public repo is the only workaround and it both exposes the ticket and loses
  comment history / the stable number. Net: while home repos are public and the backlog is
  private, **centralizing issues in the private `knowledge` repo is the correct state**, and
  the board's `Repo` field provides the per-repo focus view. Lazy migration (decision §
  "Migration path" step 6) applies only to private→private, or where a ticket's public
  exposure is acceptable (e.g. a genuinely open-source repo's own bug). Repos that do not
  yet exist (`iris`, `hermes`, the KPM-W targets) keep their tickets in `knowledge` as a
  holding pen until created.

## Reopen Triggers

- **Board split:** open item count reaches 300 and search/prioritization degrades.
- **Plan upgrade:** a team fork of this setup needs > 5 native auto-add workflows;
  reconsider native vs Action-based auto-add at that point.
- **Vault backfill:** if the vault `11-tasks.md` pattern is needed for offline-only
  work (air-gapped machine, no GitHub), re-evaluate the fallback model.
- **Repo homing unblocked:** a private per-stream repo is created (e.g. `iris` / `hermes`
  born private), or a stream's tickets become acceptable to expose publicly — at which point
  that stream's issues move to their home repo and the `add-to-project` workflow re-aggregates
  them onto the board (see the visibility-constraint note under Consequences › Negative/debt).

## References

- `docs/adr/adr-009-multi-agent-runtime.md` — AGENTS.md as cross-agent SSOT.
- `docs/adr/adr-016-deploy-canonical-agents-to-vault.md` — vault deploy strategy.
- `00_meta/patterns/pattern-knowledge-placement` — corrected by this ADR.
- `00_meta/patterns/pattern-three-layer-proposal-lifecycle` — the proposal flow this PR follows.
- GitHub Projects v2 research: [#244](https://github.com/mlorentedev/dotfiles/issues/244) comment-2.
- Industry precedent: teledyne-e2v platform programme board (one board per programme, per-stream sub-views).
- docs.github.com/issues/planning-and-tracking-with-projects
- github.com/actions/add-to-project
