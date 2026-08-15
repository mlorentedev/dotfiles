---
tags: [spec, tasks, templates]
created: "2026-08-13"
---

# Tasks - HARNESS-071-reviewer-pool

> TDD order. One task = one focused commit. Tick as you go.

## Setup

- [x] Branch created from main: `feat/reviewer-pool` (worktree `dotfiles-wt-review`)
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] Work-gate linked: #955

## Slice 1 — the gate (this is the layer that enforces)

- [x] [AC3] `harness/reviewer-pool.json`: ordered allow-list of model **ids**,
      not tool names. Separate file rather than a key in `manifest.json` (which
      doctor's harness-drift section reads) and rather than `model-map.json`
      (H-044's future schema, a different shape — squatting on the name would
      force H-044 to migrate this file).
- [x] [P] [AC1] Test: a reviewer outside the pool blocks the archive, and the
      refusal names both the found reviewer and the whole pool.
- [x] [P] [AC2] Test: absent pool skips the check; malformed pool refuses
      (4 shapes: bad JSON, empty array, missing key, blank id).
- [x] [P] Test: exact match after trimming; a near-miss id (`…-lite`) refuses;
      any pool entry works, not only the first; `--force-without-review` still
      overrides.
- [x] Observe all of the above go red before implementing — confirmed: the three
      refusal tests failed, the permission tests passed (no gate = allow).
- [x] [AC1] Implement `loadReviewerPool` + `checkReviewerPool`, wired last in
      `checkReviewGate` because it is the only check that asks *who* reviewed
      rather than what they concluded.
- [x] [AC8] Mutation-test the predicate — 5 mutants, **5 detected**, with a
      harness that aborts when a mutation fails to apply.
- [x] Live smoke against the REAL pool file, not a fixture: a `claude-opus-5`
      signature is refused with the pool printed; `nan/deepseek-v4-flash`
      archives cleanly.

## Slice 2 — the launcher

> **Slice 1 shipped separately, in #958.** It is the layer that enforces, and it
> is independently verifiable on its own; splitting kept both PRs within the
> sizing policy. This PR (#959) carries slices 2 and 3, so the boxes below are
> this PR's own checklist rather than a declared remainder.

- [x] [AC4] `dotf spec review <spec-id>`: resolve the primary from the pool and
      pin the model **explicitly**, per runner rather than uniformly — `pi` takes
      `--provider` *and* `--model`, while `agy` has no `--provider` at all and
      selects the family through `--model` alone. An entry missing what its own
      runner needs is an error, never a fall-back to that runner's default.

      Not optional polish: BUG-074 round 3 was pinned only because
      `~/.pi/agent/settings.json` on this machine happens to default to nan, and
      `pi --help` documents its own default provider as `google`. That file is
      unversioned per-machine state.
- [x] Flags verified against the installed binaries rather than assumed, twice
      over. An earlier draft invented a `--prompt-file` that neither runner has;
      a later one assumed both took the prompt positionally, which is true of
      `pi` but not of `agy`, whose `--print` consumes the prompt as its value.
- [x] [AC5] Named tmux session `review-<spec-id>` so the run is watchable while
      it happens. Windows / no-tmux degrades to foreground and says so.
- [x] [AC6] Machine-readable transcript teed beside `review.md` (`pi --mode
      json`, `agy --output-format stream-json`).
- [x] Raise `agy --print-timeout` to 90m — it defaults to 5m and round 3 took
      ~25m, so the fallback dies on defaults. A concrete instance of
      "configured is not exercised".
- [x] Shell-quote every wrapped argument: tmux re-parses its command through a
      shell and so does `sh -c`, and the prompt carries quotes, backticks and
      newlines. Unquoted, a `$(…)` in a prompt would execute.
- [x] Refuse an out-of-pool model at the launcher too — defence in depth, and it
      names what IS available rather than only what is forbidden.
- [x] Mutation-test the launcher — 6 mutants, **6 detected**.
- [x] [AC7] Prove the Gemini arm with one **real review**. CLEARED on 2026-08-14
      by a run with verdict FAIL at `reviewed_sha b24e105`, recorded in commit
      **`a561342`** — cite the sha, because the review of record that gates this
      archive overwrites `review.md` and the transcript on disk, so the FAIL that
      cleared AC7 lives in history rather than in the working tree.

      An earlier revision ticked this on the grounds that a run had been
      *launched*. That was the wrong claim: the criterion says review, and a
      `[x]` beside "prove it with a real review" reads as done no matter what
      the prose underneath says. So the bar for ticking it now is **reach**, not
      the existence of an artifact — the third blind attempt produced a
      well-formed all-A PASS having executed nothing, which is precisely the
      artifact a naive reading of this box would have accepted.

      Reach verified three ways, not assumed:

      1. The transcript carries this worktree's actual HEAD
         (`b24e105e78429b23d82a49f5dca5c0c08c2b4119`), so `git rev-parse` ran
         inside the repo under review rather than in agy's install dir.
      2. It runs `go test` eight times with real pass/fail output across 147
         recorded steps — the suite was reachable and was used.
      3. None of the three blind-run signatures appear: no "not a git
         repository", no `antigravity-cli` cwd, no empty response at exit 0.

      The verdict being FAIL is *stronger* evidence than a PASS would have been.
      A reviewer that cannot act produces agreeable output; this one produced a
      Blocker that reproduces in four git commands (see the dispositions in
      `verification.md`). NaN cleared the same bar in BUG-074 round 3 by
      re-running the spec's own mutation battery rather than trusting its table.
      A fallback never observed doing the job is decoration — this one has now
      been observed doing it, and finding something.

## Slice 3 — the standing rule where agents read it

- [x] Amend the skill's "Do NOT prescribe which agent, model, or IDE" line: it
      now says the pool is binding where one exists, and the no-pool case — where
      the choice really is still the human's — is preserved unchanged. Edited in
      the **vault source** and re-rendered with `compile-harness.sh --refresh`;
      editing the repo record directly would have been reverted by the next
      refresh. Vault was clean and on master before committing.
- [x] Add a "Launching a review" section to the skill. An agent that knows it
      must not review its own change and does not know how to hand the job off
      simply stops, so the rule needed the command beside it.
- [x] Carry the same statement to the two surfaces agents actually read:
      `AGENTS.md`'s verification-window trigger and `.claude/CLAUDE.md`'s
      workflow section.

## Closing

- [x] Every acceptance criterion covered by at least one test
- [x] `features.json` entries with non-vacuous verification commands — 10
      criteria, each running real tests (1/2/6/2/1/5/3/3/3/1)
- [x] `go build ./...`, `go vet ./...`, `golangci-lint run` (pinned 2.12.2) — 0 issues
- [x] `verification.md` filled in
- [x] PRs opened referencing this spec folder: #958 (gate, merged), #959 (launcher)
- [x] CodeRabbit's review on #959 triaged: 2 Majors on the shell wrappers and 2
      on the pool/AC7 applied; the table-driven-tests Major deferred with a
      stated reason (see below)
- [x] Fresh adversarial review — on a pooled model, which this spec is about.
      No longer blocked on AC7: `agy/gemini-3.1-pro-high` reviewed this spec on
      2026-08-14 and returned FAIL with a reproducible Blocker plus a Major, both
      fixed above with guards. That review is the mechanism reviewing itself and
      finding a real defect in its own gate.

      Freshness and verdict are not claimed by this box, because a box cannot
      enforce them. `dotf spec archive` refuses this folder unless `review.md`
      carries a passing verdict, a `reviewed_sha` no contract file postdates,
      and a `reviewer:` inside `harness/reviewer-pool.json`. The archive commit
      landing at all is the evidence; the box only records that the review
      happened rather than being waived.

## Machine-readable features

See `features.json`. States stay at `pending`: an agent may not write `passing`,
only the harness may, after running each `verification` command and capturing
exit 0.
