---
id: lesson-019-s-is-not-posix-use-space-in-bash-regex
type: lesson
status: active
created: "2026-03-12"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 019: \s is not POSIX — use `[[:space:]]` in bash regex

**Context**: `utils.sh` used `\s` in `[[ =~ ]]` regex patterns inside `load_env_file()` and `debug_print_env()`.

**Problem**: Bash's `[[ =~ ]]` uses POSIX Extended Regular Expressions (ERE), which do not define `\s` as a whitespace shorthand. `\s` is a Perl-Compatible Regular Expression (PCRE) extension. On some bash versions/platforms, `\s` silently fails to match whitespace, causing functions to behave incorrectly without any error message.

**Solution**: Replace all `\s` with the POSIX character class `[[:space:]]`, which is universally supported in ERE and functionally equivalent.

**Rule**: Never use `\s`, `\d`, `\w` or other PCRE shorthands in bash `[[ =~ ]]` or `grep -E` patterns. Use POSIX character classes instead: `[[:space:]]`, `[[:digit:]]`, `[[:alnum:]]`. Only `grep -P` (PCRE mode) supports the shorthand forms.
