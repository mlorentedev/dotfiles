---
generated: true
generated_from: 00_meta/skills/autopilot/SKILL.md
generated_sha: 1aee3400a0436b9b
id: autopilot-skill
type: skill
status: active
created: '2026-10-14'
owner: manu
name: autopilot
description: Use when the operator announces an absence (sleeping, working, eating, stepping away, workouts, AFK) and leaves sessions in pilot mode, or when explicitly switching an agent to unattended backlog execution. Covers autonomous ticket checkout, advisor escalation, deterministic worktree isolation, in-flight documentation, and pre-authorized PR landing under strict conditions. Not for interactive pairing sessions.
keywords: [autopilot, pilot-mode, pilot mode, modo piloto, unattended, afk mode, autonomous loop, unattended session, sesiones en modo piloto, ausentarme, dormir, comer, trabajar, tiempo fuera, coge otro ticket, I'll be away, keep working while I'm out, work the board]
paths: []
requires: [using-git-worktrees, handoff, pr-review-triage, test-driven-development]
---

# Autopilot — Autonomous Unattended Developer Loop

> The operator announces an absence (sleeping, working, eating, stepping away, workouts — any bounded time away). Sessions keep working the backlog autonomously: check out tickets, implement with the repo's engineering standards, document in-session, land PRs under the standing grant, and leave a clean state plus a report for the operator's return.
>
> **Core Invariant:** Autonomy is granted by scope and bounded by deterministic machinery (Bitácora issue claims, physical Git worktrees, green CI on head SHA, branch protection, the merge-grant file, and circuit breakers) — never by asking the model to "be careful". The operator's trust is spent once at activation; the protocol must not spend more than was granted.

---

## The One-Line Contract

When the operator says (any variant): *"Me voy a ausentar / a dormir / a trabajar / a comer, dejo las sesiones en modo piloto. En caso de duda consultad al advisor y si acabas la tarea coge otro ticket de la misma naturaleza o del mismo epic y ve documentando, haciendo handoff, lessons, adr y demas... Tienes permiso para mergear PRs que abras con 4 condiciones: te hayas coordinado con las otras sesiones para no entorpeceros, CI está verde, no hay conflictos, y comentarios de reviewers están atendidos."*

...this skill converts that announcement into a safe, bounded, unattended work loop. The operator's sentence IS the grant; nothing in it may be widened.

---

## 1. Activation & Cold-Start Ritual (Run Once at Session Start)

1. **Tooling & CLI Health:**
   ```bash
   dotf doctor --quick
   gh auth status
   ```
2. **Clean Base Tree & Cold-Start Test:**
   ```bash
   git status --short
   git pull --rebase
   ```
   *Cold-start verification:* Run the repo's test suite or verify script on the base tree before making changes. If tests fail before you write code, record the pre-existing defect in your session journal — never mistake an inherited failure for your own regression.
3. **Check the PR Triage Queue:**
   ```bash
   dotf pr triage-queue
   ```
   If pending review items exist on existing PRs, disposing them takes precedence over picking new work.
4. **Establish Run Budget & Boundaries:**
   - **Ticket Limit:** Maximum 3–5 tickets per unattended session (unless explicitly given another limit).
   - **Per-Ticket Time Budget:** 30 minutes max. If a ticket exceeds 30 minutes, trip the circuit breaker (§4.2).
   - **Token / Turn Budget:** Set `--max-turns` or harness spend limit if available.
   - **Ask Degrades to Deny:** In unattended mode, any ambiguity, missing input, or interactive prompt degrades to **Deny / Escalate / Blocked**, never to silent approval ("idle is not authorization").

---

## 2. Ticket Selection & Eligibility (Decide BEFORE Claiming)

A ticket is pilot-eligible only if ALL hold:

- **The 3 a.m. Test:** *If this runs while the operator is away and does exactly what the ticket literally says, is the operator comfortable with the result unreviewed?* If no, leave an issue proposal or comment instead of executing, and move on.
- **A Plan Already Exists:** Spec, task list, or mechanical nature (bug fix <20 lines with obvious cause, tests, docs, small refactor). Never invent an architectural plan unattended.
- **Same Epic / Nature Continuity:** Select unassigned tickets belonging to the same active epic or domain:
  ```bash
  gh issue list --search "is:open is:issue no:assignee sort:created-asc" --limit 10
  # Or filter by current epic label:
  gh issue list --search "is:open is:issue no:assignee label:<epic-label>" --limit 5
  ```
- **No Path Contention:** Check open PRs and branches (`gh pr list --state open`, `git branch -r`) to ensure no peer session is editing the same files/subsystems.
- **Atomic Claim:** Self-assign the ticket to lease it:
  ```bash
  gh issue edit "$ISSUE_NUM" --add-assignee "@me"
  gh issue comment "$ISSUE_NUM" --body "Picked up in autonomous pilot mode. Branch: feat/<slug>"
  ```
  *Note:* Self-assignment updates the Bitácora status machine to `In Progress`.
- **One Ticket at a Time:** Never multitask across multiple tickets in parallel within the same session.

---

## 3. The Autonomous Execution Loop (Per Ticket)

### Step 1: Worktree Isolation & Idempotent Checkout

Each ticket MUST run in an isolated sibling worktree per `using-git-worktrees`.
Check if the worktree already exists (crash recovery / resumability):

```bash
SLUG="issue-${ISSUE_NUM}-<short-topic>"
BRANCH="feat/${SLUG}"
REPO_ROOT="$(git rev-parse --show-toplevel)"
WT_DIR="../$(basename "$REPO_ROOT")-wt-${SLUG}"

if [ -d "$WT_DIR" ]; then
  cd "$WT_DIR"
  git pull --rebase origin "$BRANCH" 2>/dev/null || true
else
  dotf worktree add "$SLUG" -b "$BRANCH"
  cd "$WT_DIR"
fi
```

### Step 2: Idempotent Implementation & Search-Before-Create

- **Search Before Create:** Before writing a helper, script, or module, search the codebase (e.g. `rg "<pattern>"`) to verify it doesn't already exist. Duplicate code is a leading failure mode of autonomous loops.
- **TDD & Small Diffs:** Write the failing test first, make it pass, adhere to the ~300 LOC cap.
- **Zero Manual Operations:** Never perform ad-hoc manual mutations on remote systems or servers. All configuration must be reproducible, idempotent IaC (`changed=0` on re-run).
- **In-Flight Documentation (Zero Debt):**
  - Record lessons in `docs/lessons/` (run `dotf lessons fmt` or update `docs/lessons/_index.md`).
  - Record ADRs in `docs/adr/` if making an architectural or non-obvious design decision.

### Step 3: Machine-Checkable Verification Gate

Run the deterministic test/lint suite. The only accepted proof of progress is command output, never self-report:

```bash
# Language/repo-specific verify command:
make verify # or npm test, go test ./..., cargo test, pytest
```

If verification passes, proceed. If verification fails, engage the Fix-Loop Breaker (§4.2).

### Step 4: PR Creation

Push the branch and open a draft PR:

```bash
git push -u origin "$BRANCH"

PR_URL=$(gh pr create --draft --title "feat(<scope>): <short-description>" \
  --body "Closes #${ISSUE_NUM}

## Summary
<description>

## Verification
Evidence:
\`\`\`
$(make verify 2>&1 | tail -15)
\`\`\`

## Knowledge
- Lesson: docs/lessons/lesson-<num>-<slug>.md
- ADR: none: <reason>
- Runbook: none: <reason>")

PR_NUM=$(echo "$PR_URL" | grep -oE '[0-9]+$')
echo "Opened PR #${PR_NUM} at ${PR_URL}"
```

*Note:* Opening as a draft prevents reviewer noise while checks run. Mark ready when checks stabilize: `gh pr ready "$PR_NUM"`.

---

## 4. Advisor Escalation & Circuit Breakers

### 4.1 When to Escalate to the Advisor

Escalate immediately upon:
- Unresolvable architectural forks or ambiguity.
- Uncovered security/secrets boundaries.
- Repeated failures tripping the 3x breaker.

**Harness-Agnostic Advisor Invocations:**
- **Cross-Agent / CLI:**
  ```bash
  dotf agent run --role architect --tier top --task "Evaluate architectural crossroads for #${ISSUE_NUM}: [Context and Options]"
  ```
- **Claude Code:** Invoke the experimental advisor tool:
  ```json
  advisor({ "prompt": "Review architecture and unblock #${ISSUE_NUM}: [Context]" })
  ```
- **Antigravity / AGY:** Dispatch a high-thinking subagent (`chat-base` or `research`).

### 4.2 The 3x Fix-Loop Breaker & Parking Tickets

- **Max 3 Fix Iterations:** If the same test or linter failure recurs across 3 rounds, **STOP grinding**.
- **Park the Ticket Cleanly:**
  1. Add a detailed root-cause comment:
     ```bash
     gh issue comment "$ISSUE_NUM" --body "Blocked after 3 fix iterations. Root cause: <details>. Tried: <attempts>. Requires operator attention."
     ```
  2. Move the Bitácora project card to `Blocked`:
     ```bash
     dotf ticket update "$ISSUE_NUM" --status Blocked
     ```
  3. Record the ruling in your session journal and move to the next eligible ticket.

---

## 5. The Standing Merge Grant (4 + 2 Conditions)

An agent may merge a PR **it opened** ONLY when all six conditions hold on the **exact head commit** produced this turn:

| # | Condition | Verification Evidence Required This Turn |
|---|---|---|
| 1 | **Coordinated with Peers** | `git fetch --prune` + checking open PRs/branches for path overlap. No concurrent sessions touching identical files. |
| 2 | **CI is Green** | Every required check passes or is skipped on the **current head SHA** (`gh pr checks "$PR_NUM"`). An ancestor's green check is the *stale-green fallacy*. |
| 3 | **No Conflicts** | Merge state is `CLEAN` against current `main`/`master`. Rebase if behind. |
| 4 | **Reviewers Triaged** | All reviewer comments disposed under a `## Review triage` PR comment (applied, ticketed, or declined with documented reason). |
| 5 | **Grant in Force** | The operator authorized pilot mode AND the repository carries `.github/merge-grant.yml` on its base branch. |
| 6 | **Scope Unchanged** | Head SHA matches the reviewed commit. No unreviewed commits injected during conflict resolution. |

### Landing Execution:

```bash
# 1. Verify queue clear
dotf pr triage-queue

# 2. Land PR via deterministic CLI
dotf pr land "$PR_NUM"
```

### The Always-Human Floor (Exclusions):

**NEVER** merge or auto-land PRs that touch:
1. `.github/merge-grant.yml` (agents cannot widen their own authority).
2. `release-please` or version bump / tagging workflows.
3. CI workflows (`.github/workflows/`), security policies, or secret management configs.
4. CODEOWNERS paths requiring human approval.

*Rule:* If a PR touches any of these, leave it open for human review.

---

## 6. Exit Protocol & Pilot Report

When the operator returns, the time window expires, the backlog empties, or a stop condition triggers:

1. **Clean Worktree Hygiene:**
   ```bash
   cd "$REPO_ROOT"
   dotf worktree done "$SLUG"
   ```
   *(Never delete your own launch worktree if running inside one).*
2. **Multi-Session Memory Handoff:**
   Update your session thread under `## Session Handoff` in `MEMORY.md`:
   ```bash
   dotf mem handoff-write --agent <agent-name> --memory "$REPO_ROOT/MEMORY.md"
   ```
3. **Deliver the Pilot Report in Conversation:**
   ```markdown
   ### ✈️ Pilot Mode Session Report — [Absence Window]
   - **Tickets Completed:** #123, #124
   - **Tickets Parked (Blocked):** #125 (Root cause: 3x breaker tripped on db lock)
   - **PRs Landed:** #250 (SHA: `a1b2c3d`)
   - **PRs Awaiting Human Review:** #251 (Draft/Release boundary)
   - **Knowledge Recorded:**
     - Lesson: `docs/lessons/lesson-438-...md`
     - ADR: `docs/adr/0042-...md`
   - **Next Recommended Action for Operator:** Review #251, inspect Blocked #125.
   ```

---

## 7. The Stop List (What Stops You — and ONLY This)

1. **Irreversible Operations:** Production deploys, dropping tables/databases, deleting git history or branches outside your own worktree, force-pushing to trunk.
2. **Secrets Handling:** Never expose secrets to stdout or write plaintext credentials to files.
3. **Ask Degrades to Deny:** In unattended mode, any ambiguous requirement or interactive confirmation stops and escalates to the Advisor/Blocked.

---

## 8. Rationalization Table

| Excuse | Reality |
|---|---|
| "CI was green an hour ago" | Stale-green fallacy. Re-check the current head SHA. |
| "Reviewer comments are nitpicks" | Every comment must be applied, ticketed, or declined with documented rationale. |
| "It's a tiny fix, I don't need a worktree" | Worktrees prevent uncommitted file collisions between sessions. Always use a worktree. |
| "I'll document the lesson when the operator returns" | In-flight documentation is non-negotiable. Hygiene happens in-session. |
| "The test failed 4 times, but this next tweak will work" | Tripped breaker. Past 3 rounds, failure is structural. Park it. |
| "No assignees on the issue, so I don't need to assign myself" | Assignment is your lease claim for peer sessions. Always assign before working. |
| "I can merge this quick PR modifying CI" | The always-human floor. CI workflows and merge grants are never auto-merged. |

---

## 9. Red Flags — STOP Immediately

- You are about to merge with any of the 6 conditions unevidenced this turn.
- You are editing `.github/merge-grant.yml`, CI workflows, or release automation.
- You are in round 4 of a fix loop or widening the diff to force a pass.
- You opened a second ticket while the first is unfinished.
- You ran a command without an isolated git worktree.
- You cannot produce the command output that proves your completion claim.
