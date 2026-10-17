# Run gh-tools-20261017T1822Z

**run gh-tools-20261017T1822Z — ended queue-empty after 3 iterations / about 2h05 (18:22Z to 20:27Z) — 3 done, 1 parked half, 3 merged, 0 unmerged.**

What waits on a human: the Windows half of #2136 (one decision, A or B below) and the release that carries #2721, which PLAT-003 AC8 needs. Everything landed is reversible with a revert. Nothing was released or published.

## Waiting on you — irreversible first

- **Landed under the standing grant**, each with a `## Review triage` comment, CI green on its head, and `dotf pr triage-queue --pr N` at exit 0:
  - #2708 → `f178fc62` (#2136, the Linux/macOS half)
  - #2721 → `dc1e7d9b` (#2688)
  - #2725 → `39a3289c` (#2320)
- **Release.** #2721 reaches installed boxes only through a release. PLAT-003 AC8 (`dotf vm linux`) waits for it; noted on #2344 (issuecomment-6240269109).
- **Envelope widening.** The owner added #2688 to the run. dotfiles-20 committed it at `f5c5635f`, because this session's own attempt was refused as self-modification. The committed file matches the patch byte for byte, and `ssh`/`scp` joined `allowed_commands` in that commit.

## Parked, with the question restated

- **#2136, the Windows half.** Board: Blocked (exit 0). Options are on the issue (issuecomment-6239500812):
  - **(A), recommended:** retire `setup-windows.ps1`'s winget row in the Windows-empirical batch (PLAT-001c P5b), once mise is on there.
  - **(B):** keep a winget `system` entry for gh on Windows only. This needs the one-owner guard in `tests/packages-json.bats` to accept a per-OS owner.

  Cheapest unblock: choose A or B. Either way it needs the work-lab box. #2136 stays open on purpose; #2708 was `Refs`, not `Closes`.

## Done but unmerged

None.

## Evidence index

| Item | Verification | Exit |
|---|---|---|
| #2136 / #2708 | `go test ./...`, `golangci-lint run` (0 issues), `GOOS=windows go vet ./...`; bats: 27 files, 573 tests; mutations 2/2 killed | 0 |
| | Mac `dotf converge --only tools`: plan → apply → re-plan, 0 changes. The mise config diff is only `gh = "2.102.0"` plus `GH_NO_UPDATE_NOTIFIER`. `mise exec -- gh auth status` | 0 |
| #2688 / #2721 | Go tests and lint as above; mutations 5/5 killed | 0 |
| | msi `env PATH=/usr/bin:/bin ~/.local/bin/dotf converge --only records-harness` (installed 0.69.0, the control) | 1 (FAIL, as in the issue) |
| | msi, the same tree without the fix (counter-control) | 1 |
| | msi `env PATH=/usr/bin:/bin /tmp/dotf-2688/dotf converge --only records-harness`, run twice: `[ OK ] records-harness 0 changed` | 0, 0 |
| #2320 / #2725 | `go build/vet/test ./...`, `golangci-lint run ./...` (0 issues), `GOOS=windows go vet ./...`; mutations 7/7 killed; read-only doctor on the Mac and msi (nothing to report on either; `--fix` not run) | 0 |

The full outputs are in each PR body.

## Deviations and gaps, disclosed

- **The binary path on msi.** The owner said `/tmp/dotf-2688`. The child steps look `dotf` up *by name*, so a binary named `dotf-2688` cannot prove the fix. I used `/tmp/dotf-2688/dotf` instead, and the fix gained a name guard (lesson 475). The temporary binaries were removed afterwards.
- **One reviewer per PR.** On all three PRs CodeRabbit posted only "Review limit reached" notices, so pr-agent was the one reviewer (claude-haiku-5-5 on #2708, glm5.3-flash on #2725).
- **GraphQL secondary rate limit.** `gh pr ready` (GraphQL-only) was refused for about 30 minutes while the REST counter showed 4994 remaining. Retries spaced 20 to 30 minutes apart got through at 19:19Z.
- **A stale attestation status.** `dotf pr land 2708` first refused on `review-attestation: fail`, because that check had run before the review existed. It re-evaluated to attested at 19:33Z, and the land then went through.
- **Wake-lock gap.** The first `caffeinate` (`b3402w8ym`) died with exit 144 at about 19:02Z. It was relaunched within minutes (`caffeinate -i -t 33500`). The host did not sleep: work continued across the gap.
- **Peer overlap.** #2716 (a peer's, open) touches `cli/internal/cmd/converge.go`, as #2721 did. #2721 merged with a clean state. The peer rebases.
- **Lane order.** `dotf lane plan` lists the drafts #2681 and #2699 and other sessions' PRs (#2683, #2716, #2717) ahead of this run's PRs. None of them was ready to land under this run, and none is this session's.
- **This report's push** goes through the skip-sdd escape: the run branch carries only `.agent/` (see #2698).

## Knowledge

- Lesson 474: a baseline run from the change's worktree measures the change, not the baseline (`convergeCheckout` resolves the repo from the cwd first).
- Lesson 475: a fix found by name is proven only by a binary of that name.
- ADR: none.
- Runbook: `docs/runbooks/tool-installation.md` (the gh example now uses eza; "Copies left in `~/.local/bin`").

## Debt filed

None new. The one real review finding (Windows coverage, #2708) is held by #2136's park.
