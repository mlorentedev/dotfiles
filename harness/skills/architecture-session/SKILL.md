---
id: architecture-session-skill
type: skill
status: active
created: "2026-05-28"
name: architecture-session
description: "Run a pure-architecture / definition session on a project. Triggers on /architecture-session, /arch, \"sesion de arquitectura\", \"arch session for X\", \"definir arquitectura de X\", \"revisar arquitectura de X\", \"evaluar opciones para X\". Six phases A-F: state verification, multi-reference audit (Regla del 3 gate), constraint formalization, options + rejection list, decision (ADR + plan + vault patch in-session), recap. Refuses to advance past Phase B without N>=2 references audited for any decision affecting cross-instance reuse. Pairs with /spec init (downstream implementation) and /adversarial-review (post-implementation gate)."
allowed-tools: [Bash, Read, Edit, Write, Grep, Glob, mcp__hive__vault_query, mcp__hive__vault_search, mcp__hive__vault_write, mcp__hive__vault_patch, mcp__hive__capture_lesson]
---

# Architecture Session

> Pure architecture / definition session: no implementation, no code. The output is **decisions persisted in the vault** (ADR + plan revision + 11-tasks update), not commits. Implements the discipline crystallized in `pattern-decision-persistence` + `pattern-workflow-protocol` (Exit Phase 2) + ADR-015 "Regla del 3 en abstraccion".
>
> **Companion to** `/spec init|fill|archive` (per-feature SDD): architecture sessions produce ADRs and roadmap shifts; specs produce code. Different granularity, different cadence.

## When to use

- `/architecture-session` (or `/arch`) explicitly.
- "Sesion de arquitectura para X" / "arch session for X" / "definir arquitectura de X" / "revisar arquitectura de X" / "evaluar opciones para X" / "decidir entre X e Y".
- Memory has gone stale (>30d since last architecture review on the area) AND a non-reversible decision is on the table.
- Multiple options exist for a cross-cutting concern (tooling, layering, distribution, governance) and you need a structured walk before committing.

## When NOT to use

- Implementing a feature whose architecture is already settled -> use `/spec init`.
- Single-file bug fix or mechanical change -> just do it.
- Pure brainstorming with no commitment intent -> a free chat is fine; do not formalize.
- A decision that can be safely reversed in <1 day of work -> skip ADR ceremony.

## Environment (resolve once per invocation)

- `$VAULT_PATH` -- env var. Linux/macOS: `$HOME/Projects/knowledge`. Windows: `%USERPROFILE%\Projects\knowledge`.
- `$PROJECT_AREA` -- vault path of the project area in scope. Examples:
  - `10_projects/<repo>/`
  - `50_work/45-development/<family>/<component>/`
- `$WORKSPACE` -- on-disk workspace path (CWD or explicit). May or may not be a git repo at session time.
- `$SESSION_DATE` -- today in ISO `YYYY-MM-DD`.
- If `$VAULT_PATH` is unreachable, fail fast and instruct the user to set it. A pure-architecture session is meaningless without vault writes.

---

## Phase A -- State Verification (no memory, evidence only)

**Goal:** discard stale beliefs and ground the session in current reality.

**Steps:**

1. Read these vault files in order (Hive `vault_query`):
   - `$PROJECT_AREA/00-context.md` (technical stack, repo map, hotspots)
   - `$PROJECT_AREA/10-roadmap.md` (active phase, what is done vs pending)
   - `$PROJECT_AREA/11-tasks.md` (progress bar + last "Updated" frontmatter date)
   - the repo's `docs/adr/` directory (latest ADR number, accepted/deferred status — GitHub self-indexes; legacy vault-only projects still use `$PROJECT_AREA/30-architecture/`)
   - `$PROJECT_AREA/30-architecture/plan-*.md` -- newest mtime first
   - `$PROJECT_AREA/memory/MEMORY.md` -- Session Handoff section
2. If a `$PROJECT_AREA/30-architecture/session-protocol.md` exists, read it -- it is the project-specific overlay (constraints, rejection list, pending ADRs).
3. **Verify reality, not memory.** For each repo mentioned in `00-context.md`:
   - Resolve its on-disk path. Read `pyproject.toml` / `package.json` / equivalent for current version.
   - If git, `git log -1 --format=%cI` for last commit date and `git status --short` for dirty state.
4. **Surface drift** as a structured table BEFORE the user picks the day's topic:

   | Layer | Memory says | Reality | Drift? |
   |---|---|---|---|
   | Repo X version | v0.7.0 (per 00-context) | v0.7.4 (per pyproject) | YES (stale 4 patches) |
   | Repo Y branches | clean | 3 zombie local branches | YES |
   | Vault 11-tasks "Updated" | 2026-03-18 | (today minus N days) | YES if N > 30 |

5. **If drift detected**, ask the user: "First action: patch the stale items in vault, or note drift and continue?" Do not advance to Phase B until the user picks.

**Output of Phase A:** a 5-10 line "Estado real verificado (no memoria)" block, the same shape as `plan-architecture-revision.md` opens with.

**Phase A is mandatory.** A session that skips state verification re-decides what was already decided or builds on facts that have shifted.

---

## Phase B -- Multi-Reference Audit (Regla del 3 -- BLOCKING)

**Goal:** prevent single-reference bias by forcing N>=2 reference instances to be audited before any cross-instance generalization.

**When this phase fires (gate):**

- The day's topic affects **cross-instance reuse** (template, framework class, shared library, generalized config). Example: defining the sensor SDK template; selecting bus abstractions; designing a config schema that will serve N callers.
- The day's topic does NOT affect cross-instance reuse: SKIP Phase B and go to Phase C. Example: refactoring an internal module of a single SDK; tightening one repo's CI; choosing a logging format for one service.

**Steps:**

1. **Inventory the available references.** Read the project's `30-architecture/sensor-reference-audit-matrix.md` (or equivalent inventory file). If absent, create it from `00_meta/templates/` and offer to the user.
2. **Pick the audit set.** Minimum N=2 references plus the gold standard (if one exists). For the platform case (`python-sensor-sdk-platform`), gold standard is Hydra3D; pick 2+ additional sensors maximizing paradigm diversity (per ADR-015 decision criteria).
3. **For each reference, populate the audit matrix.** The matrix columns are project-defined (in `session-protocol.md`); typical fields:
   - Paradigm / domain category
   - Bus / transport (USB3, PCIe, Ethernet, ...)
   - Register / API surface size
   - Config format (binary, JSON, MATLAB .dat, ...)
   - Whether MATLAB SDK / Python POC / firmware ref exists
   - Key divergence flag (1-line "what makes this one different")
4. **Emit a divergence-log section** at the end of the audit: list patterns that emerge from the intersection (these are **template candidates**) and patterns unique to one reference (these are **NOT template candidates**).

**Refusal condition.** If the user wants to skip Phase B because they "already know" what is generic, respond once: "ADR-015 / Regla del 3 says the audit IS the evidence. Audit at least N=2 references or we are speculating. Continue, or skip with `--force-no-audit` flag (NOT RECOMMENDED, will be tagged in the resulting ADR)."

**Output of Phase B:** updated audit matrix + 5-15 line divergence-log written to the matrix file.

---

## Phase C -- Constraint Formalization

**Goal:** make implicit constraints explicit so the options analysis (Phase D) can be evaluated against them mechanically.

**Steps:**

1. **Read existing constraints** from `$PROJECT_AREA/30-architecture/session-protocol.md` if present. These persist across sessions (numbered `C1`, `C2`, ...). Example for `python-sensor-sdk-platform`: C1 self-contained artifacts; C2 onboard new team <30min; C3 one-command bootstrap; ... C10 escape hatch for divergent branches.
2. **Surface any new constraint** introduced by today's topic. Number it sequentially. Source it from a user statement, an ADR, or a lesson -- not from speculation.
3. **Render the constraint table** to the user for confirmation:

   | # | Constraint | Origin | Today's topic? |
   |---|---|---|---|
   | C1 | ... | constraint usuario | yes / no |
   | ... | | | |

4. **Persist new constraints** by patching `session-protocol.md` via `vault_patch`. The constraint table is the project's SSOT for "rules to evaluate options against".

**Output of Phase C:** updated `session-protocol.md` (constraint table), reflected back in conversation.

---

## Phase D -- Options + Rejection List

**Goal:** generate 3-5 candidate options, evaluate them against the constraint table, and check the rejection list before re-debating an already-rejected alternative.

**Steps:**

1. **Read the rejection list** in `session-protocol.md` (section "Alternativas evaluadas y descartadas"). Each row: alternative + reason discarded + date. If the user proposes one of these, surface the prior reason and ask: "Has the trigger to reopen this changed?"
2. **Generate 3-5 options.** For each: 1-line summary, pros, cons. Lean on the patterns catalog (`00_meta/patterns/_index.md`) for vocabulary; do not invent terms when a canonical one exists.
3. **Map options to constraints.** Render the matrix:

   | Option | C1 | C2 | C3 | ... | Cn |
   |---|---|---|---|---|---|
   | A | ok | ok | gap | ... | ok |
   | B | ok | ok | ok | ... | gap |
   | ... | | | | | |

4. **Ask the user for selection.** If user picks, advance to Phase E. If user says "defer", treat the decision as **explicit deferral with triggers** and still write an ADR with `status: deferred` (per ADR-016 template).

**Output of Phase D:** options-vs-constraints matrix in the conversation, plus the user's selection (or explicit deferral with triggers).

---

## Phase E -- Decision (PERSIST IN-SESSION, blocking)

**Goal:** materialize the decision so it is not lost. A decision that lives only in conversation is not a decision (per `pattern-decision-persistence`).

**Work-project mode (IMPORTANT).** If the project is a **work / team** project under the docs-as-code model (repo + GitHub Project as SSOT — see `pattern-platform-governance`; detect via the project's `AGENTS.md` / CLAUDE overlay), then:
- Write the ADR to the **repo** `docs/adr/adr-NNN-<slug>.md` (cross-cutting in the platform repo; repo-specific in the owning repo) via a PR — **NOT** the vault `30-architecture/`.
- Track tasks/roadmap in the **GitHub Project + Issues** — NOT the vault `11-tasks`.
- The vault keeps only personal/methodology lessons + AI memory + drafts.
For **personal** projects on the placement model, ADRs ALSO go to the repo `docs/adr/` (per [[pattern-knowledge-placement]]); only the `11-tasks.md` / `memory/` steps below stay in the vault. The work-vs-personal split is now only about *tracking* (GitHub Project vs issues), not ADR placement.

**Steps:**

1. **Author the ADR draft.** Use `$VAULT_PATH/00_meta/templates/adr.md`. Path: the repo's `docs/adr/adr-NNN-<slug>.md` where NNN = (highest existing ADR + 1). (Legacy pre-migration project still vault-only: `$PROJECT_AREA/30-architecture/`.)
   - Required sections: Status, Date, Context, Options Considered, Decision, Rationale, Consequences (positive/negative/neutral), References.
   - If decision is deferred: status `deferred`, plus explicit "Triggers to Reopen Decision" section (per ADR-016 model).
   - **GitHub issue link:** if a GitHub Project issue or task tracks this architectural decision, populate `issue: <repo>#NNN` in the ADR frontmatter before writing.
2. **Update the plan** of record. If a `plan-*.md` motivated this session, patch it: append a "Decision recorded" line linking the new ADR, and update the "Next steps" section if the decision changes it.
3. **Index:** the repo `docs/adr/` self-indexes (GitHub renders the directory) — no separate index file to patch. (Legacy vault-only project: patch the project `_index.md`.)
4. **Update `11-tasks.md`** if the decision creates, closes, or reshapes tasks. Apply backlog ticks `- [x]` and add new tasks with the ADR-NNN reference inline.
5. **Capture a lesson** (optional) via `mcp__hive__capture_lesson` if the discussion surfaced a non-obvious insight that future sessions should not re-derive.

**Blocking rule.** Phase E does not exit until at least the ADR file is written. The user CANNOT defer the write to "later" -- per `pattern-decision-persistence` "Anti-Patterns", `I'll update the vault later` = the decision is lost.

**Override.** If the user genuinely cannot author an ADR this turn (interruption, missing data), Phase E creates a tracked task in `11-tasks.md` named `**ADR-NNN-draft**: write ADR for <topic>` with `(spec: ADR-NNN-...)` placeholder. Verbal deferral never counts.

**Output of Phase E:** new ADR file (repo `docs/adr/`, self-indexing) + patched plan + patched 11-tasks.

---

## Phase F -- Recap

**Goal:** persist session continuity (handoff) and propagate cross-cutting deltas to the working guide if a trigger fired.

**Steps:**

1. **MEMORY.md Session Handoff.** OVERWRITE the `## Session Handoff` section in `$PROJECT_AREA/memory/MEMORY.md` (max 8 lines, format per `~/.claude/CLAUDE.md` Auto-Maintenance Rules).
2. **CURRENT-STATE.md delta** (Exit Phase 2, conditional). Check `pattern-workflow-protocol` trigger dimensions:
   - Skill added / removed / renamed -> propose delta to CURRENT-STATE.md section 1
   - Automation changed -> section 2
   - Lifecycle diagram changed -> section 3
   - Path topology shifted -> section 4
   - Mental model branch / new anti-pattern -> section 5 / 7
   - Date close marker (sprint boundary) -> section 6

   Emit a `vault_patch` proposal for ONLY the affected sections; do not full-rewrite CURRENT-STATE.md.
3. **claude-mem observation.** The session hook captures conversation automatically; no manual `observation_add` needed (per `~/.claude/CLAUDE.md` claude-mem section).
4. **Schedule follow-ups (no `/schedule`).** Architecture sessions rarely produce timer-based follow-ups. If the decision has an explicit reopen trigger (per Phase D deferral), the trigger lives in the ADR itself -- not in `/schedule`.

**Output of Phase F:** patched MEMORY.md handoff + (conditional) CURRENT-STATE.md delta proposal.

---

## Integration with other skills

- **Downstream `/spec init`.** If the new ADR creates feature-level work, propose `/spec init <feature-id>` for the first task at the end of Phase E. The ADR is the "Why" upstream of the spec.
- **Downstream `/adversarial-review`.** When the implementation of the ADR's decision lands, run `/adversarial-review` before archiving the spec -- this catches drift between ADR intent and code reality.
- **Upstream `enrich-us`.** If a backlog entry that motivates this session is too thin, run `enrich-us` BEFORE Phase A -- avoids re-asking what should already be in the ticket.
- **Sibling `/crystallize`.** Architecture sessions can surface lesson-promotion candidates. Use `capture_lesson` in Phase E or queue a `/crystallize` for next session.

## Cross-OS notes

- Linux / macOS: agent uses POSIX commands via `Bash`. Path joining uses `/`.
- Windows: agent uses PowerShell tool. Path joining uses `\`. Hard-copy deployment (not symlinks) per dotfiles `setup-windows.ps1` -- the skill in `~/.claude/skills/architecture-session/` is a COPY of vault SSOT, NOT a symlink. After editing the vault SSOT, the user re-runs `setup-windows.ps1` to refresh.

## Vault connections

| Phase | Reads | Writes |
|---|---|---|
| A | `00-context.md`, `10-roadmap.md`, `11-tasks.md`, repo `docs/adr/`, `plan-*.md`, `memory/MEMORY.md`, `session-protocol.md` | nothing (read-only verification) |
| B | `sensor-reference-audit-matrix.md` (or analog) | patches matrix + divergence-log |
| C | `session-protocol.md` (constraints section) | patches constraint table |
| D | `session-protocol.md` (rejection list), `00_meta/patterns/_index.md` | nothing (or patches rejection list if new alternatives discarded) |
| E | `templates/adr.md` | new repo `docs/adr/adr-NNN-*.md` (self-indexing), patches `plan-*.md`, patches `11-tasks.md` |
| F | (nothing) | patches `memory/MEMORY.md` handoff; OPTIONAL delta proposal for `CURRENT-STATE.md` |

## Anti-patterns this skill blocks

- **Phase A skipped** -- re-deciding what was already decided, or building on stale state. Phase A is the cheapest insurance.
- **Phase B speculated** -- generalizing from 1 reference, then rewriting at sensor 2. Regla del 3 is not advice.
- **Phase E deferred** -- decisions captured in MEMORY.md or "I'll write the ADR later" -- per `pattern-decision-persistence`, these are lost.
- **Conflating with `/spec init`** -- architecture session produces ADRs; spec produces code. If the day's work is one feature, use `/spec init`, not this.
- **No rejection list discipline** -- re-debating alternatives discarded last sprint. The rejection list in `session-protocol.md` is the SSOT for "we already evaluated this".

## References

- Pattern: `$VAULT_PATH/00_meta/patterns/pattern-decision-persistence.md`
- Pattern: `$VAULT_PATH/00_meta/patterns/pattern-workflow-protocol.md` (Exit Phase 2)
- Pattern: `$VAULT_PATH/00_meta/patterns/pattern-spec-driven-development.md` (downstream after Phase E)
- Pattern: `$VAULT_PATH/00_meta/patterns/pattern-architecture.md` (project structure conventions)
- Template: `$VAULT_PATH/00_meta/templates/adr.md`
- Template: `$VAULT_PATH/00_meta/templates/system-design.md`
- Sibling skill: `$VAULT_PATH/00_meta/skills/spec/SKILL.md`
- Sibling skill: `$VAULT_PATH/00_meta/skills/adversarial-review/SKILL.md`
- Sibling skill: `$VAULT_PATH/00_meta/skills/enrich-us/SKILL.md`
- Origin and dogfood: ADR-019 (python-sensor-sdk-platform) -- this skill adopted for the platform project.
