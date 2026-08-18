---
id: lesson-052-powershell-single-quoted-strings-grep-bre-backslas
type: lesson
status: active
created: "2026-05-21"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 052: PowerShell single-quoted strings + grep BRE: backslash counting

**Context:** BUG-017 PR #84 first CI run failed on bats test 471. The grep pattern was `'hooks\\\\hooks\.json'` in single-quoted bash to match a PowerShell single-quoted path `'hooks\hooks.json'`. With 4 backslashes the regex matched `hooks\\hooks.json` (two literal backslashes) — wrong. Reducing to 2 backslashes made it match `hooks\hooks.json` (one literal backslash) — correct.
**Problem:** The "double the backslashes" rule of thumb (count literal `\` then × 2 for regex escape) over-applies when the source string is ALREADY single-quoted bash. In bash single-quotes, every char is literal — no shell-level escape happens. So `'\\'` is exactly 2 chars sent to grep. Grep BRE treats `\\` as one literal `\`. Hence 2 backslashes in bash single-quote == 1 literal backslash matched. Adding 4 backslashes gives 2 literal matches, which over-shoots when the source has only one.
**Solution:** Counting rule for grep BRE inside bash single-quote: target literal backslashes × 2 (NOT × 4). Examples: `'\\'` → matches `\`; `'\\\\'` → matches `\\`; `'\\\\\\\\'` → matches `\\\\`. Tip: extract the pattern into a `pat=$'...'` ANSI-C variable for double-quote interpolation rules instead, OR use `grep -F` (fixed string, no regex) when you don't need wildcards — sidesteps the entire escape rabbit hole.
**Tags:** `#bash` `#powershell` `#grep` `#regex-escape` `#bats`
