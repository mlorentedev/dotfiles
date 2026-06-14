---
id: adversarial-review-skill
type: skill
status: active
created: "2026-05-14"
name: adversarial-review
description: Independent red-team verification pass on a spec-driven change BEFORE archiving. Triggers on /adversarial-review, "red-team this", "devil's advocate AI-001", "independent verification before archive". Reads `specs/<feature-id>/{proposal,tasks,verification}.md` + diff/PR, refutes acceptance criteria, classifies findings Blocker/Major/Minor, issues PASS/PASS-WITH-GAPS/FAIL verdict. Pairs with /spec archive lock. Ported and adapted from LIDR-academy/lidr-specboot.
allowed-tools: [Bash, Read, Grep, mcp__hive__vault_query, mcp__hive__vault_search]
---

# Adversarial Review (Red-Team Gate)

> Act as an **independent adversarial reviewer**: assume gaps, flaws, or unsafe behavior may exist until you have argued against them with evidence. Intended for the **verification window** of spec-driven development (after implementation, BEFORE running `/spec archive` or `dotf spec archive`), ideally run by a different agent/session than the one that implemented the change.
>
> **Do NOT** prescribe which agent, model, or IDE to use. That is the human's choice. Default to skepticism over agreement; refuse to rubber-stamp.
>
> **Origin:** ported from [LIDR-academy/lidr-specboot](https://github.com/LIDR-academy/lidr-specboot/blob/main/ai-specs/skills/adversarial-review/SKILL.md) (`adversarial-review`, MIT). Adapted: OpenSpec references replaced with `pattern-spec-driven-development` artifacts (`specs/<feature-id>/{proposal,tasks,verification}.md`); archive command updated to vault-rooted `/spec archive`.

## When to use

- `/adversarial-review <feature-id-or-PR>` explicitly.
- "Red-team this change" / "devil's advocate pass on AI-001" / "independent verification before archive".
- Verification window: after implementation merged / about-to-merge, BEFORE archiving the spec.
- **Pair with**: `/spec archive` lock — that command refuses to archive while any `[AGENT-DRAFT]` tags remain; this skill catches the gaps *before* that final gate.

## When NOT to use

- During implementation (use `enrich-us` or `/spec fill` instead — wrong phase).
- For trivial changes that bypassed SDD per Skip rules.
- As a single-agent self-review (the value is *independence* — different session/agent from the implementer).

## Inputs

Auto-detected, in this resolution order:

1. **Explicit feature-id** (e.g. `SDD-014`, `AI-001-ollama-public`) → load `specs/<feature-id>/{proposal,tasks,verification}.md` from the current repo.
2. **PR reference** (`owner/repo#42`, full URL) → use `gh pr view` + `gh pr diff` for diff scope; cross-reference with the linked spec.
3. **Current active work** — branch name often matches feature-id; infer.

If ambiguous, ask. Do not guess.

## Mindset (adversarial review)

- **Try to break the system**, not only to confirm happy paths.
- **Hunt incorrect assumptions** about data shape, timing, ordering, authz, idempotency, error handling.
- **Trace cross-boundary risks**: pieces that look fine in isolation but fail together (multi-file, API + UI, retries + side effects, race conditions).
- **Treat the diff as incomplete context**: missing tests, missing negative paths, or spec drift can hide issues.
- **Calibrate depth to risk**: auth, payments, PII, privilege boundaries, data mutation, secrets — strictest scrutiny. Read-only telemetry — lower bar.

## Workflow

### Step 1 — Load the specification side first

1. Read `specs/<feature-id>/proposal.md` → extract **acceptance criteria** and **explicit non-goals**. List what must be true for "done."
2. Read `specs/<feature-id>/tasks.md` → check which tasks claim `[x]` vs `[ ]`. A `[x]` without diff evidence is a finding.
3. Read `specs/<feature-id>/verification.md` → check the user/agent's own verification artifacts. Do they prove the criteria, or only happy paths?
4. Note anything **underspecified** (ambiguous acceptance, missing error cases, missing security constraints) — these are first-class findings, not waivers.

### Step 2 — Load the implementation side

1. If a **PR** was provided: `gh pr view <ref> --json title,body,files` + `gh pr diff <ref>`. Read the full diff scope (not only the default file ordering). Map files-and-changes to spec sections and tasks.
2. If no PR: `git diff <base>...HEAD` against the merge base (typically `main`/`master`).
3. Also check `git log <base>..HEAD` for context the diff hides (commit messages may reveal incomplete work or reverts).

### Step 3 — Adversarial pass (refute, do not rubber-stamp)

For each acceptance criterion / scenario:

1. State how the implementation **could still fail** while the author believed it passed (wrong input, partial failure, double-submit, stale cache, wrong role, race, empty state, oversized payload, timezone drift, off-by-one, integer overflow).
2. Check **negative and abuse cases**: validation bypass strings, IDOR-style access patterns, replay, conflict handling, command injection if shell-adjacent, SSRF if URL-adjacent.
3. Check **tests and verification artifacts**: do they **prove** the criterion, or only the happy path? Lack of negative tests is a Major finding by default.
4. Record **spec vs code mismatches** (spec says X, code does Y) as first-class findings — never silently accept code overriding spec.

## Severity and recommendations

Classify each finding:

- **Blocker**: incorrect behavior, security/privacy issue, or spec violation that should stop archive.
- **Major**: likely bug or significant gap; fix or spec update required before archive.
- **Minor**: clarity, maintainability, or low-risk gap; can follow up.
- **Question / assumption**: needs human or author confirmation.

For each finding, state whether the fix belongs in **code**, **tests**, **spec artifacts** (proposal/tasks/verification), or **vault** (new ADR / pattern promotion candidate).

## Evaluator Rubric (quantitative — SDD-028c)

Orthogonal to the Blocker/Major/Minor severity of individual findings, grade the change across **six dimensions on an A-D scale**. The severity axis answers "how bad is this specific issue"; the rubric axis answers "how solid is the change overall by dimension". Both are required.

The rubric is mechanically aggregable — useful when N agents review N changes in parallel and trendlines matter (e.g. "verification grade is consistently C for one team / one repo / one phase").

### Dimensions

| Dimension | A (Exemplary) | B (Solid) | C (Below bar) | D (Failing) |
|---|---|---|---|---|
| **Correctness** | All acceptance criteria verified, negative paths covered, no observed defects | Criteria met on happy path, minor negative-path gaps | Criteria partially met OR substantial negative-path gaps | Criteria not met OR defects present |
| **Verification** | Evidence proves each criterion with reproducible commands + outputs | Evidence covers criteria but not reproducible without context | Anecdotal or unverifiable evidence | No verification artifacts |
| **Scope** | Diff matches proposal exactly; no creep | Diff mostly matches; minor side-changes documented | Significant unrelated changes mixed in | Diff materially diverges from proposal |
| **Reliability** | Error paths handled, idempotent, rollback safe | Most error paths handled, partial idempotency | Several error paths unhandled / unclear | Crashes or silent failures on common errors |
| **Maintainability** | Clear naming, ≤40-line fns, no dead code, comments explain WHY | Acceptable structure with minor smells | Confusing structure, smells, or no tests for new logic | Unreviewable; needs rewrite |
| **Handoff-readiness** | Spec updates included, lessons captured if applicable, clear next steps | Spec updates included but lesson capture deferred | Implementation only; spec stale | No spec touch; lessons lost |

### Aggregation rule (rubric → verdict)

The rubric and the verdict are joined via this rule (no judgment, mechanical):

- Any **D** in any dimension → **FAIL** verdict.
- Any **C** + no D → **PASS WITH GAPS** minimum.
- All **B** or above → **PASS**.
- All **A** → **PASS** (note "Exemplary" optionally).

The severity findings axis (Blocker/Major/Minor) can still escalate independently — a single Blocker forces FAIL regardless of rubric grades. Use the more severe of the two paths.

## Verdict

End with a clear verdict:

- **PASS (adversarial)** — no blockers or majors; rubric all B or above; minors listed optionally.
- **PASS WITH GAPS** — minors only OR rubric has at least one C (no D); both tracked.
- **FAIL** — at least one blocker or major OR rubric has at least one D, until addressed.

## Output format

```markdown
## Adversarial review

**Scope**: <feature-id / PR>
**Sources**: <spec paths + PR/diff reference>

### Spec and task alignment
- ...

### Findings

| Severity | Area | Finding | Evidence | Fix location (code / tests / spec / vault) |
|----------|------|---------|----------|---------------------------------------------|
| Blocker  |      |         |          |                                             |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        |  |  |
| Verification       |  |  |
| Scope              |  |  |
| Reliability        |  |  |
| Maintainability    |  |  |
| Handoff-readiness  |  |  |

### Verdict
PASS | PASS WITH GAPS | FAIL

### Recommended next steps (before archive)
- ...
```

## Guardrails

- **Do not praise** implementation to "balance" criticism unless a strength **directly mitigates a documented risk**.
- **Do not skip** reading spec artifacts when they exist. Skipping the proposal/tasks/verification triad is a process failure, not a shortcut.
- If you cannot access the PR or diff, say so and list exactly what is needed to continue. Do not fabricate findings from imagined diffs.
- Refuse to issue PASS if `[AGENT-DRAFT]` or `[AGENT-SUGGESTION]` tags remain in any spec file — that is `/spec archive`'s own gate, but flag it here too so the human catches it earlier.

## Completion

Always end with: (a) the verdict, (b) whether `dotf spec archive` / `/spec archive` is **advisable** in the current state, (c) if FAIL, the minimum set of actions that would flip it to PASS.
