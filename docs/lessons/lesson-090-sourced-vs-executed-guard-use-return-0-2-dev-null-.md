---
id: lesson-090-sourced-vs-executed-guard-use-return-0-2-dev-null-
type: lesson
status: active
created: "2026-06-13"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 090: Sourced-vs-executed guard: use `(return 0 2>/dev/null)`, not a `BASH_SOURCE`-vs-`$0` compare

**Context**: CLI-009 `scripts/install-dotf.sh` (named `install-dot.sh` until the CLI-010 rename) is both *sourced* (by `setup-linux.sh` and by its bats test) and *executed* directly (standalone `./install-dotf.sh` upgrade). It needs a guard so `install_dotf "$@"` runs only on direct execution, not on source.

**Problem**: The first guard was the common `if [ "${BASH_SOURCE[0]:-$0}" = "$0" ]`. It fired on *source* in the Bash-tool / harness context — `install_dotf` ran at source time with empty `$@`, printing a spurious `no version given` error (and would side-effect under setup). The `${BASH_SOURCE[0]:-$0}` fallback also makes it wrong under zsh: `BASH_SOURCE` is unset there, so the expansion becomes `$0` and the comparison is trivially true whether sourced or executed. A standalone diagnostic with `bash -c '. probe.sh'` showed "not equal" (correct) while the actual harness invocation showed it firing — i.e. the idiom's correctness is context-dependent, which is itself disqualifying for a guard.

**Solution**: `if ! (return 0 2>/dev/null); then install_dotf "$@"; fi`. `return` is only valid in a sourced script (or function), so the subshell exits 0 when sourced and non-zero when executed — a context-independent signal. Verified across `bash -c` source, script-source (the setup path), and direct execution.

**Rule**: To gate a script's "run only when executed directly" block, use `(return 0 2>/dev/null)` (sourced → succeeds, executed → fails), not a `BASH_SOURCE`-vs-`$0` string compare. The string compare aligns the two values in some shells (zsh, where `BASH_SOURCE` is unset) and some harnesses, so it fires on `source` and auto-runs the script's main path as a side effect.
