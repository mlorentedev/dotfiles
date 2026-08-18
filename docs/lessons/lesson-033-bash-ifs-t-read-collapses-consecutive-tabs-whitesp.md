---
id: lesson-033-bash-ifs-t-read-collapses-consecutive-tabs-whitesp
type: lesson
status: active
created: "2026-05-15"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 033: bash `IFS=$'\t' read` collapses consecutive tabs (whitespace IFS chars never preserve empty fields)

**Context:** Building `scripts/doctor.sh` to iterate `env-contract.json` entries. Used jq's `@tsv` to emit one TSV record per env-var, with `(.required_on // "")` for an optional column, then parsed with `while IFS=$'\t' read -r name required required_on default validation`. On entries where `required_on` was empty (most of them), every subsequent column shifted by one — `$default` got the validation value, `$validation` got an empty string, and the script ran silently wrong for ~10 minutes before a `bash -x` trace pinpointed it. Raw TSV had the right columns (`cat -A` confirmed); the read loop was eating them.

**Problem:** Bash's `read` treats *whitespace* IFS characters specially: even when you explicitly set `IFS=$'\t'`, consecutive tabs are collapsed into a single delimiter (the same rule applies to space and newline). So a TSV row like `name<TAB>true<TAB><EMPTY><TAB>$HOME/.dotfiles<TAB>path_exists` is read as if the empty third column did not exist, shifting every later assignment by one slot. POSIX behaviour, deeply confusing in practice because the documentation calls IFS "the field separator" and a quick reader assumes "tab means tab". The bug is silent: the script never errors, it just operates on wrong data.

**Solution:** Use a *non-whitespace* delimiter for TSV-like output whenever any column can be empty. Switched jq to `... | join("|")` and bash to `IFS='|' read -r ...`. Non-whitespace IFS chars do NOT collapse, so empty fields are preserved exactly. Pipe is safe for our values (paths, version strings, regex patterns) but for arbitrary content prefer the ASCII Unit Separator `$'\x1f'` — chosen by ASCII itself for this purpose, guaranteed absent from any sane string. Either way, **never use `IFS=$'\t'` (or `' '` or `$'\n'`) for `read` if an empty field is even possible**.

**Rule:** When a `read` loop must preserve empty fields, the IFS character must be non-whitespace (`|`, `;`, `:`, or `$'\x1f'`). For jq pipelines: replace `@tsv` with `[...] | join("|")`. For ad-hoc shell: never assume `IFS=$'\t' read` round-trips a TSV with empty columns — it doesn't, and the failure is invisible.

**Tags:** `#bash` `#ifs` `#read` `#tsv` `#silent-failure` `#jq` `#posix-gotcha`
