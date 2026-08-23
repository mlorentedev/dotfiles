---
generated: true
generated_from: 00_meta/skills/catchup/SKILL.md
generated_sha: 93ee453629e40867
id: catchup-skill
type: skill
status: active
created: "2026-08-23"
owner: manu
name: catchup
description: Instant session resumption and Senior SRE orientation briefing.
  Audits memory continuity, live git/worktree state, PR review triage queue, bitácora
  board issues, architectural context, and technical debt. Enforces zero debt, IaC
  idempotency, and outputs an executive briefing with ONE clear next action.
  Triggers on /catchup, /briefing, /standup, /next-action, "catch me up",
  "dame briefing", "estado de la sesion", "que sigue", "donde nos quedamos", "orientacion".
allowed-tools: [Bash, Read, Grep, mcp__hive__session_briefing, mcp__hive__vault_query]
keywords: [catchup, briefing, standup, session start, next action, catch me up, where were we, que sigue, orientacion]
paths: ["**/MEMORY.md", "**/context.md", "docs/adr/**"]
---

# /catchup — Senior SRE Session Briefing & Resumption

> Audits cognitive memory and ground-truth runtime state to deliver a zero-overhead, high-signal executive briefing with exactly ONE prioritized next action.

## When to Use

- At the beginning of any interactive or autonomous agent session (`/catchup`, `/briefing`, `/standup`, `/next-action`).
- When returning from context compaction or switching between tasks.
- Whenever context drift or ambiguity arises ("what should I do next?").

---

## The 6-Step Resumption Protocol

### Step 1: Memory Continuity Ingestion
1. Resolve target repo from working directory or explicit parameter.
2. Read `$VAULT_PATH/10_projects/<repo>/memory/MEMORY.md` (target the `## Session Handoff` block at EOF).
3. Extract:
   - `Last task` (commits, PRs touched)
   - `Decisions` (architectural rulings)
   - `Open threads` (in-flight items)
   - `Recorded next action`

### Step 2: Live Git & Worktree Audit (Ground Truth)
Run local git verification:
```bash
# 1. Active branch & dirty working tree
git branch --show-current
git status --short

# 2. Audit worktrees (detect orphan or merged trees)
git worktree list

# 3. Prune remote refs and detect gone branches
git fetch --prune
git branch -vv | grep ": gone]" || true

# 4. Open PRs authored by agent/user
gh pr list --author @me --state open --json number,title,headRefName,isDraft,mergeable
```

### Step 3: PR Review Triage Queue Check (DoD §4)
Verify whether any open PR requires triage before starting new feature code:
```bash
dotf pr triage-queue
```
- **Exit 0:** Queue is clear.
- **Exit 1:** Un-triaged reviewer output pending. Triaging this PR takes precedence over new work.

### Step 4: Bitácora Board Audit (GitHub Project #1)
Query active tasks on the project board:
```bash
gh project item-list 1 --owner mlorentedev --format json --limit 100 | python3 -c "
import json, sys
items = json.load(sys.stdin).get("items", [])
active = [i for i in items if i.get("status") in ["In Progress", "Blocked"]]
for i in active:
    c = i.get("content", {})
    print(f"[{i.get('status').upper()}] #{c.get('number')} - {c.get('title')} (ID: {i.get('iD', '-')}, Priority: {i.get('priority', '-')})")
"
```

### Step 5: Architectural Context & Zero-Debt Audit
1. Read `10_projects/<repo>/context.md` frontmatter (`phase`, `focus`, `blocked_by`, `recent_adrs`).
2. **Two-Exit Debt Rule:**
   - Minor defect / smell in scope (<50 LOC) $\\to$ fix immediately with guard/test.
   - Non-trivial defect / out-of-scope $\\to$ file ticket immediately via `/new-ticket` on Project #1.
3. **Idempotency Gate:** Confirm any infrastructure/config change re-runs cleanly with `changed=0`.

### Step 6: Executive Synthesis (The Rule of One)
Deliver the structured briefing below, concluding with **exactly ONE unambiguous next step**.

---

## Output Template: Executive Briefing

Deliver a structured, high-signal briefing matching this exact template:

```markdown
# 🧭 Catchup Briefing — <repo>
**Health Status:** 🟢 READY | 🟡 BLOCKED / TRIAGE PENDING | 🔴 SYSTEM DEGRADED

### 1. Memory Continuity
- **Last Session:** <Summary of last completed task & commits>
- **Key Decisions:** <Durable constraints established>
- **Open Threads:** <In-flight items requiring closure>

### 2. Live Runtime & Git Reality
- **Branch / Status:** `<branch>` (<clean | N modified files>)
- **Active Worktrees:** `<count>` active (`<list of paths>`)
- **Open PRs:**
  - `PR #NNN`: `<Title>` — [CI: `<status>` | Review: `<triaged | pending>`]
- **Triage Queue:** `dotf pr triage-queue` → **<CLEAR (Exit 0) | PENDING (Exit 1)>**

### 3. Board & Architecture State
- **Bitácora In-Progress:** Issue `#N` (`<AREA-NNN-slug>`)
- **Project Phase / Focus:** `<phase>` → `<focus>`
- **Blockers:** <None | Description of blocker>

### 4. Technical Debt & Doctrine Watch
- **Debt Findings:** <None detected | N items ticketed/triaged>
- **SRE Guardrails:** IaC Idempotent (`changed=0`), Zero AI Attribution, Atomic PR Cap (<300 LOC).

---

### 🎯 Immediate Next Action (The Rule of One)
> **`<Single, clear, unambiguous step to execute right now>`**
```bash
<Exact command to run or file to edit>
```
```
