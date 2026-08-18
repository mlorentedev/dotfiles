---
id: "TOOL-017-deterministic-review-triage"
type: spec
status: draft # draft | implementing | verifying | archived
created: "2026-08-18"
issue: "mlorentedev/dotfiles#1062"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# TOOL-017-deterministic-review-triage

## Why

<!-- from issue #1062 -->

`pr-review-triage` is a sound protocol executed entirely by judgement. Nothing triggers it, nothing verifies it ran, and its highest-value step — deciding whether a reviewer's claim is **true** — is encoded nowhere. Every PR receives whatever quality of triage the agent happened to apply that day.

The incident is #1059. PR-Agent reported a finding and classified it **REAL**: that a flagless `apt-get install age` escapes the guard's regex, because `[^\n]*` greedily consumes the space. Measured against the actual matcher, all four forms were already caught — the regex backtracks. **The stated mechanism was false.**

Testing the claim is what found the real defect: `grep` is line-based, so the multi-line continuation form escapes entirely, and `tests/Dockerfile.integration` is written in exactly that shape.

So the sequence was: right instinct, wrong diagnosis, real defect — and **only the third is visible without reproducing the claim.** A flow that auto-applied REAL findings would have swapped a working guard for a worse one *and* left the actual hole open. That is worse than no flow, because it carries the authority of having run.

## What

**The rule: reproduce the defect a finding claims. If you cannot reproduce it, it is not a finding.** The mutation discipline this repo already applies to guards, applied to reviews — a reviewer's severity is a hypothesis, and the flow's job is to test it rather than route on it.

Four pieces:

1. **A trigger.** Triage begins from a signal, not from an agent remembering. The reviewer-finished signal already exists — GUARD-003 wired `workflow_run` for exactly this, because `GITHUB_TOKEN` comments emit no events.

2. **Review-vs-notice decided by the existing registry, not re-derived.** `harness/review-attestation.json` already answers "did a review happen" via `declined_markers` and `review_markers`. Triage must consult it rather than re-implement the distinction in prose. One source, already audited.

3. **A classification seam with a fail-closed default.** `.pr_agent.toml` instructs the reviewer to classify **REAL / THEORETICAL / SPECULATIVE**; nothing consumes it. It is the natural hook — *and it is only prose*. Measured on #1059: the literal `REAL.` sits inside a `<details>` body, and the run publishes **zero artifacts**. There is no JSON, no label, no structured field.

   So the contract is **"extract from a declared marker"**, never "parse the severity". Free text drifts with model, prompt and version — the reasoning that already made `declined_markers` match HTML comments rather than translated prose. A finding whose marker is absent is **unclassified**, and unclassified needs a human. It does not decay into a skip.

4. **A closing guard.** A PR cannot close or merge carrying a reviewer comment with no recorded disposition.

### Dispositions

| class | disposition | note |
|---|---|---|
| **REAL** | reproduction required before apply | a failed reproduction is **not** a skip — the claim is wrong *and the probe may still have found something*, which is precisely #1059 |
| **THEORETICAL** | ticket, no reproduction demanded | |
| **SPECULATIVE** | one line, closed | |
| **unclassified** | escalate to a human | never automatic |

The REAL row is the whole spec. Every other row is bookkeeping.

## Out of scope

- **Applying fixes autonomously.** This produces a *verified, dispositioned* triage; a human still decides what lands. `pr-review-triage` already forbids merging, and nothing here relaxes that.
- **Changing what PR-Agent reviews.** `.pr_agent.toml`'s `extra_instructions` are TOOL-013's surface.
- **Making the classification structured upstream.** Emitting a machine-readable severity is a change to PR-Agent itself; this spec consumes what exists and fails closed when it is absent.
- **Retrofitting past PRs.** The guard binds from adoption forward.
- **Editing `pr-review-triage`'s SKILL.md directly.** It renders into `harness/skills/` from a vault source; a protocol change edits the vault, and another session is currently in the harness. Coordinated, not taken.

## Risks / open questions

- **A reproduction harness is itself a check, and can be vacuous.** Tonight produced both halves of that: a mutation that never reached the code, and (in a parallel session) a harness that compared against `HEAD` on a dirty tree and so reported an invalid control as applied. **An invalid reproduction and a real refutation produce the same result.** Mitigation: the harness must verify its own mutation landed — checksum before and after — and report "did not land" as a distinct state from "did not reproduce".
- **What counts as a reproduction is not uniform.** A regex claim reproduces by running the regex; a race condition does not reproduce on demand. Proposal: reproduction is required where the claim is *mechanically checkable*, and a claim that is not gets treated as THEORETICAL regardless of its stated class. This needs a boundary that is written down rather than judged per case.
- **The classification marker does not exist yet.** Consuming prose is what this spec forbids, so either the reviewer is asked to emit a stable marker (a `.pr_agent.toml` change, TOOL-013's surface) or every finding is unclassified and every PR escalates — which is fail-closed but useless. **This is the open decision, and it gates the value of the whole spec.**
- **Escalation could become the normal path**, the same way `declined` became normal for GUARD-003's check-run. If most findings arrive unclassified, the guard degrades into noise. Needs a measured adoption window before the closing guard is made blocking.

## Acceptance criteria

- [ ] Triage begins from the reviewer-finished signal, not from an agent electing to look.
- [ ] Review-vs-notice is decided by `harness/review-attestation.json`, not re-derived — a change to the registry changes triage behaviour with no code edit.
- [ ] A finding's class is read from a **declared marker**; a finding without one is `unclassified` and escalates to a human. Parsing free prose fails the criterion even if it works.
- [ ] A **REAL** finding is not applied until its claimed defect has been reproduced.
- [ ] A reproduction that does not land is reported as **did-not-land**, distinct from **did-not-reproduce**.
- [ ] A refuted REAL finding is recorded as refuted with its evidence — not silently skipped, because the probe may still have found something real (#1059).
- [ ] Every reviewer comment carries exactly one recorded disposition, and a PR cannot close or merge with one missing.
- [ ] Every new check has a red-direction test that fails when the thing it guards is broken.

## References

- Bitácora: `mlorentedev/dotfiles#1062`
- The incident: #1059 — REAL classification, false mechanism, real defect found only by testing the claim
- `harness/review-attestation.json` — the reviewer registry this must consume rather than duplicate
- GUARD-003 (#1052) — the reviewer-finished signal, and why `GITHUB_TOKEN` comments emit no events
- GUARD-002 (#1019) / #906 — a green check that did not mean reviewed; this is the same failure one layer up, where triage was claimed and nothing was
- `pattern-verification-fails-toward-unproven` — the class; a reviewer's stated severity is a proxy for whether a defect exists
- `pattern-track-or-fix` — the two-exits rule this makes mechanical
