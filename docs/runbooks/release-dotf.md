---
id: "dotfiles-runbook-release-dotf"
type: runbook
status: active
tags: [runbook, dotfiles, release, dotf, deploy]
created: "2026-09-25"
owner: manu
---

# Releasing and installing dotf

This runbook covers going from a merged release PR to a verified `dotf` on a machine.

- Run every step from the main checkout (`~/Projects/dotfiles`, on `main`, pulled). Never run them from a worktree whose branch predates main (HARNESS-153).
- Never edit `~/.dotfiles` by hand.

## When to cut a release

release-please keeps one release PR open and adds every merged change to it. Merging that PR cuts the release, so leaving it open is how changes accumulate, at no cost.

Merge the release PR when one of these holds:

1. **A next step needs the binary installed.** A merged change under `cli/` takes effect only once installed: a check that `dotf` runs, or a subcommand someone is about to use.
2. **A merged fix removes a live hazard that has no workaround.** The hazard must be one that sessions hit with the installed binary.

Otherwise, let it accumulate. Skill records, doctrine, CI, scripts and documentation need no release. They reach every agent through `dotf harness mirror` and `scripts/compile-harness.sh --deploy`, run from main.

While a fix waits, tell live peers about the hazard and its workaround. For #1725, that was a `--dry-run` before `dotf mem handoff-write`.

## Landing several PRs before a release

Merging ready PRs before the release PR puts them in one release instead of several. Branch protection requires a branch to be up to date with main, so each merge puts every other open PR behind. Land them one at a time:

1. **Wait for the merge state to settle.** Right after main moves, `mergeStateStatus` reads `UNKNOWN` for several seconds. Poll until it reads something else:

   ```bash
   gh pr view <N> --repo mlorentedev/dotfiles --json mergeStateStatus --jq .mergeStateStatus
   ```

2. **If it reads `BEHIND`, update the branch and read the new head:**

   ```bash
   gh api -X PUT repos/mlorentedev/dotfiles/pulls/<N>/update-branch -f expected_head_sha=<old sha>
   gh pr view <N> --repo mlorentedev/dotfiles --json headRefOid --jq .headRefOid
   ```

3. **Wait for every check on the new head.** Budget 10 to 15 minutes; the Windows test job is the slow one.
4. **Gate the merge.** Merge only when `dotf pr triage-queue` does not list the PR and every check on the new head passed. The update can bring new reviewer output, so read the queue after CI, not before. If the PR is listed, triage it first.
5. **Merge on the head you verified:**

   ```bash
   gh pr merge <N> --repo mlorentedev/dotfiles --squash --match-head-commit <new sha>
   ```

6. Go back to step 1 for the next PR.

Update only the PR you are about to merge. Updating all of them at once reruns every Windows job on each merge and saves no time. Lesson 324 records why steps 1 and 4 exist.

After the last merge, check the regenerated release PR before its merge:

- Its head sha changed. Re-read its checks for the new head, and look for `action_required` runs: a workflow run on a bot-authored push can wait for approval without any notice.
- Its CHANGELOG lists each `feat:` and `fix:` that landed. `chore:` commits are left out by design.
- Its triage record is newer than any reviewer output. CodeRabbit pauses on the release branch and PR-Agent skips release PRs, so a fresh `## Review triage` comment is usually all it needs.

## Steps

1. **Merge the release PR** (`chore(main): release X.Y.Z`). Manu merges it; an agent does not.
2. **Wait for the binaries.** The `cli` workflow's goreleaser job attaches them to the release; for 0.59.0 that took 5 minutes after the tag. Check with:

   ```bash
   gh release view vX.Y.Z --repo mlorentedev/dotfiles --json assets --jq '.assets | length'
   ```

3. **Announce to live peers.** List them, then send each one a message: mirror, deploy and binary swap are starting, so they should hold deploys and installs.
4. **Mirror the records before installing the binary that reads them** (ADR-038):

   ```bash
   git pull --ff-only && dotf harness mirror
   ```

5. **Deploy** to every harness target:

   ```bash
   scripts/compile-harness.sh --deploy
   ```

6. **Install** the version pinned in `versions.conf`:

   ```bash
   scripts/install-dotf.sh
   ```

7. **Verify by effect, not by the version string alone:**
   - `dotf version` prints `X.Y.Z`.
   - The prompt hook suggests no retired skill. Run it with a prompt that routed to a retired skill before the release (the one below is an example), and check that none of the skills it prints is on the retirement's list:

     ```bash
     printf '%s' '{"prompt":"rebase the branch on main"}' | dotf harness suggest --from-hook
     ```

   - After a skill retirement, `scripts/check-retired-skills.sh <retired names>` exits 0.
   - One feature from the release answers, for example `dotf mem handoff-write --help | grep -- --agent`.
   - `dotf doctor`: the only failures left are deploy-dir drift, which `setup` refreshes, and known zombie specs.
8. **Announce that it is done.**

## When it goes wrong

- **No assets after 15 minutes.** Run `gh run list --repo mlorentedev/dotfiles --workflow cli`. A failed goreleaser job leaves the tag without binaries. Fix the release; do not install a source build instead.
- **`install-dotf.sh` says the binary "drifted from pinned".** That is its normal convergence line, not an error.
- **The hook still suggests a retired skill.** Either the records were not mirrored (step 4), or the machine still runs an older binary (step 6).
