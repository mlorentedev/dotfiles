---
id: lesson-328
type: lesson
status: active
created: "2026-10-01"
owner: manu
tags: [lesson, setup, idempotence, integration, ci, agy]
---

# 328 — An idempotence test presumes the first run converged

## What happened

`POLISH-005` in `tests/verify-setup.bats` hashes every deployed config after the image's setup run, runs setup again, and asserts the two lists match. It failed on three unrelated PRs (#1703, #1870, #1850) and passed on re-run each time. In the two failures with a printed diff, the only change was `~/.gemini/config/hooks.json`, present only after the second run (#1704).

The file is idempotent. The attempt-1 job logs of both failures show the cause one step earlier. During the image build, the vendor installer (`curl … antigravity.google/cli/install.sh | bash 2>/dev/null`) failed within 0.2 s. Setup logged `agy install failed`, continued, and `dotf harness bind` reported `skip agy: agy is not installed`, which is correct for a missing binary. The test's second run retried the install, succeeded, and bound agy, so it wrote the file. A repair was read as drift.

The re-runs hid this. A re-run replaces the job log GitHub shows by default, so the failing attempt's log has to be fetched by job id (`gh api repos/<r>/actions/runs/<id>/attempts/1/jobs`).

## Rule

- Idempotence is defined on a converged first run. A setup step that warns and continues makes convergence optional, so an idempotence test needs a precondition that the first run actually reached the end state. Without it, a transient outage looks like a code defect.
- An integration image installs tools that come from an external, unpinned endpoint in its own layer, with a retry and a check that the binary exists. The repository already did this for bats. A vendor outage then fails the build at a step that names the vendor.
- When a flaky failure passes on re-run, read the failing attempt's log before theorising. The default view of the run shows only the attempt that passed.

## Guard

`tests/Dockerfile.integration` installs agy before setup, retrying five times, and fails the build if `~/.local/bin/agy` is absent. `POLISH-005` fails with a message that names a non-converged first run before it takes any snapshot.
