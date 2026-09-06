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
      {
          "status": "dry_run",
          "tier": "mid",
          "role": "planner",
          "pool": "claude",
          "model": "sonnet",
          "exit": 0,
          "output": "would dispatch role \"planner\" to claude:sonnet (no request was sent)",
          "resolution": {
              "role_from": "inferred",
              "tier_from": "inferred",
              "pattern": "pattern-bitacora-tracking"
          },
          "attempts": [{"pool": "claude", "model": "sonnet", "status": "dry_run"}]
      }

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

- [ ] **AC7 — NOT SATISFIED. Blocked, and stated rather than substituted for.**

      No real dispatch was performed. This machine's
      `~/.config/dotfiles/machine.json` contains only a `paths` block and declares
      no `machine.id`, so ADR-032 §8's identity gate denies every non-local pool:

      Error: this machine has not declared an identity, so every non-local pool is
      denied (ADR-032 §8)

      Declaring an identity would open the pools, and that is a decision about
      which pools this machine may spend quota on — the machine owner's, not the
      implementer's. Satisfying it by pointing `XDG_CONFIG_HOME` at a fixture, as
      AC1–AC6 legitimately do, would here be using a test fixture to step around
      the exact gate the dispatch contract exists to enforce. So it is left open.

      A `dry-run` quote is deliberately NOT offered in its place: AC7's whole
      content is that a REAL pool answered, and the proposal's own risk section
      says the honest claim must match the criterion.

      Filed as **#1547**: the gate is right, but nothing creates the identity,
      prompts for it, or reports it missing — `dotf doctor` has no check for it,
      and no `dotf` subcommand writes the `machine` block. The only thing that
      reports it is a dispatch refusing at the moment you try to use it.

      **This is not new to `auto`.** The same gate refuses `dotf agent run` on
      this machine and always has: `machine.json` here has only ever carried a
      `paths` block. So AC7 is blocked by a pre-existing machine state, not by
      anything this change introduced.

      To close it, declare an identity in `~/.config/dotfiles/machine.json` and
      run the command in `features.json` entry `HARNESS-120-f7`:

      $ cd cli && go run ./cmd/dotf agent auto --role reviewer \
          --task 'Reply with the single word OK and do nothing else.' --timeout 5m

      Two details of that command are deliberate and should not be "simplified".

      **The task is inert.** AC7's claim is that a pool chosen by the persona's
      OWN DECLARED tier answered, which `resolution.tier_from: inferred` proves;
      the join is AC1's business, not AC7's. Running the AC1 task text against a
      live model instead would ask a real agent to actually go and file a ticket,
      with `--cwd` defaulting to this worktree. `reviewer` is named for the same
      reason: it declares `mid`, and its record forbids it to edit anything.

      **It is `go run ./cmd/dotf`, not `dotf`.** The `dotf` on PATH is a stale
      dev source build (#1469) with no `auto` subcommand; running it would report
      "unknown command" and read like the feature is missing.

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
