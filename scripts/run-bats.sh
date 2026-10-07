#!/usr/bin/env bash
#
# run-bats.sh: run the bats suite in parallel, the same way on every OS.
#
# CI's Linux and macOS jobs both call this, so the CPU count, the preflight and
# the flags live in one place. Before it, ci.yml spelled `$(nproc)`, which does
# not exist on macOS: the command substitution came back empty and `--jobs ""`
# is an error there, not a default.
#
# Usage:
#   run-bats.sh [--filter-tags TAGS] [--expect-bash MAJOR] [--report-dir DIR] [bats-file...]
#
#   --filter-tags TAGS   run only tests carrying these bats tags (comma = AND, as
#                        bats defines it). The run FAILS when no test matches:
#                        bats itself reports zero matches as a green run, which
#                        is how a mistyped tag turns a CI leg into a no-op.
#   --report-dir DIR     also write a junit report with per-test timing into DIR
#                        (created if missing), for the CI timing artifact
#   --print-jobs         print the parallelism it would use and exit (for tests)
#   --expect-bash MAJOR  fail unless the `bash` bats will use (the one on PATH,
#                        which its `#!/usr/bin/env bash` shebangs resolve) has
#                        that major version. The macOS leg asks for 3, because
#                        /bin/bash 3.2 is what a Mac user's scripts run under
#                        and a Homebrew bash ahead of it on PATH would silently
#                        turn the leg into a second bash 5 run.
#   bats-file...         default: tests/*.bats
#
# Exit: bats' own status, or 1 for an unusable environment, 2 for bad usage.

set -euo pipefail

die() {
    # `::error::` makes GitHub Actions annotate the run; on a terminal it reads
    # as a plain prefix.
    printf '::error::run-bats: %s\n' "$1" >&2
    exit "${2:-1}"
}

# Logical CPUs. getconf answers on Linux (glibc, musl) and macOS alike; nproc is
# GNU-only and sysctl BSD-only, so they are fallbacks, not the primary.
cpu_count() {
    local n=""
    n="$(getconf _NPROCESSORS_ONLN 2>/dev/null)" || n=""
    case "$n" in
        ''|*[!0-9]*|0) n="$(nproc 2>/dev/null)" || n="" ;;
    esac
    case "$n" in
        ''|*[!0-9]*|0) n="$(sysctl -n hw.ncpu 2>/dev/null)" || n="" ;;
    esac
    case "$n" in
        ''|*[!0-9]*|0) return 1 ;;
    esac
    printf '%s\n' "$n"
}

tags=""
expect_bash=""
print_jobs=0
report_dir=""
while [ "$#" -gt 0 ]; do
    case "$1" in
        --filter-tags) [ "$#" -ge 2 ] || die "--filter-tags needs a value" 2; tags="$2"; shift 2 ;;
        --report-dir) [ "$#" -ge 2 ] || die "--report-dir needs a value" 2; report_dir="$2"; shift 2 ;;
        --print-jobs) print_jobs=1; shift ;;
        --expect-bash) [ "$#" -ge 2 ] || die "--expect-bash needs a value" 2; expect_bash="$2"; shift 2 ;;
        --) shift; break ;;
        -*) die "unknown option: $1" 2 ;;
        *) break ;;
    esac
done

# Relative file arguments stay relative to the caller; only the default suite is
# found from the script's own location.
if [ "$#" -gt 0 ]; then
    files=("$@")
else
    root="$(cd "$(dirname "${BASH_SOURCE[0]:-$0}")/.." && pwd)"
    files=("$root"/tests/*.bats)
fi

# `bats --jobs` shells out to GNU parallel. moreutils ships a different program
# of the same name, which bats cannot drive, so the banner is checked and not
# just the name. The runner images install it; nothing here does, so its absence
# fails loudly instead of as an obscure bats error.
command -v parallel >/dev/null 2>&1 \
    || die "GNU parallel is missing from the runner image; bats --jobs needs it"
parallel --version 2>/dev/null | head -n 1 | grep -q 'GNU parallel' \
    || die "the 'parallel' on PATH is not GNU parallel (moreutils?); bats --jobs needs GNU parallel"

if [ -n "$expect_bash" ]; then
    got="$(bash -c 'printf %s "${BASH_VERSINFO[0]}"')"
    [ "$got" = "$expect_bash" ] \
        || die "bats would run under bash $got ($(command -v bash)), expected major version $expect_bash; put the intended bash first on PATH"
    printf 'run-bats: bash %s at %s\n' "$(bash -c 'printf %s "$BASH_VERSION"')" "$(command -v bash)" >&2
fi

jobs="$(cpu_count)" || die "cannot determine the CPU count (getconf, nproc and sysctl all failed)"
if [ "$print_jobs" = 1 ]; then
    printf '%s\n' "$jobs"
    exit 0
fi

args=(--jobs "$jobs" --no-parallelize-within-files)
if [ -n "$tags" ]; then
    count="$(bats --count --filter-tags "$tags" "${files[@]}")"
    [ "${count:-0}" -gt 0 ] \
        || die "no test carries the tag(s) '$tags'; a leg that selects nothing passes green and proves nothing"
    printf 'run-bats: %s test(s) tagged %s\n' "$count" "$tags" >&2
    args+=(--filter-tags "$tags")
fi

if [ -n "$report_dir" ]; then
    mkdir -p "$report_dir"
    args+=(--report-formatter junit --timing --output "$report_dir")
fi

exec bats "${args[@]}" "${files[@]}"
