<!-- DRAFT of the vault section for `pattern-change-lifecycle.md`, NOT yet applied.
     Held here instead of half-written into the shared vault tree, because two
     objections from the kubelab session are unresolved — see proposal.md. -->

## PR Stewardship

> Injected verbatim into every agent's instructions (harness `enforced` id `pr-stewardship`). It elaborates Definition of Done §4 — "an open PR is not finished work" — into the two obligations that item leaves implicit: stay, and get an independent read.

**A PR you open is watched, not abandoned.** Opening it is not the end of the change: checks report afterwards, and reviewers report after *that*. The window closes at the first of an actionable reviewer comment, or ten minutes after the checks settle. Pushing a fix reopens it, because the reviewer re-reviews. Leave with the window open and you have handed the next session a change nobody read.

**CI green is not the end of the window, and a comment is not a review.** Both halves have been observed failing here. Checks went green on one PR and the reviewer then posted four Major findings; on another, checks went green and the reviewer posted *"review limit reached — we couldn't start this review"*. That second one is a comment arriving, and it means **nobody looked**. A notice that no review ran leaves the PR unreviewed, and closing the window on it requires saying so out loud — "merged unreviewed, reviewer quota exhausted" is a disclosure; silence is a claim of review that never happened.

**A PR that touches a `specs/<id>/` folder gets an independent adversarial review by default.** That is the whole test — no judgement call about whether a change is "important" enough. It names an obligation that already binds mechanically, so the only question is whether you meet it deliberately or discover it as a refusal: the spec gate declines to merge a PR closing a spec's issue without archiving it, `spec archive` declines without a passing review, and the reviewer pool declines one signed by the wrong model. The reviewer must not be the implementer; that is what makes it independent, and it is the entire value.
