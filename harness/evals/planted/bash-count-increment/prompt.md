# Review this bash script

The script is run as `bash count.sh <files...>`, on Linux (bash 5) and macOS (`/bin/bash` 3.2).

```bash
#!/usr/bin/env bash
set -euo pipefail

# Count the regular files given as arguments and report the total.
count=0
for f in "$@"; do
  if [ -f "$f" ]; then
    ((count++))
  fi
done
echo "regular files: $count"
```

## How to answer

Answer as an adversarial review. Start with YAML frontmatter holding one field, `verdict:`, set to `PASS`, `PASS-WITH-GAPS` or `FAIL`. Then give a findings table with exactly this header, one row per defect:

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|

Severity is `Blocker`, `Major` or `Minor`. Reality is `REAL`, `THEORETICAL` or `SPECULATIVE`. Report only defects that make the code behave wrongly, each with a concrete failing input. Do not report something that is correct.
