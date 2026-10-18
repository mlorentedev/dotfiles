# Run night-20261018T1833Z

**Verdict:** run night-20261018T1833Z ended `queue-empty` after 7 iterations, about 2h40m of active work inside a window of 18:33–21:30Z. Items: 4 done, 1 parked. PRs: 5 merged, 0 unmerged. Nothing irreversible is waiting on you. Everything that landed is an ordinary squash merge of `cli/**`, `docs/lessons/**` or tests, each with green CI and a posted triage.

Envelope: `5cb9d441` on chore/autopilot-night-20261018 (never merge). Ledger: `ledger.md` next to this file.

## Waiting on you

1. **#2759, parked (board: Blocked).** Detecting squash merges on a Gitea origin needs a forge read surface. Two options are on the issue:
   - **(A)** A `MergedPRHeads` read verb behind a forge backend, using the Gitea API with the registry token. Consequence: `forgewrite` stops being write-only, or a sibling read package appears. That is an ADR or a CLI-123 placement.
   - **(B)** `git ls-remote origin 'refs/pull/*/head'` plus a containment check, with no token. Consequence: it cannot tell merged from closed-unmerged, so a branch whose PR was rejected reads as landed and is deleted.

   The message half landed as #2809 (`Refs #2759`).
2. **#2806 (new).** The autopilot skill's claim template ("Picked up in autonomous pilot mode") and its disclosure wording attribute work to an automated run, which the no-AI-attribution override forbids. pr-agent flagged it on #2798. The skill lives in the vault (`00_meta/skills/autopilot`), outside this envelope. I rewrote this run's six affected comments and one issue body by hand.
3. **#2785 (owner-land).** Lint the CLI with `GOOS=windows` in CI. It touches `.github/**`, which is frozen for this run. It came out of #2753, where a Windows-only unused import passed every local check.
4. **#2797 (new, not taken).** Five other `filepath.Glob` sites in `cli/` read an unreadable directory as no match, the same class as #2690. It is outside this run's committed filter (dotfiles-20 asked me to check), so it waits for a queue that names it.
5. **#2773:** still yours to decide: where the per-repo disposable list lives, and which copy is the authority.
6. **Worktree to remove after you read this:** chore-autopilot-night-20261018. Its branch carries only this report and the envelope. Also chore-autopilot-dropped-20261018 from the earlier run.

## Parked

- #2759: see above (A/B; the forge read placement belongs to CLI-123).

## Done

| Item | PRs | Landed | Notes |
|---|---|---|---|
| #2753 | #2786 | `5bc84df` | The Windows-only unused import moved into the `!windows` test file. Follow-up #2785 |
| #2760 | #2795, #2805 | `18725e2`, `a5f03dc` | An unparseable deployed mise config is rewritten instead of blocking the sync. The converge, doctor and `tools sync --dry-run` name the repair. Lesson 490. Split into two PRs to stay within `max_files: 8` (disclosed) |
| #2690 | #2798 | `1d21211` | An unreadable `docs/lessons/` is now one line in the brief, not silence. Lesson 491. Follow-up #2797 |
| #2759 (message half) | #2809 | `164e22a` | On a non-GitHub origin, `worktree done` names the host instead of printing gh's bare exit status. The scp, ssh-with-port and file origins are pinned by a test |
| #2670 | none | closed | Duplicate of #2750/#2773 |

## Post-release v0.75.0 verifications

- **#2749 (live verdict wait on a land):** not observed. All five lands found the attestation already settled, so none had to wait on a verdict. The evidence still comes from the first land that arrives while a verdict is in flight.
- **PLAT-003 AC8:** owned by feat-cross-os-verification-59 (they said so at arming); not run here.
- **`pr land` lock:** observed working. #2798's first land attempt was refused while another session's land held the lock (PID 35302), and the retry merged.

## Review disclosure

The PR author did every triage on this run; none was independent. pr-agent reviewed all five PRs. CodeRabbit posted only limit or pause notices, so it reviewed none. Process findings were applied or declined, with reasons, in each PR's `## Review triage` comment. #2809's follow-up commit `164e22ac` (tests only) was not reviewed separately.

## Evidence index

- Red→green tests: `TestSync_RepairsADeployedConfigThatDoesNotParse`, `TestToolsSync_RepairsADeployedConfigThatDoesNotParse`, `TestLessonsStaleness/an_unreadable_lessons_directory_is_named…` and `TestMergedPRHeadsNamesAnOriginGHCannotAsk`. Each PR body or triage comment carries the command output.
- Live runs:
  - Real `mise ls` over a broken config exits 1, while the dev `dotf tools sync --dry-run` exits 0 and names the file.
  - The dev `dotf mem session-start` over a 000 `docs/lessons` prints the new line.
  - The dev `dotf worktree done` over a `gitea.kubelab.live` origin names the host.
- `dotf pr triage-queue --pr <N>` returned exit 0 before every land.

## Budget

| Limit | Used |
|---|---|
| max_items 7 | 5 (4 queue items plus the #2670 closure) |
| max_iterations 10 | 7 |
| 45 min per iteration | none exceeded |
| max_files 8 | the largest PR touched 8 (#2795); the 9th file went into #2805 |
| max_loc 300 | the largest was #2795, well under |
| max_consecutive_failures 2 | 0 |

## Host

- **Wake-lock:** `caffeinate -i -s -t 28800`, held from 18:33Z.
- **Pause, 19:08–20:31Z:** this session did no work for about 80 minutes because the claude.ai usage limit was hit. The background CI waits kept running, and nothing was lost.
- **GraphQL limit:** a secondary GraphQL rate limit hit between 19:06 and 19:17Z.

## Tickets filed this run

#2785, #2797, #2806 (and the note on #2773).
