---
spec: "TOOL-013-pr-agent-reviewer"
verdict: "PASS"
reviewed_sha: "e31c4ee4ac24c5bbd28d3f6407108e604811777d"
reviewer: "nan/deepseek-v4-flash"
date: "2026-10-10"
---

## Adversarial review

**Scope**: TOOL-013-pr-agent-reviewer — PR-Agent on NaN inference as a second, inline PR reviewer. Round 2: the round-1 FAIL (F1, inline delivery untested) and its fix.
**Sources**: `specs/TOOL-013-pr-agent-reviewer/{proposal,tasks,verification,features}.md`; `specs/TOOL-013-pr-agent-reviewer/review.md` (round 1); the three commits the spec owns since the archive review — `883654cb` (FAIL review), `7186bc0a` (#2325, the fix), `e31c4ee4` (#2335, live evidence); `.pr_agent.toml`; `.github/workflows/pr-agent.yml`; `scripts/check-review-attestation.sh`; `harness/{review-attestation,pr-agent-upstream-contract}.json`; `harness/skills/pr-review-triage/SKILL.md`; `docs/lessons/lesson-397-a-config-guard-must-assert-the-path-that-runs.md`; `tests/pr-agent-config.bats`; `tests/check-review-attestation.bats`; upstream `The-PR-Agent/pr-agent@8e5a9295973b24af4b70cafd0b660a230811ef9e` (the pin): `pr_reviewer.py`, `inline_comment_dedup.py`, `github_provider.py`, read live from the GitHub API.

### Spec and task alignment

- **Diff scope — the launcher's base is not this change.** The stated base `2f7bfd5b` is **708 commits, 2,485 files and 195,356 insertions** behind `HEAD`. That is not an accident of a stale branch: `ResolveReviewBase` anchors on the parent of the commit that *added the spec folder* (`cli/internal/spec/review_base_test.go:71`, `TestResolveReviewBaseIsTheParentOfTheCommitThatAddedTheSpec`), TOOL-013's folder was added at `23c57169` (#1032) and the spec was revived much later by #2325 — so the anchor is 708 commits behind the work. The instruction "review the whole change" is therefore unreadable for this spec: obeying it literally means reading 195k lines of other specs' work. I reviewed the TOOL-013-owned surface at `HEAD` and re-ran its evidence, and recorded the base resolver as a finding (F-scope). This is the same question round 1 raised as Q1; it is real and it is not this change's defect.
- **All nine `features.json` verifications were exercised fresh**, not read: they are filters over two suites, both re-run here — `bats tests/pr-agent-config.bats` → **73 ok, 0 not ok, 1 skip** (the `DOTF_TEST_NETWORK=1` upstream-contract leg), `bats tests/check-review-attestation.bats` → **62 ok**. That matches `verification.md`'s stated 73/73 (one network-leg skip) and 62/62.
- **Round-1 F1 is fixed, and the fix is the route round 1 recommended.** `[pr_reviewer] inline_key_issues = true` is now set, and the guard asserts the setting *and* that `review` runs automatically — the path that actually runs. Confirmed by mutation (below), not by reading the claim.
- **`tasks.md`**: implementation, follow-on and update-contract boxes are `[x]`; the only open box is "Adversarial review passes before archive", which is this run. `proposal.md`'s AC boxes remain `[ ]` and `features.json` states remain `pending` — expected under pass-state gating (see F-state).
- **No `[AGENT-DRAFT]` / `[AGENT-SUGGESTION]` markers** in the authored spec files.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Minor | REAL | Review scope (launcher, not this change) | The stated base resolves to a 708-commit / 2,485-file diff, so "the WHOLE change" is 195k lines of unrelated work. `ResolveReviewBase` anchors on the commit that added the spec folder; TOOL-013's was added at #1032 and revived at #2325. A reviewer who obeys the instruction literally reads other specs and can miss the owned surface; one who narrows it risks under-scoping. | `git rev-list --count 2f7bfd5b..HEAD` → 708; `git diff --stat 2f7bfd5b...HEAD` → 2,485 files / 195,356 insertions; the spec owns 11 files across `883654cb`/`7186bc0a`/`e31c4ee4`. | n/a (tooling) — `TestResolveReviewBaseIsTheParentOfTheCommitThatAddedTheSpec` is the current, and the defect is that this anchor ages when a spec is revived. | vault / other spec (`HARNESS-112` lineage) — **not** TOOL-013's contract set |
| Minor | REAL | Handoff / state | `features.json` still reads `state: "pending"` for f1–f9 and all nine `proposal.md` AC boxes are `[ ]`, while the criteria are demonstrably met. The archive must run the harness to set pass states; until then the status record is unreadable as one. | `features.json` / `proposal.md` at `HEAD`; the nine verifications re-run here all pass. | the nine `features.json` `verification` commands (all pass fresh — 73/73 + 62/62) | harness pass-state run; no code |
| Question / assumption | SPECULATIVE | AC1 evidence fidelity | The inline comments read back with `line: null` from `pulls/N/comments`. Most plausibly GitHub marks them outdated after the later pushes (all six PRs were pushed again), or they anchor at file level; either way they are review comments on the diff, so AC1 holds. Worth one glance that they are line-anchored at publish time. **Does not move the verdict.** | `gh api repos/mlorentedev/dotfiles/pulls/2335/comments` → `github-actions[bot]`, path set, `line: null`; same on #2326. | UNTESTED as to anchoring | — (observation) |

### What I tried to break, and what held (evidence, not praise)

Each attack was run fresh in this session; the working tree was restored and is clean apart from the launcher's `review-request.json`.

- **F1 — inline delivery (the round-1 Blocker/Major).** Mechanism confirmed at the pin, not inferred: `pr_reviewer.py:1174` calls `_publish_key_issues_as_inline_comments` only when `config.publish_output` and `pr_reviewer.get('inline_key_issues', False)`; that method returns early unless `can_verify_inline_comment_publication(self.git_provider)` (`inline_comment_dedup.py:243`), which requires callable `get_persistent_comment_bodies` **and** `get_recent_inline_comment_bodies`; `GithubProvider` defines both (`github_provider.py:1296,1309`), so the capability is **true on the Action path** and located findings publish via `publish_code_suggestions` from the same `/review` response. **Mutation M1:** `inline_key_issues = true` → `false` turns `pr-agent: the automatic review publishes its findings inline` red with *"[pr_reviewer] inline_key_issues is not true: upstream defaults it to false…"*; reverted byte-identical. The guard now asserts the path that runs, which is exactly what round 1 found missing.
- **AC1 — live evidence independently reproduced, not read from `verification.md`.** `gh api` on `pulls/N/reviews` and `pulls/N/comments`, filtered to `github-actions[bot]`: #2335 → review `5481828782` `COMMENTED` body=0 + **2 inline**; #2326 → review `5481615000` `COMMENTED` + **2 inline**; #2333 → `reviews: []` + 0 inline (the round-1 table's stated exception). The table in `verification.md` is accurate and re-runnable.
- **Attestation change (AC6 interplay).** The new fixture `pr-agent-inline-review.json` carries an empty-body `COMMENTED` review by the declared login; `TOOL-013: PR-Agent's inline-findings review attests through reviews[]` passes and asserts it does **not** attest comment-shaped. The classifier door it uses is the pre-existing declared-login rule (`check-review-attestation.sh:201-209`, #1033) — this change documents and pins it, it does not widen it: an undeclared login still cannot attest (`#1033: an undeclared automation login cannot attest through reviews[]` passes).
- **F3 — `vendor/**`.** `.pr_agent.toml`'s `[ignore] glob` now carries `vendor/**` alongside the repo's entries; `pr-agent: the ignore list keeps upstream's vendor/** default` passes.
- **AC9 — one Action.** Three `uses: The-PR-Agent/pr-agent@…` steps exist, but their `if:` guards are mutually exclusive (`route.outputs.first == 'anthropic'`, the default leg, and `published_after_nan == 'false'`), so at most one runs; the "at most two bounded attempts…" and "ambiguous failures fail closed without invoking a second Action" guards pass.
- **AC8 network leg** was **not re-run in this session** (`DOTF_TEST_NETWORK=1`); its other half (`is bound to the action pin`) passes, and the network half is covered by round 1's blob-mutation evidence and CI.

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | A | The one unmet criterion from round 1 is now met and reproduced live; the remaining eight are held by mutants that go red on their own property. |
| Verification       | A | Evidence is reproducible with commands I ran: two suites green (73/73 + 62/62), live PR rows re-read and matched, and the new guard proven falsifiable by mutation. |
| Scope              | B | The spec-owned surface matches the proposal exactly; the 2,485-file diff is a launcher artifact, reported as F1-scope, not crept in by this change. |
| Reliability        | B | Fail-closed by construction and disclosed: inline publication degrades to the summary (not to silence) when the provider cannot verify, and the attestation gate distinguishes that from an absent review. |
| Maintainability    | B | Config comments now describe the code that runs, a named lesson (397) records the guard defect class, and the guard comments are scoped to what they assert. |
| Handoff-readiness  | B | Runbook, ADR-040/042 lineage and promotion answers stand; `features.json` states and AC ticks await the harness run (F-state). |

### Verdict

**PASS** — the round-1 REAL Major is fixed and its fix independently verified; no Blocker and no REAL Major remain. The two Minor REAL findings are a launcher tooling defect outside this change and the expected pre-archive harness state; the single SPECULATIVE observation does not move the verdict. Rubric: all B or above, two A.

### Recommended next steps

The contract set (`proposal.md` / `tasks.md` / `features.json`) is **closed** by this verdict — editing any of it invalidates the review. These are for the implementer to disposition in `verification.md` (excluded from the staleness check) or carry to a follow-up ticket:

1. **Harness, not by hand** — run the nine `features.json` verifications so the states leave `pending` and the AC boxes can be ticked; that is the mechanism that turns the spec's own status record into one (F-state).
2. **File the review-base defect** against `HARNESS-112`'s owner: a base anchored on spec-folder creation becomes a whole-repo diff when a spec is revived months later, which makes the skill's "whole change" instruction unreadable and invites under-scoping. Suggested shape: anchor on the spec's first *owned-file* change since the last archived review, or emit the owned subset and say so.
3. **Optional** — confirm the inline comments are line-anchored at publish time (the `line: null` read-back); no change required unless publishing is actually file-level.

`dotf spec archive TOOL-013-pr-agent-reviewer` is **advisable** once step 1 has run: the review is fresh against `e31c4ee4`, no `[AGENT-DRAFT]` markers remain, and the remaining archive-checklist boxes are the archive's own work.
