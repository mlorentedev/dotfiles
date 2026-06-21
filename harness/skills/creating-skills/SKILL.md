---
name: creating-skills
targets: [claude]
description: Use when creating a new skill, updating an existing skill, or verifying a skill works before deployment. Covers skill anatomy, TDD testing methodology, and Claude Search Optimization for trigger descriptions.
---

# Creating Skills

## Core Principle

Creating skills IS Test-Driven Development applied to process documentation. If you didn't watch an agent fail without the skill, you don't know if the skill teaches the right thing.

## When to Create a Skill

**Create when:**
- Technique wasn't intuitively obvious
- You'd reference this again across projects
- Pattern applies broadly (not project-specific)
- Agent consistently fails without guidance

**Don't create for:**
- One-off solutions
- Standard practices well-documented elsewhere
- Project-specific conventions (put in CLAUDE.md instead)
- Mechanical constraints enforceable with validation (automate it — save skills for judgment calls)

## Anatomy of a Skill

```
skill-name/
├── SKILL.md              # Required — frontmatter + instructions
├── scripts/              # Optional — executable code (deterministic, reusable)
├── references/           # Optional — docs loaded on demand (schemas, API docs)
└── assets/               # Optional — output resources (templates, icons)
```

### Progressive Disclosure (3 levels)

1. **Metadata** (name + description) — Always in context (~100 words). This is the TRIGGER.
2. **SKILL.md body** — Loaded when skill triggers. Keep under 500 lines.
3. **Bundled resources** — Loaded on demand. Unlimited size.

Split to reference files when approaching the 500-line limit. Reference from SKILL.md with clear "when to read" guidance.

## SKILL.md Structure

### Frontmatter (YAML) — CRITICAL

Only two fields: `name` and `description` (max 1024 chars total).

```yaml
---
name: skill-name-with-hyphens
description: Use when [specific triggering conditions and symptoms]
---
```

The **description is the primary trigger mechanism.** Claude reads it to decide whether to load the skill. The body is only loaded AFTER triggering. All "when to use" information MUST be in the description, not in the body.

See **Claude Search Optimization** section below.

### Body (Markdown)

```markdown
# Skill Name

## Overview
Core principle in 1-2 sentences.

## Core Pattern
Before/after comparison or key workflow.

## Quick Reference
Table or bullets for scanning.

## Common Mistakes
What goes wrong + fixes.
```

Do NOT include: README, CHANGELOG, installation guides, or auxiliary documentation.

## Claude Search Optimization (CSO)

### Description = When to Use, NOT What It Does

**NEVER summarize the skill's workflow in the description.** Testing revealed that when descriptions summarize workflow, Claude follows the description as a shortcut instead of reading the full skill body.

```yaml
# BAD: Summarizes workflow — Claude takes shortcut
description: Dispatches subagent per task with code review between tasks

# BAD: Too much process detail
description: Use for TDD - write test first, watch it fail, write minimal code

# GOOD: Triggering conditions only
description: Use when executing implementation plans with independent tasks

# GOOD: Symptoms and situations
description: Use when tests have race conditions or pass/fail inconsistently
```

### Description Rules

- Start with "Use when..." — focus on triggering conditions
- Include symptoms, situations, and contexts
- Write in third person (injected into system prompt)
- Keep under 500 characters
- **NEVER** summarize the skill's process or workflow
- Include keywords Claude would search for (error messages, symptoms, tool names)
- Describe the *problem*, not language-specific symptoms

### Naming Conventions

- Use active voice, verb-first: `creating-skills` not `skill-creation`
- Gerunds (-ing) for processes: `writing-plans`, `dispatching-parallel-agents`
- Hyphens only (no parentheses, special chars)

### Keyword Coverage

Include words Claude would search for:
- Error messages: "Hook timed out", "race condition"
- Symptoms: "flaky", "hanging", "zombie"
- Synonyms: "timeout/hang/freeze", "cleanup/teardown"
- Tool and library names

### Cross-Referencing Skills

```markdown
# GOOD: Explicit requirement markers
**REQUIRED:** Use test-driven-development for the RED-GREEN-REFACTOR cycle.

# BAD: Force-loads into context
@skills/test-driven-development/SKILL.md
```

Never use `@` syntax — it force-loads files, burning context before you need them.

## Skill Creation Process (TDD)

### Phase 1 — RED: Establish Baseline

1. Create pressure scenarios for the skill's domain
2. Run scenarios WITHOUT the skill using a subagent
3. Document exact behavior: choices made, rationalizations used (verbatim), failures observed
4. This IS "watching the test fail" — you MUST see what agents naturally do

### Phase 2 — GREEN: Write Minimal Skill

1. Understand the skill with concrete examples (ask user if unclear)
2. Plan reusable resources: scripts, references, assets
3. Write SKILL.md addressing the specific failures observed in Phase 1
4. Run same scenarios WITH skill — agent should now comply

**Key:** Only add content that addresses observed failures. Don't add hypothetical coverage.

### Phase 3 — REFACTOR: Close Loopholes

1. Identify new rationalizations from testing
2. Add explicit counters for each
3. Build rationalization table from all iterations
4. Create red flags list for self-checking
5. Re-test until bulletproof

### The Iron Law

```
NO SKILL WITHOUT A FAILING TEST FIRST
```

Applies to NEW skills AND EDITS to existing skills. Write skill before testing? Delete it. Start over.

- Don't keep it as "reference"
- Don't "adapt" it while writing tests
- Delete means delete. No exceptions.

## Testing by Skill Type

### Discipline Skills (rules/requirements)

**Examples:** TDD, verification-before-completion
**Test with:** Pressure scenarios combining time + sunk cost + exhaustion
**Success criteria:** Agent follows rule under maximum pressure

### Technique Skills (how-to guides)

**Examples:** condition-based-waiting, root-cause-tracing
**Test with:** Application + variation + missing-information scenarios
**Success criteria:** Agent applies technique correctly to new scenario

### Pattern Skills (mental models)

**Test with:** Recognition + application + counter-example scenarios
**Success criteria:** Agent correctly identifies when/how to apply

### Reference Skills (documentation/APIs)

**Test with:** Retrieval + application + gap scenarios
**Success criteria:** Agent finds and correctly applies information

## Bulletproofing Discipline Skills

### Close Every Loophole Explicitly

Don't just state the rule — forbid specific workarounds:

```markdown
# Weak
Write code before test? Delete it.

# Strong
Write code before test? Delete it. Start over.
- Don't keep it as "reference"
- Don't "adapt" it while writing tests
- Delete means delete
```

### Address "Spirit vs Letter" Arguments

Add early: **"Violating the letter of the rules is violating the spirit of the rules."**

### Build Rationalization Table

Every excuse agents make during testing:

| Excuse | Reality |
|--------|---------|
| "Too simple to test" | Simple code breaks. Test takes 30 seconds. |
| "I'll test after" | Tests passing immediately prove nothing. |
| "Obviously clear" | Clear to you ≠ clear to other agents. |
| "No time to test" | Untested skills waste more time fixing later. |

### Create Red Flags List

```markdown
## Red Flags — STOP and Start Over
- Code/skill content before test
- "I already manually tested it"
- "This is different because..."
All mean: Delete. Start over.
```

## Token Efficiency

The context window is a public good. Only add context Claude doesn't already have.

- **Default assumption:** Claude is already very smart. Challenge each piece: "Does this paragraph justify its token cost?"
- **Prefer** concise examples over verbose explanations
- **One excellent example** beats many mediocre ones
- **Use cross-references** instead of repeating content from other skills
- **Move details** to reference files or tool `--help`

## Deployment

The vault is the single source of truth (SDD-008). Author the skill here
(`00_meta/skills/<skill-name>/SKILL.md`); `compile-harness.sh` (invoked by setup,
or run with `--deploy` standalone) renders it into the dotfiles `harness/skills/`
record and deploys it to each agent:
- Claude: `~/.claude/skills/<skill-name>/` (full directory, regular copy)
- OpenCode: `~/.config/opencode/commands/<skill-name>.md`
- Antigravity: `~/.gemini/skills/<skill-name>/` + `~/.gemini/prompts/<skill-name>.md` (frontmatter stripped)
- Copilot: a catalog entry in `~/.copilot/copilot-instructions.md`

Add `targets: [claude]` (or any subset) to the frontmatter to limit which agents
receive the skill; absent = all agents. Edit the skill here and run
`compile-harness.sh --deploy` (or re-run setup) — no manual per-agent deployment.
