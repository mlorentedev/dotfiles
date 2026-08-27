---
generated: true
generated_from: 00_meta/skills/handoff/SKILL.md
generated_sha: c7f721197379048f
id: handoff-skill
type: skill
status: active
created: '2026-05-31'
owner: manu
name: handoff
description: Run a complete, standardized session handoff at session end (or on demand).
  Triggers on /handoff, "do handoff", "haz handoff", "wrap up the session", "session
  handoff", "close out the session". Updates the MEMORY.md continuity block, then
  vault hygiene (capture lessons), bitácora board-status reconciliation, repo/worktree/branch
  state verification, an artifact/PR summary, and a concrete next action. Cross-agent
  via AGENTS.md indirection.
allowed-tools: [Bash, Read, Edit, Write, mcp__hive__vault_query, mcp__hive__vault_search,
  mcp__hive__vault_write, mcp__hive__vault_patch]
keywords: [handoff, wrap up, session handoff, close session, cerrar sesion, haz handoff]
paths: ['**/MEMORY.md', sessions/**]
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

If in doubt, run it — a thin handoff is cheaper than a lost thread.

---

## The Handoff Checklist (Run in Order)

### 1. Continuity block — `## Session Handoff` in `MEMORY.md`

Maintain exactly ONE `## Session Handoff` block with these fields, in this exact order:

- `> Updated: YYYY-MM-DD`
- `**Last task:** [what was worked on, with PR/commit refs]`
- `**Decisions:** [durable decisions, or "None"]`
- `**Open threads:** [unfinished work + who owns each, or "None"]`
- `**Next action:** [concrete first step for the next session]`

**Rules:**
- **Dense & Bounded:** ~8 lines. Convert relative dates to absolute.
- **Markdown hard breaks:** End each handoff field line with two trailing spaces (`  `) for clean rendering in Obsidian.
- **Path resolution (Target Repo vs Session CWD, HARNESS-066):** Target is `$VAULT_PATH/10_projects/<target-repo>/memory/MEMORY.md`. The `<target-repo>` MUST be resolved from the **repositories actually modified/worked during the session** (via git remotes and worktrees touched), NOT blind session `cwd`. If multiple repos were modified, write the continuity block to the primary worked repo and record an explicit cross-pointer in the others. Resolve `$VAULT_PATH` via: (1) env var, (2) `dotf env path VAULT_PATH`, (3) `~/.config/dotfiles/machine.json` `paths.VAULT_PATH`, (4) Fail closed if unset. Never hardcode literal paths.
- **Independence caveat:** Never leak full design rationale or spec argumentation into a repo's auto-memory that would bias an independent `/adversarial-review` (write a pointer and task ref instead).
- **Placement (Cache-stable, HARNESS-029):** The block is the **LAST** section of `MEMORY.md`, *after* the stable index content. Keeping volatile handoff text out of the prefix prevents prompt cache thrashing.
- **Write mechanics — RUN THE COMMAND, do not Edit the file (HARNESS-088, #1278):**

  ```bash
  dotf mem thread                       # which session am I: wt-<slug>, or main
  printf '%s' "$BODY" | dotf mem handoff-write --memory "$MEMORY_MD"
  ```

  The section holds **one sub-block per thread**, keyed by worktree
  (`### wt-pi-harness`). The command replaces only *your* thread and leaves every
  other byte-identical.

  **This replaces HARNESS-028's merge-by-hand rule because that rule kept
  losing.** It said "if concurrent writes occurred, merge threads", and it was
  violated three times in one evening — by three different sessions, none of
  which noticed. Last-writer-wins produces a well-formed file and a successful
  edit, so the failure is invisible to the writer and only surfaces when a later
  session follows a pointer into a block that no longer exists. One shared
  mutable slot with N concurrent writers is a data-structure problem; editing it
  more carefully was never going to fix it.

  End your thread with `Journal: sessions/<file>` so the record is reachable from
  the index.

### 1b. Session record — append-only history (`sessions/<date>-<project>-<agent>-<thread>.md`)

In ADDITION to the replaced-in-place continuity block, record the session journal:

- **Path:** `<target-repo-area>/sessions/<YYYY-MM-DD>-<target-repo>-<agent>-<thread>.md`, where `<thread>` is what `dotf mem thread` prints. Ask for the whole name rather than assembling it:

  ```bash
  dotf mem thread --date "$(date +%F)" --project <repo> --agent claude
  ```

  **The thread is in the name because the collision it prevents already
  happened.** Naming was `<date>-<project>-<agent>.md`, so two *worktrees* on one
  day collided into `-2` and `-3` suffixes encoding nothing — six such files
  exist across two days, and no session could tell which one was its own. With
  the worktree in the name it is derivable from the working directory.

  Under `10_projects/<target-repo>/sessions/`, derived from the repository where
  work was executed. Never under `00_meta/sessions/`.
- **Frontmatter Law:**
  ```yaml
  ---
  id: session-YYYY-MM-DD-<project>-<agent>
  type: session
  status: active
  created: "YYYY-MM-DD"
  owner: manu
  tags: [session, <project>]
  ---
  ```
  *(Emit `id`, `type`, `status` as the first three keys for deterministic `vault_health` validation).*
- **Content:** Frontmatter per above + journal body sections:
  - `## Context & Objectives`
  - `## Work Completed` (with issue, PR, commit, and artifact links)
  - `## Decisions`
  - `## Next Actions`
  *(Unlike the ~8-line continuity block snapshot in step 1, the session record is the durable append-only journal).*
- **Append-only:** One file per session, and the thread in the name keeps concurrent
  sessions apart. A same-day `-2`/`-3` suffix is now only for a genuine SECOND
  SITTING in the SAME worktree — never for two worktrees, which is what those
  suffixes were silently absorbing before.

### 1c. Shared surfaces — write via the owning command only

A worktree isolates the repository tree, and the Bash sandbox already refuses
cross-worktree git. What it does **not** isolate is a short, knowable list. With
several sessions running at once, these are the places two of them can collide:

| Surface | Write it via |
|---|---|
| `MEMORY.md` handoff | `dotf mem handoff-write` (never an Edit) |
| Vault `sessions/` | the thread-scoped filename from `dotf mem thread` |
| `~/.dotfiles` deploy dir | `setup-linux.sh` / `dotf deploy` |
| `~/.claude/settings.json` and peers | the harness deploy path, which merges by marker |
| The git stash | never bare `git stash` — see the standing order |

Everything else a session touches lives inside its own worktree or its own
per-session state. No lock file and no scheme: the fix for each row above is that
one command owns the write, and the list is short enough to state.

### 2. Knowledge harvest & documentation sync (Standing Order #3)

Systematically sweep the session for durable knowledge and route each artifact:

| Produced this session… | Destination |
|---|---|
| A **project** lesson / gotcha / post-mortem | repo `docs/lessons/lesson-NNN-<slug>.md` (+ update `docs/lessons/_index.md`) |
| A **cross-project / methodology** lesson | `00_meta/patterns/` (promote to pattern) |
| An **architectural decision** | repo `docs/adr/adr-NNN-*.md` |
| A **runbook** (procedure run >1x) | repo `docs/runbooks/` (cross-project → `00_meta/runbooks/`) |
| A **troubleshooting** entry | repo `docs/troubleshooting/` |
| A **system discovery** | Relevant README section or existing ADR/runbook |

- **Repo docs sync:** If commands, behavior, or architecture changed, update `README.md` and repo `docs/` in-session.
- **Foreign repo edits:** Land changes via isolated sibling worktree (`../<repo>-wt-<slug>`).

### 2b. Bitácora status reconciliation (Standing Order #8)

Every issue touched this session must match board reality ([Project #1](https://github.com/users/mlorentedev/projects/1)):

- **Worked, still open →** Set to `In Progress` (self-assign issue), or `Blocked` with a comment naming the blocker.
- **Finished, closed →** Verify `Done` status. Confirm open PRs include `Closes #N` / `Fixes #N` keywords.
- **Audit command:**
  ```bash
  gh issue view <N> --json state,assignees,title
  ```
- *Best-effort:* If `gh project` query fails due to sandbox/auth, note the pending status in **Open threads**.

### 3. Repo / worktree / branch state & housekeeping

1. **Verify Git clean state:** No dangling uncommitted work unless explicitly listed in **Open threads**.
2. **Audit Worktrees:**
   ```bash
   git worktree list
   # For worktrees associated with merged PRs:
   git worktree remove ../<repo>-wt-<slug>
   ```
3. **Prune Remote Refs:** Run `git fetch --prune` on all touched repos.
4. **Delete Merged Local Branches:**
   ```bash
   # Identify gone branches from merged PRs:
   git branch -vv | grep ': gone]'
   # Delete verified merged branch:
   git branch -D <branch-name>
   ```
   *(Never delete branches with active PRs or backing existing worktrees).*

### 3b. Context refresh (conditional — `/context-refresh`)

If the session wrote an ADR, closed a phase milestone, or changed project focus, run `/context-refresh <project>` to update `context.md` frontmatter (`phase`, `focus`, `recent_adrs`, `last_updated`). Skip for tactical task changes.

### 4. Artifact verification (HARNESS-011)

Verify all produced artifacts before asserting completion:
- Verify commit hashes: `git cat-file -e <hash>^{commit}`
- Verify PR states: `gh pr view <N>`
- Unpushed or uncommitted items must be logged as **uncommitted WIP** in Open threads, never claimed done.

### 4b. PR review triage (Definition of Done §4 & pr-stewardship)

Before concluding any session touching or creating PRs, verify that no open PR is awaiting a disposition:

```bash
dotf pr triage-queue
```

- **Exit 0 (`[OK] no reviewer output is awaiting a disposition`):** Queue is clear, proceed.
- **Exit 1 (PRs listed awaiting disposition):** You MUST run `/pr-review-triage` (or post the `## Review triage` table on each listed PR) before claiming the session or PR is finished. Never leave reviewer findings or unrecorded empty reviews floating.

### 5. Next action

Formulate exactly ONE concrete, actionable first step for the next session (e.g. "Review and merge PR #1048, then run `dotf deploy`").

### 6. Completion gates

- **Tests / Lints:** 100% green before concluding.
- **Lessons Delta:** Every bug fix or non-obvious solution captured in `docs/lessons/lesson-NNN-*.md` + `_index.md`.
- **Behavioral Rule Proposal:** If the user gave a behavioral correction, output a proposed addition to `AGENTS.md` as a code block for human review.

---

## Response Format (Chat Summary)

Deliver the final handoff summary to the user using this standardized block:

```markdown
### 🏁 Session Handoff Summary

- **Continuity Block:** Updated in `$VAULT_PATH/10_projects/<repo>/memory/MEMORY.md`
- **Session Record:** Written to `10_projects/<repo>/sessions/<YYYY-MM-DD>-<project>-<agent>.md`
- **PRs & Commits:**
  - `PR #NNN`: `<Title>` (`<state>`)
  - `Commit <hash>`: `<message>`
- **Knowledge & Lessons:** `docs/lessons/lesson-NNN-*.md` (or "None this session")
- **Bitácora Board:** Issue `#N` -> `<Status>`
- **Worktree / Branch State:** Clean / Pruned (`<details>`)
- **Next Action:** `<Concrete first step for next session>`
```
