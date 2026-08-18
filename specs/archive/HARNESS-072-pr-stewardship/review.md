---
spec: "HARNESS-072-pr-stewardship"
verdict: "PASS WITH GAPS"
reviewed_sha: "a9b2063d7440c152a1997c20f5d78ee4b5261998"
reviewer: "nan/deepseek-v4-flash"
date: "2026-08-15"
---

## Adversarial review

**Scope**: HARNESS-072-pr-stewardship
**Sources**: `specs/HARNESS-072-pr-stewardship/{proposal,tasks,verification}.md`, `harness/manifest.json`, `harness/enforced/pr-stewardship.md`, `AGENTS.md`, `ai/claude/CLAUDE.md`, `scripts/compile-harness.sh`, `harness/skills/pr-review-triage/SKILL.md`, `tests/compile-harness.bats`

### Spec and task alignment

- All eight acceptance criteria have corresponding features (f1–f8) and implemented tasks.
- Every task is ticked `[x]`. The diff evidence is consistent with the task claims.
- No `[AGENT-DRAFT]` or `[AGENT-SUGGESTION]` tags remain in any spec file.
- The spec's Risks section names the coverage-class bug (the partial-injection case), and the implementation ships the fix (`check_coverage` + AC8) in the same PR — this satisfies `feedback_incident_to_guard`.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Minor | REAL | Verification gap | `features.json` f2 verifies AC2 ("positive per-surface grep") by checking only the **committed** targets (AGENTS.md, CLAUDE.md) and the manifest doctrine.inject list. It does **not** grep the rendered doctrine payloads at `~/.gemini/GEMINI.md` and `~/.codex/AGENTS.md`, which AC2 specifically requires. The current state is correct (manually confirmed in-session), but no automated gate prevents a regression if `--deploy` silently fails or a future change drops pr-stewardship from the doctrine payload. | UNTESTED (no automated check on the rendered doctrine files) | spec: update features.json f2 to also grep the doctrine payloads after deploy, or document this as an accepted design limit of the offline-check model |
| Minor | THEORETICAL | Coherence | The PR Stewardship region and the existing "Hand the PR over; don't watch CI" rule (AGENTS.md lines 263 vs 273 — 10 lines apart) express different timing expectations for the same transition. The region says "the default mechanism is to stay" and sets a 10-minute window; the existing rule says "move to the next piece of work — never sit in a watch loop." The region does provide an escape ("a project that already tells you when to look back — the human notifies, a hook fires — has met this, and its instruction wins"), and the existing rule qualifies here. But an agent reading both without careful synthesis could conclude they contradict, especially since the phrase "default mechanism is to **stay**" directly opposes "**move** to the next piece of work." | UNTESTED (no behavioural test exists for agent instruction comprehension) | spec: add a bridging sentence in the region connecting the two rules ("This does not contradict 'Hand the PR over; don't watch CI' because…") or relocate the "Hand the PR over" rule into the region itself so the reader sees them as one unit |
| Minor | THEORETICAL | Timing ambiguity | The region uses "ten minutes after the checks settle" as its default window. The `pr-review-triage` skill (which this spec amends) says "wait a couple of minutes" / "two minutes" before reading comments. These describe different timers for different phases (window-close vs first-read), but nothing in either document explains how they compose: the agent reads comments at t+2 min, then has until t+10 min to produce a disposition. A reader could conflate them or wait the wrong amount of time. | UNTESTED | spec: add a note in either document clarifying that "two minutes" (first look) is a sub-phase of the "ten minutes" (window close) |
| Minor | SPECULATIVE | Default trigger | The `pr-review-triage` skill says it triggers "by default once a PR you opened has come back." But no automated mechanism calls it — the trigger relies on agent self-invocation. The enforced region is the binding mechanism; the "by default" trigger language is aspirational and could cause an agent to expect an automatic call that never fires. Not a blocker — the region's obligation is the fix — but it creates a false expectation. | UNTESTED | spec: clarify in the skill's preamble that "by default" means "when the agent judges it is time," not "automatically" |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness | B | All acceptance criteria verified and passing; one verified-but-unguarded gap in automated coverage of doctrine payloads (Minor/REAL). |
| Verification | B | Evidence in `verification.md` is thorough and reproducible. The automated `features.json` gate has a gap (f2 doesn't check rendered doctrine), keeping this at B rather than A. |
| Scope | A | Diff matches the proposal exactly. The GUARD-002-review-attestation work is a separate commit on the branch, not merged into this spec's diff. The six unrelated drifted records are a committed sync, declared in tasks.md. |
| Reliability | B | The coverage guard (AC8) and region diff work correctly. Mutation tests confirmed both `check_coverage` and region-diff catch the cases they're designed for. No error-path gaps found in compile-harness.sh changes. |
| Maintainability | A | Code is clean, functions short, tests have readable names with descriptive comments explaining *why* they exist. The `check_coverage` function has a prose explanation of why region-diff alone is blind to its gap. |
| Handoff-readiness | A | Spec artifacts are complete. Verification.md records decisions, promotion candidates, and evidence. No stale tags remain. Only the findings above would benefit from resolution before archive. |

### Verdict
PASS WITH GAPS

### Recommended next steps (before archive)

1. **Address the verification gap (Minor/REAL):** Update `features.json` f2 to also grep the rendered doctrine payloads at `~/.gemini/GEMINI.md` and `~/.codex/AGENTS.md` for "the disposition, not the waiting." Alternatively, document this as an accepted limit of ADR-013's offline-check model (committed-only verification) and close the gap in a follow-up.
2. **Resolve the coherence tension (Minor/THEORETICAL):** Either add a bridging sentence in the PR Stewardship region or relocate the "Hand the PR over; don't watch CI" rule into the region itself, so an agent reading both sees them as one consistent instruction rather than two contradicting ones.
3. **Clarify the two timers (Minor/THEORETICAL):** Add a short note in either the region or the skill stating that the "two minutes" (first comment read) is a sub-phase of the "ten minutes" (window close), so the two don't appear to conflict.

None of these are blockers. `dotf spec archive` is **advisable** once item 1 (the automated verification gap) has a resolution path recorded — either as a fix to `features.json` or as a documented acceptance in the spec artifacts. The change is correct and the guard works; the gaps are in documentation completeness and automated verification breadth, not in the implementation logic.