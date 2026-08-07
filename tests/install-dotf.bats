#!/usr/bin/env bats
# Tests for scripts/install-dotf.sh — the dotf release fetch/verify/install.
#
# The script is sourced and its functions driven against a file:// fixture, so
# there is no network here. A fake `dotf` binary that echoes its version stands
# in for the real release artifact.

setup() {
    SCRIPTS_DIR="$BATS_TEST_DIRNAME/../scripts"
    TMP="$(mktemp -d "/tmp/bats_installdotf_XXXXXX")"

    # shellcheck source=/dev/null
    . "$SCRIPTS_DIR/install-dotf.sh"

    VERSION="9.9.9"            # no real dotf reports this -> idempotence stays off
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
    STUB="$TMP/stub"
    mkdir -p "$STUB"
    cat > "$STUB/dotf" <<EOF
#!/bin/sh
echo "dotf version $VERSION"
EOF
    chmod +x "$STUB/dotf"

    export PATH="$STUB:$PATH"
    run install_dotf "$VERSION" "$DEST" "$BASE"
    [ "$status" -eq 0 ]
    [[ "$output" == *"already installed"* ]]
    [ ! -e "$DEST/dotf" ]
}

@test "converges over a running dotf: a live binary in dest is replaced, not refused" {
    # BUG-037: writing onto a *running* binary fails with ETXTBSY, so the upgrade
    # path broke in exactly the situation dotf is in daily use — the long-lived
    # `dotf secrets run -- <agent>` wrappers hold the binary open for hours.
    ( cd "$FIXTURE/v$VERSION" && sha256sum "$ART" > checksums.txt )

    mkdir -p "$DEST"
    # A shell script never triggers ETXTBSY (the kernel does not hold it as an
    # executable text image), so stand in a real ELF and keep it running.
    cp "$(command -v sleep)" "$DEST/dotf"
    chmod 0755 "$DEST/dotf"
    "$DEST/dotf" 30 &
    BUSY_PID=$!

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
    pinned="$(grep -m1 '^DOTF_VERSION=' "$SCRIPTS_DIR/../versions.conf" | cut -d= -f2)"
    run env -u DOTF_VERSION DOTF_RELEASE_BASE="file://$TMP/nope" "$SCRIPTS_DIR/install-dotf.sh"
    [[ "$output" != *"no version given"* ]]
    [[ "$output" == *"$pinned"* ]]
}
