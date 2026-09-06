---
generated: true
generated_from: 00_meta/skills/adversarial-review/SKILL.md
generated_sha: 4696ee5e688ab476
id: adversarial-review-skill
type: skill
status: active
created: '2026-05-14'
owner: manu
name: adversarial-review
description: Independent red-team verification pass on a spec-driven change BEFORE
  archiving. Triggers on /adversarial-review, "red-team this", "devil's advocate AI-001",
  "independent verification before archive". Reads `specs/<feature-id>/{proposal,tasks,verification}.md`
  + diff/PR, refutes acceptance criteria, classifies findings Blocker/Major/Minor,
  issues PASS/PASS-WITH-GAPS/FAIL verdict. Pairs with /spec archive lock. Ported and
  adapted from LIDR-academy/lidr-specboot.
allowed-tools: [Bash, Read, Grep, mcp__hive__vault_query, mcp__hive__vault_search]
keywords: [adversarial review, red team, devils advocate, review spec, independent
    verification, review gate, revisar spec]
paths: [specs/**/review.md, specs/**/verification.md]
requires: [verification-before-completion]
---
# Adversarial Review (Red-Team Gate)

> Act as an **independent adversarial reviewer**: assume gaps, flaws, or unsafe behavior may exist until you have argued against them with evidence. Intended for the **verification window** of spec-driven development (after implementation, BEFORE running `/spec archive` or `dotf spec archive`), ideally run by a different agent/session than the one that implemented the change.
>
> **Do not invent** which agent, model, or IDE to use — but do not treat it as open either. Where a repo declares a **reviewer pool**, the human has already made that choice and recorded it, and the pool is binding. Read it; do not re-litigate it. Where a repo declares none, the choice remains the human's and you propose rather than pick. Default to skepticism over agreement; refuse to rubber-stamp.
>
> **Origin:** ported from [LIDR-academy/lidr-specboot](https://github.com/LIDR-academy/lidr-specboot/blob/main/ai-specs/skills/adversarial-review/SKILL.md) (`adversarial-review`, MIT). Adapted: OpenSpec references replaced with `pattern-spec-driven-development` artifacts (`specs/<feature-id>/{proposal,tasks,verification}.md`); archive command updated to vault-rooted `/spec archive`.

## When to use

- `/adversarial-review <feature-id-or-PR>` explicitly.
- "Red-team this change" / "devil's advocate pass on AI-001" / "independent verification before archive".
- Verification window: after implementation merged / about-to-merge, BEFORE archiving the spec.
- **Pair with**: `/spec archive` lock — that command refuses to archive while any `[AGENT-DRAFT]` tags remain, **and** (since CLI-034) refuses without a fresh, passing `review.md`. This skill produces that artifact; skipping it now blocks the archive rather than merely weakening it.
- **Proposed, not only requested** (since HARNESS-064): an always-on trigger in `AGENTS.md` ("Spec-Driven Development" → *Proactive (verification window)*) makes an agent offer this review while the PR is still open. See "Agent-Side Activation Rule" below for how that decision is made and phrased.

## When NOT to use

- During implementation (use `enrich-us` or `/spec fill` instead — wrong phase).
- For trivial changes that bypassed SDD per Skip rules.
- As a single-agent self-review (the value is *independence* — different session/agent from the implementer).
- On a model outside the repo's reviewer pool, where one exists. Not merely discouraged: `dotf spec archive` refuses such a review, so it is wasted work.

## Launching a review

Where the repo declares `harness/reviewer-pool.json`, there is one entry point and it does the pinning for you:

```bash
dotf spec review <feature-id>                             # a pool member drawn at random (the launch line names it)
dotf spec review <feature-id> --reviewer <pool-id>        # a specific pool member
dotf spec review <feature-id> --dry-run                   # print the command, run nothing
dotf spec review <feature-id> --foreground                # no detached session
```

It resolves the model from the pool, passes provider and model **explicitly** to the runner, launches in a detached tmux session named `review-<feature-id>` so the run can be watched (`tmux attach -t review-<feature-id>`), and tees a machine-readable transcript beside `review.md`.

Three things not to work around:

- **Never fall back to a runner's default model.** A runner's configured default lives in unversioned per-machine state, so a review that "ran on the right model" on one box silently runs on another elsewhere — the pin has to be on the command line. This is a real incident, not a hypothetical: a review once counted as independent only because one machine's config happened to agree with the intent, while the tool's documented default was a different provider entirely.
- **The tool is not the guarantee; the model id is.** A single agent CLI can serve several providers, including the one you are trying to avoid. "Run it with `<tool>`" constrains nothing on its own.
- **Record `reviewer:` exactly as the pool spells it.** It is self-reported and matched exactly, so a different spelling of the right model is refused for a string mismatch — burning a whole review round on punctuation.

Without a pool, launch it however the human directs, and keep the prompt thin: the feature-id and the repo, no design rationale, or the independence is cosmetic.

## Agent-Side Activation Rule

> **Proactive mode.** This skill is otherwise *reactive* — it runs when a human types `/adversarial-review …`. This rule makes the agent *proactive*: when a spec's implementation is complete and its PR is about to be opened or merged, the agent PROPOSES the review itself. The always-on trigger that primes this lives in `AGENTS.md` ("Spec-Driven Development" → *Proactive (verification window)*); this section is the SSOT for *how* the agent decides and *how* it phrases the proposal.
>
> The agent proposes; it does **not** supply the verdict for a change it implemented. That is the single-agent self-review forbidden above. Where the repo declares a pool, the choice of reviewer was made by the human in advance and written down, so the agent may launch it without asking who — what it must never do is *be* it. Where there is no pool, the proposal names the choice instead of making it.

### Checks the agent runs

When implementation looks finished — tasks ticked, tests green, a PR being drafted or about to merge — silently evaluate:

1. **Active spec** — does `specs/<feature-id>/` exist with `status:` not yet `archived`? No spec, no proposal.
2. **Implementation complete** — are the `tasks.md` implementation boxes ticked and `verification.md` carrying evidence? Proposing mid-implementation is the wrong phase.
3. **No fresh review** — is `specs/<feature-id>/review.md` absent, or present with a `reviewed_sha` older than the last change to `proposal.md` / `tasks.md` / `features.json`? A fresh, passing review already meets the requirement.
4. **Not waived** — does `proposal.md` declare `review: waived` with a non-empty reason? A declared waiver is a decision already taken; do not re-litigate it.

If 1–3 hold and 4 does not → propose.

### How to phrase the proposal

State the evidence, name the consequence, and name the command. Where a pool exists the reviewer is already decided, so proposing means offering to *run* it — not asking who should. Template:

> `<feature-id>` is implemented and its PR is about to merge — this is the **verification window**. `dotf spec archive` will refuse without a fresh, passing `specs/<feature-id>/review.md`, and I implemented this change, so **I cannot be the reviewer**. The pool's primary is `<pool-primary-id>`. Run `dotf spec review <feature-id>`? It launches detached, so you can watch it with `tmux attach -t review-<feature-id>`.

Without a pool, the choice is still the human's, and the older phrasing applies: offer to run `/adversarial-review <feature-id>` in a separate session, with a deliberately thin prompt.

- **Say which session you are.** If you implemented the change, that fact is what makes the proposal necessary; leading with it is the evidence, not a disclaimer.
- **Name the escapes honestly.** If the review genuinely does not fit, the declared paths are `review: waived` + a reason in `proposal.md`, or `--force-without-review`. Surface them; never take one unilaterally.
- **Once per change.** If the user declines, proceed and do not re-propose for the same spec.

### When NOT to propose

Silence is correct when ANY of these hold:

- **No active spec** — the change was Skip-SDD, or the spec is already archived.
- **Still implementing** — wrong phase; this review reads a finished change.
- **A fresh, passing `review.md` already exists** for the current contract files.
- **`review: waived`** is declared with a reason.
- **Already declined** for this spec in the current thread (once-per-change debounce — do not nag).

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
- **Major**: likely bug or significant gap; fix or spec update required, and a re-review after it.
- **Minor**: clarity, maintainability, or low-risk gap; can follow up.
- **Question / assumption**: needs human or author confirmation.

For each finding, state whether the fix belongs in **code**, **tests**, **spec artifacts** (proposal/tasks/verification), or **vault** (new ADR / pattern promotion candidate).

**Where the fix lands decides when it can be applied, so say which set it is in.** `proposal.md`,
`tasks.md` and `features.json` are the **contract set**: the archive gate measures the review's
staleness against exactly those three, so editing any of them after the review invalidates it. That
is correct and not a defect — a verdict is about a state, and the review's own findings quote text
that must still be there when the spec archives. Everything else — code, tests, scripts committed
beside the spec, and `verification.md` — is outside the set and can be changed freely.

So a Blocker or a REAL Major in the contract set means **fix, then re-review**: the next round is the
mechanism, not an inconvenience. Never ask for a contract edit alongside a passing verdict; if the
change is worth blocking on, the verdict is FAIL.

### Reality classification — REAL / THEORETICAL / SPECULATIVE (HARNESS-004)

Orthogonal to severity, tag every finding by how much *evidence* says it can actually happen — so a reviewer cannot inflate the verdict with hypotheticals, and a real past-incident risk cannot be waved off as "just theoretical":

- **REAL** — a past incident, a reproduction, or a failing test demonstrates this can occur (**cite it**). The strongest class; a REAL Blocker is non-negotiable.
- **THEORETICAL** — a concrete, plausible failure path argued from the code/spec, but never observed and not yet reproduced. Legitimate, but must be labelled as such.
- **SPECULATIVE** — a hypothesis with weak evidence ("could maybe happen if…"). Allowed, but **never escalate a verdict on a SPECULATIVE finding alone** — surface it, or convert it into a "Question / assumption" to confirm.

Rule: weight each finding by `severity × reality`. A **REAL** Blocker/Major forces FAIL / PASS-WITH-GAPS; a **SPECULATIVE** finding cannot, by itself, move the verdict below PASS. State the reality tag **and its evidence** for every finding.

### Test traceability gate — named coverage or UNTESTED (HARNESS-004)

For each finding, name the test that *proves* it (or proves its fix). A finding with no named test is **UNTESTED**, and that gap is itself first-class:

- Cite the **named test** that exercises the finding's path — a `bats` case (`test/*.bats` → `@test "…"`), a pytest id, or a Go test func — **by name**, not "the test suite".
- A finding with **no named covering test → mark it `UNTESTED`**. An UNTESTED Blocker/Major is **not** resolved by the implementer's claim alone; it needs a named regression test before PASS.
- This closes the gap where `verification.md` asserts "tested" but no *named* test maps to the specific risk.

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
| **Maintainability** | Clear naming, ≤40-line fns, Cyclomatic Complexity ≤10 per function (checked with `cyclomatic-complexity`), no dead code, comments explain WHY | Acceptable structure with CC ≤15 | CC >15, confusing structure, smells, or no tests for new logic | Unreviewable; needs rewrite |
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

**Severity alone does not decide this — `severity × reality` does**, exactly as the Reality
classification above states. Read the two together; they are one rule:

- **PASS (adversarial)** — no blockers or majors; rubric all B or above; minors listed optionally.
- **PASS WITH GAPS** — minors only, OR every open Major is **THEORETICAL/SPECULATIVE**, OR rubric
  has at least one C (no D). All of them tracked, each with a disposition line.
- **FAIL** — at least one **Blocker** (any reality), at least one **REAL** Major, or rubric has at
  least one D, until addressed.

> **Why this is spelled out twice.** This list previously read *"FAIL — at least one blocker or
> major"*, which contradicts the Reality rule above (*"a SPECULATIVE finding cannot, by itself,
> move the verdict below PASS"*). A reviewer following this list instead of that rule returns FAIL
> on a Major it has itself labelled THEORETICAL — measured on BUG-093 round 4, where a
> race-window-narrowing suggestion with no demonstrated exploit cost a full merge-and-re-review
> cycle. The reality tag is not decoration; if a finding cannot be shown to happen, it is tracked,
> not blocking.
>
> A **Blocker** is deliberately exempt from this softening: "incorrect behavior, security/privacy
> issue, or spec violation" stops an archive whatever its reality tag, because the cost of being
> wrong about a Blocker is the thing the gate exists to prevent.

## Output format

**Persist the verdict — do not only print it.** Write the block below to
`specs/<feature-id>/review.md` in the reviewed repo, with the frontmatter shown. `dotf spec archive`
reads that frontmatter as its second pre-flight (CLI-034); a review that exists only in the chat
transcript does not satisfy the gate, and the archive will refuse.

```yaml
---
spec: "<feature-id>"                  # MUST equal the containing folder name — a review copied
                                      # from a sibling spec is refused as describing another change
verdict: "PASS"                       # PASS | PASS-WITH-GAPS | FAIL
reviewed_sha: "<40-hex commit sha>"   # the commit you actually examined (`git rev-parse HEAD`)
reviewer: "<agent or model id>"       # e.g. claude-opus-5, deepseek-v4-flash
date: "YYYY-MM-DD"
---
```

`reviewed_sha` is not bookkeeping: the gate rejects the review as **stale** if `proposal.md`,
`tasks.md` or `features.json` changed after it. Record the sha you read, never a later one.

The body of the file is the markdown below, unchanged.

```markdown
## Adversarial review

**Scope**: <feature-id / PR>
**Sources**: <spec paths + PR/diff reference>

### Spec and task alignment
- ...

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Blocker  | REAL    |      |         |          |                           |                                             |

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

### Recommended next steps
- ...
```

**Route every recommendation by set, and say which.** On a **FAIL**, contract-set changes are the
point and a re-review follows. On a **PASS** or **PASS WITH GAPS**, the contract set is closed: the
verdict means the gaps are tracked, not fixed, and an edit to `proposal.md`, `tasks.md` or
`features.json` would invalidate the verdict that permitted the archive. Write those recommendations
for the implementer to disposition in `verification.md` — applied, ticketed, or declined with a
reason — or to carry into a follow-up ticket. `verification.md` is excluded from the staleness check
for exactly this purpose.

Do not head the list "before archive". Measured on BUG-093 round 5 (#1516): a PASS-WITH-GAPS whose
recommendations asked for four contract edits, all of which the archive gate then refused, on a spec
that had already spent four rounds.

### Worked example (severity × reality × test coverage)

| Severity | Reality | Area | Finding | Evidence | Test | Fix location |
|---|---|---|---|---|---|---|
| Blocker | REAL | concurrency | Two back-to-back `/handoff` runs clobber the `## Session Handoff` block (last-writer-wins) | observed; HARNESS-028 | UNTESTED | code + tests (add `test/handoff-concurrency.bats`) |
| Major | THEORETICAL | path-resolution | `$VAULT_PATH` unset on a fresh machine falls back to a hardcoded literal | code read of session-start; no repro | `@test "resolves VAULT_PATH via machine.json"` | tests |
| Minor | SPECULATIVE | perf | `rglob` over a very large vault *could* be slow | none | UNTESTED | — (surface only; do not gate) |

**Verdict: FAIL** — one **REAL** Blocker (concurrency) that is **UNTESTED** → needs a named regression test before it can flip to PASS. The SPECULATIVE perf finding is surfaced but does not move the verdict.

## Guardrails

- **Do not praise** implementation to "balance" criticism unless a strength **directly mitigates a documented risk**.
- **Do not skip** reading spec artifacts when they exist. Skipping the proposal/tasks/verification triad is a process failure, not a shortcut.
- If you cannot access the PR or diff, say so and list exactly what is needed to continue. Do not fabricate findings from imagined diffs.
- Refuse to issue PASS if `[AGENT-DRAFT]` or `[AGENT-SUGGESTION]` tags remain in any spec file — that is `/spec archive`'s own gate, but flag it here too so the human catches it earlier.

## Completion

1. **Write `specs/<feature-id>/review.md`** with the frontmatter and body from "Output format". This is the deliverable; the chat summary is not.
2. Always end with: (a) the verdict, (b) whether `dotf spec archive` / `/spec archive` is **advisable** in the current state, (c) if FAIL, the minimum set of actions that would flip it to PASS.

If the change genuinely does not warrant a review, do not skip silently — the archive will refuse. Declare it in `proposal.md` frontmatter as `review: waived` with a non-empty `review_waived_reason:`, so the decision is auditable in the diff rather than invisible in a habit.
