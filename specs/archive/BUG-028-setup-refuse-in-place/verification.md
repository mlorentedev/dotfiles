---
id: "BUG-028-setup-refuse-in-place"
type: spec
status: implementing
created: "2026-07-09"
tags: [spec, verification]
template_version: "1.0"
---

# Verification — BUG-028-setup-refuse-in-place

## Reproduction (pre-fix, #695)

Cloning into `~/.dotfiles` and running setup: the env-contract `cp -f X X`
aborts under `set -euo pipefail`, and `deploy_git_hooks` empties the dispatcher
while logging `[SUCCESS]`.

## Automated evidence (this branch)

| Check | Command | Result |
|---|---|---|
| Preflight smoke | minimal `$HOME/.dotfiles` fixture, `bash setup-linux.sh` | exit 1, "Refusing in-place install" + `~/dotfiles-repo` remediation |
| In-place refusal | `bats tests/setup-linux.bats` (#695 test) | pass |
| Same-dir no-op | `bats tests/install-git-hooks.bats` (#695 test) | pass — dispatcher + lib subtree intact |
| Bash syntax | `bash -n setup-linux.sh`, `bash -n scripts/install-git-hooks.sh` | clean |

### Pre-existing failures ruled out (Windows-local only)

`setup-linux.bats` "valid zsh syntax" and `install-git-hooks.bats` tests 3/4
(`core.hooksPath`) fail on this Windows box — but they fail identically on
**pristine main** (zsh not installed; MSYS git-config behavior). They are not
regressions from this change and pass in Linux CI.

## CI evidence (post-push, T6)

- [ ] `lint` (shellcheck) green.
- [ ] `test` (bats incl. setup-linux.bats + install-git-hooks.bats) green on
      ubuntu-latest.
- [ ] `integration` green — the copy layout (`~/dotfiles-repo`) is unaffected;
      the new preflight only fires on the in-place layout, which the container
      does not use.

## Guard rationale (incident -> guard)

Two guards land with the fix: a behavioral test proving setup exits 1 (not
corrupts) on the in-place layout, and a unit test proving the git-hooks mirror
cannot empty its own dispatcher. Both encode the exact failure the audit found.
