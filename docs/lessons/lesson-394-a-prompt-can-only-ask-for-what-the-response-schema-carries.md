---
id: "lesson-394-a-prompt-can-only-ask-for-what-the-response-schema-carries"
type: lesson
status: active
title: "A prompt can only ask for what the response schema carries"
created: "2026-10-11"
---

# A prompt can only ask for what the response schema carries

## Context
PR-Agent's review is a YAML document under a fixed schema: a `review:` mapping with declared
fields (`key_issues_to_review`, `security_concerns`, `ticket_compliance_check` when a ticket is
linked, and so on). `.pr_agent.toml`'s `extra_instructions` told the reviewer to "open every
review" with a HARNESS COMPLIANCE pass and to report it "even when everything passes", because a
silent section cannot be told apart from a skipped one (#786).

## The Trap
No field could hold that section, so each model improvised. Across seven reviews:

- five dropped the pass, so the silence it was meant to rule out was the usual result;
- two filed it as a pseudo-ticket in `ticket_compliance_check`, a field that exists only when
  the PR links a ticket;
- one wrote it as YAML comments above the mapping and then emitted the review's keys without the
  `review:` root. PR-Agent rejected the response ("did not contain a non-empty review mapping"),
  and because that was the pool's last attempt, the PR got no review at all
  (run 38047143517, #2287).

The instruction read as rigour. Nothing checked that the output had a place for it.

## The Solution
- A compliance FAIL is reported like any other finding: an entry of `key_issues_to_review` with
  `issue_header: "Harness compliance"` and the offending file and lines. An artifact that is not a
  file (branch, PR title, commit message) is named as `relevant_file`, with lines 0, which
  PR-Agent keeps in the summary instead of anchoring it to a random line.
- Passes are not reported. That loses nothing: in practice five of seven reviews never reported
  them anyway.
- `num_max_findings` rises from 3 to 5, so a compliance FAIL does not push a defect out.
- `tests/pr-agent-config.bats` fails if the instructions ask the review to open with, or carry, a
  section, or ask for passes to be reported.

## Takeaways
- Before telling a model what to write, find the field it will write it in. If the schema has
  none, the instruction asks for an improvisation, and you will get several.
- A rare format failure matters most on the last attempt. There it costs the whole result, not a
  retry.
- "Report it even when it passes" only makes the absence visible if the output has a slot for
  the report.
