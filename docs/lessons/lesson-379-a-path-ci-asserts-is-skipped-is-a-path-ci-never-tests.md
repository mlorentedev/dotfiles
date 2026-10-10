---
id: "lesson-379-a-path-ci-asserts-is-skipped-is-a-path-ci-never-tests"
type: lesson
status: active
title: "A path CI asserts is skipped is a path CI never tests"
created: "2026-10-10"
---

# A path CI asserts is skipped is a path CI never tests

## Context
`dotf tools install` took over the npm catalog (bw, yarn, opencode, copilot) from setup's shell
blocks. Setup already installed pi with `npm install -g --prefix "$HOME/.local"`, because a bare
`-g` writes to the first npm's global prefix: an nvm per-version tree when nvm is loaded, root-owned
`/usr/local` when it is not ([[lesson-105-an-npm-global-cli-under-nvm-is-invisible-to-gui-ad]]).

## The Trap
The Go port ran a bare `npm install -g`, and on the first upgrade on a box without nvm loaded every
npm catalog tool failed with EACCES (#2251). Three things kept it out of CI. The integration
container had no Node, so the npm catalog skipped. A test asserted that skip as correct behaviour
(`copilot config NOT deployed ... the container has no Node`), turning the gap into a green check.
And setup wraps `dotf tools install` in `|| log_warning`, so even a run that reached npm would have
warned, not failed. The same port lost a second rule: `uv tool install` refuses to replace an entry
point another installer left in `~/.local/bin`, which setup had never met because it never upgraded.

## The Solution
npm installs take `--prefix <Dest's parent>` on Linux and macOS (Windows keeps `%APPDATA%\npm`), and
uv installs take `--force`. The integration container installs Node from apt as root, which
reproduces a real box: npm's prefix is `/usr`, not writable by the test user. `verify-setup.bats`
asserts that precondition, that each npm catalog tool is in `~/.local/bin` owned by the user, and
that a second `dotf tools install` exits 0, since setup's own exit status hides the failure.

## Takeaways
- **A test that asserts a skip should name what would have to change for the path to run**, and that
  change belongs on a ticket. Otherwise the green check certifies that the path is never exercised.
- **Porting a shell block to Go ports its environment assumptions too.** Grep the old block's flags
  (`--prefix`, `--force`, `--ignore-scripts`) and decide each one explicitly in the port.
- **A best-effort step (`|| log_warning`) needs a guard that checks its result**, because its exit
  status never will.
