---
spec: "TOOL-039-forge-pr-draft-ready"
verdict: "PASS-WITH-GAPS"
reviewed_sha: "bb25fe23626f33a19a3d1d9457e62ca64a23ef4a"
reviewer: "nan/mimo-v2.6-flash"
date: "2026-11-14"
---

## Adversarial review

**Scope**: TOOL-039-forge-pr-draft-ready — diff `99cd03ae17f591bad1260d050b23d6d63b63095d...HEAD`
(bb25fe23). The range also contains one unrelated commit merged in between —
`9553caa3` feat(lane) lane doctor (#3473) — judged for interaction only: its file set
(lane/audit, lane_doctor, hooks, runbooks, HARNESS-195 spec) is disjoint from the forge
work, and the full suite is green at HEAD across both.

**Sources**: `specs/TOOL-039-forge-pr-draft-ready/{proposal,tasks,verification,features}.json`,
`review-request.json`; commits `f0aea7a6` (implementation) and `bb25fe23` (verification note);
`docs/adr/adr-052-one-forge-write-surface-in-dotf.md` (amended); PR #3479.

### Spec and task alignment

- **Contract integrity**: `dotf spec digest TOOL-039-forge-pr-draft-ready` matches all three
  `contract_digests` in `review-request.json` exactly (features.json `b5940582…`, proposal.md
  `70e9b5b8…`, tasks.md `ffd06a1f…`). Nothing in the contract set changed since the request.
- **AC1** (Gitea `pr create --draft` prefixes `WIP: `, fails naming the URL when the answer says
  `draft: false`): code in `gitea.go CreatePR`; tests `TestGiteaCreatePRDraftSendsAPrefixedTitle`,
  `TestGiteaCreatePRDraftFailsWhenTheInstanceIgnoresThePrefix`,
  `TestGiteaCreatePRWithoutDraftKeepsTheTitle`, `TestGiteaCreatePRDraftDoesNotPrefixTwice`.
  Ran green. Mutation probe (drop the `draft` read-back after create) turns
  `TestGiteaCreatePRDraftFailsWhenTheInstanceIgnoresThePrefix` red — reproduced, then reverted.
- **AC2** (Gitea `pr ready` strips `WIP:`/`[WIP]` any case, succeeds on `draft: false`, fails on
  still-`true`, no write when already ready): tests `TestGiteaReadyPRStripsThePrefix` (four
  spellings), `TestGiteaReadyPRFailsWhenTheTitleStillReadsAsDraft`,
  `TestGiteaReadyPRWritesNothingForAReadyPullRequest`. Ran green. One branch of AC2 is UNTESTED —
  see findings.
- **AC3** (GitHub reads `node_id`/`draft` over REST, sends `markPullRequestReadyForReview`,
  skips the mutation when ready): tests `TestGitHubReadyPRSendsTheMutationWithTheNodeID`,
  `TestGitHubReadyPRSkipsTheMutationForAReadyPullRequest`,
  `TestGitHubReadyPRFailsWhenTheAnswerIsNotReady`. Ran green. Mutation probe (drop the ready
  short-circuit) turns the skip test red — reproduced, then reverted. The GraphQL transport
  (`gh api graphql -f query=…`) matches the existing precedent at
  `cli/internal/forge/preflight.go:85`.
- **AC4** (verb exists, exactly one positive number, help names both forges):
  `parseForgeNumber` rejects non-numeric and `n < 1`; tests `TestForgePRReadyMarksTheNumberedPullRequest`,
  `TestForgePRReadyTakesExactlyOnePositiveNumber`, `TestForgePRHelpNamesDraftAndReadyOnBothForges`
  all ran green.
- **All four `features.json` verification commands ran verbatim, exit 0.** `state: pending` with
  empty `evidence` is correct at review time (only the harness writes `passing`).
- **Every command I ran for this review** (fresh, at this reviewed sha):
  `go build ./... && go vet ./...` → exit 0; `GOOS=windows go vet ./...` → exit 0;
  `golangci-lint run ./internal/forgewrite/... ./internal/cmd/...` → 0 issues;
  `go test ./... -count=1 -timeout 5m` → every package ok, exit 0.
- **No `[AGENT-DRAFT]` / `[AGENT-SUGGESTION]` tags** anywhere in the spec folder (grep: no match).
- **Test deletion retention bar** — `TestGiteaCreatePRRefusesDraftBeforeAnyRequest`
  (`forgewrite_verbs_test.go`), deleted by `f0aea7a6`:
  1. *Exact test / failure it detects*: a `--draft` create on Gitea reaching the forge instead of
     refusing — it enforced "never guess the instance's prefix silently".
  2. *Stronger remaining proof, shown*: the contract was deliberately inverted by this spec, and
     the replacement risk (a prefix sent but unverified) is guarded — mutation probe M1 (drop the
     read-back) turns `TestGiteaCreatePRDraftFailsWhenTheInstanceIgnoresThePrefix` red. Shown in
     this session, not assumed.
  3. *Non-test callers / production code unlocked*: `CreatePR` is called only from
     `cmd/forge_write.go` create; the refusal branch in `gitea.go` is what the diff removes.
  4. *Why it exists*: `git log -S` shows it was added by `89aca1d7` (TOOL-030) with the comment
     "guessing a title prefix would be silent behaviour" — the silence, which the read-back now
     eliminates, is the property to keep.
  5. *Focused command*: `go test ./internal/forgewrite/ -run 'TestGiteaCreatePRDraft|TestGiteaReadyPR' -count=1` — ran, exit 0.
- **Refuted risk (recorded so it is not re-raised):** a title-only PATCH to Gitea's edit
  endpoint could have wiped the PR body. Verified against the exact instance version
  (Gitea v1.27.3, `routers/api/v1/repo/pull.go EditPullRequest`): `if form.Body != nil { …
  ChangeContent … }` — an omitted `body` field leaves `form.Body` nil, so the description is
  untouched; `if len(form.Title) > 0` applies the title. This also matches dotf's own
  `editPayload` pattern ("keeps unset fields out of the request"). No defect.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location |
|----------|---------|------|---------|----------|---------------------------|--------------|
| Minor | THEORETICAL | tests | AC2's post-PATCH still-draft path is untested: if stripping a *known* prefix leaves a title the instance still marks draft (e.g. a doubly-prefixed title, or an unknown prefix behind the known one), the `if p.Draft` error after the PATCH is reached — no test sends a PATCH whose answer stays `draft: true`; only the no-PATCH path is exercised | code read of `gitea.go ReadyPR`; no case in `draft_test.go` carries two prefixes into the PATCH | **UNTESTED** (nearest named: `TestGiteaReadyPRFailsWhenTheTitleStillReadsAsDraft` — its title fails the strip, so it never PATCHes) | tests (a `prServer` case with a doubly-prefixed title) |
| Minor | REAL | spec (tasks) | Two closing boxes in `tasks.md` are stale: `verification.md filled in` and `PR opened referencing this spec folder` are unticked although `verification.md` is filled and PR #3479 exists with body "Spec: `specs/TOOL-039-forge-pr-draft-ready/`" | `tasks.md` read; `gh pr view 3479` output in this session | n/a (process artifact) | spec — contract set: disposition here / in `verification.md`, do **not** edit alongside this verdict |
| Minor | THEORETICAL | spec (tasks) | The manual-check task is ticked with the claim the scratch PR was "closed", but `verification.md` says the owner closes `personal/resume#811` by hand; a live read (`GET /api/v1/repos/personal/resume/pulls/811`) returns **403** unauthenticated, so closure is **UNVERIFIED** from this run | curl exit with HTTP 403; `verification.md` text "the owner closes #811 … by hand" | UNTESTED | spec — a disposition line in `verification.md` (owner closes #811) or follow-up ticket |
| Minor | SPECULATIVE | code | GitHub `ReadyPR` REST read failure discards the underlying error (no `%w`) and omits the ERROR/WHY/FIX shape used on the sibling failure paths: `the answer is not a pull request` with no cause or remedy | `github.go` `json.Unmarshal(out, &pr)` error branch | `TestGitHubReadyPRFailsWhenTheAnswerIsNotReady` covers GraphQL answers only, not REST decode failure — path itself UNTESTED | code (follow-up; not contract) |

No Blocker, no Major. Test traceability: every AC maps to named tests that ran green here; the
four findings above each name their covering test or are marked UNTESTED.

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | A | All four ACs verified by named tests, two mutation probes red-green, and a live Gitea smoke check; negative and idempotency paths covered |
| Verification       | A | features.json commands reproduced verbatim (exit 0), full suite + lint + cross-OS vet green at this sha, mutations reproduced as claimed |
| Scope              | A | Spec commit touches exactly forge code, tests, ADR-052 amendment and spec files; the one other commit in the range (lane doctor #3473) is disjoint and interacted only via the shared suite, which is green |
| Reliability        | B | Timeouts, idempotent `ready` on both forges, ERROR/WHY/FIX on the mandated failure paths; minus for the dropped error wrap (finding 4) |
| Maintainability    | A | `golangci-lint` 0 issues, all functions under 40 lines, WHY-comments on the prefix and mutation decisions, tests named by behavior |
| Handoff-readiness  | B | Proposal/tasks/verification/ADR and follow-up ticket #3477 all updated in-session; minus for the stale `tasks.md` ticks (findings 2–3) |

Aggregation: no D, no C → all B or above. Severity axis: findings are minors only, each with a
disposition → the more severe of the two paths is **PASS WITH GAPS**.

### Verdict

**PASS WITH GAPS** — no blockers, no majors; four minors tracked above, none of which gates the
archive. Severity × reality: all four are minor by severity, and the two strongest tags
(THEORETICAL/REAL-process) are tracked, not blocking.

### Recommended next steps

Route by set — this verdict permits the archive, so the **contract set is closed**: the first two
items are *dispositions to record*, not edits to make alongside this verdict.

- (spec, disposition) Record in `verification.md` that `tasks.md`'s two stale closing ticks and
  the "#811 closed" tail are acknowledged — owner closes #811/branch `chore/forge-draft-check`,
  ticks reconciled at archive time; checkbox text is ignored by the digest normalisation, so this
  is housekeeping, not contract drift.
- (tests, follow-up) Add the doubly-prefixed-title case for the post-PATCH still-draft error in
  `draft_test.go` — cheap, closes the one UNTESTED branch of AC2.
- (code, follow-up) Wrap the REST decode error in GitHub `ReadyPR` with `%w` and the ERROR/WHY/FIX
  shape. Code is outside the contract set; can land any time.
- Second opinion: this run is pinned to the pool draw `nan/mimo-v2.6-flash`; the harness offered
  no stronger-model second opinion inside a pinned review run, and subagent delegation was not
  operator-requested — stating that rather than silently skipping it.

**(a) Verdict: PASS WITH GAPS. (b) `dotf spec archive TOOL-039-forge-pr-draft-ready` is advisable
once the implementer dispositions the four minors above (the gate itself was observed in this
session: it parses this frontmatter and refuses a `PENDING` verdict by name, exit 1). (c) Not FAIL
— no minimum action set applies.**
