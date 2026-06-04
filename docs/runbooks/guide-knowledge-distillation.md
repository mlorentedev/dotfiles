---
id: "guide-knowledge-distillation"
type: runbook
status: active
tags: [ai-workflow, claude, knowledge-management, neural-hive]
owner: manu
created: "2026-03-28"
---

# Guide: AI Knowledge Distillation

How to keep the Neural Hive knowledge loop healthy, prevent token bloat, and ensure lessons propagate from ephemeral Claude sessions into durable vault documentation.

---

## Why This Matters

Claude Code sessions are stateless. Without active maintenance, valuable decisions and bug fixes accumulate in `claude-mem` observations and MEMORY.md but never reach the vault where they can be reused across sessions, projects, and team members.

**The cost of neglect:**
- Baseline token usage grows: each session loads ~20k tokens of stale context
- Repeated problem-solving: the same bugs get re-debugged because lessons aren't findable
- Onboarding friction: new team members can't access institutional knowledge

**The target state:**
- MEMORY.md: < 23 lines (user preferences + dynamic state only)
- Lessons: project-specific → repo `docs/lessons.md`; cross-project/methodology → vault `00_meta/` patterns
- 00_meta/patterns/: reusable patterns promoted from repeated lessons
- Baseline session tokens: ~7k (65% reduction from 20k)

> **Placement note:** the `10_projects/<repo>/90-lessons.md` (vault) references later in this guide are the **legacy** location. Per the knowledge-placement model, project lessons live in the repo `docs/lessons.md`; only cross-project/methodology lessons stay in the vault `00_meta/`. Realigning the distillation tooling (`crystallize`/`insights` skills, which are vault-SSOT) is tracked in `RFD-001-vault-placement-cleanup`.

---

## The Knowledge Loop

```
Claude Session
     │
     ▼
claude-mem observations (🔴 bugs, ⚖️ decisions, 🟣 features)
     │
     ▼ /crystallize promotes these
10_projects/<repo>/90-lessons.md  ◄── project-specific lessons
     │
     ▼ if lesson appears in >1 project
00_meta/patterns/pattern-<topic>.md  ◄── global reusable patterns
     │
     ▼ patterns inform
CLAUDE.md / ai-protocol.md  ◄── Claude's standing instructions
```

---

## Components

### `scripts/knowledge-crystallize.sh`

**What:** Automated MEMORY.md maintenance. Updates dates, warns on size, prints checklist.
**When:** After each sprint, or in a post-work automation.
**Cross-platform:** Bash/zsh on Linux. PowerShell equivalent for Windows: see [Windows section](#windows-support).

```bash
# Run for current project
./scripts/knowledge-crystallize.sh

# Run for a specific project
./scripts/knowledge-crystallize.sh ~/Projects/kubelab

# Auto-discover and process ALL projects from ~/.claude/projects/ (recommended)
./scripts/knowledge-crystallize.sh --all
```

The `--all` flag auto-discovers every project Claude Code has ever touched on this machine by scanning `~/.claude/projects/`. It decodes each project path and skips entries that don't exist on disk (stale entries from deleted projects or other machines).

### `/insights` skill

**What:** Read-only weekly audit. Reports unvaulted observations, MEMORY.md health, backlog status.
**When:** Weekly check-in. Takes ~2 minutes.
**Output:** Structured report with specific action items.

### `/crystallize` skill

**What:** Full AI-assisted crystallization ritual. Mines observations, updates vault, proposes patterns, trims MEMORY.md.
**When:** When `/insights` shows issues, or after completing a significant feature or sprint.
**Output:** N lessons added, N pattern proposals, MEMORY.md before/after line counts.

### `/vault-doctor` skill

**What:** Vault structural maintenance. Diagnoses and fixes unresolved links, missing frontmatter, orphan notes.
**When:** When `/insights` reports structural issues (unresolved links, frontmatter violations, orphans).
**Output:** Severity report + fixes applied + remaining items for manual review.
**Prerequisite:** Hive MCP (always available). obs-cli (optional, requires Obsidian GUI).

### `scripts/vault-maintenance-weekly.sh` / `.ps1`

**What:** Automated weekly maintenance. Runs `knowledge-crystallize.sh --all` + `vault-health.sh`, sends desktop notification with results.
**When:** Cron/Task Scheduler fires every Sunday 10:07 AM. Can also be run manually.
**Log:** `~/.local/share/vault-maintenance/latest.log` (Linux) / `%LOCALAPPDATA%\vault-maintenance\latest.log` (Windows).
**Deployed by:** `setup-linux.sh` (crontab) / `setup-windows.ps1` (Register-ScheduledTask).

### `scripts/claude-session-start.sh` / `.ps1`

**What:** SessionStart hook. Injects vault health context at the start of every Claude session. Also checks knowledge staleness and warns if MEMORY.md > 150 lines or Last Crystallized > 14 days. **Auto-creates missing memory symlinks/junctions** so new projects synced from another machine work immediately.
**Deployed by:** `setup-linux.sh` / `setup-windows.ps1` → registers in `~/.claude/settings.json`.

---

## Cross-Machine Memory Sync

### How It Works

Memory files live in the vault. Setup scripts create **symlinks** (Linux) or **junctions** (Windows) from Claude Code's internal path to the vault. Obsidian git plugin syncs the vault across machines.

```
Machine A: Claude writes MEMORY.md
              |  (symlink/junction)
              v
        ~/Projects/knowledge/<scope>/<project>/memory/MEMORY.md
              |  (Obsidian git sync, every 10 min)
              v
        Remote vault repo
              |  (Obsidian git pull on Machine B)
              v
        ~/Projects/knowledge/<scope>/<project>/memory/MEMORY.md
              |  (symlink/junction, auto-created by session hook)
              v
Machine B: Claude reads MEMORY.md
```

### Critical Assumptions

The entire sync system depends on two path conventions. If these don't hold, junctions/symlinks break silently and Claude works with local-only memory.

| Assumption | Expected Value | Used By |
|------------|---------------|---------|
| **Vault location** | `~/Projects/knowledge` | Setup scripts, session hooks, crystallize scripts |
| **Personal project repos** | `~/Projects/<name>` where `<name>` matches `10_projects/<name>/` | Setup scripts, session hooks |
| **Work projects CWD** | The vault path itself (`~/Projects/knowledge/50_work/.../<name>`) | Session hook fallback |
| **Obsidian running** | Git plugin active with auto-commit/pull | Cross-machine sync |

If the vault moves to a different path or projects don't follow the naming convention, update both setup scripts and both session hooks.

### What Happens Automatically

| Event | What Fires | Result |
|-------|-----------|--------|
| Run setup script | Scans `10_projects/*/memory/` and `50_work/**/memory/` | Creates all junctions/symlinks |
| Open Claude in a project | Session hook (`claude-session-start`) | Creates missing junction/symlink for that project |
| Claude writes MEMORY.md | Junction/symlink | Change lands directly in vault |
| Obsidian auto-commit (10 min) | Git plugin | Pushes vault changes to remote |
| Obsidian auto-pull (10 min) | Git plugin | Pulls vault changes from remote |

### What Requires Manual Action

| Situation | Action |
|-----------|--------|
| New machine | Clone vault + run setup script (once) |
| Username change (e.g. `Manu` to `mlorente`) | Re-run setup script (re-creates all junctions with new encoded paths) |
| Vault path moved | Update hardcoded `~/Projects/knowledge` in setup scripts and hooks, re-run setup |
| Obsidian not running | Open Obsidian (changes accumulate locally until sync resumes) |
| Want all junctions pre-created (not just current project) | Run `setup-windows.ps1` / `setup-linux.sh` |

### Failure Modes

| Symptom | Cause | Fix |
|---------|-------|-----|
| Claude creates local MEMORY.md, not in vault | Junction/symlink missing | Open Claude in that project (hook auto-creates) or re-run setup |
| Changes on Machine A don't appear on Machine B | Obsidian not syncing | Check Obsidian is running with git plugin on both machines |
| Junction exists but points to nothing | Vault project deleted or moved | Remove stale junction, re-run setup |
| `MEMORY.md not found` from crystallize script | Path encoding mismatch | Check `ls ~/.claude/projects/ \| grep <name>` matches expected encoding |
| Vault on OneDrive / network drive | Junctions require local paths | Move vault to local disk |

---

## Recommended Cadence

| Frequency | Action | Tool |
|-----------|--------|------|
| Every Sunday (automated) | MEMORY.md maintenance + vault health | `vault-maintenance-weekly.sh` (cron) |
| Weekly (manual) | Audit what needs attention | `/insights` (quick mode) |
| When insights shows knowledge gaps | Crystallize observations to vault | `/crystallize` |
| When insights shows structural issues | Fix links, frontmatter, orphans | `/vault-doctor` |
| Post-sprint / end of week | Deep audit + crystallize | `/insights full` then `/crystallize` |
| When lesson recurs in 2nd project | Promote to global pattern | `/crystallize` (auto-proposes) |

---

## MEMORY.md Health Guidelines

| Metric | Healthy | Warning | Action |
|--------|---------|---------|--------|
| Line count | < 100 | 100–150 | > 150: run `/crystallize` |
| Last Crystallized | < 7 days | 7–14 days | > 14 days: run `/crystallize` |
| Sections | User Prefs + CI + Bugs + Backlog | + Key Files (ok if brief) | If duplicating CLAUDE.md: trim |

**What belongs in MEMORY.md vs CLAUDE.md:**

| Content | Location |
|---------|----------|
| "Never commit directly" (user pref) | MEMORY.md |
| CI pipeline structure | MEMORY.md (dynamic state) |
| Bugs found this sprint | MEMORY.md (until vaulted) |
| Backlog status | MEMORY.md (dynamic) |
| Shell compatibility rules | CLAUDE.md (stable, instructional) |
| Key file paths | CLAUDE.md (stable reference) |
| Test commands | CLAUDE.md (stable reference) |

---

## Team Onboarding

### New machine (existing team member)

1. Clone the vault to `~/Projects/knowledge`
2. Run `setup-linux.sh` or `setup-windows.ps1` — deploys hooks, creates all memory symlinks/junctions
3. Install Obsidian and enable the git plugin (auto-commit + auto-pull)
4. Open any project in Claude Code — the hook fires automatically
5. Run `/insights` in the dotfiles project to verify the knowledge system is healthy

### New team member

1. Clone dotfiles and vault, run the setup script for your OS
2. Read `~/Projects/knowledge/00_meta/patterns/pattern-ai-protocol.md` — the master AI workflow governance doc
3. Read `~/Projects/knowledge/10_projects/<repo>/00-context.md` for each project you join
4. After your first sprint, run `/crystallize` to add your first vault lessons

### What to do when starting a new project

1. Create `~/Projects/knowledge/10_projects/<new-repo>/` with standard files:
   - `00-context.md` — project overview and AI config
   - `11-tasks.md` — active backlog
   - `90-lessons.md` — lessons learned
   - `memory/` — directory for Claude Code memory (create empty)
2. Run Claude Code in the new repo — the session hook auto-creates the symlink/junction to `memory/`
3. Run `./scripts/knowledge-crystallize.sh ~/Projects/<new-repo>` after your first session

> **Work projects** (`50_work/`): Same structure but under `50_work/<area>/<project>/`. The session hook detects CWD inside the vault and maps accordingly.

---

## Windows Support

Full parity with Linux. All components have PowerShell equivalents:

| Component | Linux | Windows |
|-----------|-------|---------|
| Setup | `setup-linux.sh` (symlinks) | `setup-windows.ps1` (junctions, no admin) |
| Session hook | `claude-session-start.sh` | `claude-session-start.ps1` |
| Crystallize script | `knowledge-crystallize.sh` | `knowledge-crystallize.ps1` |
| Weekly maintenance | `vault-maintenance-weekly.sh` (crontab) | `vault-maintenance-weekly.ps1` (Task Scheduler) |
| Memory link type | Symlink (`ln -s`) | Junction (`New-Item -ItemType Junction`) |
| Notification | `notify-send` | `System.Windows.Forms.NotifyIcon` |
| Skills (`/insights`, `/crystallize`, `/vault-doctor`) | Work unchanged | Work unchanged |

```powershell
# Run crystallize for current project
.\scripts\knowledge-crystallize.ps1

# Run for all projects
.\scripts\knowledge-crystallize.ps1 -All
```

**Windows-specific notes:**
- Junctions are bidirectional and require no admin privileges
- Junction removal uses `cmd /c rmdir` to avoid deleting target contents
- Path encoding: `C:\Users\Manu\Projects\dotfiles` becomes `C-Users-Manu-Projects-dotfiles`

---

## Token Budget Reference

| Component | Before cleanup | After cleanup | Savings |
|-----------|---------------|---------------|---------|
| MEMORY.md | ~400 tokens | ~150 tokens | ~250t |
| claude-mem index | ~16,000 tokens | ~7,000 tokens (after vaulting) | ~9,000t |
| CLAUDE.md | ~2,500 tokens | ~2,500 tokens | — |
| **Session baseline** | **~20,000 tokens** | **~10,000 tokens** | **~50%** |

The largest lever is keeping 90-lessons.md current so the claude-mem index stays small (fewer unprocessed observations).

---

## Verification: Post-Overhaul Testing

After running `setup-linux.sh` or `setup-windows.ps1`, verify the skills ecosystem is correctly deployed:

### 1. Skill Sync

```bash
# Source should have 17 skills
ls -1 ~/Projects/dotfiles/ai/skills/ | wc -l

# Deployed should match (no stale skills from previous versions)
ls -1 ~/.claude/skills/ | wc -l

# These should NOT exist in deployed:
# brainstorming, prd, qa-plan, doc, refactor, backlog,
# using-superpowers, skill-creator, writing-skills
ls ~/.claude/skills/brainstorming 2>/dev/null && echo "STALE: brainstorming still deployed" || echo "OK"

# Gemini prompts should also match
ls -1 ~/.gemini/prompts/*.md 2>/dev/null | wc -l
```

### 2. Weekly Maintenance Script

```bash
# Run manually (no cron needed for testing)
./scripts/vault-maintenance-weekly.sh

# Check the log
cat ~/.local/share/vault-maintenance/latest.log

# Verify cron is installed (Linux)
crontab -l | grep vault-maintenance
# Expected: 7 10 * * 0 .../vault-maintenance-weekly.sh

# Verify Task Scheduler (Windows PowerShell)
# Get-ScheduledTask -TaskName "DotfilesVaultMaintenance"
```

### 3. Skills Pipeline

Run in any active project (e.g. dotfiles):

```
/insights              # Quick mode: MEMORY.md + vault health + backlog
/insights full         # Full mode: + observations + decisions + patterns
```

If insights reports issues:
- Knowledge gaps (unvaulted observations) -> `/crystallize`
- Structural issues (unresolved links, frontmatter) -> `/vault-doctor`

### 4. Full Cycle Across All Projects

```bash
# Step 1: automated MEMORY.md maintenance
./scripts/knowledge-crystallize.sh --all

# Step 2: open Claude in each active project, run /insights
# Active projects: dotfiles, kubelab, hive, youtube-toolkit (adapt to your list)

# Step 3: fix what insights reports per project
```

### 5. Standing Orders Verification

Open a new Claude session in any project and verify:
- CLAUDE.md shows "Standing Orders" section with 6 rules
- Ask Claude to propose a quick hack -- it should refuse and propose an enterprise pattern instead
- Ask Claude to suggest manual steps for a repeatable task -- it should propose automation

### 6. CSO Verification

In any Claude session, the skill descriptions should auto-trigger correctly:
- Say "I need to create a new skill" -> should invoke `creating-skills`
- Say "This project needs hardening" -> should invoke `project-maturation`
- Say "The vault has broken links" -> should invoke `vault-doctor`
- Say "I'm about to commit this fix" -> should invoke `verification-before-completion`

---

## Troubleshooting

### "No MEMORY.md found" from knowledge-crystallize script

The path encoding must match what Claude Code uses:

- **Linux:** `path → tr '/' '-'` — e.g. `/home/manu/Projects/kubelab` becomes `-home-manu-Projects-kubelab`
- **Windows:** `path.Replace('\','-').Replace(':','')` — e.g. `C:\Users\Manu\Projects\kubelab` becomes `C-Users-Manu-Projects-kubelab`

Check: `ls ~/.claude/projects/ | grep kubelab`

### "Knowledge crystallization never run" warning in session hook

Run `./scripts/knowledge-crystallize.sh` from within the project directory. This stamps the `## Last Crystallized:` line that the hook checks.

### MEMORY.md keeps growing

Common causes:
- claude-mem auto-adds context sections — trim them with `/crystallize`
- Duplicate `# currentDate` entries — fixed automatically by `knowledge-crystallize.sh`
- Backlog Status becoming too detailed — keep it to 3-4 bullet points maximum
