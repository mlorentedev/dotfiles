---
id: "TOOL-013-pr-agent-reviewer"
type: spec
status: implementing # draft | implementing | verifying | archived
created: "2026-08-16"
issue: "mlorentedev/dotfiles#786"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# TOOL-013-pr-agent-reviewer

## Why

<!-- from issue #786: PR-Agent on NaN inference as a uniform reusable review workflow -->

Review capacity, not review quality, is the binding constraint on throughput here.
CodeRabbit's free tier allows roughly **one review per hour, account-wide**. Measured
2026-08-16: three PRs opened by three parallel sessions each received *"Review limit
reached — we couldn't start this review"* instead of a review, `gh pr checks` reported
`CodeRabbit  pass` for all three, and two were merged unreviewed. Neither pushing new
commits nor an explicit `@coderabbitai review` reclaims a slot while the quota is spent.

GUARD-002 (#1019) made that state *visible*. It cannot make it stop: a gate that reports
"nobody reviewed this" is only useful when someone could have. A reviewer with no
per-hour account quota removes the queue instead of scheduling around it.

## What

An inline PR reviewer running on NaN cluster inference, on **this repo only**, alongside
CodeRabbit rather than replacing it.

After this change a pull request receives inline review comments generated through NaN,
path exclusions demonstrably keep excluded files out of the model call, and a provider
failure degrades the review into a **visible** absence rather than a silent pass —
because GUARD-002 classifies a missing review as `declined` and turns the PR red.

## How this spec was produced

Disclosed rather than left for a reviewer to infer from commit timestamps: the
config and workflow were **drafted before this spec existed**. The spec-gate
refused the push at 165 production LOC, which is when the spec was written — and
it was written before the PR opened, not after.

That ordering is not the intended one, and the record should say so plainly. What
it cost is visible in the criteria: AC5 exists *because* the first draft pinned a
tag that does not exist and a repository name that has since been renamed, both
found only when the work was checked against reality. A proposal written first
would have asked "does this action exist at this tag?" before the answer mattered.

## Out of scope

- **Rollout to other repos.** #786's full scope is uniform delivery across every personal
  repo via a reusable workflow. This slice proves the shape on one repo first.
- **Retiring CodeRabbit.** The standing rule caps autonomous reviewers at two, and #786
  requires recording which tool found what over a bounded window before either is
  retired. Both run in parallel.
- **The Gitea leg** (#1005), which is gated on kubelab#1077 anyway.
- **Prompt tuning.** #786 ranks it last by leverage, and it needs real reviews to tune
  against. `extra_instructions` here does only noise control — telling the reviewer not
  to restate what shellcheck, bats, golangci-lint and gitleaks already report.

## Risks / open questions

- **Delivery is blocked by #983, and not by anything in this change.** `dotf secrets sync
  ci` refuses the whole batch when any entry fails its liveness check, and `BITACORA_PAT`
  is a dead token (HTTP 401). So `NAN_API_KEY` cannot reach GitHub Actions secrets until
  #983 is resolved or the operator passes `--skip-verify`. Worth noting as a shape, not
  only an instance: one dead credential blocks delivery of every other, which is the same
  batch-abort defect as #1004.
- **Cannot be validated from a worktree** (#939): `dotf` resolves `DOTFILES_REPO_DIR`
  ahead of the cwd, so `secrets sync ci` reads the registry in the main checkout. The
  registry change here is unverifiable until it merges.
- **The empirical acceptance test needs a live PR.** #786 is right that the real test is
  opening a PR with a known defect and confirming it is found. Fixtures cannot substitute
  for that, and it cannot run before merge.
- **Cost of a second reviewer on every PR.** Two bots commenting is noise if both are
  verbose. Mitigated by `persistent_comment` and by the parallel window being bounded —
  but if the noise is worse than the queue, that is a finding and the window should end
  early.
- **Open question — does the model see enough?** `custom_model_max_tokens` is set to 200k
  against an advertised 1M context. Deliberately a floor: LiteLLM has no registry entry
  for NaN models, so an unset budget silently truncates the diff, and a reviewer reading
  half a change reports on half a change. Raise on measured need, not preemptively.

## Acceptance criteria

- [ ] **AC1** A PR in this repo receives inline review comments generated through NaN.
      Proven by a live PR, not by fixtures.
- [ ] **AC2** The reviewing model is a reasoning-class one and there is **no cheap
      fallback**: `fallback_models` is empty. `harness/reviewer-pool.json` excludes
      `qwen3.6` from the adversarial-review pool by name — *"a reviewer that PASSes
      cheaply is worse than no gate"* — and two files in this repo must not hold opposite
      policies on who may review.
- [ ] **AC3** `sensitive/**` never enters the model call. Asserted in config and pinned by
      a test, because the diff leaves this infrastructure only at that call.
- [ ] **AC4** The workflow holds **exactly one** inference credential, delivered by
      `consumers: ci:<repo>` rather than ambiently. #1025 is the counter-example: the
      spec-review path injects the whole registry to authenticate one model, so one broken
      item mapping takes down authentication for everything.
- [ ] **AC5** The action is pinned to a release tag that exists, at the project's current
      name. Both were wrong in the first draft — `qodo-ai/pr-agent` now redirects to
      `The-PR-Agent/pr-agent`, and `@v0.30` does not exist.
- [ ] **AC6** A provider failure degrades the review rather than blocking the PR, **and
      the absence is visible** — GUARD-002 reports `declined` and the PR goes red. The
      second clause is what distinguishes this from #786's original wording, which as
      written describes a silent green.
- [ ] **AC7** `AGENTS.md` enters the review prompt, so the repo's standing orders become
      review criteria with no additional wiring.

## References

- Bitácora board: `mlorentedev/dotfiles#786` (see the `issue:` frontmatter field)
- `#1019` / `specs/GUARD-002-review-attestation` — the gate that makes a failed review visible
- `#1005` (TOOL-014) — the Gitea leg, gated on this one
- `#983` — the dead `BITACORA_PAT` blocking `sync ci`
- `#1025` — whole-registry injection on the spec-review path
- `#939` — `DOTFILES_REPO_DIR` outranking the cwd
- `harness/reviewer-pool.json` — the reviewer policy this config must not contradict
- `ai/nan/README.md` — NaN endpoint and key material
