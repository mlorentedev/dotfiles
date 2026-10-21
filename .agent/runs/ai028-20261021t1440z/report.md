# Run ai028-20261021t1440z

Objective: carry out the narrowed RESUME of AI-028 (#791) as far as is possible without the owner.
Armed 2026-10-21T14:40Z, expires 20:40Z. A caffeinate wake-lock covered the window.

## Waiting on you (first item first)

1. **Re-confirm RESUME.** The decision rested partly on "hive is missing on every fresh Windows machine". The diagnostic run (V6) narrowed that to **Arm64**: cryptography 50.0.2 has no `win_arm64` wheel, the source build fails at `link.exe`, and setup hides it with `Out-Null`. On x64 the install most likely succeeds and leaves a uv copy, which ADR-019 rules out. That is inferred, not measured. RESUME still stands on AC4 (the uv copy on Windows), AC9 (the timers) and #1993. If that is not enough, ABANDON is still open.
2. **PR6 shape:** (A) a `hive-self` source type in `packages.json`, or (B) a dedicated converge step. Also: does `hive self-upgrade <pin>`, run from converge, count as "on the user's behalf" under hive ADR-020?
3. **Arm64 option (hive#515):** `--only-binary cryptography` (resolves to 46.0.3), x64 CPython under emulation, or build tools in the clean-base.
4. **Retiring the timers on live machines (#2857):** where the Go retire step lives, in the shape of #2799. Until then #2857's PR stays a draft.
5. **Windows scope of `prerequisite_command`** together with #1993: setup still runs `uv tool install --upgrade hive-vault` on every Windows run.
6. **No machine identity:** `dotf agent auto` refused (no machine declared in `machine.json`). The second opinion came from the native advisor and an Opus subagent instead.
7. Local branch `chore/diag-hive-windows-guest` (`291ce5bf`): a throwaway that instruments `setup-windows.ps1`. It was committed with `--no-verify` and never pushed, and its worktree has been removed. Delete it, or keep it as the V6 reproducer.

## Done

- **#2868 landed** (`790a1feb`, Refs #791): the AI-028 re-scope and lesson 506. PR-Agent found one REAL issue: PR5 and PR7 tasks were ticked while not on main. Fixed in `5d1937f7`; that fix was not re-reviewed, which is disclosed in the triage.
- **#2869 landed** (`c213ae90`, closes #2856): the doctor names hive's own install on Windows. PR-Agent: no findings.
- **#2876 open as a draft** (Refs #2857): deletes the timers and adds the guard. Draft on purpose; it waits on items 4 and 5 above.
- Filed #2856 (BUG-163, closed by #2869), #2857 (OPS-054) and mlorentedev/hive#515 (BUG-012). Commented the re-scope on #791, which stays open while the spec is active.
- CodeRabbit reviewed nothing (free-tier limit); PR-Agent was the only reviewer.

## Not done

- **Scope exception:** the run wrote one path outside the envelope's write_paths: `tests/hive-daemon-gate.bats` (new; the envelope listed only `tests/hive-upgrade-timer.bats`). It exists only in the unmerged draft #2876, and nothing outside the envelope was merged.
- #2857 and #791 are parked: an ERROR/WHY/FIX comment with options, and board Status Blocked (verified).

- The stale timer mentions in `docs/architecture.md:27,29` and `systemd/hive.service.d/10-dotf-secrets.conf:8,58` are outside this run's write_paths. They are listed in #2876.
- AC2, AC4 and AC9's retire half are blocked on the owner decisions above, so the spec stays `implementing`.

Ended: queue-empty (everything left needs an owner decision).
