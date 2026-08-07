---
id: "BUG-028-setup-refuse-in-place"
type: spec
status: implementing
created: "2026-07-09"
tags: [spec, tasks]
template_version: "1.0"
---

# Tasks — BUG-028-setup-refuse-in-place

- [x] T1. setup-linux.sh: early preflight refusing `CURRENT_DIR == DOTFILES_DIR`
  with an actionable exit-1 message, before any directory create/copy.
- [x] T2. install-git-hooks.sh: `deploy_git_hooks` no-ops on `src -ef dest`
  (skip rm -rf + cp; refresh exec bits) as defense in depth.
- [x] T3. README.md: Linux Quick Start clones to `~/dotfiles-repo`; note deploy
  target and `git`/`curl` requirement.
- [x] T4. Tests: behavioral in-place refusal (setup-linux.bats) + same-dir no-op
  (install-git-hooks.bats).
- [x] T5. Local verification: `bash -n` both scripts; new bats tests green;
  confirmed pre-existing zsh/hooksPath failures are Windows-env only (pass on
  pristine main too).
- [ ] T6. CI green (`lint`, `test` incl. both suites, `integration`).
