---
tags: [spec, verification]
created: "2026-09-05"
---

# Verification - HARNESS-120

## Evidence

Every criterion below was exercised against the SHIPPED records and the shipped
`model-map.json`, not against fixtures, except where a fixture is the only way
to produce the state (AC4 needs a persona declaring an illegal tier, and no
shipped record does).

The identity gate was satisfied with a fixture `machine.json` under a scratch
`XDG_CONFIG_HOME`. That is not a bypass: none of AC1–AC6 spends quota — five
resolve or refuse before any backend is reached, and AC6 dispatches to a stub
binary. AC7 is the one that needs a real pool, and it is NOT satisfied; see
below.

- [x] **AC1** — `role: planner`, `tier: mid`, neither supplied.
      Test `TestAgentAuto_DerivesBothRoleAndTierFromTheTask`; commit `201afe3`.

      $ dotf agent auto --task "open a ticket for the bitacora" \
          --backend dry-run --timeout 60s --repo-root <checkout>
      {"status":"dry_run","tier":"mid","role":"planner","pool":"claude","model":"sonnet","exit":0,"duration_ms":0,"output":"would dispatch role \"planner\" to claude:sonnet (no request was sent)","truncated":false,"resolution":{"role_from":"inferred","tier_from":"inferred","pattern":"pattern-bitacora-tracking"},"attempts":[{"pool":"claude","model":"sonnet","status":"dry_run"}]}

      Quoted byte-exact, unpretty-printed, as the command emits it — round 2 of
      review caught that the earlier block had been reformatted and had dropped
      `duration_ms` and `truncated`. A quote a reader cannot diff against their
      own run is a claim, not evidence.

      The matched pattern is `pattern-bitacora-tracking`, which is what
      `proposal.md` predicted before the code existed. That is the criterion
      being reproducible from the spec file rather than fitted to the result.

- [x] **AC2** — two personas, nothing dispatched, both named, exit 1.
      Test `TestAgentAuto_BothRefusalsDispatchNothingAndDiffer`; commit `201afe3`.

      $ dotf agent auto --task "run a spec-driven development review of this change" ...
      Error: this task resolves to 2 personas and a dispatch runs one: planner, reviewer

      Choose with --role. They are not ranked, deliberately: each declares a skill
      the task matched, and which one is right is a judgement about the work rather
      than a property of the rules
      exit=1

- [x] **AC3** — no rule matched, and the refusal DIFFERS from AC2's.
      Same test; the assertion is `ambiguous.Error() != unmatched.Error()`,
      never two fixed strings, so a reworded message cannot fail it and a
      collapsed one cannot pass it.

      $ dotf agent auto --task "xyzzy plugh frobnitz" ...
      Error: this task matched no trigger rule, so no persona can be derived from it

      Name one with --role, or rephrase the task using the vocabulary the rules
      declare — the match is on keywords, not on meaning
      exit=1

- [x] **AC4** — an illegal tier and an absent one are each refused, never
      defaulted. Test `TestResolveTierForPersona`; commit `ccd681d`.
      Produced by copying the roster to a fixture root and editing planner's
      `model:`, since no shipped record declares an illegal tier.

      # model: enormous
      Error: persona "planner" declares tier "enormous", which harness/model-map.json
      does not: it declares low, mid, top

      Either fix `model:` in <fixture>/harness/agents/planner/AGENT.md or add the
      tier to the map — but not by guessing here, because a dispatch routed to a
      tier with no chain has nowhere to go
      exit=1

      # model: (empty)
      Error: persona "planner" declares no tier, so there is no chain to walk for it

      Add one to <fixture>/harness/agents/planner/AGENT.md:

          model: mid

      harness/model-map.json declares these: low, mid, top. This is refused rather
      than defaulted because a tier chosen here is a route nobody picked, and
      afterwards it is indistinguishable from one someone did
      exit=1

      Note the legal tiers are read from the map's `chains` block and are not a
      constant in Go. A hardcoded `top|mid|low` would be a second place the tiers
      are true.

- [x] **AC5** — each override honoured, each reported, and overriding one does
      NOT mark the other. Test
      `TestAgentAuto_OverridesAreHonouredAndReportedAsDictated`; commit `201afe3`.

      $ ... --role architect          $ ... --tier low
      {                               {
       "role": "architect",            "role": "planner",
       "tier": "top",                  "tier": "low",
       "resolution": {                 "resolution": {
        "role_from": "dictated",        "role_from": "inferred",
        "tier_from": "inferred"         "tier_from": "dictated",
       }                                "pattern": "pattern-bitacora-tracking"
      }                                }
      }

      `--role architect` still reads architect's OWN declared tier (`top`), so the
      override skips the join without also dictating the tier. `pattern` is absent
      for a dictated role because no rule was consulted — reporting one would be a
      claim about a derivation that did not happen.

- [x] **AC6** — the persona's record reaches the dispatched PROCESS.
      Test `TestAgentAuto_SendsThePersonasOwnRecordToTheBackend`; commit `201afe3`.

      Asserted in the test against the `Request` the backend RECEIVED, never
      against stdout: `DryRun` echoes the role and the route and never the task,
      so a preamble that was composed and then dropped would leave every
      assertion on the emitted record passing.

      Reproduced at the CLI with a stub `claude` on PATH that prints its argv and
      stdin verbatim — so the evidence is the bytes that actually crossed the
      process boundary, not a Go value:

      $ dotf agent auto --task "check this diff" --role reviewer \
          --backend subprocess --timeout 60s --repo-root <checkout>
      ARGV: -p --model sonnet
      --- STDIN AS RECEIVED ---
      You are operating as the `reviewer` persona.

      Verify-phase persona. Invoke to check a change against what it claims —
      before a merge, and always before a spec archives. Reviews and refutes;
      never fixes, because the independence is the whole value.

      Everything above the TASK delimiter is your operating instruction for this
      dispatch. Follow it as written, including its boundaries — especially the
      work it tells you NOT to do.

      # Reviewer
      ...
      ## Boundaries

      You review; you do not edit, and you hold no write capability on purpose — a
      reviewer who fixes what they find has stopped being independent of it. ...

      ===== TASK =====

      check this diff

      3219 bytes sent where the bare task is 15. The task arrives LAST and intact.

- [x] **AC7 — SATISFIED.** One real dispatch, not `dry-run`, answered from the
      pool the persona's OWN declared tier chose.

      Unblocked by the machine owner declaring an identity in
      `~/.config/dotfiles/machine.json` — the decision ADR-032 §8 exists to make
      deliberate, and the reason this sat open through round 1 of review.

      $ dotf agent auto --role reviewer \
          --task 'Reply with the single word OK and do nothing else.' --timeout 5m
      {
          "status": "ok",
          "tier": "mid",
          "role": "reviewer",
          "pool": "claude",
          "model": "sonnet",
          "exit": 0,
          "duration_ms": 12705,
          "output": "OK\n",
          "truncated": false,
          "resolution": {
              "role_from": "dictated",
              "tier_from": "inferred"
          },
          "attempts": [{"pool": "claude", "model": "sonnet", "status": "ok"}]
      }

      **`tier_from: inferred` is the whole criterion.** Nobody passed `--tier`.
      `reviewer`'s record declares `model: mid`, `ResolveTierForPersona` read it,
      and the walk took `chains.mid`'s first entry. A live model answered in
      12.7 s with the one word it was asked for.

      The task is deliberately inert and the persona is `reviewer`, which is
      read-only by its own mandate. AC7 asserts that a pool chosen by the
      DECLARED tier answered — the join is AC1's business — so running AC1's
      "open a ticket for the bitacora" against a live model would have asked a
      real agent to go and file one, with `--cwd` defaulting to this worktree.

      The `features.json` command for `HARNESS-120-f7` was then run verbatim,
      `jq -e` predicate and all: **exit 0**.

## Test status

- `cd cli && go build ./... && go vet ./... && go test ./...` → all packages pass.
- `GOOS=windows go vet ./...` → clean. The Windows leg fails the whole package on
  one error, and `splitRecord`'s CRLF branch is Windows-only behaviour.
- `golangci-lint run` at the `versions.conf` pin (v2) → `0 issues.` Run with the
  pinned binary, not a local one: a different major reports 0 on code CI rejects
  (BUG-071). It caught one real finding on the way (ST1008) which was fixed.
- No regressions: the full suite passed before and after. The `agent run`
  refactor is behaviour-preserving, and `TestAgentRun_DoesNotComposeAPreamble`
  pins the one behaviour it would have been easy to change by accident.

**Mutation-checked, because a passing test proves nothing until it can fail:**

| Mutation | Result |
|---|---|
| Send the bare task instead of the composed preamble (the pre-HARNESS-120 behaviour) | `TestAgentAuto_SendsThePersonasOwnRecordToTheBackend` FAILS — "the persona did not travel" |
| Make the ambiguous case return its first candidate instead of refusing | `TestAgentAuto_BothRefusalsDispatchNothingAndDiffer` and `TestResolveOneDispatchesOrRefuses` both FAIL |

## Decisions made during implementation

- **`agent run` does NOT compose the preamble; only `auto` does.** AC6's
  `--role reviewer` is `auto`'s flag, which AC5 settles by describing the same
  flag on the same command. `run` is the primitive that takes a route as given:
  its `--role` is a label, not a roster lookup, and making it one would refuse
  names it has always accepted (its own tests pass `--role r` against temp roots
  with no roster at all). The residual is real and stated rather than hidden:
  **`dotf agent run --role reviewer` still dispatches a generic agent.** Pinned by
  `TestAgentRun_DoesNotComposeAPreamble` so the two cannot drift together
  unnoticed, and filed as **#1548** so the boundary stays revisitable rather than
  buried here.

- **`frontmatterBlock` became `splitRecord` returning both halves**, rather than
  the preamble getting its own body splitter. The duplicate had already been
  written and carried the bug that broke the build — a literal U+FEFF where the
  `"﻿"` escape belonged, which Go rejects anywhere but byte 0 of a file. The
  deeper reason to delete rather than patch it is the one its own comment stated:
  two ideas of where a record ends is how one of them starts being wrong, which
  is exactly how `check-roster-consistency.py` came to report "no skills" in
  silence.

- **Composition happens at the command layer, before the walk.** `Dispatch`
  retries across pools; composing per attempt could send different bytes to the
  second pool than the first, making a comparison of the two answers a comparison
  of two different questions.

- **`prepareDispatch` extracted from `run` before `auto` was written.** The
  identity gate and deny list are fail-closed, and a second command re-deriving
  them is a second place they can be forgotten — where forgetting looks exactly
  like working.

- **`dispatchRoot` is `env.ResolveHarnessRoot`, deliberately not the suggest
  hook's `harnessRoot`.** The latter falls back to `~/.dotfiles` — the deployed
  copy rather than the checkout — and `run` has never resolved that way. Both
  commands share one resolution so a task cannot be routed by one root's map
  while being run as another root's persona.

- **`Record` widened with `role` and `resolution`.** Additive and `omitempty`,
  but it is a contract change for JSON readers and is called out in the PR body.
  `run` emits no `resolution` because it derived nothing.

## Adversarial review round 1 — FAIL

`agy/gemini-3.1-pro-high`, 2026-09-06, against `reviewed_sha` `7b105e1`.
Artifact: `review.md` (signed; never edited). Rubric: Scope A, Reliability A,
Maintainability A, Correctness B, Handoff-readiness B, Verification C.

Dispositioned here rather than by editing the contract files, per
`docs/lessons/lesson-275-a-review-that-demands-contract-edits-invalidates-itself.md`.
Note that lesson's distinction applies in my favour only for the *second*
finding: under a **PASS** verdict a contract edit is self-defeating, but under
**FAIL** the sequence is explicitly fix-then-re-review, so a `proposal.md` edit
is a legitimate exit for the Blocker if the owner chooses it.

| # | Severity / Reality | Finding | Disposition |
|---|---|---|---|
| 1 | **Blocker / REAL** | AC7 unverified — no end-to-end dispatch against a live model | **OPEN, owner's decision.** Correctly identified, and it is the same gap this file already declared before the review ran. Two legitimate exits: supply the evidence (needs `machine.id` declared — a pool-spend decision, **#1547**), or amend AC7 in `proposal.md` and re-review. `review: waived` is NOT an exit here: lesson 275 records that `checkReviewGate` returns on a waiver *before* calling `FindReview`, so it declares the requirement inapplicable rather than satisfied |
| 2 | Minor / SPECULATIVE | A dictated `--tier` bypasses `ResolveTierForPersona`, so an invalid one fails later in `ResolveChain` with a less descriptive error | **APPLIED, with a corrected fix.** Measured first: the old message was `tier "colossal" has no chain in harness/model-map.json` — accurate and fail-closed, but it did not list the legal tiers, which the persona path did. So the finding is real and its severity is right. **The reviewer's proposed fix was wrong** — reusing `ResolveTierForPersona`'s error would blame a *persona's record* for a value a human typed. Fixed in `ResolveChain` instead, where both paths already meet, so `agent run` gains it for free. Pinned by `TestDictatedTierRefusalNamesTheLegalOnes` |

**Observation about the review run itself, recorded not ticketed** (meta-work is
capped while opened > closed): the reviewer left a 108 KB `scratch-diff.patch` in
the **repo root**. Nothing in this repo writes that name — `grep -rn scratch-diff`
across Go, shell, markdown and JSON returns nothing — so the reviewing agent
created it in the working tree itself. It is not in `.gitignore`, so a `git add -A`
would commit it. Deleted by hand. Not worth a `.gitignore` line, because the name
was chosen by an external agent and the next one will pick a different one; the
real guard is the standing rule to stage by explicit path. Worth knowing before
anyone else launches a review.

Verified after the fix, both paths now identical:

    $ dotf agent auto --tier colossal ...
    Error: tier "colossal" has no chain in harness/model-map.json: it declares low, mid, top
    $ dotf agent run --tier colossal ...
    Error: tier "colossal" has no chain in harness/model-map.json: it declares low, mid, top

## Adversarial review round 2 — PASS

`nan/deepseek-v4-flash`, 2026-09-06, against `reviewed_sha` `9d3a7ee` (= HEAD of
the contract set). Artifact: `review.md`, signed, never edited. Rubric:
Verification A, Reliability A, Handoff-readiness A; Correctness B, Scope B,
Maintainability B. **No Blocker, no REAL Major.**

A different reviewer from round 1 — the launcher draws from the pool at random,
so the two verdicts are independent draws rather than one model reconsidering.
Round 2 independently reproduced AC1-AC6 at the CLI, rebuilt and re-ran the whole
Go layer plus `GOOS=windows go vet` and `golangci-lint` at the pin, and
**mutation-checked two of my paths itself** — the ambiguous case returning its
first candidate, and the bare task sent instead of the preamble — confirming both
tests fail when the behaviour is removed.

Round 1's two findings are both closed: the Blocker by the real dispatch above,
the Minor by the `ResolveChain` fix in `6d8db91`.

| # | Severity / Reality | Finding | Disposition |
|---|---|---|---|
| 1 | Minor / REAL | The `--role` refusal cannot distinguish "not declared" from "declared but `kind: autonomous`" — `--role hermes-nan` says no such persona, and there is one | **TICKETED, #1562.** Behaviour is correct and must not change: an autonomous steward is not a dispatchable phase. The message is the defect, and it is the same shape as the vacuous allow reason #1534 removed from the gate. The `kind: autonomous` path is UNTESTED and the ticket says so |
| 2 | Minor / REAL | AC1's quoted output was not byte-exact — reformatted, missing `duration_ms` and `truncated` | **APPLIED.** `verification.md` is outside the staleness set, so this is the one recommendation safe to act on. Re-captured raw. The finding is fair and slightly worse than cosmetic: AC7 was quoted verbatim and AC1 was not, so a reader could not tell which quotes were transcribed |
| 3 | Minor / REAL | `resolvePersonaForTask` is ~48 lines against the repo's 40-line rule (CC 9, within limits) | **TICKETED, #1562.** Noted there that `golangci` does not enforce the 40-line rule, so `0 issues` says nothing about it |
| 4 | Minor / SPECULATIVE | `taskDelimiter` could collide with a task or record body containing `===== TASK =====` | **TICKETED as surface-only, #1562, explicitly do-not-gate.** No repro; the delimiter was chosen for that unlikelihood. Refusing a dispatch over a string that has never occurred trades a real capability for a hypothetical one |
| 5 | Question | Round 2 did not independently re-run the live AC7 dispatch — it verified the deterministic half by dry-run and declined to spend pool quota | **ACCEPTED as stated.** Correct restraint, and the confirmation it did perform is the load-bearing half: `--role reviewer` resolving to `mid`/`claude:sonnet` with `tier_from: inferred` is what AC7 asserts. The live run remains quoted above, from this session |

**`tasks.md`'s review checkbox is left UNTICKED on purpose, and this line is
where that is recorded.** Ticking it would edit the contract set after
`reviewed_sha`, and the archive gate measures staleness against exactly
`proposal.md`, `tasks.md` and `features.json` — so the act of marking the review
done is the act that invalidates it. I staged that edit before catching it,
having read lesson 275 an hour earlier, which is a fair measure of how natural
the mistake is. The review's own text closes the box: *"the one unticked box is
the adversarial review itself, which this run closes."*

**The three code findings are tracked, not fixed, and that is deliberate.** The
review passed against `9d3a7ee`; applying code changes afterwards would ship code
no reviewer saw, using a PASS earned by different code. Only the `verification.md`
edit was applied, which the review names as outside the contract set.

**One scope observation from the reviewer worth preserving:** Scope was graded B
rather than A because the reviewed diff carries upstream HARNESS-111 changes
(`SKILL.md`, a bats guard, lesson 275) pulled in by the `main` merge `511c77d` —
not authored here. That is a property of reviewing a merge commit rather than a
defect in this change, and it is worth knowing that a `main` merge into a branch
dilutes what the next review reads as "the diff".

## Promotion candidates

- [x] Lesson for the repo's `docs/lessons.md`? **Yes** — a literal U+FEFF inside a
      Go string literal is a hard compile error for the whole package
      (`invalid BOM in the middle of the file`), and it is invisible in every
      editor and in `git diff`. The general form is worth recording: when a helper
      is written to "mirror" an existing parser, delete the mirror instead — the
      duplication is the defect and the transcription bug is only its symptom.
- [ ] ADR-worthy decision? **No** — ADR-032 and ADR-035 already govern the
      dispatch contract and the routing registry; nothing here changes either.
- [ ] New pattern candidate for `00_meta/patterns/`? **No** — the fail-closed
      refusal shape is already recorded, and this is one project's application of
      it, not a second occurrence of a new idea.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/HARNESS-120/` -> `specs/archive/HARNESS-120/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
- [ ] **AC7 satisfied or explicitly waived by the spec owner.** It is the only
      criterion asserting the chain works end to end against a real pool; archiving
      with it open is a decision, not an oversight, and must be recorded as one.
