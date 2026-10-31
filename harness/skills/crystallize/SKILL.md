---
generated: true
generated_from: 00_meta/skills/crystallize/SKILL.md
generated_sha: 9e882ec50469d6dc
id: crystallize-skill
type: skill
status: active
created: '2026-05-30'
owner: manu
name: crystallize
description: Use when /insights shows unpersisted decisions, stale MEMORY.md, or
  after completing a significant sprint. Addresses knowledge gaps between session
  journals and vault lessons.
keywords: [crystallize, cristalizar, promote lesson, pattern promotion, leccion aprendida]
paths: [00_meta/patterns/**, docs/lessons/**]
---
# /crystallize — Knowledge Crystallization Ritual

Full maintenance ritual for the Neural Hive knowledge loop. Run when `/insights` shows issues or after a major sprint.

## Protocol

### Step 1 — Audit State
- Read the project's MEMORY.md at `$VAULT_PATH/10_projects/<repo>/memory/MEMORY.md` (keep `$VAULT_PATH` if already set and non-empty; otherwise assign it from `dotf env path VAULT_PATH`, and stop and ask the user if that is empty)
- Note: line count, Last Crystallized date, days elapsed since last run
- Read the repo's `docs/lessons/` (and `dotf lessons index` for the table, where the repo uses dotf's format) for current **project** lesson state (project lessons live in the repo — see [[pattern-knowledge-placement]]); read `$VAULT_PATH/00_meta/patterns/` for cross-project state

### Step 2 — Mine Recent Activity
There is no observation store; the continuity layer is the vault (see [[pattern-dual-memory]]). Mine the window since the last crystallization:

```bash
V="$VAULT_PATH"
git -C "$V" log --since="<Last Crystallized>" --name-only --pretty=format: \
  | grep -E '/sessions/|/memory/' | sort -u              # journals + memory files to read
awk -v s="<Last Crystallized>" 'FNR==1{d=""} /^> Updated: /{d=$3} /^\*\*Decisions:\*\*/ && d>=s' \
  "$V"/10_projects/*/memory/MEMORY.md                   # handoff decisions from threads updated since then
```

- Read the journals and list each fix or decision with its source (journal path or thread name)

### Step 3 — Gap Detection & Orthogonal `INV` / `DOM` Classification
- Compare mined fixes and decisions against the repo's `docs/lessons/` (project lessons) and `00_meta/` (cross-project)
- Identify which bugs/decisions are NOT yet documented in the vault
- Classify each candidate along two orthogonal axes before writing (`VAULT-060`, extracted from `harness2` Epoch 1 synthesis):
  - **`INV` (Task-Agnostic Invariant):** Cross-domain loop mechanics (verification gates, workspace hygiene, stop conditions, obligation closure). Must never contain file-type, framework, or tool-specific recipes.
  - **`DOM` (Domain-Scoped Procedure):** Tool-, stack-, or format-specific workflows (e.g., Kubernetes manifests, PDF generation, SQLite migrations). Strictly forbidden from entering always-on `AGENTS.md` prose; must route to a project `docs/lessons/` file, an on-demand `00_meta/skills/<domain>/SKILL.md`, or an executable validator script.

### Step 4 — Evidence Admission Gate & Vault Update
- **Causal & Evidence Admission Gate (`NO_ENTRY` rule):** Every lesson candidate must satisfy a non-empty 4-tuple before writing:
  1. **Cue (`Context`):** Concrete observable trigger condition (when to apply).
  2. **Lesson (`Problem` + `Solution` + `Why`):** The actionable mechanism and root cause.
  3. **Evidence:** Verifiable artifact citation (commit SHA, PR/issue `#N`, failing-then-passing test command, or session journal path).
  4. **Confidence & Causal Attribution:** `confidence: tentative | supported | confirmed` paired with a 4-bucket causal classification (`helped | hurt | no observed effect | confounded`).
  - **Refusal (`NO_ENTRY`):** If a candidate lacks a concrete artifact citation or rests solely on `confounded` observations (where the outcome shifted due to external environment drift rather than the fix), discard it (`NO_ENTRY`) rather than writing speculative prose into `docs/lessons/` or `00_meta/`.
- For each admitted gap, write a **project** lesson to the repo's `docs/lessons/lesson-NNN-<slug>.md` with the standard frontmatter, then format it with `dotf lessons fmt` only where the repo uses dotf's lesson format; a repo with its own lesson formatter (named in its `docs/lessons/_format.md` or its pre-commit hooks) uses that one, because `dotf lessons fmt` rewrites every lesson in the tree (577 files in kubelab, #2470) (a genuinely **cross-project** lesson goes to `00_meta/patterns/`):

```markdown
---
id: "lesson-NNN-<slug>"
type: lesson
status: active
created: "YYYY-MM-DD"
owner: manu
confidence: supported # tentative | supported | confirmed
tags: [tag1, tag2]
---

# <Title>

**Context:** <observable cue — what you were doing when you hit this>
**Problem:** <what went wrong or what decision was needed>
**Solution:** <what fixed it or what was decided>
**Why:** <root cause or rationale>
**Evidence:** <commit SHA, PR/issue #N, test output, or session journal path (causal bucket: helped | hurt | no observed effect; a confounded candidate is NO_ENTRY)>
```

### Step 5 — Pattern & Harness Promotion (Five-Tier Lever Cost Hierarchy)
- For each new lesson, ask: "Does this appear in >1 project, or can it be mechanized?"
- When promoting a lesson into the cross-project harness (`00_meta/`), select the target lever strictly by the **5-Tier Lever Cost Hierarchy** (lowest recurring token tax and highest determinism first):
  1. **Tier 1 — Executable Script / Static Validator (`00_meta/scripts/*.py`, `dotf`):** Best for invariant checks (`_check_script_refs`, schema gates, linters). Zero prose interpretation error and zero turn-token tax.
  2. **Tier 2 — On-Demand Skill (`00_meta/skills/<name>/SKILL.md`):** Best for `DOM` procedures and multi-step workflows. Zero always-on token cost; loaded only when its trigger fires.
  3. **Tier 3 — Subagent Delegation:** Use **only** when the subagent's `input_context : output_summary` ratio is high (heavy exploration returning a compact verdict). Never introduce a subagent step when the parent must pass the full artifact across the boundary anyway.
  4. **Tier 4 — Lifecycle Hook (`pre_tool_use` / `stop`):** Deterministic gate at turn boundaries when a check must fire unprompted.
  5. **Tier 5 — Always-On `AGENTS.md` / Global Rule Prose (Last Resort):** Reserved exclusively for `INV` rules that must be active on every turn of every task before any tool or skill fires. Because always-on prose enters the context of every turn, adding to Tier 5 near the token ceiling requires compressing or retiring equal weight (`KEEP_BASE` unless net step/score improvement is proven).
- If a passive cross-project principle is needed (`Tier 2` / pattern) and no matching file exists in `$VAULT_PATH/00_meta/patterns/` → propose creating `pattern-<topic>.md`

### Step 6 — MEMORY.md Trim
- Remove any sections from MEMORY.md that are already covered verbatim by the repo's agent instructions (`AGENTS.md`)
- Keep: User Preferences, CI Pipeline, Bugs Found, Backlog Status, Last Crystallized
- Update `## Last Crystallized: YYYY-MM-DD` to today

### Step 7 — Stamp
- Run: `dotf vault crystallize` (add `--all` to stamp every project)
- This updates currentDate and Last Crystallized automatically

### Step 8 — Report
Print a summary:
```
Crystallization complete:
  - X project lessons added to repo docs/lessons/ (cross-project -> 00_meta/patterns/)
  - Y pattern proposals (list titles)
  - MEMORY.md: N lines before → M lines after
  - Last Crystallized: <today>
```

## When to Use

| Trigger | Action |
|---------|--------|
| `/insights` shows unpersisted decisions or lesson gaps | Run `/crystallize` |
| MEMORY.md > 150 lines | Run `/crystallize` |
| Last Crystallized > 14 days ago | Run `/crystallize` |
| After completing a major sprint | Run `/crystallize` |

## Difference from /insights

`/insights` is **read-only** — it tells you what needs attention.
`/crystallize` is **write mode** — it actually fixes the issues.

Run `/insights` weekly. Run `/crystallize` when insights shows problems.
