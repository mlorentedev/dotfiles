---
id: spec-skill
type: skill
status: active
created: "2026-05-13"
name: spec
description: "Manage Spec-Driven Development per-feature artifacts. Triggers on /spec, \"create a spec for\", \"scaffold spec X\", \"bootstrap substrate for X\", \"fill proposal for X\", \"archive spec X\". Four subcommands: init (scaffold, gated on an open GitHub issue per ADR-018), bootstrap (optional 4-section substrate contract), fill (Socratic 5-question proposal), archive (move + selective vault promotion). Cross-OS Linux/Windows, cross-agent Claude/Copilot via AGENTS.md indirection."
allowed-tools: [Bash, Read, Edit, Write, mcp__hive__vault_query, mcp__hive__vault_search, mcp__hive__vault_write, mcp__hive__vault_patch]
---

# Spec Workflow

> Implements `pattern-spec-driven-development`. Four subcommands: `init`, `bootstrap` (optional), `fill`, `archive`.
> **Core principle:** every spec is downstream of an OPEN GitHub issue on the bitácora Project — the work-gate per ADR-018. The vault keeps templates and patterns; task state lives in GitHub.

## When to use

- `/spec init|bootstrap|fill|archive <feature-id>` explicitly.
- "Create a spec for X" / "scaffold spec X" / "start working on X" -> `init`.
- "Bootstrap substrate for X" / "the substrate for X doesn't exist yet" / "new runtime needed before features" -> `bootstrap`.
- "Fill in proposal for X" / "help me write the proposal" -> `fill`.
- "Archive spec X" / "close spec X" -> `archive`.

## When NOT to use

- Trivial change (typo, comment-only, mechanical rename) — per pattern's "Skip SDD" rules.
- Strategic planning across milestones -> use `/writing-plans`.
- If you cannot point to ANY work-gate justifying this spec (open GitHub issue, ADR, roadmap) — don't run the skill yet; open the bitácora issue first.

---

## Agent-Side Activation Rule

> **Proactive mode.** The subcommands below are *reactive* — they run when a human types `/spec …`. This rule makes the agent *proactive*: when work is being scoped in conversation and a Discipline Gate trigger is met, the agent applies the Skip-SDD heuristic itself and PROPOSES `/spec init <id>` — it does not wait to be asked. The always-on trigger that primes this behavior lives in `AGENTS.md` ("Discipline Gate"); this section is the SSOT for *how* the agent decides and *how* it phrases the proposal.

### Checks the agent runs

When a change is being planned or has just been described, silently evaluate it against the Discipline Gate triggers (authoritative list: `AGENTS.md` → "Discipline Gate (NON-NEGOTIABLE)" / `pattern-spec-driven-development.md` → "Trigger Criteria"):

1. **Size** — does the change look like ~50–300+ LOC of production diff (excluding tests, generated files, lockfiles)?
2. **Public contract** — does it touch an API, CLI flag, exported type, alias name, file path, deployed config schema, or *agent behavior*?
3. **Dependency** — does it add or remove a dependency?
4. **Multi-PR** — is it the first step of a multi-PR sequence?
5. **Socratic** — does it warrant a Socratic Guardrail pause (architecture, schema design, concurrency, breaking change)?

If ANY trigger fires AND none of the "When NOT to propose" conditions hold → propose a spec.

### How to phrase the proposal

Surface a short, skippable proposal that states the evidence — don't just assert. Template:

> This looks like a Discipline-Gate trigger: **<which trigger(s) fired>** (e.g. "~120 LOC + touches a deployed config schema"). I ran the Skip-SDD checks — it does **not** qualify for skip. Propose `/spec init <feature-id>` before writing code? (work-gate = an open GitHub issue — reuse one or open it now.)

- **List the checks you ran**, not just the verdict — the evidence is the value (mirrors the Model-Tier "the proposal IS the value" rule in `AGENTS.md`).
- **Offer, don't impose.** The human decides. If they decline, proceed without the spec and do not re-propose for the same change.
- **Suggest the id.** Derive `<feature-id>` from the gating GitHub issue if one matches; otherwise propose opening the issue first (the work-gate).

### When NOT to propose

Silence is correct when ANY of these hold — proposing here is noise:

- **Trivial change** per the Skip-SDD list: typo, comment-only, mechanical rename, doc-only, or an obvious bugfix <20 LOC.
- **Already inside an active spec** — `specs/<id>/` exists for this work; just keep implementing and ticking `tasks.md`.
- **User already declined** a spec for this change in the current thread (once-per-change debounce — do not nag).
- **Pure exploration / question** — no change is being committed yet.
- **Emergency / hotfix** the user has explicitly fast-tracked.

When unsure whether a change crosses the threshold, ASK rather than assume (`AGENTS.md`: "When in doubt, ASK the user") — a one-line "want a spec for this?" is cheaper than either a missed spec or an unwanted one.

---

## Environment (resolve once per invocation)

- `$VAULT_PATH` — env var. Cross-OS default `$HOME/Projects/knowledge` (Linux/macOS) or `%USERPROFILE%\Projects\knowledge` (Windows).
- `$REPO_ROOT` = `git rev-parse --show-toplevel`.
- `$REPO_NAME` = basename of `$REPO_ROOT`.
- Fail fast if `$VAULT_PATH` unresolvable AND a vault operation is required.

---

## Subcommand: init

**Purpose:** Scaffold `$REPO_ROOT/specs/<feature-id>/` from vault templates. Work-gated per ADR-018: requires an OPEN GitHub issue (bitácora) before scaffolding.

**Signature:** `/spec init <feature-id> [--issue <number>] [--force-no-gate]` (`--force-no-vault` is a deprecated alias of `--force-no-gate`)

**Steps:**

1. **Validate id** matches `^[A-Z]+-\d+[a-z]?(-[a-z0-9-]+)?$` OR `^\d{4}-\d{2}-\d{2}-[a-z0-9-]+$`. Reject otherwise.
2. **No clobber:** fail if `$REPO_ROOT/specs/<feature-id>/` exists. Warn if `specs/archive/<feature-id>/` exists.
3. **Work-gate pre-flight (mandatory):**
   - The gate is an OPEN GitHub issue on the repo (tracked in the bitácora Project). Verify via `gh issue view <number> --json state,title`: the issue must exist and be `OPEN`.
   - If `--issue` was not given, ask the user which issue gates this work (suggest candidates via `gh issue list --state open` if helpful).
   - If the issue is missing or closed -> present three options to user:
     - **(a)** Open (or reopen) the gating issue now, add it to the bitácora Project, then proceed with its number.
     - **(b)** Cancel. User sorts out the issue manually, then re-runs `/spec init`.
     - **(c)** Force-proceed via `--force-no-gate` flag (NOT RECOMMENDED — violates the work-gate discipline; emit warning).
   - Wait for choice. Note the gating issue for step 7.
4. `mkdir -p $REPO_ROOT/specs/<feature-id>/`.
5. Read 3 templates from `$VAULT_PATH/00_meta/templates/spec-{proposal,tasks,verification}.md`.
6. Substitute placeholders in memory:
   - `<feature-id>` -> actual id
   - `{TITLE}` -> derived from id (drop ticket prefix, title-case the slug). E.g. `AI-001-ollama-public` -> `AI-001: Ollama public`.
   - `{{date:YYYY-MM-DD}}` -> today (UTC).
   - `template_version: "1.0"` -> hardcoded v1.
7. **Inject issue context** in proposal `## Why` as HTML comment: `<!-- from issue #<number>: <issue-title> -->`, and set the `issue:` frontmatter field to `<repo>#<number>`. Always injected when the gate was verified.
8. Write the 3 substituted files.
9. **Output:**
   - Paths created.
   - Gating issue used (or `none — force-proceeded`).
   - Next step: `/spec fill <feature-id>`.

**Edge cases:**
- `gh` unavailable or unauthenticated: gate cannot be verified -> fail with instructions (install/auth `gh`) or `--force-no-gate`.
- Templates missing in vault -> fail with path hint.
- Id collides with archived spec -> warn but allow (user may be reviving).
- Mechanical fallback: `dotf spec init <id> --issue <N>` implements this same gate for non-interactive use (cross-platform Go CLI on PATH).

---

## Subcommand: bootstrap

**Purpose:** Author `bootstrap-contract.md` for specs that scaffold a NEW substrate (new repo, new worker runtime, new microservice). Normative doc — a peer with this file + repo HEAD on a clean machine must reach passing smoke test. Differs from `00-context.md` (descriptive, vault).

**Signature:** `/spec bootstrap <feature-id>`

**When to apply:** spec scaffolds a new runtime substrate that must exist before feature code can run (e.g., new Go motor, new Python worker pool, new external integration with its own dev env).

**When to SKIP:** feature work inside an existing runnable codebase; doc-only / config-only / one-shot changes. Most specs skip `bootstrap`.

**Pre-flight:**

1. Verify `$REPO_ROOT/specs/<feature-id>/proposal.md` exists. If not -> suggest `/spec init` first.
2. If `bootstrap-contract.md` already has non-placeholder content, ask: overwrite / append / abort.
3. Copy template from `$VAULT_PATH/00_meta/templates/bootstrap-contract.md` into the spec folder. Substitute `<feature-id>`, `{TITLE}`, `{{date:YYYY-MM-DD}}` per `init` semantics.
4. Read context silently: patterns referenced in `proposal.md`, sister bootstrap contracts in `$REPO_ROOT/specs/archive/` (if any) for tone.

**The 4 questions (one at a time, wait for each answer):**

| # | Section | Question prompt |
|---|---|---|
| Q1 | Runnable environment | "Single command that brings the substrate up locally. Paste the command + any required prerequisites (env vars, ports, host binaries)." |
| Q2 | Passing smoke test | "Single command that exits 0 iff substrate works end-to-end. Must run right after Q1." |
| Q3 | Contract document | "Normative interfaces this substrate exposes. Reference existing patterns / `features.json` IDs — don't redefine. Stability guarantee?" |
| Q4 | Task breakdown | "Decompose substrate work into ≤5 atomic tasks. Each row: outcome + verification step." |

**Capture rules:** identical to `/spec fill` — `skip`/`idk`/`TODO` -> `[AGENT-DRAFT — review before archive]`; partial answers -> `[AGENT-SUGGESTION — accept or remove]`; archive lock applies (overridable via `--force-with-drafts`).

**Closing:**
- Restate acceptance signal: peer + clean machine + this doc -> passing smoke test.
- Next step: `/spec fill <feature-id>` (if not done), then execute Q4 task breakdown until smoke test green; after that, return to feature-level `tasks.md`.

**Edge cases:**
- All 4 answers `skip` -> warn: bootstrap likely not needed; suggest deleting `bootstrap-contract.md` and going straight to `/spec fill`.
- Smoke test command does not exist yet (Q2) -> Q2 stays `[AGENT-DRAFT]`; the FIRST row of Q4's breakdown should be "implement smoke test command".
- User invokes `bootstrap` after `fill` (out of order) -> allowed; warn that this implies the substrate was implicit and now needs to be made explicit retroactively. No data loss either way.

---

## Subcommand: fill

**Purpose:** Socratic 5-question dialog to fill `proposal.md`. NOT autonomous — agent asks, user answers, agent captures verbatim.

**Signature:** `/spec fill <feature-id>`

**Pre-flight:**

1. Verify `$REPO_ROOT/specs/<feature-id>/proposal.md` exists. If not -> suggest `/spec init` first.
2. For each section: if non-placeholder content present, ask: "Section `<X>` is already filled. Overwrite, append, or skip?"
3. Read context for grounding (silent, not shown to user):
   - `$VAULT_PATH/10_projects/$REPO_NAME/11-tasks.md` (find feature entry).
   - `$VAULT_PATH/10_projects/$REPO_NAME/10-roadmap.md` (strategic frame).
   - Referenced ADRs (frontmatter only via Hive).
   - Up to 3 sister specs in `$REPO_ROOT/specs/archive/` (for tone consistency).
4. **GitHub issue link:** if the `issue:` frontmatter field is empty and a GitHub Project issue exists for this spec, ask: "What is the GitHub issue for this spec? (e.g. `kubelab#123` or `skip`)" and populate the field before proceeding.

**The 5 questions (one at a time, wait for each answer):**

| # | Section | Question prompt |
|---|---|---|
| Q1 | Why | "In 2-3 sentences: what user or business pain does this solve? What breaks if we don't ship this?" |
| Q2 | What | "What new endpoint, API, or system behavior exists? Give 1-3 concrete outputs the system produces." |
| Q3 | Out of scope | "List up to 3 things that should NOT sneak into this PR (avoid scope creep)." |
| Q4 | Risks / open questions | "Top 1-3 risks or open questions. Mark which MUST be resolved before any code is written." |
| Q5 | Acceptance criteria | "Give 2-4 testable, observable outcomes. Each must be verifiable by a test or smoke check." |
| Q6 | Completeness review | "Looking at all sections: any typically-expected items missing? Agent will suggest based on context (e.g. rate limit, regression test, idempotency, cert provisioning, cost guard). Add any, or skip." |

**Capture rules:**
- Write each answer verbatim into the corresponding section via `Edit`.
- **`skip` semantics:** if user types `skip`, `idk`, or `TODO`, agent generates a draft and writes it tagged as `[AGENT-DRAFT — review before archive]`. User accepts, modifies, or deletes before archive. Tag is visible and persists until removed.
- **Partial answers:** if user provides fewer items than the question asks for (e.g., 2 acceptance criteria when 4 were requested), agent generates suggestions for the remaining items tagged as `[AGENT-SUGGESTION — accept or remove]`. User decides.
- **Off-script:** if user asks an unrelated question or writes an essay, redirect to current Q. Don't write essays to the spec.
- **Archive lock:** `/spec archive` REFUSES to archive if any `[AGENT-DRAFT]` or `[AGENT-SUGGESTION]` tags remain. User must convert tagged content to plain content (accept, edit, or delete) before archive. Override only via `--force-with-drafts` flag (NOT RECOMMENDED — flagged in archive output).

**Closing:**
- Summary: list TODO markers (if any), list sections captured.
- Ask: "Open questions in Risks section — any that BLOCK tasks.md being frozen?" If yes, flag them.
- Output: "proposal.md drafted. Next: resolve open questions, then start implementing per tasks.md."

**Edge cases:**
- User wants to redo a question -> re-ask, overwrite the section.
- Vault unreachable during context fetch -> degrade gracefully (run without context, ask user to provide manually).
- All 5 answers are `skip` -> warn user that spec has no content; offer to delete via `archive --abandoned` (v2 feature).

---

## Subcommand: archive

**Purpose:** Close a spec. Interactive promotion to vault, then mechanical archive.

**Signature:** `/spec archive <feature-id> [--pr <url>] [--abandoned]`

**Steps:**

1. **Pre-flight:**
   - Read `$REPO_ROOT/specs/<feature-id>/proposal.md`, `tasks.md`, `verification.md`.
   - **Tag check:** scan all three files for `[AGENT-DRAFT]` or `[AGENT-SUGGESTION]` markers. If any found, REFUSE to archive unless `--force-with-drafts` is passed. Output list of files + lines with unresolved tags.
   - Count unchecked acceptance criteria. If >0, warn: "N criteria still unchecked. Continue?" Ask.
   - If `--abandoned` flag: skip step 2 entirely, mark as `status: abandoned`, route to `specs/archive/_abandoned/<id>/` in step 3.

2. **Promotion candidates (always interactive — never autoparse):**
   For each of the three promotion types, ASK the user regardless of `verification.md` marker (the marker is a hint, not authoritative):
   - **Lesson?** "Any non-obvious lesson worth recording? 2-sentence summary, or `no`."
     - If non-`no`: compose lesson entry -> append to the **repo's** `docs/lessons.md` (project lessons live in the repo — see [[pattern-knowledge-placement]]; never a vault `90-lessons.md`). A genuinely cross-project / methodology lesson goes to `00_meta/` (promote to a pattern).
   - **ADR-worthy?** "Any architectural decision that future-you needs to remember? ADR title, or `no`."
     - If non-`no`: ask for ADR number (query existing ADRs in the **repo's** `docs/adr/` first to suggest next sequential) and 1-line decision summary. Create skeleton at the **repo's** `docs/adr/adr-XXX-<slug>.md` (ADRs live in the repo — see [[pattern-knowledge-placement]]; the vault keeps only cross-project decisions in `00_meta/`).
   - **Pattern candidate?** "Does this approach recur in >1 project? Pattern name, or `no`."
     - If non-`no`: create skeleton at `$VAULT_PATH/00_meta/patterns/pattern-<name>.md` via Hive.

3. **Mechanical archive:**
   - Standard path: `mkdir -p $REPO_ROOT/specs/archive/`.
   - Abandoned path: `mkdir -p $REPO_ROOT/specs/archive/_abandoned/`.
   - `mv $REPO_ROOT/specs/<feature-id>/ <target>/<feature-id>/`.
   - In archived `proposal.md`: `Edit` `status: <whatever>` -> `status: archived` (or `abandoned`).

4. **Update vault backlog:**
   - Via Hive `vault_patch` on `$VAULT_PATH/10_projects/$REPO_NAME/11-tasks.md`:
     - Replace `- [ ] **<feature-id>**:` -> `- [x] **<feature-id>**:` and append ` ✓ <today>` plus ` (PR: <url>)` if `--pr` provided.
     - If `--abandoned`: replace `- [ ] **<feature-id>**:` -> `- [-] **<feature-id>**:` (or strikethrough convention) and append ` ✗ <today> (abandoned)`.

5. **Output:**
   - List of artifacts created/moved (paths).
   - Reminder: "Stage the archive in repo. Vault edits auto-committed via Hive."

**Edge cases:**
- User aborts mid-promotion -> DO NOT execute step 3 (mechanical archive). Leave spec in place. State recoverable.
- ADR number collision -> query existing ADRs via Hive, suggest next available.
- Lesson append fails (Hive unavailable) -> save draft to `$REPO_ROOT/specs/<feature-id>/_pending-lesson.md` for retry; do not lose user input.

---

## Integration with existing skills

- **Pre-archive:** recommend invoking `verification-before-completion` for evidence audit.
- **Post-archive deeper crystallization:** `crystallize` can promote a lesson further to a pattern if recurrence detected.
- **Independent of** `code-review`, `commit-commands:commit-push-pr`, `pr-review-toolkit:*` — those are pre/post-merge gates, separate axis.

## Cross-OS notes

- Linux/macOS: agent uses POSIX commands via `Bash` tool.
- Windows native (Copilot): agent reads this SKILL via `AGENTS.md` indirection at `$VAULT_PATH/00_meta/skills/spec/SKILL.md` and invokes the `dotf spec` CLI from dotfiles (`dotf spec init` / `dotf spec archive`) where mechanical ops are needed.
- Path joining: never hardcode `/` or `\`; agent uses platform-appropriate joining.

## Vault connections (5 touchpoints)

| Subcommand | Reads | Writes |
|---|---|---|
| `init` (pre-flight) | GitHub issue via `gh` (work-gate, not a vault read) | nothing |
| `init` (substitution) | `00_meta/templates/spec-*.md` | (filesystem only — repo specs/) |
| `bootstrap` (template) | `00_meta/templates/bootstrap-contract.md`, sister contracts in `specs/archive/` | (filesystem only — repo specs/<id>/bootstrap-contract.md) |
| `fill` (grounding) | `11-tasks.md`, `10-roadmap.md`, referenced ADRs, sister specs | nothing |
| `archive` (promotion) | `verification.md` flags | repo `docs/lessons.md`, repo `docs/adr/adr-XXX.md`, `00_meta/patterns/` (cross-project only) |
| `archive` (backlog tick) | `11-tasks.md` | `11-tasks.md` (tick + PR link) |

## References

- Pattern: `$VAULT_PATH/00_meta/patterns/pattern-spec-driven-development.md`
- Templates: `$VAULT_PATH/00_meta/templates/spec-{proposal,tasks,verification}.md`
- Bootstrap template (optional): `$VAULT_PATH/00_meta/templates/bootstrap-contract.md`
- Cross-OS env: `$VAULT_PATH/00_meta/patterns/pattern-workflow-protocol.md`
- ADR template: `$VAULT_PATH/00_meta/templates/adr.md`
- Lesson template: `$VAULT_PATH/00_meta/templates/lesson.md`
- Decision Persistence: `$VAULT_PATH/00_meta/patterns/pattern-decision-persistence.md` — reinforces vault-first principle.
