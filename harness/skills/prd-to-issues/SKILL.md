---
generated: true
generated_from: 00_meta/skills/prd-to-issues/SKILL.md
generated_sha: fbd3a700210d814a
id: prd-to-issues-skill
type: skill
status: active
created: '2026-05-30'
owner: manu
name: prd-to-issues
description: Use when creating GitHub issues from a PRD, syncing an existing PRD to
  GitHub, or converting requirements documents into tracked issues.
keywords: [prd to issues, convert prd, requirements to tickets, sync prd]
paths: ['**/prd*.md', '**/requirements*.md']
---
# PRD to GitHub Issues

Convert a PRD into GitHub issues (epics + stories/tasks) using the `gh` CLI. Supports initial creation and re-sync when the PRD changes.

## Prerequisites

Before starting, verify:
- `gh` CLI is installed and authenticated (`gh auth status`)
- A PRD exists in `docs/prd/` (created via the `prd` skill)
- The project has a GitHub remote (`git remote -v`)

If any prerequisite fails, stop and tell the user what to fix.

## Label Setup

Create these labels before creating issues (idempotent — safe to re-run):

```bash
gh label create "epic" --color "3B0A8C" --description "Epic: high-level feature group" --force
gh label create "story" --color "0E8A16" --description "Story: user-facing deliverable" --force
gh label create "task" --color "1D76DB" --description "Task: technical work item" --force
gh label create "must-have" --color "B60205" --description "MoSCoW: Must have" --force
gh label create "should-have" --color "D93F0B" --description "MoSCoW: Should have" --force
gh label create "could-have" --color "FBCA04" --description "MoSCoW: Could have" --force
gh label create "wont-have" --color "CCCCCC" --description "MoSCoW: Won't have (this time)" --force
```

## Initial Create Mode

When no `<!-- GH-ISSUE:` markers exist in the PRD:

### 1. Parse PRD

- Read the PRD from `docs/prd/`
- Extract Epics, FRs, acceptance criteria, priorities
- Present a summary table to the user for confirmation before creating anything

### 2. Create Labels

Run the label setup commands above.

### 3. Create Epic Issues

For each Epic:

```bash
gh issue create \
  --title "EPIC-001: [Epic Title]" \
  --label "epic,[priority]-have" \
  --body "$(cat <<'EOF'
## Epic: [Title]

[Epic description from PRD]

### Acceptance Criteria

- [ ] [Criterion 1]
- [ ] [Criterion 2]

### Functional Requirements

- FR-001: [Description]
- FR-002: [Description]

---
> Source: docs/prd/<project>-prd.md
EOF
)"
```

### 4. Create Story/Task Sub-Issues

For each FR within an Epic, create a sub-issue:

```bash
gh issue create \
  --title "[FR-001] [Short description]" \
  --label "story,[priority]-have" \
  --body "$(cat <<'EOF'
## User Story

As a [persona], I want to [action] so that [benefit].

### Acceptance Criteria

- [ ] [Criterion from PRD]

### Details

- **Priority:** [Must/Should/Could]
- **Estimate:** [S/M/L]
- **Epic:** EPIC-001 (#N)
- **FR:** FR-001

---
> Source: docs/prd/<project>-prd.md
EOF
)"
```

Then link to the parent epic:

```bash
gh issue develop <story-number> --issue-parent <epic-number>
```

If `gh issue develop` is not available, add a comment linking to the parent:

```bash
gh issue comment <story-number> --body "Parent epic: #<epic-number>"
```

### 5. Embed Markers in PRD

After creating issues, insert HTML comment markers in the PRD next to each Epic/FR:

```markdown
### EPIC-001: Authentication <!-- GH-ISSUE: owner/repo#42 -->

| FR-001 | User login | Must | EPIC-001 | <!-- GH-ISSUE: owner/repo#43 --> |
```

**Marker format:** `<!-- GH-ISSUE: owner/repo#N -->`

Placement rules:
- Epic markers: end of the Epic heading line
- FR markers: end of the FR table row or as a new column

## Re-Sync Mode

Triggered when `<!-- GH-ISSUE:` markers already exist in the PRD.

### 1. Parse Existing Markers

Extract all `<!-- GH-ISSUE: owner/repo#N -->` markers from the PRD using:

```
Pattern: <!-- GH-ISSUE:\s*([^#]+)#(\d+)\s*-->
```

### 2. Fetch Issue State

For each marker, fetch the current issue state:

```bash
gh issue view <number> --json title,state,body,labels
```

### 3. Diff PRD vs Issues

Compare PRD content against issue content. Classify each item:

| Status | Meaning |
|--------|---------|
| **UNCHANGED** | PRD and issue match |
| **NEW** | In PRD but no marker (new requirement) |
| **MODIFIED** | PRD content differs from issue body |
| **REMOVED** | Marker exists but FR removed from PRD |
| **CLOSED** | Issue was closed on GitHub |

### 4. Present Diff Summary

Show the user a summary table before making any changes:

```
| # | Issue | Status | Action |
|---|-------|--------|--------|
| 1 | #42 EPIC-001 | UNCHANGED | — |
| 2 | #43 FR-001 | MODIFIED | Update issue body |
| 3 | — FR-004 | NEW | Create issue |
| 4 | #45 FR-003 | REMOVED | Close issue? |
```

### 5. User Confirms

Ask: "Which actions should I apply? (all / select by number / none)"

**Never auto-apply re-sync changes.**

### 6. Execute and Update Markers

Apply confirmed actions:
- **NEW:** `gh issue create` + add marker to PRD
- **MODIFIED:** `gh issue edit <N> --body "..."`
- **REMOVED:** `gh issue close <N>` (only if user confirms)

Update markers in the PRD file after execution.

## Issue Body Template

All issues follow this structure:

```markdown
## [User Story | Epic Description]

[Content from PRD]

### Acceptance Criteria

- [ ] [Criterion]

### Details

- **Priority:** [Must/Should/Could/Won't]
- **Estimate:** [S/M/L/XL]
- **Epic:** [EPIC-ID] (#N)
- **FR:** [FR-ID]

---
> Source: docs/prd/<project>-prd.md
```

## Rules

1. **gh CLI only** — never use the GitHub API directly or curl; always use `gh`
2. **Confirm before creating** — show the summary table, wait for user approval
3. **Never auto-apply re-sync** — always present the diff and ask for confirmation
4. **Labels are idempotent** — use `--force` flag so re-runs do not fail
5. **Markers are sacred** — never delete or modify markers manually; only this skill manages them
6. **One PR per epic** — suggest (do not enforce) branching strategy: one branch per epic
7. **Traceability** — update the PRD traceability matrix after creating issues
8. **Error handling** — if `gh issue create` fails, report the error and continue with remaining issues
