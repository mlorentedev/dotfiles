---
id: "BUG-047-commit-msg-hook-windows"
type: spec
status: archived # draft | implementing | verifying | archived
created: "2026-08-07"
issue: "dotfiles#794"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
review: waived
review_waived_reason: "Work shipped and merged under #794; the issue was then closed by hand, so the archive-on-merge gate (keyed on a PR closing keyword) never saw it and the spec was left active. A retroactive adversarial review cannot gate code already on main, so the waiver is recorded instead of manufacturing one. Backlog reconciliation 2026-08-19."
---

# BUG-047: Commit-msg hook — scoped commits and Windows

<!-- from issue #794: the pre-commit test hook tests the deployed tree, not the committed one — and cannot pass on Windows -->

## Why

The local hook stack does not gate what it claims to gate. `validate-commit-msg.sh` matched `^[a-z]+: .+` against the whole commit message, which **rejects every scoped Conventional Commit** — the exact form this repo uses and release-please consumes. `feat(tmux):`, `docs(spec):`, `chore(specs):` and `fix(setup):` are all real subjects on `main` and all four were refused. The validator only ever appeared to work because PRs are squash-merged on GitHub, where a local `commit-msg` hook never runs; it fired against local commits alone. Separately, its `#!/bin/sh` shebang made it unrunnable on native Windows (`Executable /bin/sh not found`), as did `#!/bin/bash` in `check-spec-gate.sh` for the pre-push gate — so on Windows the hooks were not lenient, they were absent.

## What

The commit-msg validator accepts the repo's actual commit grammar and runs on every supported platform:

1. `type(scope)!: subject` is accepted; scope and the breaking-change `!` are optional.
2. Only the **subject** is validated, so a conforming body line can no longer rescue a malformed subject.
3. The hook scripts invoked by pre-commit resolve their interpreter via `env`, so they execute on Windows as well as Linux/macOS.
4. Git-generated subjects (`Merge `, `Revert "`, `fixup!`, `squash!`) are exempt rather than rejected.

## Out of scope

- `scripts/test.sh` resolving `DOTFILES_DIR` to the deploy target instead of the repo, and its deployment assertions running inside a repo-integrity suite — findings 1 and 2 of #794, deliberately left open there.
- The remaining ~10 non-hook scripts still carrying `#!/bin/bash`. They do not block anything; noted on #794.
- Changing the commit convention itself, or what release-please consumes.

## Risks / open questions

- **Permissive type vs allow-list.** `[a-z]+` accepts any lowercase type, including typos like `feet:`. An allow-list would be stricter but would reject the repo's real non-release types (`wip:`, and the `plan:` the old error text advertised). Resolved: stay permissive; the gate's job here is shape, and release-please already ignores types it does not know.
- **`env sh` availability.** Resolves through PATH on Windows via Git Bash and natively elsewhere. Verified empirically on Windows — this branch's own commit passed the hook.
- **zsh absent on Windows** made the pre-existing cross-shell tests fail. Resolved by skipping rather than deleting, so the contract keeps its teeth where zsh exists.

## Acceptance criteria

- [x] **AC1** — Scoped Conventional Commits are accepted, including the four real subjects currently on `main`, plus breaking-change `!` and scopes containing `.`, `/`, `-`.
- [x] **AC2** — Malformed subjects are still rejected: no type, capitalised type, missing space after the colon, empty subject.
- [x] **AC3** — Only the subject is validated; a conforming body line does not rescue a malformed subject.
- [x] **AC4** — Git-generated subjects (`Merge`, `Revert "`, `fixup!`, `squash!`) are exempt.
- [x] **AC5** — The hook runs on native Windows: the shebang resolves via `env`, and a real commit on Windows passes it.
- [x] **AC6** — `scripts/check-spec-gate.sh` executes on Windows, so the pre-push SDD gate is enforced there instead of erroring out.
- [x] **AC7** — The cross-shell syntax contract (sh/bash/zsh) is preserved, and the suite is green on a host without zsh.

## References

- Bitácora board: [dotfiles#794](https://github.com/mlorentedev/dotfiles/issues/794)
- Blocked by this: [dotfiles#791](https://github.com/mlorentedev/dotfiles/issues/791) (AI-028) — its spec commit could not be made until the hooks ran
- Related: BUG-036 (pre-commit under global hooksPath), ADR-025 (`DOTFILES_DIR` vs `DOTFILES_REPO_DIR`)
