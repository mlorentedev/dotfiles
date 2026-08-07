---
tags: [spec, verification]
created: "2026-08-06"
---

# Verification - TOOL-013-pr-agent-nan-review

> **Status: draft — planned criteria only.** No evidence is recorded below because implementation
> has not started. Each line is a checkable command to run at implementation time, not a claim.

## Evidence (planned)

- [ ] AC1 — reusable workflow exists and is called
      -> `gh api repos/mlorentedev/dotfiles/contents/.github/workflows/pr-review.yml --jq .name`
      and at least one caller references it (`grep -r "dotfiles/.github/workflows/pr-review.yml@"`)
- [ ] AC2 — inline review comments produced via NaN
      -> `gh api repos/mlorentedev/biwenger-mvp/pulls/<N>/comments --jq 'length'` is non-zero on the
      pilot PR, and the run log shows the NaN base URL was the endpoint used
- [ ] AC3 — ignore globs actually filter
      -> a PR touching an excluded path; PR-Agent verbose log lists the file as ignored and it does
      not appear in the model payload
- [ ] AC4 — provider failure degrades, never blocks
      -> re-run with an invalid key; the review step is red or skipped while
      `gh pr view <N> --json mergeable` still reports mergeable
- [ ] AC5 — secret present everywhere, never printed
      -> `gh secret list -R mlorentedev/<repo>` shows `NAN_API_KEY` for each participating repo;
      `gh run view <id> --log | grep -c "$NAN_API_KEY"` returns 0
- [ ] AC6 — no public repo runs this on the self-hosted runner
      -> `grep -L "self-hosted"` across every public-repo caller returns all of them
- [ ] Fork-PR behaviour -> a fork PR run completes as a no-op, with no failure annotation
- [ ] `actionlint` clean on the reusable workflow and every caller

## Test status

Not run — draft.

## Decisions made during implementation

To be filled in during implementation. Decisions already fixed at design time live in
`proposal.md`; the ones expected to need recording here:

- The exact LiteLLM model string that resolved against the NaN endpoint
- Final `custom_model_max_tokens` / `max_model_tokens` values and how they were derived
- The outcome of the `resume` PII open question
- The runner choice actually taken for private repos, and why
- The evaluation-window verdict: which reviewer was retired, on what evidence
