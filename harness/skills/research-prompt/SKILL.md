---
generated: true
generated_from: 00_meta/skills/research-prompt/SKILL.md
generated_sha: f4bbe9b20654b38f
id: research-prompt-skill
type: skill
status: active
created: '2026-07-07'
owner: manu
name: research-prompt
description: Turn a vague research ask ("look into X", "what's the state of Y", "is
  there a better way to do Z") into one tight, sourceable research brief BEFORE dispatching
  it to a research agent or subagent. Triggers on /research-prompt, "help me phrase
  this research question", "what should the researcher look for", "turn this into
  a research brief", "tighten this research ask", and whenever a research/explore/task
  subagent is about to be launched from an under-specified request. Adapted from davidondrej/skills'
  research-prompt, re-targeted at this system's research subagent + web_fetch instead
  of a third-party research API.
allowed-tools: [Read, Grep, Glob]
keywords: [research prompt, research brief, tighten research, pregunta de investigacion]
paths: ['**/research/**', '**/prestudy/**']
---
# Research Prompt — sharpen the ask before you dispatch it

> A fuzzy research request ("look into X") produces a fuzzy report — the agent has to guess scope, sources, and what "done" means, and usually guesses wrong at least one of the three. This skill is a five-minute sharpening pass run BEFORE launching a `research` (or `explore`/`task`) subagent, not a replacement for one.

## When to use

- Before delegating to the `research` subagent (or `explore` for a lighter lookup) when the request is one sentence and open-ended.
- When the user asks "what should I have the researcher look for?" or hands you a half-formed question.
- Before any deep-research web session where getting the scope wrong wastes a long-running agent's full run.

## When to skip

- The ask is already scoped (specific question, named sources, clear output format) — dispatch directly.
- It's a quick factual lookup answerable in 1-2 `web_fetch`/`grep` calls — don't ceremony a two-minute lookup into a five-minute brief.

## The five slots (fill all before dispatching)

1. **Question** — one sentence, falsifiable. Not "look into observability tools" but "which of Prometheus/Victoria Metrics/Grafana Mimir handles our current cardinality (~2M series) on the homelab's resource budget?"
2. **Scope boundary** — what's explicitly OUT. Time range, geography, tech stack, versions. The fastest way to blow a research budget is an unbounded "also check...".
3. **Sources to check first** — named repos, docs sites, standards bodies, or "GitHub code search for X" — not just "the internet". If you don't know good sources, say so explicitly rather than silently omitting this slot.
4. **Output shape** — inventory table? prioritized shortlist? a single recommendation with tradeoffs? Match the shape to what happens next (a shortlist feeds a decision; a table feeds a comparison doc).
5. **Done-criteria** — what makes the report sufficient vs. what would make you ask for another pass. ("Cite at least one production case study" / "must cover the last 12 months only".)

## Procedure

1. Read the user's raw ask. Identify which of the five slots are already implied and which are missing.
2. For each missing slot, either infer a reasonable default from context (state the assumption explicitly) or ask ONE targeted question — never all five at once.
3. Write the tightened brief as a short paragraph or a 5-line list, not a form — it should read naturally to whichever subagent receives it.
4. Dispatch: `research` for external/citation-heavy work, `explore` for fast internal-codebase lookups, `task` for command-output-style checks. Include the brief verbatim in the subagent prompt (sub-agent prompts are stateless — full context every time).
5. State placement of the resulting artifact **before** the agent starts, per [[pattern-knowledge-placement]]'s research sub-rule: project-bound research → that repo's `docs/` (or `10_projects/<project>/research/` if genuinely pre-repo/decide-layer); cross-project tooling/methodology research → vault `00_meta/research/`. Don't leave the placement to be decided after the report lands.

## Anti-patterns

- **Ceremony tax** — running the full five-slot pass on a question you could answer yourself in one `grep`. Match the process to the cost of getting it wrong.
- **Silent scope creep** — accepting "also look into whatever else seems relevant" as a slot-2 answer. That's not a boundary, it's the absence of one; push back once.
- **Brief without a home** — dispatching before slot 5's artifact placement is decided, producing an orphan report that gets asked "where does this go?" after the expensive part is already done.

## References

- [[pattern-knowledge-placement]] — research artifact placement sub-rule (project-bound vs cross-project).
- Adapted from `davidondrej/skills/research-prompt` (external skills audit, 2026-07-07) — re-targeted at the `research`/`explore`/`task` subagents and `web_fetch` instead of a third-party research API, since this system has no DeepAPI-equivalent dependency.
