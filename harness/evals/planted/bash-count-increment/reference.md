---
verdict: FAIL
---

## Adversarial review

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Major | REAL | errexit | `((count++))` evaluates to the old value, 0, on the first regular file, so the arithmetic command returns status 1 and `set -e` exits the script before it prints anything | bash 5: `bash count.sh /etc/hosts; echo $?` prints nothing from the script and exits 1. macOS `/bin/bash` 3.2 prints `regular files: 1` and exits 0, because errexit only applies to `(( ))` from bash 4.1, so the Mac hides it | UNTESTED | code: `count=$((count + 1))` or `((++count))` |
| Minor | REAL | semantics | `[ -f ]` follows symlinks, so a symlink to a regular file is counted as a regular file | `ln -s /etc/hosts l; bash count.sh l` counts 1 | UNTESTED | code: add `&& [ ! -L "$f" ]` if links must not count |

### Verdict
FAIL
