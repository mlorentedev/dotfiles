#!/usr/bin/env bats
# Tests for scripts/install-dot.sh — the dot release fetch/verify/install.
#
# The script is sourced and its functions driven against a file:// fixture, so
# there is no network here. A fake `dot` binary that echoes its version stands
# in for the real release artifact.

setup() {
    SCRIPTS_DIR="$BATS_TEST_DIRNAME/../scripts"
    TMP="$(mktemp -d "/tmp/bats_installdot_XXXXXX")"

    # shellcheck source=/dev/null
    . "$SCRIPTS_DIR/install-dot.sh"

    VERSION="9.9.9"            # no real dot reports this -> idempotence stays off
    DEST="$TMP/bin"
    FIXTURE="$TMP/release"
    BASE="file://$FIXTURE"
    ART="dot_${VERSION}_$(_dot_os "$(uname -s)")_$(_dot_arch "$(uname -m)").tar.gz"
    mkdir -p "$FIXTURE/v$VERSION"

    # Fake artifact: a tarball whose `dot` echoes the version.
    PKG="$TMP/pkg"
    mkdir -p "$PKG"
    cat > "$PKG/dot" <<EOF
#!/bin/sh
echo "dot version $VERSION"
EOF
    chmod +x "$PKG/dot"
    tar -czf "$FIXTURE/v$VERSION/$ART" -C "$PKG" dot
}

teardown() {
    [ -z "${TMP:-}" ] || rm -rf "$TMP"
}

@test "arch mapping: known machines map, unknown is rejected" {
    run _dot_arch x86_64;  [ "$status" -eq 0 ]; [ "$output" = "amd64" ]
    run _dot_arch amd64;   [ "$status" -eq 0 ]; [ "$output" = "amd64" ]
    run _dot_arch aarch64; [ "$status" -eq 0 ]; [ "$output" = "arm64" ]
    run _dot_arch arm64;   [ "$status" -eq 0 ]; [ "$output" = "arm64" ]
    run _dot_arch i686;    [ "$status" -ne 0 ]
}

@test "happy path: verifies sha256 and installs the binary" {
    ( cd "$FIXTURE/v$VERSION" && sha256sum "$ART" > checksums.txt )

    run install_dot "$VERSION" "$DEST" "$BASE"
    [ "$status" -eq 0 ]
    [ -x "$DEST/dot" ]

    run "$DEST/dot"
    [ "$output" = "dot version $VERSION" ]
}

@test "checksum mismatch aborts the install, leaving no binary" {
    printf '%s  %s\n' "deadbeefdeadbeef" "$ART" > "$FIXTURE/v$VERSION/checksums.txt"

    run install_dot "$VERSION" "$DEST" "$BASE"
    [ "$status" -ne 0 ]
    [ ! -e "$DEST/dot" ]
}

@test "missing checksum entry aborts the install" {
    : > "$FIXTURE/v$VERSION/checksums.txt"

    run install_dot "$VERSION" "$DEST" "$BASE"
    [ "$status" -ne 0 ]
    [ ! -e "$DEST/dot" ]
}

@test "idempotent: a no-op when the pinned version is already on PATH" {
    ( cd "$FIXTURE/v$VERSION" && sha256sum "$ART" > checksums.txt )
    STUB="$TMP/stub"
    mkdir -p "$STUB"
    cat > "$STUB/dot" <<EOF
#!/bin/sh
echo "dot version $VERSION"
EOF
    chmod +x "$STUB/dot"

    export PATH="$STUB:$PATH"
    run install_dot "$VERSION" "$DEST" "$BASE"
    [ "$status" -eq 0 ]
    [[ "$output" == *"already installed"* ]]
    [ ! -e "$DEST/dot" ]
}
