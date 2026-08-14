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

- [ ] [AC4] `dotf spec review <spec-id>`: resolve the primary from the pool and
      pass `--provider`/`--model` **explicitly**. Not optional polish — BUG-074
      round 3 was pinned only because `~/.pi/agent/settings.json` on this machine
      happens to default to nan; pi's own default provider is `google`, and that
      file is unversioned per-machine state.
- [ ] [AC5] Named tmux session `review-<spec-id>` so the run is watchable while
      it happens. Windows / no-tmux degrades to foreground and says so.
- [ ] [AC6] Machine-readable transcript beside `review.md` (`pi --mode json`,
      `agy --output-format stream-json`). Check the size before making it a habit.
- [ ] Raise `agy --print-timeout` — it defaults to 5m and round 3 took ~25m, so
      the fallback dies on defaults. A concrete instance of "configured is not
      exercised".
- [ ] [AC7] Prove the Gemini arm with one real review. NaN's evidence already
      exists (BUG-074 round 3); a fallback never observed working is decoration.

## Slice 3 — the standing rule where agents read it

- [ ] Amend the skill's "Do NOT prescribe which agent, model, or IDE" line: the
      pool records the human's standing choice and the gate enforces it, so the
      skill should point at the pool rather than imply the reviewer is free to
      choose. **Edit the vault source** (`00_meta/skills/adversarial-review/`),
      direct to master, then `compile-harness.sh --refresh` — the repo copy is
      generated. Check no other session has staged vault work first.

## Closing

- [ ] Every acceptance criterion covered by at least one test
- [ ] `features.json` entries with non-vacuous verification commands
- [ ] `go build ./...`, `go vet ./...`, `golangci-lint run` (pinned)
- [ ] `verification.md` filled in
- [ ] PR opened referencing this spec folder
- [ ] Fresh adversarial review — on a pooled model, which this spec is about

## Machine-readable features

See `features.json`. States stay at `pending`: an agent may not write `passing`,
only the harness may, after running each `verification` command and capturing
exit 0.
