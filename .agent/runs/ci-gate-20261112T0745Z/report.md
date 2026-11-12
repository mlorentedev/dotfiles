# Run ci-gate-20261112T0745Z

Objective: close CI-018 (#3147). Envelope: `.agent/unattended.yml` at 6c5c5296.

## Done

- mlorentedev/ci-workflows#99 merged under that repository's grant (06fa706b): ADR-001 decision 5 links `docs/runbooks/guide-fleet-lint.md`. Reviewed by pr-agent; CodeRabbit rate-limited; the PR author triaged.
- Fleet adoption: `dotf lint fleet` on each default-branch tip, one issue per repository in that repository: kubelab#3354, iris#63, pdf-modifier#240, hive#581, pollex#130, ts-bridge#744, yt-metrics-cli#120, svqtriana.github.io#42, leaving-denver#281, web#659, mail2markdown#33, garsync#164, and two on the self-hosted forge. Index comment on ci-workflows#8.
- Filed #3384: `--fleet` reads each checkout's working tree, not its default branch (measured: 16 vs 14 and 51 vs 57 findings).
- CI-018 closing commit dd4009de on `chore/ci018-close`: AC1-AC5 ticked with evidence, features.json real commands, lesson 590 (`git archive` honours `export-ignore`). Spec review launched (nan/qwen3.8-flash, random draw).
- CI-017 (#3306) claimed with the two findings (name `ci-gate`; AC2 of the issue unreachable); spec `specs/CI-017-gate-required-check`; draft PR #3387 (owner lands).

## Held for the owner

- #3335, #3336, #3387 (`.github/**`).
- CI-017 step 2: protection flip, and whether `integration`/`test-macos`/`pi-nan-package` become blocking.
- #3328: deferred, the guests are claimed tonight by other sessions.
