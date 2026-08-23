#!/usr/bin/env bats
# `dotf agent run` — the machine contract, smoked through the real binary
# (CLI-042 AC1).
#
# The Go tests already assert the record's shape in-process. What they cannot
# assert is the thing the criterion is actually about: that a SHELL pipeline
# sees the record on stdout with nothing else mixed in. captureRealStreams swaps
# os.Stdout for a pipe inside one process; this runs the compiled binary and
# pipes it, which is how every consumer will reach it.
#
# Skips (never fails) when the Go toolchain is absent, so a shell-only checkout
# still runs the rest of the suite. CI installs Go for the `test` job precisely
# so this does not silently skip there.

setup() {
    REPO_ROOT="$BATS_TEST_DIRNAME/.."
    # Slot state goes to a per-test dir, never the machine's real one. The
    # semaphore is deliberately machine-scoped (two checkouts share one
    # subscription), so a smoke that used the default would take real slots from
    # whatever else is dispatching on this box.
    SEM_DIR="$BATS_TEST_TMPDIR/slots"
}

# Build once per file run into a cached location, so the cases do not pay a
# compilation each.
_build_dotf() {
    command -v go >/dev/null 2>&1 || skip "go toolchain not installed"
    DOTF_BIN="${BATS_FILE_TMPDIR:-/tmp}/dotf-agent-run"
    export DOTF_BIN
    if [ ! -x "$DOTF_BIN" ]; then
        ( cd "$REPO_ROOT/cli" && go build -o "$DOTF_BIN" ./cmd/dotf ) \
            || skip "go build failed"
    fi
}

# python3 parses the record in three cases. It is probed like `go` and `jq` are,
# so a checkout without it SKIPS rather than fails — the file header promises
# that, and an unprobed dependency turns a missing interpreter into a red suite.
_need_python() {
    command -v python3 >/dev/null 2>&1 || skip "python3 not installed"
}

_run_dry() {
    _build_dotf
    "$DOTF_BIN" agent run \
        --role reviewer --task 'smoke the contract' --tier "${1:-mid}" \
        --backend dry-run --timeout 30s --repo-root "$REPO_ROOT" \
        --semaphore-dir "$SEM_DIR"
}

@test "agent run: stdout alone is a parseable JSON object" {
    _build_dotf
    _need_python
    # stderr deliberately NOT merged: the claim is that a consumer capturing
    # only stdout gets the whole record. Merging 2>&1 would let a record written
    # to the wrong stream pass.
    run bash -c "'$DOTF_BIN' agent run --role reviewer --task smoke --tier mid \
        --backend dry-run --timeout 30s --repo-root '$REPO_ROOT' --semaphore-dir '$SEM_DIR' 2>/dev/null"
    [ "$status" -eq 0 ]
    printf '%s' "$output" | python3 -c 'import json,sys; json.load(sys.stdin)'
}

@test "agent run: the record names status, route and duration" {
    _need_python
    run _run_dry mid
    [ "$status" -eq 0 ]
    local parsed
    parsed="$(printf '%s' "$output" | python3 -c '
import json, sys
r = json.load(sys.stdin)
missing = [k for k in ("status", "tier", "pool", "model", "exit", "duration_ms", "output") if k not in r]
print("MISSING:" + ",".join(missing) if missing else r["status"] + " " + r["pool"] + ":" + r["model"])
')"
    [ "${parsed#MISSING:}" = "$parsed" ]
    [ "${parsed%% *}" = "dry_run" ]
}

@test "agent run: a dispatcher can read the status through a pipe" {
    _build_dotf
    command -v jq >/dev/null 2>&1 || skip "jq not installed"
    # The documented consumer, verbatim from the command's own Long text.
    run bash -c "'$DOTF_BIN' agent run --role reviewer --task smoke --tier mid \
        --backend dry-run --timeout 30s --repo-root '$REPO_ROOT' --semaphore-dir '$SEM_DIR' | jq -r .status"
    [ "$status" -eq 0 ]
    [ "$output" = "dry_run" ]
}

@test "agent run: the top tier resolves to its single declared entry" {
    _build_dotf
    _need_python
    # Through a file rather than re-interpolating $output into another shell:
    # the record contains quotes, and a case that breaks on its own payload
    # fails for the wrong reason.
    local rec="$BATS_TEST_TMPDIR/top.json"
    run bash -c "'$DOTF_BIN' agent run --role architect --task decide --tier top \
        --backend dry-run --timeout 30s --repo-root '$REPO_ROOT' --semaphore-dir '$SEM_DIR' >'$rec' 2>/dev/null"
    [ "$status" -eq 0 ]
    run python3 -c '
import json, sys
r = json.load(open(sys.argv[1]))
assert len(r["attempts"]) == 1, r["attempts"]
print(r["pool"])
' "$rec"
    [ "$status" -eq 0 ]
    [ "$output" = "claude" ]
}

@test "agent run: a missing --timeout is refused and writes nothing to stdout" {
    _build_dotf
    run bash -c "'$DOTF_BIN' agent run --role r --task t --tier mid \
        --backend dry-run --repo-root '$REPO_ROOT' --semaphore-dir '$SEM_DIR' 2>/dev/null"
    [ "$status" -eq 1 ]
    [ -z "$output" ]
}

@test "agent run: --timeout is named in the refusal, on stderr" {
    _build_dotf
    run bash -c "'$DOTF_BIN' agent run --role r --task t --tier mid \
        --backend dry-run --repo-root '$REPO_ROOT' --semaphore-dir '$SEM_DIR' 2>&1 >/dev/null"
    [ "$status" -eq 1 ]
    printf '%s' "$output" | grep -q -- '--timeout is required'
}

@test "agent run: no backend is a loud refusal, not a silent dry run" {
    _build_dotf
    run bash -c "'$DOTF_BIN' agent run --role r --task t --tier mid \
        --timeout 30s --repo-root '$REPO_ROOT' --semaphore-dir '$SEM_DIR' 2>&1 >/dev/null"
    [ "$status" -eq 1 ]
    printf '%s' "$output" | grep -q -- '--backend is required'
}

@test "agent run: a saturated pool is skipped and the chain advances to the next entry" {
    _build_dotf
    _need_python
    # fcntl is POSIX-only; the suite runs under bash and zsh on Linux, but the
    # probe keeps the case a skip rather than a failure anywhere it is absent.
    python3 -c "import fcntl" 2>/dev/null || skip "python3 fcntl unavailable"
    # Hold every dispatchable nan slot with real flocks, then dispatch at the
    # low tier, whose chain is nan:qwen3.6 -> claude:haiku. nan declares
    # concurrency 5 with reserve_interactive 2, so 3 are takeable. This proves
    # the semaphore through the compiled binary and the real lock primitive, not
    # only through the Go-level seam.
    local helper="$BATS_TEST_TMPDIR/saturate.py" rec="$BATS_TEST_TMPDIR/saturated.json"
    cat > "$helper" <<'PY'
import fcntl, os, subprocess, sys
sem_dir, dotf, repo_root, out = sys.argv[1:5]
pool_dir = os.path.join(sem_dir, "nan")
os.makedirs(pool_dir, exist_ok=True)
held = []
for i in range(3):                                  # concurrency 5 - reserve 2
    f = open(os.path.join(pool_dir, "slot-%d.lock" % i), "a+b")
    fcntl.flock(f.fileno(), fcntl.LOCK_EX | fcntl.LOCK_NB)
    held.append(f)
with open(out, "wb") as fh:
    r = subprocess.run([dotf, "agent", "run", "--role", "r", "--task", "t",
                        "--tier", "low", "--backend", "dry-run", "--timeout", "30s",
                        "--repo-root", repo_root, "--semaphore-dir", sem_dir],
                       stdout=fh, stderr=subprocess.DEVNULL)
sys.exit(r.returncode)
PY
    run python3 "$helper" "$SEM_DIR" "$DOTF_BIN" "$REPO_ROOT" "$rec"
    [ "$status" -eq 0 ]

    run python3 -c '
import json, sys
r = json.load(open(sys.argv[1]))
assert r["status"] == "dry_run", r["status"]
assert r["pool"] == "claude", r["pool"]          # nan was full, so the chain advanced
assert r["attempts"][0]["pool"] == "nan", r["attempts"]
assert r["attempts"][0]["status"] == "pool_unavailable", r["attempts"]
print("advanced")
' "$rec"
    [ "$status" -eq 0 ]
    [ "$output" = "advanced" ]
}
