# Run report: tickets-20261022T0735Z

- Armed: 2026-10-22T07:35Z by manu. Expires 15:35Z.
- Ended: 2026-10-22T10:32Z. **reason: queue-empty**, 2 of 2 items landed.
- Objective: close the guard-no-gui pid-substring flake (#2872) and add `dotf lessons renumber` (#2837).

## Items

| Item | PR | Result | Merge commit |
|---|---|---|---|
| #2872 guard-no-gui pid substring flake | #2933 | landed under the grant | `dbfa9688` |
| #2837 `dotf lessons renumber` | #2938 | landed under the grant | `0fd77c1b` |

### #2872 → #2933
- **Fix:** both stray-detector tests compare the detector's output on the pid field through one helper, `_found_has_pid`. A new test pins the helper against the substring cases.
- **CI caught a miss:** a bare `!` assertion that `guard-bats-negation` rejects. My single-file run did not cover that guard. Fixed in `d757f0c8`.
- **Full local suite:** 1999/2000. The one failure is `--refresh against the real vault`, a drift of local vault records already ticketed in #2913; it is unrelated.
- **Review:** pr-agent (haiku) reviewed it with no findings. CodeRabbit hit its rate limit.

### #2837 → #2938
- **Command:** `dotf lessons renumber <file> --next|--to N`, in `cli/internal/lessons/renumber.go` plus the `cmd` wiring.
- **Design:** the advisor was consulted before implementing.
  - The scan tolerates a number used twice; that is the state after main is merged.
  - `--next` takes max+1, never a gap.
  - It rewrites only links that name this lesson and no other one.
  - An ambiguous `[[lesson-N]]` is left alone and printed.
  - Tracked files move with `git mv`.
  - A second run is a no-op.
- **Live run:** on a copy of the real tree (513 lessons), lesson 212 moved to 514, the links in 213 and 214 were rewritten, and `fmt --check` printed [OK].
- **spec-gate:** 294 production LOC. Labelled `skip-sdd` with the rationale "the issue is the plan", on the advisor's recommendation; `spec init` is closed while the active specs are over the WIP limit. This was a stated decision, not an oversight.
- **Review:**
  - CodeRabbit: 2 real findings, both applied in `0f4d07a0`: a relative `--dir`, and `Normalize` running after the move.
  - pr-agent (glm5.3-flash): 1 applied in `66e9f527`, the error message after a partial write. 2 declined: the body was truncated, so it did not see the skip rationale or the Knowledge section.
  - `66e9f527` was not re-reviewed: pr-agent does not review a push that follows a triage (#1756). Disclosed on the PR.

## Filed during the run
- #2939: `pr land` should name `dotf lessons renumber` in its taken-number refusal. It is a one-line change in `cli/internal/prland`, outside this envelope's paths.

## For the owner
- `machine.json` declares no identity, so `dotf agent auto` refuses. The in-harness advisor was used instead once its rate limit had passed.
- #2909 (Windows secrets wrapper) was excluded at the start because it overlaps SEC-009; it needs your call or the advisor's.
- #2907 (harness `--refresh` rewrites other records) is yours.
- Remove the worktrees `feat-plat003-close` (the launch worktree) and `chore-autopilot-tickets-20261022` after this session. Also `chore-autopilot-vm-20261021` from the previous run.

## Host
- A `caffeinate -i -t 28800` wake-lock covered the run.
- A usage-limit pause around 10:05–10:30Z delayed landing #2938; nothing was lost.
