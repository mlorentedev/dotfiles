# Run npm20261111t0745z

**Verdict:** ended `budget-exhausted` after 2 guest runs / 5h04m (07:50Z to 12:54Z of a 12h window) — 0 done, 1 parked (#2438), 0 merged, 0 unmerged PRs. Waiting on a human: nothing irreversible; no PR was opened and nothing was merged (`merge: never`).

## Waiting on you
- **#2438 parked, two questions:**
  - Cut the release. The pins cannot land before one past 0.79.0 is the `DOTF_VERSION` pin (guard in `tests/versions-conf.bats`).
  - Decide opencode: #3329. If `opencode-ai` has no windows-arm64 binary, option (A) keep opencode on its current channel on Windows Arm64, or (B) drop Windows Arm64 for opencode. Reproduce first; the cause is not diagnosed.
- **PR B is bigger than "pins and deletions"** and part of it is outside what a run may touch: an ADR-036 amendment (`tests/versions-conf.bats` test 11 says opencode's pin lives in packages.json), `ci-npm-cli-cache.bats` test 6 with the CI npm cache step in `.github/workflows`, `copilot-config.bats` test 27, and `TestCheckVersionMatch` (`checks_test.go` reads the checkout's packages.json instead of its fixture). Listed in `specs/PLAT-001c-tools-via-mise/tasks.md`.

## Parked
- **#2438** — reason: two defects found on the clean guest. Options: (A) fix #3328 (Git Bash lookup after a same-process git install) and #3329, then re-measure; (B) take opencode out of the move. Rung reached: pins + deletions + vcredist + doctor fallback built and measured; mise, node, bw, copilot, yarn ran on the guest. Cheapest unblock: reproduce opencode in a named session (`dotf vm windows up --session s`, then `exec`).
- Board: #2438 set to Blocked (exit 0).

## Defects found while working (ticketed, not worked)
- #3328 — a from-zero Windows converge cannot find Git Bash for bats because git was installed earlier in the same process. Root cause known; two fix options on the ticket.
- #3329 — opencode through mise's npm backend does not run at its pin on a clean Arm64 guest. Cause not diagnosed.

## Done but unmerged
Nothing opened. The branch `feat/mise-npm-pins` (5fb410a5) is a measurement branch, not a PR.

## Evidence index
- Guest run 1: `dotf vm windows --ref feat/mise-npm-pins --dotf <arm64 build> --keep`, exit 1: `mise-v2026.9.13-windows-arm64.exe does not run on windows/arm64` (no vcredist entry). Log in the session scratchpad (`vm-run1.log`).
- Guest run 2: same command, exit 1: `after mise install, not running at their pin: bats, opencode`; `3 changed, 0 ok, 18 skipped, 1 failed (after 58 changes)`. Log `vm-run2.log`.
- Go: `go test ./...` ran once on the pins branch: only `TestCheckVersionMatch` failed (named above). bats: `versions-conf.bats` tests 11 and 25 fail by design.
- Advisor: the native advisor was rate-limited at arming; `dotf agent auto --role reviewer --tier top` answered twice (envelope review; the stop decision).

## Budget spent
2 iterations of 6, 5h04m of 12h, 1 item of 3. No dollar figure exists: the provider exposes none to this session.
A usage-limit pause fell inside the window; run 1's log was read only after it, so the 07:50Z-12:01Z timeline is reconstructed from file times, not observed.

## Host-suspend line
caffeinate held a 12h lock until I stopped it at 12:54Z. Both guest runs completed (55 and 45 min), which a sleeping host would not have done; no suspend was observed.

## The one next action
Cut the release; then fix #3328, diagnose #3329 in a named guest session, and re-run the measurement.
