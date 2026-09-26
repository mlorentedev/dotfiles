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
