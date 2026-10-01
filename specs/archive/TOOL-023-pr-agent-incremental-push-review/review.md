---
spec: "TOOL-023-pr-agent-incremental-push-review"
verdict: "PASS WITH GAPS"
reviewed_sha: "a157cb52c01995bce461d381a80b5414cb64bc8e"
reviewer: "agy/gemini-3.1-pro-high"
date: "2026-09-30"
---
## Adversarial review

**Scope**: TOOL-023-pr-agent-incremental-push-review
**Sources**: \`specs/TOOL-023-pr-agent-incremental-push-review/{proposal,tasks,verification}.md\`, git diff \`4dad5eadd8857c084499e2474a67b4ac87024ea7...HEAD\`

### Spec and task alignment
- All acceptance criteria are verified and mapped to evidence.
- The gate successfully filters by threshold and correctly distinguishes between full and incremental modes.
- Failsafe mechanisms are well-implemented; unexpected errors default to running a full review.
- The rebase blind spot and CWE-345 author filtering fixes are applied.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Major | THEORETICAL | security / gate logic | An attacker can bypass future incremental reviews by injecting a fake \`pr-agent-review-state:v1\` block with a recent \`head_sha\` into a PR comment or description. If an incremental review quotes this payload, the script \`split("pr-agent-review-state:v1") \| last\` picks it up (since incremental reviews don't append a genuine block), reads the attacker's \`head_sha\`, and counts only commits after it, skipping review for unreviewed code. | Code inspection: The state block extraction does not validate that the block originated from the bot's own footer rather than quoted text in the middle of the comment. | UNTESTED | code + tests |
| Minor | SPECULATIVE | gate robustness | The gate relies on \`last_run.head_sha\` being present only when intended. If upstream PR-Agent changes its behavior and includes this field in incremental reviews, the logic will shift back to position-based counting automatically, which might diverge from upstream's own date-counting logic. | Code inspection: The script falls back to date-counting only when \`head_sha\` is absent. | UNTESTED | — (surface only) |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | B | Criteria met on happy path, but a state-block spoofing vector remains unhandled. |
| Verification       | A | Evidence proves each criterion with reproducible commands and robust offline bats tests. |
| Scope              | A | Diff matches proposal exactly; follow-up issues directly address the triage. |
| Reliability        | B | Excellent fail-open logic on parse errors, but partial vulnerability to quoted payloads. |
| Maintainability    | A | Clear structure, \`jq\` scripts are decoupled and readable, thorough test coverage. |
| Handoff-readiness  | A | Spec updates and follow-ups are comprehensively documented. |

### Verdict
PASS WITH GAPS

### Recommended next steps
- **code + tests**: Harden the state block extraction in \`scripts/pr-agent-push-gate.sh\`. Ensure the \`pr-agent-review-state:v1\` block is only matched if it appears at the very end of the comment (e.g., anchoring the regex or checking the offset), avoiding payloads quoted inside markdown blocks. Add a regression test where a fake state block is injected into an incremental review.
