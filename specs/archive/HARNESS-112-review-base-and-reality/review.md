---
spec: "HARNESS-112-review-base-and-reality"
verdict: "PASS"
reviewed_sha: "53e9c846ff2f6383a93bf3288a1f916a3fe7b106"
reviewer: "nan/deepseek-v4-flash"
date: "2026-09-05"
---

## Adversarial review

**Scope**: HARNESS-112-review-base-and-reality
**Sources**: `specs/HARNESS-112-review-base-and-reality/{proposal,tasks,verification,features}.json` + `git diff 1fe3531798120056a75fb9b2d66741ec1635e46f...HEAD` (5 commits, 30 files).

### Spec and task alignment

- **Acceptance criteria 1–5** (base resolution, refusal, prompt, sidecar): all implemented and covered by named tests I ran (see below).
- **AC6 / AC8** (verdict list + Major definition honour the reality axis; both skill copies carry the routing rule): implemented, guarded by `tests/guard-review-verdict-honours-reality.bats` (7/7) and verified in all three deployed copies.
- **AC7** (build/test/vet/lint clean, guard mutation-proven): reproduced.
- **AC9** (staleness refusal names the review-preserving exit first): implemented, guarded by `TestStaleRefusalOffersTheExitThatKeepsTheReview`.
- Tasks: all implementation boxes `[x]`. The "Not done, deliberately" items (backfilling `base_sha`, softening Majors/Blockers, field-granular staleness, parsing the findings table in Go) are each argued in the proposal or tasks.md and are genuinely out of scope.
- No `[AGENT-DRAFT]` / `[AGENT-SUGGESTION]` tags in any contract file.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Minor | REAL | Scope | The diff `base...HEAD` carries two commits outside the proposal's stated "What": #1539 (`fix(pi)`, `ai/pi/models.json` + `tests/guard-pi-models-schema.bats`) and #1541 (`chore(spec)`, the BUG-093 archive move). Both are legitimate, tested changes tied to the review infrastructure, but neither is named in the proposal's What/acceptance criteria, so the reviewed scope is broader than the spec. | `git log 1fe353...HEAD` shows 5 commits; only #1535/#1543/#1549 are HARNESS-112. The proposal's "What" names only the base and reality work. | UNTESTED (no test asserts review scope contains only the spec's work) | none required (observation); HARNESS-112 changes are correct |
| Minor | THEORETICAL | Spec artifact (skill) | The skill's "Inputs" section (line 127) still reads "If no PR: `git diff <base>...HEAD` against the merge base (typically `main`/`master`)". This is the exact guidance root 1 aims to eliminate. The prompt overrides it ("do NOT substitute `main` or guess one"), so a reviewer following the prompt is safe, but the generic skill text still invites the substitution. | `grep -n 'typically' ~/.pi/agent/skills/adversarial-review/SKILL.md` → line 127; the prompt (`ReviewPrompt`) is explicit and comes last. | UNTESTED (no guard asserts the Inputs section no longer says "typically main/master") | skill (repo record `harness/skills/` + vault source `00_meta/skills/`), outside the contract set |
| Minor | THEORETICAL | Code | `ResolveReviewBase` assumes the spec folder is created within the change being reviewed. If a spec folder pre-exists (created in an earlier merged PR and still active), the base resolves to the parent of the ORIGINAL folder-add commit, so `git diff base...HEAD` includes unrelated changes accumulated since. The proposal's "Risks / open questions" acknowledges the "trusts the spec folder's add-commit" assumption. | code read of `ResolveReviewBase` (`git log --diff-filter=A -- <specDir>` returns the earliest add-commit, whose parent is the base). | UNTESTED (no test covers revising a pre-existing spec folder) | code (if deemed worth addressing); currently an accepted limitation |
| Minor | THEORETICAL | Code | The `baseSHA == headSHA` refusal branch is unreachable through `ResolveReviewBase`: the base is always the parent of a commit in HEAD's ancestry, so it can never equal HEAD (a descendant). The branch is exercised only via the test seam (`withReviewBase`). Defensive, but dead in production. | reasoning over the resolver; `TestSpecReviewRefusesWhenTheBaseIsTheHead` drives the seam, not the resolver. | `TestSpecReviewRefusesWhenTheBaseIsTheHead` (via seam, not the resolver) | none (defensive); note for maintainers |
| Minor | THEORETICAL | Code (comment) | In `ResolveReviewBase`, the root-commit case comment says "the empty tree is the honest base" but the code returns `""`, which the caller treats as "cannot review". Comment and behavior disagree. | code read. | UNTESTED | code comment |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness | A | All 9 acceptance criteria verified with named tests; negative paths covered (no-base, base==head, empty-diff anchor); no observed defects. |
| Verification | A | Every command in features.json / verification.md is reproducible — I re-ran build, `go test`, vet (windows/darwin), golangci-lint, the bats guard, and the mutation proofs, and they match the recorded output. |
| Scope | B | The HARNESS-112 changes match the proposal exactly; the diff additionally carries #1539 (pi fix) and #1541 (BUG-093 archive), legitimate related side-changes not named in the proposal's "What". |
| Reliability | A | Refusal paths handled (no base, base==head), deterministic local resolution, safe (git output only, no shell injection), idempotent. |
| Maintainability | A | Small functions, clear naming, thorough WHY comments; cyclomatic complexity well under threshold. |
| Handoff-readiness | A | proposal/tasks/verification updated in-session; lesson 275 written to `docs/lessons/`; contract set and routing rule stated. |

### Verdict

**PASS**

No Blocker. No Major. Every finding is Minor; the two that carry real evidence (scope breadth, the skill's residual "typically main/master") are tracked and do not move the verdict. The rubric is all B or above.

The change does what it claims. `ResolveReviewBase` returns the parent of the spec-folder add-commit and is correct in both the work-branch and post-squash cases (verified against this repo's history: base `1fe353` = parent of the commit that added `specs/HARNESS-112-review-base-and-reality/`). The launcher refuses an unscopeable review, the prompt states the base and forbids substitution, the sidecar records `base_sha`, the verdict list and the Major definition now both honour the reality axis, and the staleness refusal names the review-preserving exit first. The guard is real: reintroducing the old unqualified FAIL line turns 3 of 7 assertions red, and reverting the Major definition or the staleness message breaks their named tests.

### Recommended next steps

Route by set. The contract set (`proposal.md`, `tasks.md`, `features.json`) is **closed** by this verdict; no recommendation below asks for an edit to it. Disposition these in `verification.md` or a follow-up ticket.

- **(skill, outside contract set)** Consider dropping "typically `main`/`master`" from the skill's "Inputs" section (line 127) in both the repo record and the vault source, so the generic text no longer invites the exact substitution root 1 eliminates. The prompt already overrides it.
- **(code, outside contract set)** If the pre-existing-spec-folder case can occur in practice, record it as an explicit limitation or track it; it is currently an accepted risk argued in the proposal.
- **(observation, no action)** The `baseSHA == headSHA` refusal is unreachable via the resolver; it is defensive. Note for maintainers.
- **(code comment, outside contract set)** Align the root-commit comment in `ResolveReviewBase` with the `""` return.
