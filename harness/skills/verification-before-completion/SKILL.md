---
generated: true
generated_from: 00_meta/skills/verification-before-completion/SKILL.md
generated_sha: da15afc3f14995da
id: verification-before-completion-skill
type: skill
status: active
created: "2026-05-30"
owner: manu
name: verification-before-completion
description: Use when about to claim work is complete, fixed, or passing -- before committing, creating PRs, or closing tasks. Evidence before assertions.
---

# Verification Before Completion

## Overview

Claiming work is complete without verification is dishonesty, not efficiency.

**Core principle:** Evidence before claims, always.

**Violating the letter of this rule is violating the spirit of this rule.**

## The Iron Law

```
NO COMPLETION CLAIMS WITHOUT FRESH VERIFICATION EVIDENCE
```

If you haven't run the verification command in this message, you cannot claim it passes.

## The Gate Function

```
BEFORE claiming any status or expressing satisfaction:

1. IDENTIFY: What command proves this claim?
2. RUN: Execute the FULL command (fresh, complete)
3. READ: Full output, check exit code, count failures
4. VERIFY: Does output confirm the claim?
   - If NO: State actual status with evidence
   - If YES: State claim WITH evidence
5. ONLY THEN: Make the claim

Skip any step = lying, not verifying
```

## Common Failures

| Claim | Requires | Not Sufficient |
|-------|----------|----------------|
| Tests pass | Test command output: 0 failures | Previous run, "should pass" |
| Linter clean | Linter output: 0 errors | Partial check, extrapolation |
| Build succeeds | Build command: exit 0 | Linter passing, logs look good |
| Bug fixed | Test original symptom: passes | Code changed, assumed fixed |
| Regression test works | Red-green cycle verified | Test passes once |
| Agent completed | VCS diff shows changes | Agent reports "success" |
| Requirements met | Line-by-line checklist | Tests passing |

## Red Flags - STOP

- Using "should", "probably", "seems to"
- Expressing satisfaction before verification ("Great!", "Perfect!", "Done!", etc.)
- About to commit/push/PR without verification
- Trusting agent success reports
- Relying on partial verification
- Thinking "just this once"
- Tired and wanting work over
- **ANY wording implying success without having run verification**

## Rationalization Prevention

| Excuse | Reality |
|--------|---------|
| "Should work now" | RUN the verification |
| "I'm confident" | Confidence ≠ evidence |
| "Just this once" | No exceptions |
| "Linter passed" | Linter ≠ compiler |
| "Agent said success" | Verify independently |
| "I'm tired" | Exhaustion ≠ excuse |
| "Partial check is enough" | Partial proves nothing |
| "Different words so rule doesn't apply" | Spirit over letter |

## Key Patterns

**Tests:**
```
✅ [Run test command] [See: 34/34 pass] "All tests pass"
❌ "Should pass now" / "Looks correct"
```

**Regression tests (TDD Red-Green):**
```
✅ Write → Run (pass) → Revert fix → Run (MUST FAIL) → Restore → Run (pass)
❌ "I've written a regression test" (without red-green verification)
```

**Build:**
```
✅ [Run build] [See: exit 0] "Build passes"
❌ "Linter passed" (linter doesn't check compilation)
```

**Requirements:**
```
✅ Re-read plan → Create checklist → Verify each → Report gaps or completion
❌ "Tests pass, phase complete"
```

**Agent delegation:**
```
✅ Agent reports success → Check VCS diff → Verify changes → Report actual state
❌ Trust agent report
```

## Why This Matters

From 24 failure memories:
- your human partner said "I don't believe you" - trust broken
- Undefined functions shipped - would crash
- Missing requirements shipped - incomplete features
- Time wasted on false completion → redirect → rework
- Violates: "Honesty is a core value. If you lie, you'll be replaced."

## When To Apply

**ALWAYS before:**
- ANY variation of success/completion claims
- ANY expression of satisfaction
- ANY positive statement about work state
- Committing, PR creation, task completion
- Moving to next task
- Delegating to agents

**Rule applies to:**
- Exact phrases
- Paraphrases and synonyms
- Implications of success
- ANY communication suggesting completion/correctness

## The Closing Pass

The Iron Law above covers **evidence**. Evidence is one of five checks a finished change owes, and it is the only one this skill used to enforce — which is why the other four kept being skipped while verification passed. Before claiming done, walk the Definition of Done (the harness injects it verbatim into your instructions file; it is authored in `pattern-change-lifecycle.md`) and produce a verdict per item, not a feeling:

| Check | What to actually do | Unmet means |
|---|---|---|
| **Debt** | List every defect you noticed and did not fix. | Fix it in scope, or file a ticket with root cause. A mention in chat is not an exit. |
| **Knowledge** | Name what you learned that is not obvious from the diff. | Write it where it belongs — repo `docs/` for build/operate, the store for cross-project — in this session. |
| **Board** | Read the ticket's real status. | Move it. Picked up when you start, blocked when blocked, closed by the change that closed it. |
| **Review** | Check whether an open PR has checks or comments waiting. | Triage them. Each comment is applied, ticketed, or declined with a reason. |
| **Evidence** | The Iron Law above. | Run the command. |

Three rules keep this from becoming theatre:

1. **Report the verdict, not the intention.** "Filed as #123" and "no debt found" are both verdicts. "I should ticket that" is not.
2. **A skip is a stated decision.** Any of the five may be skipped when it does not apply — say which and why. Silence is not a skip, it is the failure mode.
3. **Do not paraphrase the standing orders.** This checklist binds them to a moment; it is not a second source of truth. When they disagree, the standing order wins and this table is wrong.

## The Bottom Line

**No shortcuts for verification.**

Run the command. Read the output. THEN claim the result.

This is non-negotiable.
