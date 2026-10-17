# Run eve-20261017T1830Z

**Verdict:** ended `queue-empty` at about 20:22Z, after 3 iterations and about 1h50 of the 10h window. Result: 3 worked, 2 parked, 2 merged, 0 unmerged.

**Waiting on you:** two parks, each a question with two options (section 3). Nothing irreversible is pending beyond the two merged PRs below, both test-only or warn-only.

## 1. Landed (irreversible)

- **#2707** `test(pi): require pi on every deploy entry under ~/.pi`. Merged as `079d7fe` (head `26a53af`). Refs #2102 (the Linux half).
  - Adds a `tests/pi-config.bats` guard: every `ai/deploy.json` entry whose dst is under `{HOME}/.pi/` must say `requires: pi`. It replaces a stale comment that claimed pi-compaction was deliberately ungated.
  - Reviewed by pr-agent ("No major issues detected"). Triage: https://github.com/mlorentedev/dotfiles/pull/2707#issuecomment-6239752798
- **#2723** `fix(harness): warn when a deploy's checkout lacks main's last harness change`. Merged as `84bc390` (head `7ccdadb`). Closes #1712.
  - `compile-harness.sh --deploy` now warns, naming the commit and subject, when the checkout lacks the last `harness/` change on `origin/main`; it still deploys, and skips the check where it cannot judge (no git, no origin/main). New `tests/compile-harness-deploy-base.bats`.
  - Reviewed by pr-agent ("Fully compliant" with #1712). Two findings, both declined with references: `type -P` is the file's established bash form, and the `## Knowledge` section was present. Triage: https://github.com/mlorentedev/dotfiles/pull/2723#issuecomment-6240380464

## 2. Waiting on your merge

None.

## 3. Parked

| Issue | Why | (A) | (B) |
|---|---|---|---|
| #2102 Windows half | Needs a live Windows check. `setup-windows.ps1:819` creates `~/.pi/agent` and `:859` copies `AGENTS.md`, both ungated on pi. | Delete both lines and rely on the harness manifest deploy (`harness/manifest.json:188`), as Linux already does. Recommended; verify on the work-lab box. | Gate both on `Get-Command pi`. Keeps the twin, smaller change. |
| #2462 TOOL-028 | Needs a spec slot. WIP is 13/10, and no `--over-wip-limit` was granted. | Grant a slot or the override for TOOL-028 and run it as specified. | Split off a doctor WARN only, under 50 LOC, that excludes mise's own directories (shims plus `installs/` would otherwise warn on all 39 pins). |

Both are `Blocked` on the board; the comments are https://github.com/mlorentedev/dotfiles/issues/2102#issuecomment-6238567755 and https://github.com/mlorentedev/dotfiles/issues/2462#issuecomment-6238574069.

## 4. Evidence index

- #2707: `bats tests/pi-config.bats` 24/24 ok. A mutation removing `requires` from pi-compaction turned the new test red ("deployed on a box without pi: pi-compaction"). CI green on `26a53af0`. `dotf pr triage-queue --pr 2707` exit 0, `dotf pr land 2707` exit 0 (`[MERGED] #2707 at 26a53af`).
- #2723: `bats tests/compile-harness-deploy-base.bats` 6/6 ok after rebasing on main (`7ccdadb7`); test 1 was red before the fix. The compile-harness bats files: 483 ok, 1 pre-existing failure (section 5). `shellcheck -S warning` clean; `bash -n` passes under bash 5 and `/bin/bash` 3.2. CI on `7ccdadb7`: 21 success, 6 skipped. `dotf pr triage-queue --pr 2723` exit 0, `dotf pr land 2723` exit 0 (`[MERGED] #2723 at 7ccdadb`).

## 5. Things you should know

- **review-attestation lagged pr-agent by about two minutes on #2707.** The status was still the failure set at 19:24:57Z, before pr-agent reviewed at 19:36Z, so the first `dotf pr land` was refused (`review-attestation: fail`). Its re-evaluation runs sat in the queue on main behind other sessions' runs and turned green at 19:41:20Z; the second land succeeded. Nothing to fix: the gate failed closed and recovered without intervention.
- **The GraphQL user-level limit** blocked `gh pr ready 2707` from about 18:44Z to 19:21Z, although `gh api rate_limit` showed headroom. REST kept working; a background retry marked it ready.
- **#1712 warns rather than refuses.** Refusing a stale deploy would block every other session's deploy with no override in place; the warning names the missing commit and its subject.
- **A pre-existing red test on main is not this run's:** `harness-doctrine-records.bats`, "--refresh against the real vault leaves the working tree's records unchanged". It waits on the peer's #2664.
- **The repo-wide `dotf pr triage-queue`** lists only peer PRs (#2716, `feat/sec-014-target-record`), not this run's.
- **The orchestrator dotfiles-20 was messaged at the end**, best-effort: cross-session messages need approval and can expire silently.

## 6. Budget

- 3 of 6 iterations, 3 of 3 queue items, about 1h50 of the 10h window. No dollar figure exists for this run.

## 7. Host suspend

- The first `caffeinate -i` (task b8ecg2vk7) died with exit 144 at about 19:07Z. It was restarted at once as `caffeinate -i -t 33000` and holds until about 04:20Z, which covers the window. The timestamps show no gap in the work.

## 8. One next action

- Answer the #2102 park (A or B) at the work-lab Windows box; then decide the WIP question for #2462.
