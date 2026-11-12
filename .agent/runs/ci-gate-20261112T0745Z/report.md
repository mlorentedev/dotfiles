# Run ci-gate-20261112T0745Z

Objective: close CI-018 (#3147). Envelope: `.agent/unattended.yml` at 6c5c5296. Result: **objective met**, #3147 closed by #3391.

## Done

- mlorentedev/ci-workflows#99 merged under that repository's grant (06fa706b): ADR-001 decision 5 links `docs/runbooks/guide-fleet-lint.md`. Reviewed by pr-agent; CodeRabbit rate-limited; the PR author triaged.
- CI-018 closed and archived: #3391 merged (fd30d57c). Independent review nan/qwen3.8-flash, PASS WITH GAPS, seven Minor findings: six ticketed (#3388, #3389 P2, #3390), one applied. PR review: lesson number collision (590 -> 591), adoption counts reconciled, board line ticked; triaged on the PR. Lesson 591: `git archive` honours `export-ignore`.
- Fleet adoption: `dotf lint fleet` on each default-branch tip, one issue per repository in that repository: kubelab#3354, iris#63, pdf-modifier#240, hive#581, pollex#130, ts-bridge#744, yt-metrics-cli#120, svqtriana.github.io#42, leaving-denver#281, web#659, mail2markdown#33, garsync#164, and two on the self-hosted forge. Index comment on ci-workflows#8.
- Filed #3384: `--fleet` reads each checkout's working tree, not its default branch.
- CI-017 (#3306): claimed with two findings (name `ci-gate`; the issue's AC2 unreachable); spec `specs/CI-017-gate-required-check`; PR #3387, CI fully green including `ci-gate` (6929dd90), AC1-AC2 evidenced (108952bf). Left as draft: `gh pr ready` was denied by the permission classifier.

## Held for the owner

- #3335, #3336 (`.github/**`): still open.
- #3387: mark ready, review, land. Then step 2 (protection flip + `test-windows` job-level `if:`), deciding whether `integration`, `test-macos` and `pi-nan-package` become blocking.
- ADR-001 decision 3 names the check `gate`; dotfiles uses `ci-gate` because `spec-gate.yml` already reports `gate`.
- #3328: deferred, the guests were claimed tonight by other sessions.

## Notes

- Board fields not set on #3388-#3390: `dotf forge issue create --board` failed with `unknown owner type` and a GraphQL rate limit; the issues were created through REST and the repository's add-to-project workflow places them.
- A usage-limit pause stopped the run from about 08:50Z to 11:20Z.
