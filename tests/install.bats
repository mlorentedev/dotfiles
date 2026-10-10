#!/usr/bin/env bats
# Tests for install.sh — the root entrypoint: dotf release fetch/verify/install,
# then the hand-off to `dotf converge`.
#
# The script is sourced and its functions driven against a file:// fixture, so
# there is no network here. A fake `dotf` binary that echoes its version stands
# in for the real release artifact; when DOTF_ARGS_LOG is set it also records
# the arguments it was called with, which is how the hand-off is observed.

# bats file_tags=os-sensitive

load 'lib/os'

setup() {
    REPO_DIR="$(cd "$BATS_TEST_DIRNAME/.." && pwd)"
    TMP="$(mktemp -d "/tmp/bats_installdotf_XXXXXX")"

    # shellcheck source=/dev/null
    . "$REPO_DIR/install.sh"

    # Neutralise the ONE lookup that reaches outside the fixture. Without this
    # the tests read whatever dotf the developer has installed: a source build
    # answers `dev`, install_dotf takes its "leave it in place" branch and
    # returns 0 without installing, and six tests below fail on a clean tree
    # (#1409 -- measured: 6 failures with ~/.local/bin on PATH, 0 without).
    #
    # Empty is the default because it means "nothing installed", which is the
    # precondition every install test wants. Tests that care about the branches
    # set FAKE_CURRENT; the real implementation is exercised separately, in a
    # subshell that sources the script unstubbed -- a stub that is the only
    # thing ever tested proves nothing about the code it stands in for.
    _dotf_current_version() { printf '%s\n' "${FAKE_CURRENT:-}"; }

    VERSION="9.9.9"
    DEST="$TMP/bin"
    FIXTURE="$TMP/release"
    BASE="file://$FIXTURE"
    ART="dotf_${VERSION}_$(_dotf_os "$(uname -s)")_$(_dotf_arch "$(uname -m)").tar.gz"
    mkdir -p "$FIXTURE/v$VERSION"

    # Fake artifact: a tarball whose `dotf` echoes the version.
    PKG="$TMP/pkg"
    mkdir -p "$PKG"
    cat > "$PKG/dotf" <<EOF
#!/bin/sh
[ -z "\${DOTF_ARGS_LOG:-}" ] || printf '%s\n' "\$0 \$*" >> "\$DOTF_ARGS_LOG"
echo "dotf version $VERSION"
EOF
    chmod +x "$PKG/dotf"
    tar -czf "$FIXTURE/v$VERSION/$ART" -C "$PKG" dotf
}

teardown() {
    # A test that parks a live binary in dest must not leak the process.
    [ -z "${BUSY_PID:-}" ] || kill "$BUSY_PID" 2>/dev/null || true
    [ -z "${TMP:-}" ] || rm -rf "$TMP"
}

@test "arch mapping: known machines map, unknown is rejected" {
    run _dotf_arch x86_64;  [ "$status" -eq 0 ]; [ "$output" = "amd64" ]
    run _dotf_arch amd64;   [ "$status" -eq 0 ]; [ "$output" = "amd64" ]
    run _dotf_arch aarch64; [ "$status" -eq 0 ]; [ "$output" = "arm64" ]
    run _dotf_arch arm64;   [ "$status" -eq 0 ]; [ "$output" = "arm64" ]
    run _dotf_arch i686;    [ "$status" -ne 0 ]
}

@test "happy path: verifies sha256 and installs the binary" {
    ( cd "$FIXTURE/v$VERSION" && sha256sum "$ART" > checksums.txt )

    run install_dotf "$VERSION" "$DEST" "$BASE"
    [ "$status" -eq 0 ]
    [ -x "$DEST/dotf" ]

    run "$DEST/dotf"
    [ "$output" = "dotf version $VERSION" ]
}

@test "checksum mismatch aborts the install, leaving no binary" {
    printf '%s  %s\n' "deadbeefdeadbeef" "$ART" > "$FIXTURE/v$VERSION/checksums.txt"

    run install_dotf "$VERSION" "$DEST" "$BASE"
    [ "$status" -ne 0 ]
    [ ! -e "$DEST/dotf" ]
}

@test "missing checksum entry aborts the install" {
    : > "$FIXTURE/v$VERSION/checksums.txt"

    run install_dotf "$VERSION" "$DEST" "$BASE"
    [ "$status" -ne 0 ]
    [ ! -e "$DEST/dotf" ]
}

@test "idempotent: a no-op when the pinned version is already on PATH" {
    ( cd "$FIXTURE/v$VERSION" && sha256sum "$ART" > checksums.txt )

    FAKE_CURRENT="$VERSION"
    run install_dotf "$VERSION" "$DEST" "$BASE"
    [ "$status" -eq 0 ]
    [[ "$output" == *"already installed"* ]] || false
    [ ! -e "$DEST/dotf" ]
}

@test "a source build on PATH is left alone: dev short-circuits before the version check" {
    # THE BRANCH NOTHING EXERCISED (#1409). It sits ahead of the version check,
    # so no choice of VERSION can reach past it -- the old comment here read
    # "no real dotf reports 9.9.9 -> idempotence stays off", which disabled the
    # SECOND gate while the first one fired on every developer machine. A
    # deliberate, correct behaviour with no test read as six broken tests for
    # about ten sessions.
    ( cd "$FIXTURE/v$VERSION" && sha256sum "$ART" > checksums.txt )

    FAKE_CURRENT="dev"
    run install_dotf "$VERSION" "$DEST" "$BASE"
    [ "$status" -eq 0 ]
    [[ "$output" == *"source build"* ]] || false
    # The point of the branch: the release must NOT replace a source build.
    [ ! -e "$DEST/dotf" ]
}

@test "the real version lookup parses a dotf on PATH, on stdout or stderr" {
    # Drives the PRODUCTION _dotf_current_version, not setup()'s stub: sourced
    # fresh in a subshell so the stub is absent. Without this the stub would be
    # the only thing under test, and the seam could drift from the code it
    # stands in for while every test stayed green.
    STUB="$TMP/stub"; mkdir -p "$STUB"

    # Old binaries answer on stderr (BUG-070/#915), current ones on stdout.
    # Both must parse, which is why the call merges the streams.
    printf '#!/bin/sh\necho "dotf version 1.2.3"\n'      > "$STUB/dotf"; chmod +x "$STUB/dotf"
    run bash -c ". '$REPO_DIR/install.sh'; export PATH=\"$STUB:\$PATH\"; _dotf_current_version"
    [ "$output" = "1.2.3" ]

    printf '#!/bin/sh\necho "dotf version dev" >&2\n'    > "$STUB/dotf"; chmod +x "$STUB/dotf"
    run bash -c ". '$REPO_DIR/install.sh'; export PATH=\"$STUB:\$PATH\"; _dotf_current_version"
    [ "$output" = "dev" ]

    # A `go install …@latest` build prints its pseudo-version. The lookup must
    # read it as 0.0.0: neither `dev` (which would skip the install) nor the pin.
    printf '#!/bin/sh\necho "dotf version 0.0.0-20260930012845-db2b904794b4"\n' > "$STUB/dotf"; chmod +x "$STUB/dotf"
    run bash -c ". '$REPO_DIR/install.sh'; export PATH=\"$STUB:\$PATH\"; _dotf_current_version"
    [ "$output" = "0.0.0" ]

    # Unrecognisable output must yield empty, so the caller converges rather
    # than matching a branch by accident.
    printf '#!/bin/sh\necho "not a version"\n'           > "$STUB/dotf"; chmod +x "$STUB/dotf"
    run bash -c ". '$REPO_DIR/install.sh'; export PATH=\"$STUB:\$PATH\"; _dotf_current_version"
    [ -z "$output" ]
}

@test "no dotf installed is survivable under set -euo pipefail" {
    # setup-linux.sh runs `set -euo pipefail` and sources this script, so the
    # version lookup executes under those flags in production. With `pipefail`,
    # `dotf version | grep | head` fails at the FIRST element when dotf is not
    # installed, the pipeline reports non-zero, and `set -e` aborts the whole of
    # setup at the assignment -- on a fresh machine, which is the one path where
    # dotf is guaranteed absent. The `command_exists` guard inside the seam is
    # what prevents that, and it looks inert until you run it under the flags:
    # measured identical (empty, silent) without them, fatal with them.
    empty="$TMP/nothing"; mkdir -p "$empty"

    run bash -c "set -euo pipefail; . '$REPO_DIR/install.sh'; \
                 export PATH='$empty:/usr/bin:/bin'; \
                 v=\$(_dotf_current_version); printf 'survived:[%s]' \"\$v\""
    [ "$status" -eq 0 ]
    [ "$output" = "survived:[]" ]
}

@test "a present but unusable dotf survives set -euo pipefail and never reads as dev" {
    # The sibling above covers the ABSENT binary, which `command_exists` guards.
    # A PRESENT one reaches the same flags by a route nothing covered, and the
    # existing "unrecognisable output yields empty" test cannot see it: that one
    # asserts the VALUE and runs without the flags, while production asserts the
    # PROCESS and runs with them. Measured on this branch before the fix: exit 1,
    # `survived:` never printed — i.e. setup-linux.sh aborts.
    unusable="$TMP/unusable"; mkdir -p "$unusable"

    # Runs fine, says nothing version-shaped: grep matches nothing and exits 1,
    # pipefail promotes that to the pipeline's status, set -e kills the caller.
    printf '#!/bin/sh\necho "not a version"\n' > "$unusable/dotf"
    chmod +x "$unusable/dotf"
    run bash -c "set -euo pipefail; . '$REPO_DIR/install.sh'; \
                 export PATH='$unusable:/usr/bin:/bin'; \
                 v=\$(_dotf_current_version); printf 'survived:[%s]' \"\$v\""
    [ "$status" -eq 0 ]
    [ "$output" = "survived:[]" ]

    # Cannot run at all. Its message lands on the stream this function reads --
    # the 2>&1 merge is deliberate, for old stderr-answering builds — and `dev`
    # is a substring of ordinary paths, so this parsed as a source build and the
    # installer kept a binary that cannot run, permanently. Empty, not "dev".
    printf '#!/bin/sh\necho "dotf: /home/dev/x: not found" >&2\nexit 127\n' \
        > "$unusable/dotf"
    chmod +x "$unusable/dotf"
    run bash -c "set -euo pipefail; . '$REPO_DIR/install.sh'; \
                 export PATH='$unusable:/usr/bin:/bin'; \
                 v=\$(_dotf_current_version); printf 'survived:[%s]' \"\$v\""
    [ "$status" -eq 0 ]
    [ "$output" = "survived:[]" ]
}

@test "converges over a running dotf: a live binary in dest is replaced, not refused" {
    # BUG-037: writing onto a *running* binary fails with ETXTBSY, so the upgrade
    # path broke in exactly the situation dotf is in daily use — the long-lived
    # `dotf secrets run -- <agent>` wrappers hold the binary open for hours.
    ( cd "$FIXTURE/v$VERSION" && sha256sum "$ART" > checksums.txt )

    mkdir -p "$DEST"
    # A shell script never triggers ETXTBSY (the kernel does not hold it as an
    # executable text image), so stand in a real ELF and keep it running.
    # `sleep` itself does NOT work for this on every coreutils build (BUG-054):
    # on a multi-call `coreutils` binary the applet dispatch can key off the
    # RESOLVED EXECUTABLE PATH rather than argv[0] (observed with uutils
    # coreutils -- `exec -a sleep` changes argv[0] but the copy still dispatches
    # by its own path and refuses "unknown program"), so no argv[0] trick
    # recovers it once the file has been copied to a path named "dotf". A copy
    # of `bash` sidesteps the whole dispatch question: it is never a multi-call
    # binary, and a builtin busy-loop (no forked/exec'd child) keeps THIS COPY's
    # own executable text image open for the swap to contend with.
    cp "$(command -v bash)" "$DEST/dotf"
    chmod 0755 "$DEST/dotf"
    # An Apple-silicon kernel kills (SIGKILL, "Killed: 9") a copied Apple-signed
    # binary at exec: its signature is bound to the original path. Re-sign the
    # copy ad hoc so it runs; without it the fixture dies and proves nothing.
    if [ "$(uname -s)" = Darwin ]; then
        codesign --force --sign - "$DEST/dotf" >/dev/null 2>&1
    fi
    "$DEST/dotf" -c 's=$SECONDS; while (( SECONDS - s < 30 )); do :; done' &
    BUSY_PID=$!

    # A fixture that failed to hold the binary busy must fail loudly here, at
    # setup, instead of silently degrading the rest of the test into a check of
    # the ordinary replace-a-file path (BUG-054).
    sleep 0.2
    kill -0 "$BUSY_PID"

    run install_dotf "$VERSION" "$DEST" "$BASE"
    [ "$status" -eq 0 ]

    run "$DEST/dotf"
    [ "$output" = "dotf version $VERSION" ]

    # Swapping the binary must not disturb the process already running it.
    kill -0 "$BUSY_PID" 2>/dev/null
}

@test "install leaves no staging artifact behind in dest" {
    ( cd "$FIXTURE/v$VERSION" && sha256sum "$ART" > checksums.txt )

    run install_dotf "$VERSION" "$DEST" "$BASE"
    [ "$status" -eq 0 ]

    # Exactly one entry, the binary: a leftover staging file would accumulate on
    # every upgrade and (unremoved) mask a later failed swap.
    run bash -c "find '$DEST' -mindepth 1 | wc -l"
    [ "$output" -eq 1 ]
    [ -x "$DEST/dotf" ]
}

@test "a failed verify leaves an already-installed binary intact" {
    # The staging swap must never widen the failure window: an aborted install
    # has to leave the previous dotf runnable.
    printf '%s  %s\n' "deadbeefdeadbeef" "$ART" > "$FIXTURE/v$VERSION/checksums.txt"

    mkdir -p "$DEST"
    printf '#!/bin/sh\necho "dotf version 0.0.1"\n' > "$DEST/dotf"
    chmod 0755 "$DEST/dotf"

    run install_dotf "$VERSION" "$DEST" "$BASE"
    [ "$status" -ne 0 ]

    run "$DEST/dotf"
    [ "$output" = "dotf version 0.0.1" ]
}

@test "standalone (executed, no arg, no DOTF_VERSION env) resolves the pinned version from versions.conf" {
    # Regression guard: the standalone run-guard called install_dotf with no version
    # because the script read DOTF_VERSION only from the env and never sourced
    # versions.conf. Execute it directly (NOT sourced) with DOTF_VERSION unset and no
    # arg, pointed at a non-existent file:// base so it never touches the network. It
    # must get PAST the "no version given" check and name the pinned versions.conf value.
    pinned="$(grep -m1 '^DOTF_VERSION=' "$REPO_DIR/versions.conf" | cut -d= -f2)"
    # PATH and HOME are confined: executed, the script hands off to converge,
    # and a developer's pinned dotf on PATH would otherwise be the one it runs.
    run env -u DOTF_VERSION HOME="$TMP/standalone-home" PATH="/usr/bin:/bin" \
        DOTF_RELEASE_BASE="file://$TMP/nope" "$REPO_DIR/install.sh"
    [[ "$output" != *"no version given"* ]] || false
    [[ "$output" == *"$pinned"* ]] || false
}

@test "a raw installer stream installs an explicit release outside a checkout" {
    # The curl-style recovery path runs the script from stdin. Its directory is
    # /dev, not the checkout, so it cannot depend on utils.sh or versions.conf.
    ( cd "$FIXTURE/v$VERSION" && sha256sum "$ART" > checksums.txt )

    pipe_home="$TMP/pipe-home"
    pipe_path="$TMP/pipe-path"
    mkdir -p "$pipe_path"

    run bash -c "env HOME='$pipe_home' PATH='$pipe_path:/usr/bin:/bin' \
        DOTF_VERSION='$VERSION' DOTF_RELEASE_BASE='$BASE' \
        bash < '$REPO_DIR/install.sh'"
    [ "$status" -eq 0 ]
    [[ "$output" != *"command_exists"* ]] || false
    [ -x "$pipe_home/.local/bin/dotf" ]

    run "$pipe_home/.local/bin/dotf"
    [ "$output" = "dotf version $VERSION" ]
}

@test "latest-release resolver accepts a semver tag from release metadata" {
    curl() { printf '%s\n' '{"tag_name":"v9.9.9"}'; }

    run _dotf_latest_version
    [ "$status" -eq 0 ]
    [ "$output" = "$VERSION" ]
}

@test "latest-release resolver rejects malformed release metadata" {
    curl() { printf '%s\n' '{"tag_name":"not-a-version"}'; }

    run _dotf_latest_version
    [ "$status" -ne 0 ]
    [[ "$output" == *"semver"* ]] || false
}

@test "a raw installer stream resolves its release without a checkout pin" {
    ( cd "$FIXTURE/v$VERSION" && sha256sum "$ART" > checksums.txt )
    printf '%s\n' "{\"tag_name\":\"v$VERSION\"}" > "$FIXTURE/latest.json"

    pipe_home="$TMP/latest-pipe-home"
    pipe_path="$TMP/latest-pipe-path"
    mkdir -p "$pipe_path"

    run bash -c "env HOME='$pipe_home' PATH='$pipe_path:/usr/bin:/bin' \
        DOTF_RELEASE_BASE='$BASE' DOTF_RELEASE_API='file://$FIXTURE/latest.json' \
        bash < '$REPO_DIR/install.sh'"
    [ "$status" -eq 0 ]

    run "$pipe_home/.local/bin/dotf"
    [ "$output" = "dotf version $VERSION" ]
}

@test "a raw installer stream never sources a utils.sh from its current directory" {
    ( cd "$FIXTURE/v$VERSION" && sha256sum "$ART" > checksums.txt )

    pipe_home="$TMP/untrusted-pipe-home"
    pipe_path="$TMP/untrusted-pipe-path"
    untrusted="$TMP/untrusted-working-directory"
    marker="$TMP/untrusted-utils-was-sourced"
    mkdir -p "$pipe_path" "$untrusted"
    touch "$untrusted/bash"
    cat > "$untrusted/utils.sh" <<'EOF'
touch "$RAW_INSTALL_MARKER"
EOF

    run bash -c "cd '$untrusted' && env HOME='$pipe_home' PATH='$pipe_path:/usr/bin:/bin' \
        DOTF_VERSION='$VERSION' DOTF_RELEASE_BASE='$BASE' RAW_INSTALL_MARKER='$marker' \
        bash < '$REPO_DIR/install.sh'"
    [ "$status" -eq 0 ]
    [ ! -e "$marker" ]
}

@test "a raw installer stream hands its arguments to dotf converge, run from where it installed it" {
    # The one-entrypoint contract (PLAT-001b): after the install, the machine
    # belongs to converge. A fresh machine has no ~/.local/bin on PATH, so the
    # hand-off must use the binary the installer placed, not a PATH lookup.
    ( cd "$FIXTURE/v$VERSION" && sha256sum "$ART" > checksums.txt )

    pipe_home="$TMP/handoff-home"
    pipe_path="$TMP/handoff-path"
    log="$TMP/handoff.log"
    mkdir -p "$pipe_path"

    run bash -c "env HOME='$pipe_home' PATH='$pipe_path:/usr/bin:/bin' \
        DOTF_VERSION='$VERSION' DOTF_RELEASE_BASE='$BASE' DOTF_ARGS_LOG='$log' \
        bash -s -- --plan --only tools < '$REPO_DIR/install.sh'"
    [ "$status" -eq 0 ]
    # The exec probe ran the downloaded binary first, then converge got the args.
    [ "$(tail -n 1 "$log")" = "$pipe_home/.local/bin/dotf converge --plan --only tools" ]
}

@test "DOTF_BIN_DIR names the install directory, and converge runs from there" {
    ( cd "$FIXTURE/v$VERSION" && sha256sum "$ART" > checksums.txt )

    log="$TMP/bindir.log"
    run env HOME="$TMP/bindir-home" PATH="/usr/bin:/bin" DOTF_BIN_DIR="$DEST" DOTF_VERSION="$VERSION" \
        DOTF_RELEASE_BASE="$BASE" DOTF_ARGS_LOG="$log" "$REPO_DIR/install.sh"
    [ "$status" -eq 0 ]
    [ -x "$DEST/dotf" ]
    [ ! -e "$TMP/bindir-home/.local/bin/dotf" ]
    [ "$(tail -n 1 "$log")" = "$DEST/dotf converge" ]
}

@test "with the pinned dotf already on PATH, converge runs that one and nothing is installed" {
    onpath="$TMP/onpath"
    log="$TMP/onpath.log"
    mkdir -p "$onpath"
    cp "$PKG/dotf" "$onpath/dotf"

    run env HOME="$TMP/onpath-home" PATH="$onpath:/usr/bin:/bin" DOTF_VERSION="$VERSION" \
        DOTF_RELEASE_BASE="file://$TMP/nope" DOTF_ARGS_LOG="$log" "$REPO_DIR/install.sh" --plan
    [ "$status" -eq 0 ]
    [[ "$output" == *"already installed"* ]] || false
    [ ! -e "$TMP/onpath-home/.local/bin/dotf" ]
    [ "$(tail -n 1 "$log")" = "$onpath/dotf converge --plan" ]
}

@test "a failed install never hands off to converge" {
    printf '%s  %s\n' "deadbeefdeadbeef" "$ART" > "$FIXTURE/v$VERSION/checksums.txt"

    log="$TMP/failed.log"
    run env HOME="$TMP/failed-home" PATH="/usr/bin:/bin" DOTF_VERSION="$VERSION" \
        DOTF_RELEASE_BASE="$BASE" DOTF_ARGS_LOG="$log" "$REPO_DIR/install.sh"
    [ "$status" -ne 0 ]
    [ ! -e "$log" ]
}

@test "a release binary that verifies but cannot run here is never placed" {
    # The exec probe: the checksum proves the bytes are the published ones, not
    # that they run on this machine (wrong arch, a broken build). An unrunnable
    # binary must not replace the working one already in dest.
    printf '#!/bin/sh\nexit 126\n' > "$PKG/dotf"
    tar -czf "$FIXTURE/v$VERSION/$ART" -C "$PKG" dotf
    ( cd "$FIXTURE/v$VERSION" && sha256sum "$ART" > checksums.txt )

    mkdir -p "$DEST"
    printf '#!/bin/sh\necho "dotf version 0.0.1"\n' > "$DEST/dotf"
    chmod 0755 "$DEST/dotf"

    run install_dotf "$VERSION" "$DEST" "$BASE"
    [ "$status" -ne 0 ]
    [[ "$output" == *"does not run"* ]] || false
    run "$DEST/dotf"
    [ "$output" = "dotf version 0.0.1" ]
}

@test "the pinned dotf can converge from zero" {
    # From zero, install.sh hands off to a RELEASED dotf. Before the floor its
    # converge can neither clone the checkout (PLAT-001b PR 4a) nor run the
    # setup script (PR 5), so a fresh machine would get dotf and nothing else.
    # release-please raises the pin with every release; this only fails while
    # main still pins a release older than the entrypoint it ships.
    floor="0.67.0"
    pinned="$(grep -m1 '^DOTF_VERSION=' "$REPO_DIR/versions.conf" | cut -d= -f2)"
    [ -n "$pinned" ]
    run awk -v a="$pinned" -v b="$floor" 'BEGIN {
        split(a, x, "."); split(b, y, ".")
        for (i = 1; i <= 3; i++) {
            if (x[i] + 0 > y[i] + 0) exit 0
            if (x[i] + 0 < y[i] + 0) exit 1
        }
        exit 0
    }'
    [ "$status" -eq 0 ] || {
        printf 'versions.conf pins dotf %s, below %s\n' "$pinned" "$floor"
        false
    }
}
