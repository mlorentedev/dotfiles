#!/usr/bin/env bats
# tests/lib/gh-auth.bash: the real-gh suites' auth precondition must answer
# "does gh have a token" without the network (#2442). The stub gh below fails
# `auth status` the way a transient API error does, and records every call, so
# a helper that falls back to probing the API is caught.

load 'lib/gh-auth'

setup() {
    STUB="$BATS_TEST_TMPDIR/bin"
    mkdir -p "$STUB"
    CALLS="$BATS_TEST_TMPDIR/calls"
    : > "$CALLS"
    cat > "$STUB/gh" <<STUBEOF
#!/usr/bin/env bash
printf '%s\n' "\$*" >> "$CALLS"
case "\$1 \$2" in
    "auth status") echo "error connecting to api.github.com" >&2; exit 1 ;;
    "auth token") exit "\${STUB_TOKEN_RC:-1}" ;;
esac
exit 1
STUBEOF
    chmod +x "$STUB/gh"
    PATH="$STUB:$PATH"
    unset GH_TOKEN GITHUB_TOKEN
}

@test "a token in GH_TOKEN satisfies the precondition while the API is unreachable" {
    export GH_TOKEN=stub-value
    run gh_has_token
    [ "$status" -eq 0 ]
    [ ! -s "$CALLS" ]
}

@test "a token in GITHUB_TOKEN satisfies the precondition too" {
    export GITHUB_TOKEN=stub-value
    run gh_has_token
    [ "$status" -eq 0 ]
}

@test "a stored login satisfies the precondition, read offline" {
    export STUB_TOKEN_RC=0
    run gh_has_token
    [ "$status" -eq 0 ]
    grep -qx 'auth token' "$CALLS"
    run grep -c 'auth status' "$CALLS"
    [ "$output" = "0" ]
}

@test "no token at all still fails, so CI fails closed" {
    export STUB_TOKEN_RC=1
    run gh_has_token
    [ "$status" -ne 0 ]
}

@test "neither real-gh suite asks gh auth status any more" {
    for f in spec-gate-prepush-real.bats spec-gate-pr-real.bats; do
        run grep -nE '^[^#]*gh auth status' "$BATS_TEST_DIRNAME/$f"
        [ "$status" -eq 1 ]
        grep -q 'gh_has_token' "$BATS_TEST_DIRNAME/$f"
    done
}
