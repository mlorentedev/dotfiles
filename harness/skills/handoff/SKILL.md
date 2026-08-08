---
id: handoff-skill
type: skill
status: active
created: "2026-05-31"
owner: manu
name: handoff
description: Run a complete, standardized session handoff at session end (or on demand). Triggers on /handoff, "do handoff", "haz handoff", "wrap up the session", "session handoff", "close out the session". Updates the MEMORY.md continuity block, then vault hygiene (capture lessons), bitácora board-status reconciliation, repo/worktree/branch state verification, an artifact/PR summary, and a concrete next action. Cross-agent via AGENTS.md indirection.
allowed-tools: [Bash, Read, Edit, Write, mcp__hive__vault_query, mcp__hive__vault_search, mcp__hive__vault_write, mcp__hive__vault_patch]
---

# Handoff Workflow

> A complete, repeatable session handoff. Any agent, any session: "do handoff" runs the SAME checklist, so the next session (or a different agent) resumes with zero context loss.
> **Core principle:** the handoff is a *checklist*, not just a memory write. The always-on trigger ("at session end, run the handoff") lives in `AGENTS.md`; this skill is the SSOT for WHAT the handoff does.

## When to use

- `/handoff` explicitly, or "do handoff" / "haz handoff" / "wrap up the session".
- Automatically at the END of any session where meaningful work was done (per the `AGENTS.md` trigger).

## When to SKIP

Skip when the session left no durable state worth carrying forward — any of:
- A quick question or explanation; no files changed.
- A syntax / docs lookup, or a single-line / trivial edit with nothing committed.
- No new decision, no open thread, no PR or issue touched.

If in doubt, run it — a thin handoff is cheaper than a lost thread. (Mirrors the `MEMORY.md` "skip if trivial" rule.)

## The handoff checklist (run in order)

### 1. Continuity block — `## Session Handoff` in `MEMORY.md`

Maintain exactly ONE `## Session Handoff` block with these fields, in this exact order:

- `> Updated: YYYY-MM-DD`
- `**Last task:** [what was worked on, with PR/commit refs]`
- `**Decisions:** [durable decisions, or "None"]`
- `**Open threads:** [unfinished work + who owns each, or "None"]`
- `**Next action:** [concrete first step for the next session]`

Rules: dense but bounded (~8 lines — handoff, not journal); convert relative dates to absolute. **End each handoff field line with two trailing spaces** (markdown hard break) so the fields render as separate lines in Obsidian, not one crammed paragraph — vault-only convention (do NOT carry trailing spaces into code repos; pre-commit strips them). **Path resolution (MUST DO before writing):** The write target is `$VAULT_PATH/10_projects/<repo>/memory/MEMORY.md`. Resolve `$VAULT_PATH` via: (1) env var if set, (2) `dotf env path VAULT_PATH` if dotf is on PATH, (3) `~/.config/dotfiles/machine.json` `paths.VAULT_PATH`, (4) **FAIL** with instructions to set it. Never hardcode a literal path. If the target directory doesn't exist, create it (`mkdir -p`).

**Placement — cache-stable (HARNESS-029):** the block is the **LAST** section of `MEMORY.md`, *after* the stable index content. It changes every session; keeping it out of the auto-loaded KV-cache *prefix* stops it from busting the provider prompt cache on every new session. If you find it in a legacy position (e.g. the first section after the H1), relocate it to the end on this write.

**Write mechanics — concurrency-safe (HARNESS-028):** `MEMORY.md` is shared and unlocked — any session, any agent, can write it. To avoid a silent last-writer-wins clobber:
1. **Re-read `MEMORY.md` immediately before writing** — not the copy you read at session start.
2. **Replace only the block** — match-and-replace the existing `## Session Handoff` section using your tool's scoped/string-replace edit, never a full-file overwrite. A stale match then **fails loudly** instead of silently reverting the whole file.
3. **If the on-disk `> Updated:` marker changed** since that pre-write re-read, a concurrent session wrote the block — **merge** both sessions' threads into one block (or loud-fail then re-read), never overwrite blind.

This rule generalizes to **every** process that writes `MEMORY.md` (crystallize, auto-archive, any memory dispatcher), not only the handoff.

**Lifecycle (prune the block, accumulate the log):** there is exactly ONE `## Session Handoff` block — replace it in place (per the mechanics above); never add a second block or stack per-session blocks. The permanent, greppable per-session history lives in `sessions/` (step 1b), so the block stays a single latest-snapshot. Durable context that must outlive one session goes in its OWN named section *above* the handoff block (e.g. `## Older context`), keeping the volatile block last.

**Memory bridge (the vault path is the SSOT):** the write target above is the **vault** path — identical for every agent. If your agent keeps a local auto-memory mirror bridged onto the vault `…/memory/` dir and that bridge is missing or broken, write the **vault** path anyway AND repair the bridge per your agent's own setup — never write only to the agent-local copy. Agents with no such mirror write the vault path directly. (The bridge's concrete mechanics are an agent-specific detail; they live in that agent's overlay, not in this SSOT.)

### 1b. Session record — append-only history (`sessions/<date>-<project>-<agent>.md`)

In ADDITION to the replaced-in-place continuity block (step 1), write the **full session** as a new timestamped file so nothing is lost: `MEMORY.md` keeps only the *latest* handoff; this folder is the permanent, greppable history (ADR-014 + **MEMORY-003**).

- **Path:** `<project-area>/sessions/<YYYY-MM-DD>-<project>-<agent>.md` — the project's OWN vault folder (e.g. `10_projects/knowledge/sessions/`, `50_work/45-development/<sdk>/sessions/`), **never `00_meta/sessions/`** (per `feedback_sessions_in_project_folder.md`; already the convention in `ts-bridge`, `nan-video-pipeline`, `kubelab`, `iris`). Resolve the vault root via `$VAULT_PATH` — never a hardcoded literal.
- **Content:** frontmatter (`date`, `agent`, `project`, `session_id` if the runtime exposes one, `type: session`) + the same four handoff fields as step 1, but here you MAY be fuller than the ~8-line cap — this is the journal, not the snapshot.
- **Append-only:** one file per session. Never overwrite a prior session file; if a same-day file already exists, suffix `-2`, `-3`. (The `MEMORY.md` block is the only single-slot snapshot — replaced in place per step 1, never appended; this `sessions/` log is what accumulates. Satisfies the HARNESS-029 append-only-log requirement.)

**Mechanism (never skip): Hive-first, filesystem-fallback.** Prefer Hive `vault_write`; if Hive is unavailable/slow (the failure-mode protocol in `pattern-hive-first-vault-access`), fall back to a native filesystem write to the resolved `$VAULT_PATH/.../sessions/` path + a manual `vault:` commit. The session record is **never skipped** — not for a wedged Hive, a missing junction, or a sandbox denial.

### 2. Knowledge harvest & documentation sync (Standing Order #3 — in-session, never "later")

**Harvest sweep (systematic — not "capture if you remember").** Before closing, walk the WHOLE session and enumerate every piece of *durable* knowledge it produced, then route each to its home. The placement invariant decides the home: a repo's build/operate docs live in the **repo's `docs/`** (default for operate-layer artifacts), and only cross-project / methodology knowledge goes to the vault `00_meta/` (per [[pattern-knowledge-placement]]).

| Produced this session… | Lands in |
|---|---|
| A **project** lesson / gotcha / post-mortem | repo `docs/lessons.md` (Context / Problem / Solution / Tags) |
| A **cross-project / methodology** lesson | `00_meta/` → promote to a `00_meta/patterns/` pattern |
| An **architectural decision** | repo `docs/adr/adr-NNN-*.md` (cross-cutting → the platform repo) |
| A **runbook** (a procedure run more than once) | repo `docs/runbooks/`; cross-project procedure → `00_meta/runbooks/` |
| A **troubleshooting** entry (a fix worth recalling) | repo `docs/troubleshooting/` |
| A **discovery** (non-obvious fact about the system) | the doc it belongs to (README / the relevant ADR or runbook) — not a new file |

Rules for the sweep:
- **Sweep, don't cherry-pick.** Ask explicitly, genre by genre: *did this session produce a lesson? an ADR? a runbook? a troubleshooting note? a discovery?* Place each, or consciously decide it is not durable. A real artifact left unplaced is the gap this step closes (it generalizes the lessons-only gate of HARNESS-024 to every operate-layer genre).
- **Task state is NOT knowledge** — it lives on the bitácora board (ADR-018), reconciled in step 2b, not here.
- **Don't proliferate** — patch the existing doc for that genre; create a new file only when no home exists ([[feedback_no_doc_proliferation]]).
- **Foreign repo → worktree.** Landing a doc in another repo's `docs/` follows that repo's commit/PR discipline via a sibling worktree ([[feedback_worktree_for_foreign_repos]]); never write a non-current repo in place.

**Repo documentation (keep it reflecting the latest state):**
- If the session changed behavior, structure, commands, public contracts, or setup, update the repo docs that describe them: `README.md` and the repo's `docs/` (ADRs, runbooks, troubleshooting) for repos on the knowledge-placement model. ADRs in this repo live in `docs/adr/`.
- **Scope = this session's deltas, NOT a full audit.** A complete README/docs reconciliation is a dedicated task (e.g. an `AUDIT-*` ticket), not something every handoff redoes — the goal here is simply that docs do not drift behind the code the session just changed.

### 2b. Bitácora status reconciliation (Standing Order #8 — HARNESS-010)

Every issue you touched this session must reflect reality on the board ([Project #1](https://github.com/users/mlorentedev/projects/1)) — **none left in `Backlog` while actively worked**:

- **Worked, still open →** `In Progress` (self-assigning fires `bitacora-status.yml`; if you never assigned it, self-assign now or set the field by hand) — or `Blocked` with the blocker named in an issue comment.
- **Finished, closed →** the built-in workflow sets `Done` on close; confirm it landed.

Quick audit of this session's issue numbers (`N1,N2,…`):

```bash
gh issue list --repo mlorentedev/dotfiles --state open --json number,title,assignees --jq '.[] | select(.number == N1 or .number == N2) | "\(.number) \(.title) assigned:\(.assignees | length)"'
```

Or manually check each issue: `gh issue view <N> --json state,assignees,title`.

Leaving an actively-worked issue in `Backlog` is the exact gap HARNESS-010 closes. Mechanics: [[bitacora-project-setup]] §5.

**Best-effort, never a blocker:** if the `gh project` query is unavailable (auth, permissions, or a sandbox denial), do NOT stall the handoff. Instead make sure each touched issue is at least self-assigned or carries a status comment, and record the board change you couldn't apply in **Open threads** so the next session (or Manu) finishes it.

### 3. Repo / worktree / branch state

- No uncommitted work left dangling — or explicitly named in **Open threads** (never silently).
- Worktrees for MERGED PRs removed; their branches deleted; `git fetch --prune` gone refs.
- Every PR opened this session linked (number + merged/open state) in the handoff.

### 3b. Housekeeping (active cleanup — run for every repo touched this session)

Remove transient clutter before closing:

- **Stale branches (local):** follow the local-side procedure in [[pattern-github-branch-hygiene]] — do not improvise a recipe here. In short: `git fetch --prune`, take the `': gone]'` branches, **verify each against a merged PR**, then `git branch -D` (required, because squash-merge means the tip is never an ancestor). Never touch a branch that is checked out, backs a worktree, or has an open PR.
  > **Do not use `git branch --merged` / `-d`.** Squash-merge is the repo standard, so `--merged` lists nothing and `-d` refuses every branch worth deleting — the step becomes a silent no-op. This skill prescribed exactly that until 2026-08-06, which is why 18 stale branches accumulated across three repos while the handoff ran regularly. The pattern is the SSOT; this bullet only points at it.
- **Remote gone refs:** `git fetch --prune` on every repo touched. Removes stale remote-tracking refs for branches deleted upstream.
- **Done worktrees:** any worktree whose PR is merged must be removed (`git worktree remove <path>`). If not yet merged, name it in **Open threads** with PR number.
- **Temp / scratch files:** inspect `git status` for untracked `.bak`, `*.tmp`, scratch notes. **Never delete a file without explicit user confirmation** — list the candidates and ask; never `git clean -f`.
- **Empty vault stubs:** if any vault file is frontmatter-only with no real content, flag it (with its inbound links) and confirm before removing. Scope: only paths touched this session.

Scope: only repos and vault paths actually touched in this session — not a global cleanup pass.

### 3c. Context refresh (conditional — `/context-refresh`)

If this session **wrote an ADR, closed a phase milestone, pivoted direction, or changed the active focus**, run `/context-refresh <project>` for each affected project. It patches only the `context.md` frontmatter (`phase`, `focus`, `blocked_by`, `recent_adrs`, `last_updated`) so the next session orients in <400 tokens, and never touches the stable body. **Skip** when the session only changed task state — that lives in the bitácora, not `context.md`. See [[context-refresh]] (HARNESS-006).

### 4. Artifact summary

List what the session produced: PRs (number + state), key commit hashes, files created/changed, vault entries added/ticked.

**Verify before you assert (HARNESS-011):** every commit hash and PR you list must resolve — `git cat-file -e <hash>^{commit}` for each hash, `gh pr view <n>` for each PR. An artifact that does not resolve is reported as **uncommitted WIP** under Open threads, never stated as done. (A 2026-06-06 handoff claimed a commit `83c9609` that was never committed; this gate closes that false-positive.)

### 5. Next action

One concrete first step for the next session — actionable, not vague ("merge #185 then build X", not "continue").

### 6. Completion gates (verification-before-completion)

- **Outcomes:** tests/lints green; nothing claimed done-when-not. Anything incomplete belongs in **Open threads** with its owner — report faithfully. (Artifact existence is gated in step 4.)
- **Lessons delta (HARNESS-024):** if this session produced a fix, correction, or post-mortem, confirm a matching `docs/lessons.md` entry was written *this session*. If it is missing, capture it now (Standing Order #3) or name the omission in **Open threads** before completing — never let a real lesson go uncaptured. (Session-start surfaces `docs/lessons.md` staleness per touched repo; that signal lives in the session-start hook, not this skill.)
- **Behavioral-rule → `AGENTS.md` (HARNESS-009):** if this session established a new cross-agent behavioral rule — the user corrected how the agent should work, or you recorded a new agent-memory rule (e.g. a `feedback_*.md` entry) — surface a proposed `AGENTS.md` addition (target section + bullet) as a **code block for human review**. `AGENTS.md` is the cross-agent SSOT, so never auto-apply; point to the worktree PR that would carry it. Skip entirely when no new rule was established (zero noise in clean sessions).

## Cross-OS / cross-agent notes

- Step 1 targets the vault path (`$VAULT_PATH/10_projects/<repo>/memory/MEMORY.md`), writable by every agent. Resolve `$VAULT_PATH` per the cascade in step 1 — never assume.
- Steps 2-6 are agent-agnostic (vault + git).
- Path joining: never hardcode `/` or `\`; use platform-appropriate joining.
- **If the target directory doesn't exist:** create it (`mkdir -p`). Do not skip the continuity write.
- **Memory bridge (OS wrinkle):** the vault path is the SSOT (step 1). Where an agent bridges a local auto-memory mirror onto it, that bridge can be OS-specific (a Windows reparse point vs a POSIX symlink); its concrete mechanics belong to the agent's own overlay. Missing bridge → write the vault path AND repair it, per step 1.

## References

- Trigger (always-on): `AGENTS.md` — "at session end, run the handoff".
- Continuity-schema origin: a former agent-specific "Session Handoff" rule, consolidated here as the cross-agent SSOT; per-agent overlays only point to it (no duplication).
- Related: `MEMORY-001` (cross-agent session bridge), `00_meta/patterns/pattern-decision-persistence.md`, SDD-011 (the two-surface skill+trigger pattern this mirrors).
- Hardening folded in (2026-06-20): HARNESS-011 (#271, artifact verify · step 4), HARNESS-009 (#265, rule→AGENTS.md · step 6), HARNESS-024 (#387, lessons gate · step 6), HARNESS-028 (#432, concurrency-safe write · step 1), HARNESS-029 (#452, append-log + cache-stable order · steps 1/1b), HARNESS-025 (#390, noise/signal · skip-criteria + housekeeping scope). **Coordinated edit:** any per-agent overlay that hardcodes the block position ("the first section after the H1") must change to "the last section" in the same release (HARNESS-029).
