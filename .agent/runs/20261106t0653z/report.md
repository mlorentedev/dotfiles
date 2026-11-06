# run autopilot-20261106t0653z: ended queue-empty after 6 items in about 4h. 9 PRs merged, 0 parked, 1 ticket filed, 1 issue reopened twice

**Is anything waiting on you, and is any of it irreversible?** Nothing irreversible. Every PR was merged with `dotf pr land`, and only after a `## Review triage` comment on its head. Two things need you, both listed below.

## Waiting on you
1. **#3172, the Defender quarantine of 0.79.0:** you still owe the Microsoft submission IDs. #3225 makes `install.ps1` name the quarantine, but the false positive itself is a vendor case.
2. **Acceptance on real Windows after the next release.** These issues stay open on purpose, because each fix was inferred or unit-tested, not watched on a guest:
   - #3142: aws counted as installed from the registered PATH;
   - #2432: the foreign-HOME doctor FAIL;
   - #2346: an isolated Windows sync with Git Bash scoped to bats;
   - #3172: the Defender warning on a guest.

## Merged
| PR | Item | Effect |
|---|---|---|
| #3216 | closes #3208 | VM step timeout prints the step's output and what it left running; step limit 2h |
| #3217 | closes #3214 | doctor finds bash on Windows from `git --exec-path`, not the PATH (lesson 558) |
| #3225 | refs #3172 | `install.ps1` names a Defender quarantine and the false-positive form |
| #3229 | refs #3142 | a system command in a registered PATH directory counts as installed |
| #3233 | refs #2445 | winget `0x8a15000f`: one `winget source update` per run, then one retry |
| #3234 | refs #2432 | doctor FAILs when a persisted user-scope HOME differs from USERPROFILE (also in `--quick`) |
| #3235 | refs #2346 | Git Bash goes on PATH only for the bats probe and the npm:bats install |
| #3239 | refs #2346 | a bats pin failure names a missing Git Bash and why (lesson 563) |
| #3241 | lesson 563 | a closing verb in prose closes the issue too |

## Found while working
- **#2445 was closed by mistake twice, and I reopened it both times.**
  - First, #3233's squash merge carried the first commit's `Closes #2445`, although the body had been narrowed to `Refs`.
  - Then #3239's body described that mistake in prose: "closed" followed by the issue number is itself a closing keyword.
  - Lesson 563 records both, and #3241 extends it with the second one. zoxide has no `packages.json` entry, so only the setup script's winget loop installs it, and that loop has no source refresh. I filed **#3238** to move the loop into the catalog. `setup-windows.ps1` is frozen for this run, and the repo rule is not to grow it. Lesson 563 records the closing-keyword trap.
- **CodeRabbit's free tier ran out on most PRs.** pr-agent reviewed every one, and each triage names who reviewed.

## Not taken
| Issue | Why |
|---|---|
| #1874 | the cause is unknown, it reproduces only in CI, and the fix is likely in frozen `.github/**` |
| #2438 | inside #3174, which another session owns |
| #1842 | `scripts/` and `.pre-commit-config.yaml` are outside `write_paths` |
| #1834 | needs Windows empirics (which quoting agy's runner accepts) |
| #2978 | review pipeline, a sibling run's slice |
| #1387, #3238 | the fix lives in frozen `setup-windows.ps1` / `packages.json` |

## Host
- **The Windows VM was not used.** The envelope's free predicate was never true: an empty `~/.local/state/dotfiles/vm/windows/run/` directory exists with no qemu running. The predicate is stricter than intended. Checking for a live session file would be enough.
- `caffeinate -i` (pid 78632) held the machine awake for the window.
- Six sibling runs shared one GraphQL budget, and it ran out twice. `gh api rate_limit` reported quota while `gh pr view` failed, so a real GraphQL query is the only honest probe. All reads and writes went through REST, and `dotf pr land` was retried after each reset.

## Evidence index
- Every PR carries a `## Review triage` comment on its landed head. `dotf pr triage-queue --pr <n>` returned `[OK]` before each land.
- Local gates on every Go change: `go build`, `go vet` (also with `GOOS=windows`), `go test ./...` and `golangci-lint run`, all green. CI's `test (windows-latest)` ran the Windows-only tests.
- The envelope was read from commit `4d32e99e` and never changed.

## Budget
6 of 8 iterations used. 7 seed and queue items were considered and 6 worked. About 4h of the 8h window.

## Next action
Owner: send the Defender submission IDs for #3172. After the next release, run the Windows acceptance for #3142, #2432 and #2346 on the VM.
