#!/usr/bin/env bash
#
# Mutation harness for BUG-093 Gate f.
#
# Every mutation the round-3 adversarial review reported as SURVIVING must be
# CAUGHT here. Run from anywhere; paths resolve from this file's location.
#
#   bash specs/BUG-093-gate-f-fail-closed/mutate.sh
#
# Exits non-zero if an expected-CAUGHT mutation survives, so it is a check and
# not a report. The anchor is asserted before each mutation is applied: if the
# patch does not change the file, the run reports ANCHOR-MISS and never a
# result, because a no-op that reports "the tests caught it" is the defect this
# spec is about (lesson 267).

set -u

HERE="$(cd "$(dirname "${BASH_SOURCE[0]:-$0}")" && pwd)"
CLI="$(cd "$HERE/../../cli" && pwd)"

# Preflight, because the failure mode without it is a LIE rather than an error.
# This script once used `path` as a local variable name. Under zsh `path` is a
# special array tied to $PATH, so assigning a string to it destroyed the command
# search path -- and every subsequent `cp` and `md5sum` vanished, which the loop
# below reports as ANCHOR-MISS: "the pattern was not found". A broken
# environment then reads as a mutation nobody could apply. `zsh -n` passes
# clean, so nothing catches it but running it.
for _cmd in md5sum cut cp mv python3 go grep sed; do
    if ! command -v "$_cmd" >/dev/null 2>&1; then
        printf 'PREFLIGHT FAILED: %s is not on PATH.\n' "$_cmd" >&2
        printf 'Every result below would read as ANCHOR-MISS, which is not what happened.\n' >&2
        exit 2
    fi
done

pass=0
fail=0
expected_survivors=0

run_mutation() {
    expect="$1"; label="$2"; file="$3"; from="$4"; to="$5"
    target="$CLI/$file"
    before="$(md5sum "$target" | cut -d' ' -f1)"
    cp "$target" "$target.bak"

    python3 - "$target" "$from" "$to" <<'PY'
import sys
p, a, b = sys.argv[1], sys.argv[2], sys.argv[3]
s = open(p).read()
open(p, 'w').write(s.replace(a, b, 1))
PY

    after="$(md5sum "$target" | cut -d' ' -f1)"
    if [ "$before" = "$after" ]; then
        printf '  ANCHOR-MISS  %s\n' "$label"
        printf '               the pattern was not found, so nothing was tested\n'
        mv "$target.bak" "$target"
        fail=$((fail + 1))
        return
    fi

    # Not a pipe: a pipeline reports the LAST command's status, and reading a
    # verification's exit code through one is how a failing command passes.
    out="$(cd "$CLI" && go test ./internal/worktree/ -count=1 2>&1)"
    rc=$?
    mv "$target.bak" "$target"

    if [ "$rc" -ne 0 ]; then
        if [ "$expect" = "CAUGHT" ]; then
            printf '  CAUGHT       %s\n' "$label"
            pass=$((pass + 1))
        else
            printf '  CAUGHT       %s (was declared uncovered -- update the spec)\n' "$label"
            pass=$((pass + 1))
        fi
        printf '%s\n' "$out" | grep -E '^[[:space:]]+--- FAIL' | head -2 | sed 's/^/               /'
    else
        if [ "$expect" = "SURVIVES" ]; then
            printf '  SURVIVED     %s (declared in verification.md)\n' "$label"
            expected_survivors=$((expected_survivors + 1))
        else
            printf '  SURVIVED     %s  <-- REGRESSION\n' "$label"
            fail=$((fail + 1))
        fi
    fi
}

printf '=== baseline (must be green, or every result below is meaningless) ===\n'
base_out="$(cd "$CLI" && go test ./internal/worktree/ -count=1 2>&1)"
base_rc=$?
if [ "$base_rc" -ne 0 ]; then
    printf 'BASELINE RED -- aborting\n%s\n' "$base_out"
    exit 1
fi
printf '  baseline rc=0\n\n=== mutations the round-3 review found SURVIVING ===\n'

run_mutation CAUGHT "finding 3: prefix test loses its separator anchor" \
    "internal/worktree/sweep_proc_linux.go" \
    'strings.HasPrefix(absDest, absTarget+string(filepath.Separator))' \
    'strings.HasPrefix(absDest, absTarget)'

run_mutation CAUGHT "finding 4: Uninspectable++ becomes a no-op" \
    "internal/worktree/sweep_proc_linux.go" \
    'reading.Uninspectable++' \
    '_ = reading'

run_mutation CAUGHT "finding 5: reapSingleWorktree drops the Gate f re-check" \
    "internal/worktree/sweep.go" \
    'if err != nil || absWT == absCwd || hostProcessInside(absWT).Inside {' \
    'if err != nil || absWT == absCwd {'

run_mutation CAUGHT "finding 6: Linux claims it has no process discovery" \
    "internal/worktree/sweep_proc_linux.go" \
    'const processDiscoverySupported = true' \
    'const processDiscoverySupported = false'

run_mutation CAUGHT "finding 7: unresolvable target fails OPEN" \
    "internal/worktree/sweep_proc_linux.go" \
    '		// The caller is about to delete this path, so it should exist. If it
		// cannot be resolved, the comparison below cannot be trusted either.
		return GateFReading{Inside: true}' \
    '		return GateFReading{Inside: false}'

printf '\n=== regressions from earlier rounds (must stay caught) ===\n'

run_mutation CAUGHT "round 1 Blocker: EvalSymlinks resolution removed" \
    "internal/worktree/sweep_proc_linux.go" \
    '	absTarget = resolved' \
    '	_ = resolved'

run_mutation CAUGHT "round 1: caller ignores Gate f entirely" \
    "internal/worktree/sweep.go" \
    '	reading := hostProcessInside(absWT)
	return reading, !reading.Inside' \
    '	reading := hostProcessInside(absWT)
	return reading, true'

printf '\n=== declared uncovered in verification.md (expected to survive) ===\n'

run_mutation SURVIVES "AC2: filepath.Abs error fails open (unreachable by input)" \
    "internal/worktree/sweep_proc_linux.go" \
    '	absTarget, err := filepath.Abs(targetPath)
	if err != nil {
		return GateFReading{Inside: true}
	}' \
    '	absTarget, err := filepath.Abs(targetPath)
	if err != nil {
		return GateFReading{Inside: false}
	}'

printf '\n=== summary ===\n'
printf '  caught: %d   declared survivors: %d   regressions/anchor-miss: %d\n' \
    "$pass" "$expected_survivors" "$fail"

# Every mutation is reverted by its own run, so the tree must be back to green.
# A red restore means a .bak did not make it home and every result above is
# describing a tree nobody has now.
(cd "$CLI" && go test ./internal/worktree/ -count=1 >/dev/null 2>&1)
frc=$?
printf '  restored rc=%d\n' "$frc"

if [ "$fail" -ne 0 ] || [ "$frc" -ne 0 ]; then
    exit 1
fi
exit 0
