---
id: "BUG-026-setup-no-checkout-writes"
type: spec
status: archived
created: "2026-07-09"
tags: [spec, proposal, setup, idempotency, self-deploy, ci-guard]
template_version: "1.0"
---

# BUG-026-setup-no-checkout-writes

Stop setup from writing into the repo checkout, so the scheduled self-deploy
(`dotf update`) does not silently stop after the first run.

## Why

`setup-linux.sh` (and its Windows twin `setup-windows.ps1`) synced
`.github/copilot-instructions.md` from `ai/copilot/copilot-instructions.md` on
every run, writing the result **into the checkout** (`$CURRENT_DIR` /
`$DotfilesDir`). Two problems compounded:

1. **It dirties the checkout.** The committed `.github/copilot-instructions.md`
   is not byte-identical to what the sync regenerates, so every setup run left
   `git status --porcelain` non-empty. Empirically confirmed on this branch:
   the sync even used `printf '%s'` on a string containing literal `\n`, so it
   would have written backslash-n sequences and a different banner placement
   than the committed file — the "sync" produced a *broken* file.
2. **`dotf update` skips any dirty worktree — with exit 0.**
   `cli/internal/update/update.go` treats any `git status --porcelain` output as
   "dirty" and returns a benign skip (nil error). Net effect: a self-deploying
   machine runs setup once, the checkout goes dirty, and every subsequent
   scheduled `dotf update` skips forever while the systemd timer / Scheduled
   Task stays green. The machine silently stops receiving updates.

Root cause is a **misplaced invariant**. Parity between the two
copilot-instructions files was enforced in two places: a fail-loud test
(`tests/docs-drift.bats`, correct — blocks drift at merge) and a deploy-time
auto-rewrite (incorrect — mutates a committed file behind the user's back and
breaks idempotency). The auto-rewrite is redundant with the test and harmful.

Source: `docs/audits/process-audit-2026-07-07.md` §4 P1 (CONFIRMED, E3/E4);
GitHub issue #694.

## What

After this PR:

- `setup-linux.sh` no longer writes `.github/copilot-instructions.md`. The sync
  block is removed and replaced by a comment stating the invariant (*setup
  deploys to `$HOME` only; it MUST NEVER write into the checkout*) and pointing
  at `tests/docs-drift.bats` as the parity enforcement point.
- `setup-windows.ps1` gets the identical treatment for its parallel sync block
  (cross-OS parity of the fix).
- `tests/verify-setup.bats` gains a guard: after setup runs in the integration
  container, `git -C "$REPO_DIR" status --porcelain` MUST be empty. It asserts
  exactly the condition `update.go` checks, against the same checkout, and trips
  on *any* future checkout write — not just this one.
- `cli/internal/update/update.go` names the dirtying paths in its skip message
  (previously an opaque "dirty worktree — skipping"), so a silently-skipping
  scheduled run is diagnosable from its log alone. The unreadable-status branch
  is split out with its own fail-safe message.

## Out of scope

- The other fresh-machine setup bugs from the same audit (#695 in-place layout,
  #696 machine.json, #697 doctor verdicts). Separate issues, separate PRs.
- A pre-commit hook that *regenerates* `.github/copilot-instructions.md` from
  the SSOT. Not needed: `tests/docs-drift.bats` already fails loud on drift, and
  the two files are hand-synced on the rare edit. Adding a generator would
  re-duplicate the banner logic that caused this bug. Ticket if manual sync ever
  becomes a burden.
- Auditing every other setup write path. The new integration guard makes this
  empirical: if setup writes anywhere else in the checkout, CI goes red and the
  offending path is named.

## Risks / open questions

- **Guard depends on `.git` being present in the integration image.** There is
  no `.dockerignore`, so `COPY . /home/testuser/dotfiles-repo` includes `.git`;
  the checkout is owned by `testuser` and bats runs as `testuser`, so no
  `safe.directory` complaint. The guard `skip`s (does not fail) if `.git` is
  absent, so it degrades safely if the image layout ever changes.
- **Manual parity burden.** Editing `ai/copilot/copilot-instructions.md` now
  requires hand-updating `.github/copilot-instructions.md` or `docs-drift.bats`
  fails. This is the intended trade-off (fail-loud at merge over silent
  deploy-time rewrite).

## Acceptance criteria

- [ ] `setup-linux.sh` contains no write to `.github/copilot-instructions.md`
      (no `GH_COPILOT_DST` assignment, no `printf ... > "$GH_COPILOT_DST"`).
- [ ] `setup-windows.ps1` contains no write to `.github\copilot-instructions.md`
      (no `Copy-Item ... -Destination $ghCopilotDst`).
- [ ] `tests/verify-setup.bats` asserts an empty `git status --porcelain` in
      `$REPO_DIR` after setup (the #694 guard), skipping cleanly if `.git` is
      absent.
- [ ] `cli/internal/update/update.go` dirty-skip message includes the porcelain
      path list; the unreadable-status case has its own message.
- [ ] `cli/internal/update/update_test.go` asserts the dirty message names the
      offending path.
- [ ] `tests/docs-drift.bats` parity test still passes (invariant preserved,
      enforcement moved not deleted).
- [ ] `go build ./...`, `go vet`, `go test ./internal/update/` green;
      `bash -n setup-linux.sh` clean; `setup-windows.ps1` edit ASCII-only.

## References

- GH issue: [#694](https://github.com/mlorentedev/dotfiles/issues/694)
- Audit: `docs/audits/process-audit-2026-07-07.md` §4 P1
- Parity enforcement: `tests/docs-drift.bats` (SDD-005)
- Self-update contract: `cli/internal/update/update.go` (CLI-027 / AUDIT-007)
- Integration harness: `tests/Dockerfile.integration`, `tests/verify-setup.bats`
