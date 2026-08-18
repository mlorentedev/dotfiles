---
id: lesson-077-a-held-spec-can-be-obsoleted-by-a-later-adr-reconc
type: lesson
status: active
created: "2026-06-01"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 077: A held spec can be obsoleted by a later ADR — reconcile+close, don't implement-as-written

**Context:** Picked up IDEAS-007 (#103, filed 2026-05-27) to implement a 4-layer cross-provider agent harness (.agent/<id>/INSTRUCT.md design). Ran verify-before-act against git/specs before coding.
**Problem:** The spec's architecture had already shipped by other means AFTER it was written: ADR-009 (AGENTS.md SSOT) + ADR-010 (parity) + the ai/<provider>/ overlay structure realised Layers 1-2; the L3 registry + runtime discovery mechanism had zero consumer. Implementing the spec literally would have manufactured debt — a binary-name->provider detector nothing calls, plus a churning rename of a deployed convention wired into setup-linux.sh (~6 sites) + healthcheck.
**Solution:** Reconcile, don't re-implement. Produce evidence (audit.json: rule-by-rule classification confirming the split is already correct; reconciliation.md: criterion-by-criterion disposition) and close the issue with that evidence. Reject the no-consumer layers explicitly as YAGNI (Decision Hierarchy: Explicit > Implicit). Spin off the one genuine win found (data-driven setup-linux.sh provider-deploy manifest) as its own deferred ticket to keep the PR atomic. This is the IMPLEMENT-side mirror of feedback_check_existing_artifacts_first (which is propose-side) and another instance of pattern-verify-against-source-of-truth: verify the spec's PREMISE is still valid before writing code, because a later ADR can silently obsolete an open spec.
**Tags:** `#sdd` `#verify-before-act` `#yagni` `#reconciliation`
