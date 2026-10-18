# Run sec014-20261018T1835Z

**Verdict:** ended `queue-empty` at about 22:55Z, after 2 items in about 4h20 of the 8h envelope. 2 items done, 0 parked, 3 PRs merged, 0 unmerged. Waiting on you: one acceptance on a host, two A/B decisions and one question. Nothing irreversible was published: no release was cut, and every change in this run sits behind the next release.

## Waiting on you

1. **SEC-014 acceptance (#2388), after the next release, at the machine and as your user (never `sudo dotf`):** `dotf env role rm ssh-target`, then `dotf converge --only workstation-target` twice. The first run makes the change; the second skips, because no record is left. The `tasks.md` row stays `[ ]` until then.
2. **The Remote Login question (macmini and msi).** Both hosts were converged before the record existed, so their key and listener entries are `unknown`. A retirement there removes nothing and holds the sshd drop-in while sshd runs, because without the drop-in, sshd would serve its stock config, passwords included. If Remote Login was on before the target, the hold is right. If it was not, turn sshd off, then converge again.
3. **#2790, A or B:** the pf LaunchDaemon takes an unrecorded enable reference at every `bootstrap` (each apply, through `RunAtLoad`) and at every boot. #2791 covers only the references Apply takes.
4. **#2794:** retiring the firewall and power pieces (SEC-014 PR 3) needs its shape decided. Until then the runbook's Rollback section lists them as hand steps.
5. **The macmini's live pf reference.** Its 0.75.0 apply predates #2791, so the reference it took stays unrecorded until the next reboot. Nothing to do beyond being aware of it.
6. **pf root measurement (#2356):** whether `pfctl -E … -f` prints a token when the load fails. #2791 releases the token when one is printed; one that is never printed cannot be released, and that is bounded by reboot.

## Landed

| PR | Head | What |
|---|---|---|
| #2791 | `8b23333` | Closes #2755 (option A, advisor). pf records the `pfctl -E` token with the boot id in `/var/db/dotf/pf-token` and releases it in the undo and in Commit; the timer releases only its own boot's references. Three review rounds produced three fixes: the undo after a fired timer, a rejected load's reference, and the timer with no boot id. 17 tests; 10 of 10 mutations killed. |
| #2799 | `035ee63` | SEC-014 PR 2: converge retires the target when the role is removed (rules in the #2388 comment). 15 of 15 mutations killed. |
| #2807 | `a9e0f01` | The companion split out of #2799 to keep it within the 10-file budget: the record keeps the key line the step wrote, and `Serving` on systemd and Windows. 3 of 3 mutations killed. |

All three landed under the merge grant: CI green on the head, `dotf pr triage-queue --pr N` exit 0, and a `## Review triage` comment on each head pr-agent reviewed. CodeRabbit was rate-limited throughout (a notice, not a review), so pr-agent was the only reviewer.

## Declined review findings (reasons on the PRs)

- #2791 `e613679`/`5445ba11`: a kill between the token-file rewrite and the release loop. The reverse order leaves the file naming references already released, whose numbers can be reissued within the same boot.
- #2791 `2441a0f8`, `782d40e0`: raised again for code the head had already fixed. This is the carry-forward defect on #2761.
- #2799 `4e228a4a` (in part): a record holding no key line, combined with a rotated repo key, is named in the plan rather than held, because a hold would never converge. Records from #2807 on carry the line.

## Filed or recorded

- #2790 (pf LaunchDaemon references), #2794 (SEC-014 PR 3).
- #2803: another occurrence of the git-fixture flake (`cli/internal/spec` staleness fixtures: `git gc --prune=now` collides with background gc), seen on #2807.

## Envelope and budget

- Every write stayed inside `write_paths`. #2807 counts as part of item 2 (#2388), not a third item; it was split to respect `max_files: 10`.
- No root command was run and no converge was run on this host. No lesson was added: no new failure class, and the design notes are in `verification.md`.
- Host: `caffeinate -s -i -t 28800` held the wake lock for the whole window.

## Next action

After the next release: the acceptance in item 1, then a decision on #2790 and #2794.
