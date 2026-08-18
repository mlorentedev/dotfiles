---
id: lesson-148-zsh-expands-aliases-at-parse-time-and-the-resultin
type: lesson
status: active
created: "2026-08-04"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 148: zsh expands aliases at parse time, and the resulting parse error still exits 0

**Context**: `.zsh/functions.sh` held the Gemini saved-prompt helper. It had already been renamed once — `gp` → `gpr` — after colliding with oh-my-zsh's `alias gp='git push'`. The rename picked `gpr`, which oh-my-zsh's git plugin also owns (`alias gpr='git pull --rebase'`, `git.plugin.zsh:269`). `.zshrc` loads oh-my-zsh at line 13 and sources `functions.sh` at line 135, so the alias was always live first.

**Problem**: zsh expands aliases during **parsing**, not at invocation, so `gpr() {` parsed as `git pull --rebase () {` → `defining function based on alias` + `parse error near '()'`. The damage was not the message: a parse error aborts the rest of the sourced file, so the `utils.sh` load in the file's **last** block never ran, and every zsh session silently lost the shared library (`log_info`, `version_gte`, `deploy_file`). bash was unaffected — the git plugin is zsh-only. The 147-test suite missed it twice over: `tests/shell-wrapper-dedup.bats` sourced `functions.sh` in a bare `zsh -c` where no oh-my-zsh alias is live, and — decisively — **zsh returns exit status 0 after this parse error**, so the suite's `[ "$status" -eq 0 ]` assertions passed on a truncated file.

**Solution**: Renamed the helper to `agyp`, out of the `g*` namespace that the git plugin owns (~150 aliases) instead of picking a third name inside it. Added `tests/shell-alias-collision.bats`, whose tests assert **reach** — that `version_gte`, defined via the file's last block, still resolves after sourcing with `alias gp`/`alias gpr` pre-defined — rather than exit status. Verified the guard by reverting `functions.sh` to the pre-fix version: 4 of 5 tests went red, including the reach test in both bash and zsh.

**Rule**: Never name a shell function inside a namespace an installed plugin owns; when a name collides, leave the namespace rather than pick another name inside it — this repo hit the same rake twice (`gp`, then `gpr`). When testing that a sourced rc file loads correctly, assert that a symbol defined **after** the suspect line resolves; never assert exit status alone, because zsh reports a parse error on stderr and still exits 0. And source the file in an environment that has the aliases the real shell will have, or the collision cannot occur during the test.
