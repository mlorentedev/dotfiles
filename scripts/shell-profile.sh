#!/bin/bash

# shell-profile.sh: measure shell startup time and (optionally) per-function detail.
#
# Time-only mode (default): runs the chosen shell N times, reports min/median/max
# of total interactive startup. Cheap, repeatable.
#
# Detail mode (--detail): activates DOTFILES_PROFILE=1, which the rc files
# enable per-function profiling via zsh's zprof (or bash xtrace). Gives a
# breakdown of which sourced file/function consumed time.
#
# Usage: ./scripts/shell-profile.sh [--shell zsh|bash] [--detail] [--runs N]
# Exit:  0 on success, 2 on usage error

set -euo pipefail

SHELL_CHOICE="zsh"
DETAIL=false
RUNS=5

while [ $# -gt 0 ]; do
    case "$1" in
        --shell)  SHELL_CHOICE="$2"; shift 2 ;;
        --detail) DETAIL=true; shift ;;
        --runs)   RUNS="$2"; shift 2 ;;
        --help|-h)
            cat <<'EOF'
Usage: shell-profile.sh [--shell zsh|bash] [--detail] [--runs N]

Time-only mode (default): time RUNS interactive shells, report stats.
Detail mode (--detail):   one shell with DOTFILES_PROFILE=1, dump profile.

Options:
  --shell zsh|bash   Which shell to profile (default: zsh)
  --detail           Enable per-function profiling via DOTFILES_PROFILE=1
  --runs N           Number of warm-up + measurement runs (default: 5)
EOF
            exit 0
            ;;
        *) echo "Unknown argument: $1" >&2; exit 2 ;;
    esac
done

case "$SHELL_CHOICE" in
    zsh|bash) ;;
    *) echo "Error: --shell must be zsh or bash, got: $SHELL_CHOICE" >&2; exit 2 ;;
esac

if ! command -v "$SHELL_CHOICE" >/dev/null 2>&1; then
    echo "Error: $SHELL_CHOICE not in PATH" >&2
    exit 2
fi

if $DETAIL; then
    echo "=== Detail profile: $SHELL_CHOICE (DOTFILES_PROFILE=1) ==="
    echo ""
    # `|| true`: an interactive shell reports the status of the last command its
    # profile ran, so a non-zero exit is ordinary, not a failure to profile.
    # Without this, `set -e` aborts here and the run reports failure (BUG-035).
    DOTFILES_PROFILE=1 "$SHELL_CHOICE" -i -c exit || true
    exit 0
fi

# Time-only mode: collect RUNS measurements (in seconds, with decimals) and report stats.
echo "Profiling $SHELL_CHOICE startup ($RUNS runs)..."
echo ""

samples=""
for _ in $(seq 1 "$RUNS"); do
    # `time` writes to stderr in `real Xm Y.YYYs` format; capture and convert to
    # seconds. The `|| true` keeps the *profiled* shell's exit status from
    # aborting the run: under `set -euo pipefail` a non-zero exit propagated
    # through the pipeline and killed the script before any stats were printed
    # (BUG-035). A loaded interactive profile exits non-zero routinely — the
    # status reflects the last command it ran, not a failure to start — which is
    # why this only bit on real machines and never in CI, where a bare
    # `bash -i -c exit` happens to return 0. Scoping it to the shell keeps
    # `set -e` protecting the awk that parses the measurement.
    raw=$( { time "$SHELL_CHOICE" -i -c exit || true ; } 2>&1 | awk '/^real/ {print $2}')
    # Format: "0m0.234s" → 0.234
    seconds=$(printf '%s' "$raw" | awk -F'[ms]' '{ printf "%.3f", $1 * 60 + $2 }')
    printf '  run: %ss\n' "$seconds"
    samples+="$seconds"$'\n'
done

# Stats: sort, pick min/median/max, mean.
echo ""
printf '%s' "$samples" | sort -n | awk -v runs="$RUNS" '
    { values[NR] = $1; sum += $1 }
    END {
        min = values[1]
        max = values[NR]
        median = (NR % 2 == 1) ? values[(NR + 1) / 2] : (values[NR/2] + values[NR/2 + 1]) / 2
        mean = sum / NR
        printf "  min:    %.3fs\n", min
        printf "  median: %.3fs\n", median
        printf "  mean:   %.3fs\n", mean
        printf "  max:    %.3fs\n", max
    }
'

echo ""
echo "Tip: re-run with --detail for a per-function breakdown."
