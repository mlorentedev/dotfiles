---
spec: "GUARD-003-review-loop-determinism"
verdict: "PASS-WITH-GAPS"
reviewed_sha: "a97495e8ff27f6b410121ef7497c7975d433d2c7"
reviewer: "nan/qwen3.8-flash"
date: "2026-10-21"
---

## Adversarial review

**Scope**: `GUARD-003-review-loop-determinism` — round 2, re-review of the round-1 FAIL (F1 authorship
of the triage record, F2 writer-side marker guard, F4 unticked ACs). HEAD `a97495e8`.

**Sources**: `specs/GUARD-003-review-loop-determinism/{proposal,tasks,verification}.md`,
`review-round1.md` + `review-request-round1.json` (prior round), the diff `e608989e...HEAD` restricted to
this spec's surface, `.github/workflows/{review-attestation,pr-agent}.yml`,
`scripts/check-review-attestation.sh`, `harness/review-attestation.json`,
`harness/skills/pr-review-triage/SKILL.md`, `cli/internal/prtriage/{prtriage,fetch}.go` + tests,
`cli/internal/cmd/{pr,pr_land}.go`, `cli/internal/prland/*`,
`tests/{check-review-attestation,review-attestation-workflow,pr-agent-config}.bats`, live GitHub (read-only).

**Freshness**: `contract_digests` in `review-request.json` (proposal `595d08fe…`, tasks `a69de830…`,
features.json empty) match `dotf spec digest GUARD-003-review-loop-determinism` built from this tree —
the contract has not moved since launch. `dotf spec digest` is **absent from the installed 0.76.0**
(`Error: unknown command "digest" for "dotf spec"`), so the comparison ran on `go build ./cmd/dotf`
(F1 below).

### Spec and task alignment

- **T13 (round-1 F1, AC8)** — closed in `577fb05e`. `lastTriage` (`prtriage.go:255-260`) now requires
  `hasLinePrefix(marker) && hasStanding(c, reg.Triage.Standing)`; `hasStanding` (`:267-279`) upper-cases
  and trims both sides, and an unread association (`""`) is **not** standing — fail-closed in the loud
  direction. `Comment.AuthorAssociation` is populated only on the conversation-comment path
  (`fetch.go:362-366`), which is the channel the skill writes to (`dotf forge pr comment`).
- **AC8's wording** was amended in the contract set: "*written by an account with standing to write it
  (not any commenter)*" — round-1's requested contract change is present, which is what makes this
  round's verdict attach to the right claim.
- **T14 (round-1 F2, AC9)** — closed. `tests/pr-agent-config.bats:1370` "every writer of the triage
  record spells the registry's marker" checks all three writer files for the registry string, rejects
  any *drifted* `##\s*review\s*triage` spelling in the same file, and requires the marker at line start
  in the skill template; `:1406` "who may ask for a review is who may record its triage" ties
  `triage.standing` to the inline set at `pr-agent.yml:147`. Registry `standing_why` names both guards.
- **AC1/AC4/AC5/AC6** — re-checked at HEAD, not inherited from round 1: the `workflow_run` trigger is
  present and guarded (deleting it turns `review-attestation-workflow.bats` case 2 red, checksum-
  verified); the `/review` artifact path is guarded by cases 80–83; shellcheck/`bash -n`/actionlint all
  exit 0.
- **Round-1 F4** — the nine AC boxes are ticked and `proposal.md` reads `status: verifying`; the one
  open box in `tasks.md` is "Adversarial review passes before archive", which is correct while this
  review is the thing being decided.
- **Widening resistance** (the AGENTS "class declared in a file, with tests that fail if it widens"):
  adding `"NONE"` to `triage.standing` turns **both** `TestTheRealRegistryDeclaresTheDefaultStanding`
  and the bats drift guard red. `reviewerRegistry()` defaults to `prtriage.FetchRegistry`, which reads
  the registry from the **default branch**, so a PR cannot widen its own standing by editing the file
  in its own head — the base-pinning property #1019 established holds here too.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Minor | REAL | review-loop tooling (launcher prompt) | The launcher's own prompt instructs the reviewer to verify `contract_digests` with `dotf spec digest <id>`, but that subcommand exists only in this tree — the released CLI (`dotf 0.76.0`, what a reviewer has on a fresh box) has `archive/audit/init/review` and refuses with `unknown command "digest"`. A reviewer who does not think to build from source marks the digest check UNVERIFIED, so the freshness property this spec exists to add is untestable at the exact moment it is needed. Fix: print the digests into the prompt text the launcher already computed, or name the `go run ./cmd/dotf` fallback. | Ran both this session: installed `dotf spec digest …` → error; `go build ./cmd/dotf && ./dotf spec digest …` → the two matching digests above | UNTESTED — no test pins the prompt's command list to the released command set (`cli/internal/spec/review_launch.go:356`) | code (+tests); adjacent surface (#2875/SDD-042), so **ticket rather than block** |
| Minor | THEORETICAL | channel asymmetry | A `## Review triage` recorded as a **submitted review body** rather than a conversation comment can never clear the queue: `lastTriage` iterates `pr.Comments` only, and `fetchReviews` (`fetch.go:390-396`) does not carry `author_association` at all (GitHub returns it there). Today the documented writer path is `dotf forge pr comment`, so the two channels agree; the gap opens the moment anyone dispositions through the review UI or a future writer uses that channel. Failure direction is loud (PR stuck in the queue, `pr land` refuses), never silent green | Code read + `gh api /pulls/2855/reviews` shape; live queue run shows the comment path working | UNTESTED — `TestEvaluate_*` cases feed triage only via `Comments`; no case feeds the `Reviews` channel | tests + docs |
| Minor | THEORETICAL | writer-facing doctrine | `harness/skills/pr-review-triage/SKILL.md` §7 tells the agent *what* to write but not *who may write it*: a session whose `gh` is authenticated as a GitHub App posts with `author_association: NONE` (measured on #2855: `coderabbitai[bot]`/`github-actions[bot]` = NONE, `mlorentedev` = OWNER), so its triage record is invisible to the queue and `dotf pr land` will refuse that PR indefinitely with no message explaining why. The rule is documented in the registry (`standing_why`), which the person following the skill does not read | `gh api issues/2855/comments` this session (2× NONE, 2× OWNER); `hasStanding` returns false for NONE; skill text at `:174-193` | partially — `pr-agent: every writer … spells the registry's marker` guards the string, not the doctrine | docs (`harness/skills/…`, outside the contract set; `TestTheRealRegistryDeclaresTheDefaultStanding` already holds the value) |
| Minor | SPECULATIVE | SSOT drift | `.github/merge-grant.yml:13` carries a fourth spelling of `## Review triage` (in a comment, describing the `reviewer-triaged` condition) and is **not** in T14's `writers` list, so a marker change leaves it stale while every guarded file passes. Behaviour is unaffected — `reviewer-triaged` is enforced through `Facts.Untriaged` → `prtriage.FetchWithRegistry`, no Go code hardcodes the string — so this is prose drift only | `grep -rn "## Review triage" --include=*.yml` → one hit, outside the guard's list; `cli/internal/prland/grant_test.go:54` shows the condition is checked by fact, not by string | guard exists (`pr-agent-config.bats:1370`) but omits this file | tests (add the file to the writers list, or drop the literal from the comment) |
| Minor | REAL | spec artifacts | `verification.md` still carries unfilled template stubs — "Test status" reads `- Test suite: <command> -> <output / coverage %>` and `No regressions …: yes / no`, "Decisions made during implementation" is `-` twice — while the real evidence sits in the appended sections. A reader who stops at the template reads the spec as unverified. `verification.md` is outside the contract set, so this is fixable at any time | File read at HEAD | n/a | verification.md (post-review, free) |
| Minor | REAL | review scope / base resolution | Carried from round 1 (its F3), still unfixed: the launcher-resolved base `e608989e` predates ~2,000 merges, so `base…HEAD` is `3083 files / 267,925 insertions` against a spec surface of `14 files / 4,235 insertions`. Not the author's defect; recorded so the archive decision knows what was actually read: **this review read the spec's own surface, at HEAD state, not the whole base diff** | `git diff --stat e608989e…HEAD` and the same restricted to the spec's 14 surface files, both this session | n/a (measurement) | out of scope (launcher base-resolution rule, gate implementation outside this repo) |

**Round-1 findings, dispositioned**: F1 (REAL Major, authorship) — **fixed**, verified by mutation below.
F2 (Minor, writer-side guard) — **fixed**, verified by mutation. F3 (Minor, scope) — **still open**,
carried above. F4 (Minor, unticked ACs / `status: draft`) — **fixed** (ACs ticked, `status: verifying`).

Nothing in the diff reintroduces F1's class: I looked for other marker-driven decisions that ignore
authorship and found none — reviewer detection keys on declared logins (its own trust list), the
`exempt.signatures` path keys on the diff file set (deliberately not on author or branch), and
`pr land` consumes the queue rather than the comments.

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | A | All nine ACs hold at HEAD; round-1's REAL Major is closed with a named test whose non-vacuity I demonstrated by mutation, and every residual gap I found fails loud rather than green. |
| Verification       | A | Every claim I could reach reproduced exactly at HEAD — 83/83 bats, 46 Go packages ok, clean lints, the marker mutation failing with the guard's own diagnostics, the standing mutation failing two guards — and mutation landing is checksum-verified, which is the false-green AC5 names. |
| Scope              | B | The 14-file surface matches the proposal plus documented follow-ons (T10–T14), each with its issue; the outsized base diff is the launcher's, not the author's (F6). |
| Reliability        | B | Fail-closed on unread association, empty `standing`, absent `standing` (defaulted, not refused), full comment/review/inline pages, unreadable registry, non-numeric PR number; the gaps are channel/doc asymmetries (F2, F3), not silent greens. |
| Maintainability    | B | `hasStanding` is 12 lines, `lastTriage` unchanged in shape, the doctrine carried as comments beside what it constrains; `DefaultStanding` is a package-level mutable slice (copied at use, so harmless today) and the guards are inline Python in bats, which works but is a second language in the suite. |
| Handoff-readiness  | B | Lesson captured (`docs/lessons/…a-ported-copy-may-lack-must-default-not-refuse.md`), ACs and tasks ticked, promotion candidates answered; the stale template stubs in `verification.md` (F5) are the gap. |

### Verdict

PASS WITH GAPS

No Blocker and no open Major — round-1's single REAL Major (F1) is fixed, and the fix is guarded in both
directions: an undeclared author cannot clear the queue, and widening `triage.standing` breaks two
committed tests. Six Minors remain, none of which changes behaviour toward a silent green, so they are
tracked with dispositions rather than blocking the archive. Per `cli/internal/spec/review.go:47-51`
(`Blocks()` false for `PASS-WITH-GAPS`), this verdict satisfies the archive gate.

### Evidence reproduced in this session

```
git rev-parse HEAD                                   -> a97495e8ff27f6b410121ef7497c7975d433d2c7
go build ./... ; go vet ./...                        -> clean
go test -count=1 ./... (cli)                         -> exit 0, 46 packages ok
bats tests/check-review-attestation.bats tests/review-attestation-workflow.bats
                                                     -> 83 ok (last: ok 83 … a skipped reviewer run is not re-evaluated)
bats tests/pr-agent-config.bats -f triage            -> 2 ok (T14 writers guard + standing drift guard)
shellcheck / bash -n scripts/check-review-attestation.sh -> exit 0 / exit 0
actionlint review-attestation.yml pr-agent.yml       -> exit 0
go build ./cmd/dotf -> spec digest GUARD-003-review-loop-determinism
  proposal.md 595d08feca9776b0…  tasks.md a69de83046f0b70…  features.json ""   == review-request.json
installed dotf 0.76.0 -> `dotf spec digest …`       -> Error: unknown command "digest" for "dotf spec"
```

Mutations, each confirmed landed by sha256 before/after and then restored (tree clean apart from this
review's own files):

```
lastTriage: drop "|| !hasStanding(c, reg.Triage.Standing)"   prtriage.go 80b2f9598966 -> 898fead59b20
  -> TestEvaluate_AnUndeclaredAuthorDoesNotClearTheQueue FAIL for NONE, CONTRIBUTOR, FIRST_TIMER, ""
     ("a \"NONE\" triage record cleared the queue: triaged")
  -> restored (80b2f9598966), package ok again.  Proves the regression test is not vacuous.

review-attestation.yml: delete the workflow_run trigger block  aefe2b811455 -> 84a36b3fe355
  -> not ok 2 "…re-evaluates when the pr-agent reviewer finishes [AC1]"
     # no workflow_run trigger: our own reviewer finishing wakes nothing
  -> restored (aefe2b811455).  AC5's guard is live at HEAD.

review-attestation.json: triage.marker -> "## Review dispositions"   c59e029981e9 -> 7219dc9ae934
  -> not ok "every writer of the triage record spells the registry's marker", naming
     release-dotf.md, guide-pr-agent-reviewer.md and "SKILL.md: no line starts with …"
  -> restored (c59e029981e9).  Round-1 F2 is genuinely closed.

review-attestation.json: triage.standing += "NONE"                   c59e029981e9 -> (widened)
  -> not ok "who may ask for a review is who may record its triage"
     # workflow ['COLLABORATOR','MEMBER','OWNER'] != registry ['COLLABORATOR','MEMBER','NONE','OWNER']
  -> AND TestTheRealRegistryDeclaresTheDefaultStanding FAIL
  -> restored (c59e029981e9).  The widening is guarded on both sides.
```

Live GitHub, read-only:

```
dotf pr triage-queue (built from this tree)
  -> #2890 "coderabbitai reviewed again after the last triage, 2026-10-21 21:32"
     #2876 "github-actions reviewed, never triaged"   + "Nothing here applies anything"
     (a standing author's earlier record was counted, then a later review re-opened it:
      both halves of AC8 observed on real data, and the command still lists only)
gh api issues/2855/comments  -> one comment matching ^## Review triage: mlorentedev, OWNER
                            -> associations on that PR: 2x NONE (the two bots), 2x OWNER
     (the predicate does not lock out the real writer, and bot-authored records do not clear)
```

### Recommended next steps

The contract set (`proposal.md`, `tasks.md`, `features.json`) is **closed** by this verdict — nothing
below asks for it, and editing it would stale this review. Each item is for the implementer to
disposition in `verification.md` (applied / ticketed / declined with a reason) or carry into a
follow-up ticket:

- **F1 — ticket it** (`dotf spec digest` reachable from the released CLI, or digests printed in the
  launch prompt). This is #2875/SDD-042's surface, not GUARD-003's, and the fix belongs with whoever
  owns the launcher; until then a reviewer on an un-upgraded box should build from source rather than
  mark the digest check UNVERIFIED.
- **F2 + F3 — one docs/test change covers both**: add a `Reviews`-channel case to
  `cli/internal/prtriage/prtriage_test.go` stating that a review body is *not* a triage record (so the
  asymmetry becomes a specified behaviour instead of an accident of the read path), and say so in
  `pr-review-triage/SKILL.md` §7 next to the standing rule.
- **F4** — add `.github/merge-grant.yml` to the T14 guard's `writers` list, or delete the literal from
  its comment; either closes the fourth-copy drift without touching behaviour.
- **F5** — fill or delete the `verification.md` "Test status" / "Decisions" template stubs; the real
  evidence is already in that file's appended sections.
- **F6** — the base-resolution rule is the owner's call and outside this repo's reviewable scope;
  nothing to apply here beyond stating what was read, which this review does.
- **Then archive**: `dotf spec archive GUARD-003-review-loop-determinism` is **advisable** now — this
  review is fresh against the launch digests and `PASS-WITH-GAPS` does not block. It becomes stale the
  moment the contract set changes; a `tasks.md` tick for "Adversarial review passes before archive" is
  a contract edit, so leave that tick to the archive step rather than writing it first.
