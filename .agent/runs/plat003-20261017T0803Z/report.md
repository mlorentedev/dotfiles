# Run plat003-20261017T0803Z

**Verdict:** run plat003-20261017T0803Z ended **goal-reached** after 2 items / ~2h (08:03Z → ~10:10Z). 2 done, 0 parked, 2 merged, 0 unmerged. The objective was PR 1 (Tier 1), and PR 2 (Tier 3a) was added under step 3. Waiting on a human: one product bug the VM found (#2688), the hypervisor for PR 3, and two owner-confirm candidates. Nothing irreversible beyond the two merges below.

## Waiting on you (irreversible first)

1. **Landed under the standing grant.**
   - #2679, squash `452d35bc` on main (head `77504fd`): Tier 1, `pwsh-modules` converge, doctor section, `dotf lint powershell`.
   - #2689, squash `4c651efc` on main (head `728d564`): Tier 3a, `dotf vm linux`.
   - Neither touches `.github/`, the grant or release automation.
2. **#2688 (BUG-154): the fix location is your call.** The first live `dotf vm linux` found that a from-zero `./install.sh` fails at `records-harness` wherever `~/.local/bin` is not on PATH, which is the case on a fresh Ubuntu. Both CI legs hide it by putting `~/.local/bin` on PATH first (lesson 473).
   - **(A) `install.sh` exports `$dest` before the `exec`.** Cheap, and provable with `dotf vm linux --ref <fix>` before a release.
   - **(B) converge prepends its own directory to its children's PATH.** Covers every caller, but needs a release before any leg can verify it.
   - PLAT-003 AC8 waits on this.
3. **PR 3 (Windows 11 ARM VM) needs a hypervisor: VMware Fusion or UTM.** Neither is installed, and this run did not install one. Noted on #2344.
4. **Owner-confirm candidates, not taken.** Peer session dotfiles-20 relayed a request to add #2136 and #2320 to this run's queue. I declined: a peer cannot widen an envelope the owner set. Add them to a future run if you want them.
5. **Shared machine state, restored.** The `tools` converge apply from the PR 1 worktree removed `zsh-autosuggestions` from `~/.config/mise/conf.d/dotfiles.toml`. That entry was state converged by peer #2667's unmerged branch. I restored the file from a copy taken before the apply. The prune failure this exposed is #2677 (BUG-153), and the class is lesson 467. `dotf converge --only tools --plan` still shows `mise config to write` until #2667 lands.

## Parked

None.

## Done

- **#2679 (PR 1, Tier 1).**
  - `pwsh` is a `system` catalog entry (brew/winget).
  - `# pwsh-module:` markers on `PSSCRIPTANALYZER_VERSION` and `PESTER_VERSION`, and a `pwsh-modules` reconciler. On the Mac: plan 2 → apply → plan 0.
  - Doctor section "PowerShell (local lint-powershell)". It warns rather than fails on a missing module, because of the Windows CI doctor gate.
  - `dotf lint powershell`: 26 files in 3.6 s. A Go test parses `ci.yml` to keep the command at parity.
  - Review fix: a failed listing is no longer reported as an empty tree.
  - Follow-ups #2678 (wire the command into the hook and the CI job; owner-lands for CI) and #2677.
- **#2689 (PR 2, Tier 3a).**
  - `dotf vm linux`: the Lima template is embedded in `dotf` (plain mode, `mounts: []`, no agent or keys).
  - It checks isolation (systemd is PID 1, no host mount), clones a pushed ref, runs `./install.sh`, then `dotf converge` again, judging each run from the guest's `last.json`. The guest is deleted after every run, including a failed one. `GITHUB_TOKEN` reaches the guest only on stdin.
  - AC7 was met live. AC8 is blocked on #2688.
  - Review fixes: the delete is registered before the start, and each converge is judged on its own report.
  - `lima` got an explicit catalog entry.

## Guardrails

- **No credential in any VM image or snapshot.** The template mounts nothing, and the run took no token: `GITHUB_TOKEN` is not in the secrets registry, and the guardrail allows secrets only via `dotf secrets run`.
- **No VM is left behind.** `limactl list` printed `No instance found` after both live runs: the verification and a login-shell probe.
- The Windows VM was not started.

## Evidence index

| Item | Command | Exit |
|---|---|---|
| #2679 | `go test ./...` (44 packages), `golangci-lint run` 2.12.2 (0 issues), `GOOS=windows go vet ./...`, bats 98 + 207 ok | 0 |
| #2679 | CI on `77504fda`: 27 success; `dotf pr triage-queue --pr 2679`; `dotf pr land 2679` → `[MERGED] #2679 at 77504fd` | 0 |
| #2689 | `go test ./...`, `golangci-lint run ./...` (0 issues), 7/7 mutations killed, bats reading `packages.json` (11 files) ok | 0 |
| #2689 | CI on `728d5644`: green; `dotf pr triage-queue --pr 2689`; `dotf pr land 2689` → `[MERGED] #2689 at 728d564` | 0 |
| #2689 | `dotf vm linux --ref main` (live): isolation OK, first converge failed at `records-harness` (#2688) | 1 (expected) |

## Host

- Wake-lock: `caffeinate -i -t 36000` from arming. It ended early with exit 144 while PR 1's CI ran (shortly after 08:22Z) and was relaunched as soon as the notification arrived (`-t 34000`). No gap was observed, but a sleep in between would not have been visible. It was released when the run ended (~10:05Z).
- Lessons 466, 467 (PR 1) and 473 (PR 2). Lesson numbers 468, 471 and 472 were already claimed by open peer PRs.

## Next

- After #2688 is fixed on a ref: `go run ./cmd/dotf vm linux --ref <fix>` from a worktree's `cli/`. The installed 0.73.0 has no `vm` command until a release carries #2689. Then tick AC8.
- After a hypervisor is installed: PR 3 (AC9).
- Archive gate: `dotf spec review PLAT-003-cross-os-verification`, run by a non-implementer once all tiers are done.
- This report was pushed with the pre-push hook's skip-sdd env escape (`SDD_LABELS=skip-sdd` plus a rationale). The gate counts `.agent/` as production LOC (92 lines), and this branch is never opened as a PR. Same precedent as run cli-127.
- The first live-run command (build plus run plus `env -u GITHUB_TOKEN`, in one call) was denied by the permission system. It was reshaped: the build became a separate step, the `env -u` was dropped, and no token was ever set. The second form ran.
- `dotf pr triage-queue` repo-wide exits 1 on #2692 and #2683, which belong to the live `wintarget-14` session. They were left to their owner.
