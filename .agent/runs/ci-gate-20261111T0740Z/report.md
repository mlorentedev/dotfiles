# Run ci-gate-20261111T0740Z

**Verdict:** ended `window-expired` at 19:22Z of a 07:42Z–19:42Z window. Merged under the standing grant: #3319, #3323, #3330, #3332 (`dotf gate knowledge|spec` and the advisory adjacency report), #3334, #3350, #3351 (`dotf lint fleet`: eight rules, exceptions, `--fleet`, runbook). #3304 landed at the start; #3305 merged by the owner. Nothing left open by this run.

## Waiting on you
- **#3336:** dotfiles' own fleet findings. Four remain after #3305: `repo-hygiene.yml` needs `concurrency` (a fix), and `ci.yml` `test-windows` plus the two `skip-cache` steps in `cli.yml` (#2834) need entries in `.github/lint-exceptions.yml`. All `.github/**`, so yours to land.
- **#3335:** 22 checkouts keep the job token (`persist-credentials`), also `.github/**`.
- **#3306 (CI-017):** its first draft PR was not opened this run; the window went to #3147 and to a CI outage.

## Not finished from the objective
- **#3147:** the link from ci-workflows ADR-001 to `docs/runbooks/guide-fleet-lint.md` (a PR in that repository), and the spec's closing checklist, review and archive. The code is complete.
- **#3306:** not started.

## Defects found while working (ticketed)
- #3335, #3336 (above). #1084 got the recurrence of today's mirror outage.

## Review findings applied
- #3350: GitHub filter syntax (`?`, `+`, `[...]`, `\`) was read as a shell glob, so `!ma[a-z]n` did not exclude `main`; nested matrix paths and `exclude:`; booleans in any spelling (`False`); the git lookup moved behind an injected seam (`cli/AGENTS.md`).
- #3351: symlinked checkouts were skipped silently; `--fleet`'s hint for an unknown default branch named a flag `--fleet` refuses.
- Declined with reasons on the PRs: checkout matched by name (#3334), the loud exit without `origin/HEAD` (#3350), map order of stale exceptions (#3351, `Lint` sorts).

## Evidence index
- Every PR: `go test ./...`, `go vet` and `golangci-lint` on Linux and Windows (`GOOS=windows`), clean; mutations of each new branch red. `## Review triage` on every PR; `dotf pr triage-queue --pr N` exit 0 before each `dotf pr land`.
- `dotf lint fleet --fleet ~/Projects`: 221 findings in 13 repositories, none unreadable.
- CI outage: the Azure Ubuntu mirror failed `apt-get` (`Install zsh`, the integration image build) from about 13:50Z to 17:20Z on every branch; #3334 was rerun once, after main's CI passed again.

## Budget spent
11h40m of 12h. CodeRabbit was rate-limited on three of four PRs; PR-Agent reviewed all of them.

## Host-suspend line
caffeinate held the lock for the whole window (a 10h lock, then a 90 min one); no suspend observed.

## The one next action
Land #3336 and #3335 (owner), then open the ci-workflows PR linking ADR-001 to the runbook, and run CI-018's closing review and archive.
