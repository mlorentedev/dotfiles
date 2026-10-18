# Run selfdeploy-20261018T1835Z

run selfdeploy-20261018T1835Z ended queue-empty after 3 iterations, about 3h: 3 done, 0 parked, 2 merged, 0 unmerged.

What is waiting on you: nothing irreversible beyond the two merges below. The live Windows acceptance of #2696 needs the work-lab box.

## Waiting on you

- **Landed (irreversible):**
  - #2787 (`536d13e`, closes #2727): selfDeployWhen detail test, F2 row, live Linux proof on 0.75.0.
  - #2808 (`cc9e4bf`, closes #2696; board Done): `dotf update --config-home/--state-home`, which the Windows task passes as flags. Scheduled Task XML has no per-action environment.
- **Windows acceptance of #2696:** on the work-lab box, after the release that carries #2808, run `dotf converge --only self-deploy` with `XDG_STATE_HOME` set. Then check that the task's Arguments carry `--state-home` and that `last.json` lands under it.
- **#2792** (filed): `dotf vm linux` cannot declare a role, so the guest proof of #2727 declared it by hand through `limactl shell`.
- **Worktrees:** `dotf pr land` already removed fix-windows-task-xdg-homes and its branches. Never merge chore/autopilot-selfdeploy-run: it only carries this report and the envelope.

## Items

1. **#2727, landed in #2787.**
   - `TestSelfDeploy_TheDetailNamesTheJobsSlot` covers 04:15, 23:05 and nil (`daily`), and the Reconcile half through an injectable trigger.
   - The test was mutation-checked.
   - Guest proof on v0.75.0 (`dotf vm linux --ref v0.75.0 --keep`, role declared by hand): the first converge returned 12 changed / 0 failed, the second 0 changed / 12 ok.
   - The pr-agent finding (a vacuous Reconcile assertion) was applied. The fix commit was not re-reviewed by pr-agent; this was disclosed on the PR.
2. **Mac self-deploy activation (host step, no PR).**
   - dotfiles-20 activated it before this item came up.
   - I verified it: a re-run of `dotf converge --only self-deploy` reported changed=0.
3. **#2696, option A with skip-sdd, landed in #2808.**
   - `Trigger` now adds the homes to Args on Windows and to Env elsewhere.
   - `Schtasks.read` refuses a job with an environment, so this can no longer be dropped silently.
   - `dotf update` exports the flags before anything resolves the state dir, and it refuses a relative home.
   - Review round (pr-agent on `4ea62672`; the fix `31528e80` was not re-reviewed, CI green on it):
     - Applied: a drive root `C:\` was trimmed to `C:`, which is relative. It now keeps its separator, and the fix was mutation-checked.
     - Applied: the help text now says that a refused home exits before the run and records nothing.
     - Declined: the claim that the evidence was missing. The output is in the PR body, and CI on the head ran the same suite.

## Coordination

- Path constraints from dotfiles-20 were respected:
  - In `converge/`, only selfdeploy*.go was touched, never workstation.go.
  - In `cmd/`, only update.go and update_test.go were touched.
- No lesson was pushed, so the lesson-number check (#2752) did not apply.
- I triaged and landed my own PRs under the merge grant. pr-agent was the only reviewer; CodeRabbit posted rate-limit notices only.

## Host

The run held a wake-lock for 8h (`caffeinate -i -t 28800`).

## Budget

- Items: 3 of 3.
- Iterations: 3 of 6.
- Consecutive failures: 0.
- The envelope was never edited after arming.
