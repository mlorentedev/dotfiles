---
id: "GUARD-002-review-attestation"
type: spec
status: draft # draft | implementing | verifying | archived
created: "2026-08-16"
issue: "mlorentedev/dotfiles#906"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# GUARD-002-review-attestation

## Why

<!-- from issue #906: CI: CodeRabbit's free-tier quota silently skips review on some PRs, and the merge gate cannot tell -->

A green check wall currently cannot distinguish "reviewed and clean" from "nobody
reviewed this". Measured on 2026-08-16: PRs #1007, #1009 and #1013 each carried a
CodeRabbit comment saying *"Review limit reached — we couldn't start this review"*,
zero entries in `reviews[]`, and `gh pr checks` reporting **`CodeRabbit  pass`**.
Three PRs from three parallel sessions, none reviewed, all green. The `pr-stewardship`
doctrine already forbids proceeding *silently* on an unreviewed PR — but it binds an
agent's judgement, and nothing mechanical tells an agent, or a human, which case they
are in. The one signal that exists (`pass`) actively asserts the wrong one.

## What

A gate that makes a green check mean *reviewed*, by refusing to settle green until a
review is attested or its absence is declared.

After this change, every PR resolves into exactly one of three states, decided by
**content** rather than by check status:

| State | Meaning | Gate |
|---|---|---|
| `attested` | a recognized reviewer produced an actual review | pass |
| `declined` | a reviewer said it could not review (quota, outage, error) | **fail**, naming which reviewer and why |
| `pending` | no reviewer output yet | **fail**, "not reviewed yet" |

plus a declared escape: `disclosed` — the `merged-unreviewed` label **and** a non-empty
`## Unreviewed merge rationale` section in the PR body. Both, never one. That is the
same shape as `spec-gate`'s `skip-sdd` escape, chosen because it is already understood
here and because it forces the disclosure into the durable record instead of a chat
message.

The gate is **reviewer-agnostic**. Reviewers are declared in a config file — login,
plus the marker that identifies its "I could not review" notice. CodeRabbit is the
first entry; PR-Agent (#786) becomes a second entry rather than a rewrite. This is the
point of building it before #786 rather than after: a substitute reviewer inherits the
blind spot otherwise, because #786's own acceptance criterion is *"a provider failure
degrading the review rather than blocking the PR"* — which is this same silent green,
on a new tool.

A human review counts as an attestation. The question is whether a review happened, not
whether a bot performed it.

## Out of scope

- **Auto-retriggering a rate-limited reviewer** (`@coderabbitai review`). Which PR
  consumes the single hourly slot is a policy decision, and a gate that spends the
  quota by itself takes it from whichever PR a human would have chosen. v1 reports;
  the human triggers.
- **Making the check `required` in branch protection.** Landing it as required in the
  same change would block every open PR the moment it merges. Adopt first, observe,
  promote separately.
- **Judging review quality.** The gate asks whether a review happened, never whether it
  was good. A shallow review is a different problem with a different fix.
- **Replacing CodeRabbit.** That is #786; this ticket makes #786 safe to adopt.
- **Any change to `dotf spec review` / `harness/reviewer-pool.json`.** That is the
  *spec-level* adversarial reviewer and a separate gate with a separate purpose. See
  the note under Risks.

## Risks / open questions

- **Noise on freshly-opened PRs.** `pending` is red, so every PR is red for the minutes
  between opening and the reviewer landing. Mitigated by the check being non-required
  and by re-running on `issue_comment` so it settles by itself. Accepted deliberately:
  the alternative — treating "no output yet" as pass — is precisely the bug.
- **Fail-closed on API failure.** If `gh` fails, the PR is unauthenticated, or the
  reviewer's output cannot be read, the gate reports **not attested**, never pass. This
  follows `pattern-verification-fails-toward-unproven`, and it is the opposite of what
  `spec-gate-prepush.sh` does locally — a difference that is deliberate: that adapter
  falls through because "no PR yet" is an ordinary local state, whereas here an
  unreadable reviewer is exactly the condition being guarded.
- **Marker drift.** Classification leans on a vendor-emitted HTML marker
  (`<!-- ... rate limited by coderabbit.ai -->`). If CodeRabbit changes it, an
  unreviewed PR would be read as `pending` rather than `declined` — still red, so the
  drift degrades the *message*, not the verdict. Chosen over prose matching, which is
  more fragile and language-dependent.
- **Two reviewer policies could diverge.** `harness/reviewer-pool.json` already encodes
  who may sign a *spec* review, including the standing rule that no adversarial review
  runs on an Anthropic model and that latency-optimised models are excluded because
  "a reviewer that PASSes cheaply is worse than no gate". #786 currently proposes
  `fallback_models = ["qwen3.6"]`, which that file deliberately excludes. Whether the
  two policies unify is **out of scope here and flagged on #786**; this spec only
  requires that its own reviewer list is a declared file, not a hardcoded login.
- **Open question — does `pending` deserve its own exit code?** Distinguishing "not yet"
  from "declined" in the exit status would let a caller wait rather than fail. Deferred:
  the message names which state it is, and no caller needs to branch on it in v1.

- **This gate must not become an instance of the class it belongs to.** A parallel
  session named the shape while debugging #988, and it is sharper than "the check was
  wrong": **the check was the cause.** Three instances observed in this repo within one
  day —
  1. `bw serve`'s `GET /status` probe *poisoned the item reads it was verifying*: one
     status call made 10/10 subsequent item reads return HTTP 500, and `dotf secrets run`
     issued one before each of 33 secrets, so every `dotf spec review` launch poisoned
     itself (#988, fixed in #1018);
  2. a rate-limit notice renders as a green `pass` — this spec;
  3. a DR check whose only reachable branch was the no-op one, so an escrow that had
     never existed reported as SKIP (#997).

  One shape: **a signal that reports on a system it is simultaneously perturbing or
  misrepresenting.** GUARD-002 cannot fix that class — it is a property of checks in
  general, not of review checks. What it *must* do is not join it, which constrains this
  design in three concrete ways, each already reflected above:
  - it **observes** rather than acts, so it cannot perturb what it measures — this is the
    real reason auto-retriggering `@coderabbitai review` is out of scope, beyond the
    quota-policy argument. (Empirically moot as well: a parallel session confirmed the
    explicit command does *not* reclaim a slot while the quota is spent, so an
    auto-retrigger would spend attempts and change nothing.)
  - every classifier branch must be **observed failing** before it is made to pass, so no
    state is decided by a branch that cannot be reached (instance 3's defect);
  - it must **fail closed** (AC6), so "could not determine" never renders as "fine" —
    which is instance 2's defect, and the one this spec exists for.

## Acceptance criteria

- [ ] **AC1** A classifier decides `attested | declined | pending` from a PR's reviews
      and comments, **offline**, against committed JSON fixtures — including a fixture
      captured verbatim from the real 2026-08-16 CodeRabbit rate-limit comment.
- [ ] **AC2** A recognized reviewer's real review yields `attested` and exit 0. A human
      review yields `attested` too.
- [ ] **AC3** A reviewer notice that no review ran yields `declined` and a non-zero
      exit, and the message names the reviewer and the reason. A PR with no reviewer
      output yields `pending` and a non-zero exit.
- [ ] **AC4** The escape requires **both** the `merged-unreviewed` label and a non-empty
      `## Unreviewed merge rationale` section. Label alone fails; section alone fails;
      an empty section fails.
- [ ] **AC5** Adding a reviewer is a config edit, not a code edit: a second reviewer
      declared in the config is recognized by the classifier with no change to the
      script, proven by a fixture that names a reviewer other than CodeRabbit.
- [ ] **AC6** Any failure to read the PR's state (no `gh`, unauthenticated, malformed
      response) results in a non-zero exit, never a pass. Proven by a test that feeds
      the classifier unreadable input.
- [ ] **AC7** The gate runs in CI on `pull_request` and `issue_comment`, so a review
      arriving after the checks settled re-runs it without a push.

## References

- Bitácora board: `mlorentedev/dotfiles#906` (see the `issue:` frontmatter field)
- `#786` (TOOL-013) — the PR-Agent replacement reviewer this gate is designed to outlive
- `#1009`, `#1007`, `#1013` — the three simultaneously-unreviewed PRs that produced the evidence
- `scripts/check-spec-gate.sh` — the label + body-section escape shape reused here
- `harness/enforced/pr-stewardship.md` — the doctrine this gate mechanises
- Related patterns: `00_meta/patterns/pattern-verification-fails-toward-unproven.md`
