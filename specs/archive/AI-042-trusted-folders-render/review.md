---
spec: "AI-042-trusted-folders-render"
verdict: "PASS"
reviewed_sha: "ca90ca85ab910e1ef377e9ce451fdc5634f0228b"
reviewer: "agy/gemini-3.1-pro-high"
date: "2026-09-25"
---
## Adversarial review

**Scope**: AI-042-trusted-folders-render
**Sources**: `specs/AI-042-trusted-folders-render/{proposal,tasks,verification}.md`, local git diff and commit log.

### Spec and task alignment
- `paths` token expansion securely targets strings beginning with `{HOME}` and other recognized tokens, properly normalizing directory separators using `native` and `slash` formats without mangling paths embedded in standard text or URLs (AC1).
- `mergeInto` safely avoids overwriting dynamically managed user arrays (such as runtime trust entries). It does this by deep-merging inner map structures and exact-match unioning arrays via `jsonEqual` (AC2).
- The `encoding/json` decoder has been meticulously customized (`decodeJSONNumbers`) to parse numeric data as `json.Number`, ensuring `jsonEqual` accurately evaluates precision without truncating integers over 2^53 (AC2).
- `ai/copilot/config.json` and `ai/agy/settings.json` have replaced hardcoded absolute targets with templates. Verification suite strictly enforces the absence of absolute literal user profiles (AC3).
- Windows paths, including edge-case verification steps, behave cleanly inside arrays under both `dotf deploy` replacements and merges (AC4).

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| -        | -       | -    | *(No blocking or major findings discovered. Prior review feedback fully implemented.)* | -        | -                         | -                                           |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | A | Token expansion isolates replacements accurately, handling diverse JSON primitives gracefully. |
| Verification       | A | Comprehensive bats and Go test suite successfully exercises boundaries, permutations, and regressions. |
| Scope              | A | Diff perfectly aligns with the `proposal.md` bounds; no evidence of scope creep. |
| Reliability        | A | Secure array union merging. `jsonEqual` using `big.Rat` ensures idempotency is rock-solid across deploys. |
| Maintainability    | A | Well-architected separation of JSON object decoding and path expansion rules keeps the deployment logic crisp. |
| Handoff-readiness  | A | Spec is completely documented, tests are merged, and it is ready to be archived. |

### Verdict
PASS

### Recommended next steps
- Proceed with `dotf spec archive AI-042-trusted-folders-render`.
