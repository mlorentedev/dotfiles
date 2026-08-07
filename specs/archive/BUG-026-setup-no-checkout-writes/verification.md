---
tags: [spec, verification]
created: "2026-07-09"
---

# Verification — BUG-026-setup-no-checkout-writes

## Reproduction (pre-fix, from #694)

```console
git clone --depth 5 https://github.com/mlorentedev/dotfiles.git ~/dotfiles-repo
cd ~/dotfiles-repo && bash setup-linux.sh          # exit 0
git status --porcelain                              # " M .github/copilot-instructions.md"
DOTFILES_REPO_DIR=$HOME/dotfiles-repo dotf update   # "dirty worktree — skipping", exit 0
```

Locally confirmed the drift on `main`: replaying the sync logic against the
committed files produces a diff (literal `\n`, banner reordered) — proof the
sync dirties the checkout on every run.

## Automated evidence (this branch)

| Check | Command | Result |
|---|---|---|
| Go build | `go build ./...` (in `cli/`) | pass (no output) |
| Go vet | `go vet ./internal/update/` | pass |
| Go unit | `go test ./internal/update/` | `ok` — dirty case now asserts message names `setup.sh` |
| Bash syntax | `bash -n setup-linux.sh` | `OK syntax` |
| Bats parse | `bats --count tests/verify-setup.bats` | 57 (was 56); new test name unique |
| PS1 ASCII | `sed -n '1720,1728p' setup-windows.ps1 \| grep -P '[^\x00-\x7F]'` | no match (ASCII-clean) |
| Parity intact | `cmp` of both files modulo `> ` banner | identical (docs-drift.bats stays green) |

## CI evidence (post-push, T7)

- [ ] `test` (unit bats incl. `docs-drift.bats`) green — parity preserved.
- [ ] `integration` (Docker: setup-linux + `verify-setup.bats`) green — the new
      guard "setup leaves the repo checkout clean [#694]" passes, proving the
      copilot-instructions write is gone and no other checkout write exists.
- [ ] `lint-powershell` green — `setup-windows.ps1` edit clean.
- [ ] Go tests in CI green.

## Guard rationale (incident -> guard)

The integration guard asserts the exact predicate `update.go` checks
(`git status --porcelain` empty) against the exact checkout setup runs from. It
generalizes past the copilot-instructions symptom: any future setup change that
writes into the checkout — the whole bug *class* — trips it, red-teaming the
idempotency contract on every CI run.
