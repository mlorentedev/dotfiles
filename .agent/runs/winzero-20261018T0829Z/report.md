# Run report — winzero-20261018T0829Z

**Verdict:** run winzero-20261018T0829Z ended queue-empty at 16:11Z after 8 items / 7h42m of window (about 3h40m of it lost to two usage-limit pauses) — 8 done, 1 parked, 7 merged by this run, 0 unmerged.
The objective is met: a clean Windows 11 guest converged from zero and the second converge changed nothing (PLAT-003 AC9, recorded in `specs/PLAT-003-cross-os-verification/verification.md` and on #2344). Nothing irreversible waits on you beyond the merges below, which are already on main.

## Waiting on you — irreversible first

1. **Already on main (landed by this run under the standing grant):** #2770, #2775, #2780, #2782, #2783, #2781, #2772 — see "Done" below. None touches `.github/**`, `.agent/**`, release config or secret configs.
2. **Cut the release** (release-please PR). Every fix above reaches a real Windows box only through it: the guest converged with a `--dotf` build of the tree, not with a release.
3. **#2763 part 2 — pick A or B** (parked, board Blocked, comment issuecomment-6248969937):
   - (A, recommended) prove the effective sshd config with an `sshd -T` allowlist, so an ignored or overridden setting fails the plan;
   - (B) extend the ssh probe to require that password and keyboard-interactive logins are refused.
4. **Runbook edit outside this run's envelope:** `docs/runbooks/guide-workstation-ssh-target.md:361` still says `powercfg /q`; #2780 changed the code to `/qh` (a lidless machine hides LIDACTION from `/q`). One-word edit; `docs/runbooks/**` was not in `write_paths`.
5. **After the release:** PLAT-003 AC8 (`dotf vm linux`), as the feat-cross-os-verification thread says.
6. **Disk:** `~/.local/state/dotfiles/vm/windows` holds the 33 GB sealed `clean-base` plus the ISOs. Keep it for the next `dotf vm windows` run, or remove it.
7. **Worktrees to remove** (this session cannot remove its own launch worktree): `feat-vm-windows` (launch), `chore-autopilot-winzero-20261018` (this run branch; never merge it).

## Parked

- **#2763 part 2** (`sshd -T` proof). WHY: proving which settings sshd actually applies is a design choice between two mechanisms with different coverage; the run does not choose security policy unattended. Part 1 (only the relative `Include` Win32-OpenSSH reads) landed in #2783.

## Done (merged by this run)

| PR | Closes | What |
|---|---|---|
| #2770 `5e3f383d` | #2754, #2734 | winget's registered PATH reaches a portable package (uv, git) where symlinks are not allowed; checkout reaches git before the fast-forward. Lesson 488 |
| #2775 `81096247` | #2769 | the Python pin is read at install time and only a non-empty answer is cached. Lesson 487 |
| #2780 `ed4ea519` | #2764 | `powercfg /qh` reads the hidden lid setting on a lidless machine |
| #2782 `9a3fe8d8` | #2765 | a Windows firewall refusal surfaces at plan time; a failed apply claims no work it did not do |
| #2783 `95daf08a` | #2763 (part 1, Refs) | only the relative `Include sshd_config.d/*.conf` that Win32-OpenSSH reads is accepted |
| #2781 `291cc5ad` | #2758 | a missing `sshd_config` is explained (sshd writes it at first start) instead of a raw read error |
| #2772 `a436d19c` | PLAT-003 PR 3c, AC9 | `dotf vm windows --dotf <build>` converges with a source build; refuses a non-Arm64 or non-executable build before boot; a step whose dotf differs from the build fails. Lesson 489 |

#2771 was merged by the owner at 08:33Z, not by this run.

## Disclosures

- **Self-triage on every PR.** pr-agent was the only reviewer that produced reviews; CodeRabbit posted only its rate-limit notice on each. Each `## Review triage` comment says so.
- **Guest commands were denied** by the auto-mode classifier after 10:50Z. #2780's `/qh` and #2772's hash check were therefore not re-run in the guest: `/qh` rests on the earlier guest measurement (a measurement build of main with `/qh` worked), the hash check on its unit test and two killed mutations.
- **A unit test booted the real guest twice, by accident (about 15:45Z).** The red phase of a new `--dotf` refusal test (an Arm64 PE that is no executable image) ran the command past the missing check. On this Mac it found the sealed `clean-base`, booted a throwaway overlay, copied the few-byte fixture in, and failed in `install.ps1` after about 85 s. QEMU exited and the overlay was removed (checked: no `qemu-system` process). No credential or host directory reaches the guest, by construction. The refusal tests now point dotf's state at an empty directory, so a regression fails in under a second (#2772, lesson 489).
- **Two usage-limit pauses** (08:29–10:31Z and about 11:50–15:30Z). The wake-lock held; no work ran in the gaps.

## Evidence index

- Ledger: `.agent/runs/winzero-20261018T0829Z/ledger.md` (this branch).
- AC9: #2344 issuecomment-6249489992; `specs/PLAT-003-cross-os-verification/verification.md` (AC9 bullet).
- Park: #2763 issuecomment-6248969937.
- Triage comments: one `## Review triage` per PR above (#2781 issuecomment-6251759764, #2772 issuecomment-6252112236).
- Exit: `dotf pr triage-queue` → `[OK] no reviewer output is awaiting a disposition`, exit 0 (16:11Z).

## Budget

8 items worked, at `max_items` 8: #2770, #2775, #2772, #2734 (repro, closed by #2770), #2764, #2758, #2765, #2763. The queue's ninth, #2771, the owner merged. 7 PRs landed, 0 consecutive failures, no fix-loop breaker tripped. No envelope widening.

## Host

Mac on power, lid open; `caffeinate -dims -t 28800` held the whole window. The host did not suspend.

## Next action

Merge the release PR, then run `go run ./cmd/dotf vm windows` (no `--dotf`) from a worktree's `cli/` to confirm AC9 with the released dotf.
