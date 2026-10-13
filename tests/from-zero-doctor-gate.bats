#!/usr/bin/env bats
# The from-zero job's doctor gate (#2013 X1): a [FAIL] not in the known list
# fails it, a listed entry no [FAIL] matches fails it, and a doctor that exits
# non-zero without a [FAIL] line fails it. The committed list is checked against
# its own "# example:" lines, as test-windows' list is (TEST-003).

setup() {
    REPO_DIR="$(cd "$BATS_TEST_DIRNAME/.." && pwd)"
    GATE="$REPO_DIR/.github/scripts/from-zero-doctor-gate.sh"
    KNOWN="$REPO_DIR/.github/scripts/from-zero-known-failures.txt"
    TMP="$(mktemp -d "${BATS_TMPDIR:-/tmp}/fzgate_XXXXXX")"
    printf '%s\n' '# owner: test' 'darwin kubectl not in PATH' 'linux the hive daemon is down' > "$TMP/known"
}

teardown() {
    rm -rf "$TMP"
}

@test "only known failures: the gate passes and counts them" {
    printf '%s\n' '[Tools]' '  [ OK ] git' '  [FAIL] kubectl not in PATH' > "$TMP/out"
    run bash "$GATE" darwin 1 "$TMP/out" "$TMP/known"
    [ "$status" -eq 0 ] || { echo "$output"; false; }
    [[ "$output" == *"1 known FAIL line(s), 0 new, 0 stale"* ]] || false
}

@test "a failure no entry names fails the gate and is quoted" {
    printf '%s\n' '  [FAIL] kubectl not in PATH' '  [FAIL] helm not in PATH' > "$TMP/out"
    run bash "$GATE" darwin 1 "$TMP/out" "$TMP/known"
    [ "$status" -eq 1 ]
    [[ "$output" == *"new doctor failure on darwin"*"helm not in PATH"* ]] || false
}

@test "an entry no failure matches is stale and fails the gate" {
    printf '%s\n' '  [ OK ] kubectl' > "$TMP/out"
    run bash "$GATE" darwin 0 "$TMP/out" "$TMP/known"
    [ "$status" -eq 1 ]
    [[ "$output" == *"stale entry for darwin"*"kubectl not in PATH"* ]] || false
}

@test "another OS's entries neither excuse nor go stale" {
    printf '%s\n' '  [FAIL] the hive daemon is down, start it' '  [FAIL] kubectl not in PATH' > "$TMP/out"
    run bash "$GATE" darwin 1 "$TMP/out" "$TMP/known"
    [ "$status" -eq 1 ]
    [[ "$output" == *"new doctor failure on darwin"*"hive daemon"* ]] || false
    [[ "$output" != *"stale"* ]] || false
}

@test "a non-zero doctor with no FAIL line fails the gate" {
    printf '%s\n' 'panic: runtime error' > "$TMP/out"
    printf '%s\n' '# nothing known' > "$TMP/known"
    run bash "$GATE" darwin 2 "$TMP/out" "$TMP/known"
    [ "$status" -eq 1 ]
    [[ "$output" == *"exited 2 without a [FAIL] line"* ]] || false
}

@test "an empty list and a clean doctor pass: the list's steady state" {
    printf '%s\n' '  [ OK ] everything' > "$TMP/out"
    printf '%s\n' '# all fixed' > "$TMP/known"
    run bash "$GATE" linux 0 "$TMP/out" "$TMP/known"
    [ "$status" -eq 0 ] || { echo "$output"; false; }
}

@test "colour codes around the FAIL tag do not hide it" {
    printf '  \033[31m[FAIL]\033[0m kubectl not in PATH\n' > "$TMP/out"
    run bash "$GATE" darwin 1 "$TMP/out" "$TMP/known"
    [ "$status" -eq 0 ] || { echo "$output"; false; }
}

@test "a malformed entry stops the gate and names its line" {
    printf '%s\n' 'macos kubectl not in PATH' > "$TMP/known"
    printf '%s\n' '  [FAIL] kubectl not in PATH' > "$TMP/out"
    run bash "$GATE" darwin 1 "$TMP/out" "$TMP/known"
    [ "$status" -eq 2 ]
    [[ "$output" == *"known:1: macos kubectl"* ]] || false
}

@test "the gate gives the same verdicts under zsh" {
    command -v zsh >/dev/null 2>&1 || skip "zsh not installed"
    printf '%s\n' '  [FAIL] kubectl not in PATH' '  [FAIL] helm not in PATH' > "$TMP/out"
    run zsh "$GATE" darwin 1 "$TMP/out" "$TMP/known"
    [ "$status" -eq 1 ]
    [[ "$output" == *"helm not in PATH"* ]] || false
    printf '%s\n' '  [FAIL] kubectl not in PATH' > "$TMP/out"
    run zsh "$GATE" darwin 1 "$TMP/out" "$TMP/known"
    [ "$status" -eq 0 ] || { echo "$output"; false; }
}

# Each committed entry sits under an "# example:" line holding the [FAIL] text it
# was written for, and its text is in that example: an entry with a typo would
# otherwise excuse nothing and read as stale only on the next run.
@test "every committed entry is in the example line above it" {
    run awk '
        /^# example: / { ex = substr($0, 12); next }
        /^[[:space:]]*(#|$)/ { next }
        {
            text = $0; sub(/^[^ ]+ /, "", text)
            if (ex == "" || index(ex, text) == 0 || index(ex, "[FAIL]") == 0) { print FNR ": " $0; bad = 1 }
            ex = ""
        }
        END { exit bad }
    ' "$KNOWN"
    [ "$status" -eq 0 ] || { echo "entries without a matching example: $output"; false; }
}

@test "the from-zero job runs doctor through the gate" {
    grep -qF '.github/scripts/from-zero-doctor-gate.sh' "$REPO_DIR/.github/workflows/from-zero.yml"
}
