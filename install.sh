#!/usr/bin/env bash
# install.sh — the one entrypoint on Linux and macOS: install `dotf`, then
# hand the machine to `dotf converge`.
#
# The ADR-020 bootstrap step (the shell layer "detect OS/arch, fetch binary,
# PATH"): download the pinned `dotf` release from GitHub, verify its sha256
# against the release checksums.txt, prove it runs, and install it to
# ~/.local/bin. Everything after that is `dotf converge`, which clones the
# checkout when it is absent (PLAT-001b).
#
# A machine from zero (no checkout; resolves the latest release):
#     curl -fsSL https://raw.githubusercontent.com/mlorentedev/dotfiles/main/install.sh | bash
#     curl -fsSL https://raw.githubusercontent.com/mlorentedev/dotfiles/main/install.sh | bash -s -- --plan
# From a checkout (installs the versions.conf pin):
#     ./install.sh [converge flags]
#
# Executed, every argument goes to `dotf converge`. DOTF_VERSION, DOTF_BIN_DIR
# and DOTF_RELEASE_BASE override the version, the install directory and the
# release location.
#
# Sourced (setup-linux.sh), it only defines install_dotf [version] [dest_dir]
# [base_url], which bats drives against a file:// fixture with no network.
# Cross-shell: bash + zsh safe.

# Resolve the actual script directory once. A raw stream has no script file:
# never treat its current working directory as a checkout, because that could
# source an unrelated utils.sh. Raw recovery instead uses local fallbacks and
# resolves the latest release through DOTF_RELEASE_API.
_DOTF_SOURCE="${BASH_SOURCE[0]:-}"
if [ -f "$_DOTF_SOURCE" ]; then
    _DOTF_SCRIPT_DIR="$(cd "$(dirname "$_DOTF_SOURCE")" && pwd)"
else
    _DOTF_SCRIPT_DIR=''
fi

# Load logging + helpers if the caller (setup) has not already sourced utils.sh.
# A raw curl stream has no checkout-relative utils.sh, so retain a small local
# fallback for release recovery instead of failing before the verified download.
if [ -n "$_DOTF_SCRIPT_DIR" ] && [ -f "$_DOTF_SCRIPT_DIR/scripts/utils.sh" ]; then
    if ! command -v log_info >/dev/null 2>&1; then
        # shellcheck source=/dev/null
        . "$_DOTF_SCRIPT_DIR/scripts/utils.sh"
    fi
else
    log_info() { printf '[INFO] %s\n' "$*"; }
    log_success() { printf '[OK] %s\n' "$*"; }
    log_error() { printf '[ERROR] %s\n' "$*" >&2; }
    command_exists() { command -v "$1" >/dev/null 2>&1; }
    ensure_directory() { mkdir -p "$1"; }
fi

# Release location; overridable (tests pass a file:// base).
DOTF_RELEASE_BASE="${DOTF_RELEASE_BASE:-https://github.com/mlorentedev/dotfiles/releases/download}"
DOTF_RELEASE_API="${DOTF_RELEASE_API:-https://api.github.com/repos/mlorentedev/dotfiles/releases/latest}"

# _dotf_latest_version resolves GitHub's latest release to a strict semver.
# This is used only when no explicit version, environment pin, or checkout
# versions.conf is available, such as a raw curl recovery stream.
_dotf_latest_version() {
    _dotf_metadata="$(curl -fsSL "$DOTF_RELEASE_API")" || {
        log_error "install_dotf: latest-release lookup failed: $DOTF_RELEASE_API"
        return 1
    }
    _dotf_version="$(printf '%s\n' "$_dotf_metadata" |
        sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"v\([0-9.]*\)".*/\1/p' |
        head -n1)"
    if ! printf '%s\n' "$_dotf_version" | grep -Eq '^[0-9]+\.[0-9]+\.[0-9]+$'; then
        log_error "install_dotf: latest-release metadata has no semver tag"
        return 1
    fi
    printf '%s\n' "$_dotf_version"
}

# _dotf_arch <uname-m>: map host machine to the goreleaser arch token.
_dotf_arch() {
    case "$1" in
        x86_64 | amd64) printf 'amd64' ;;
        aarch64 | arm64) printf 'arm64' ;;
        *) printf 'unsupported arch: %s\n' "$1" >&2; return 1 ;;
    esac
}

# _dotf_os <uname-s>: map host OS to the goreleaser os token.
_dotf_os() {
    case "$1" in
        Linux) printf 'linux' ;;
        Darwin) printf 'darwin' ;;
        *) printf 'unsupported os: %s\n' "$1" >&2; return 1 ;;
    esac
}

# _dotf_current_version: the version of the `dotf` already on PATH, or empty.
#
# This is a function, not the inline pipeline it used to be, because it is the
# ONE step in this script whose answer comes from outside the script -- and a
# test cannot control that (#1409). With the pipeline inline, six of the eight
# tests in tests/install.bats failed on a clean tree for anyone with a
# source-built dotf on PATH, because the `dev` branch below fired against the
# developer's OWN binary and returned before any install happened. They were
# carried as "fixture isolation" for about ten sessions. Bisected: 6 failures
# with ~/.local/bin on PATH, 0 without, same commit.
#
# Empty on failure is deliberate and load-bearing. If `dotf version` errors, or
# prints something with no semver in it, the caller sees "" -- which matches
# neither `dev` nor the pinned version, so it falls through and installs. That
# is the right answer for a broken or unrecognisable binary: converge. Do not
# "fix" this into a hard failure; that would abort the install in exactly the
# case where replacing the binary is the repair.
#
# "Empty" has to be MADE empty, though, in both directions -- an earlier form of
# this function only got it right by luck of the caller's shell flags. The status
# must not escape (pipefail turns a no-match into an aborted setup) and the parse
# must not succeed on text that is not a version (an error message containing
# `dev` read as a source build). Both are asserted in tests/install.bats
# under `set -euo pipefail`, because neither is visible without those flags.
_dotf_current_version() {
    command_exists dotf || return 0

    # The stream merge (2>&1) is kept deliberately: BUG-070 (#915) fixed
    # `dotf version` to write to stdout, but this installer's whole job is to
    # run against whatever dotf is already on PATH -- including binaries built
    # before that fix, which answer on stderr. Merging both streams and regexing
    # the semver is correct for either. Do not tighten it to stdout-only; that
    # would silently break the idempotence skip on an old binary and reinstall
    # on every run.
    #
    # Capture first, parse second, and gate on the EXIT STATUS -- because the
    # merge above is what makes an error message parseable. A binary that fails
    # to run prints to the stream we read, and `dev` is a three-letter substring
    # of ordinary paths: `dotf: /home/dev/x: not found` matched the source-build
    # gate, so the installer took "leave it in place" and preserved a binary
    # that cannot run, permanently. Measured on this branch before the fix.
    # A non-zero `dotf version` means "nothing recognisable installed".
    _dotf_raw="$(dotf version 2>&1)" || return 0

    # `|| :` is the whole point of this line, not tidiness. setup-linux.sh
    # sources this under `set -euo pipefail`; when the binary runs fine but
    # prints no version, grep matches nothing and exits 1, `pipefail` promotes
    # that to the pipeline's status, and `set -e` aborts SETUP at the caller's
    # `_dotf_current="$(...)"`. The sibling guard on `command_exists` above
    # covers the absent binary; this covers the present-but-unparseable one,
    # which the flags reach identically. `|| return 0` cannot close it -- the
    # pipeline is the last command, so its status is the function's.
    printf '%s\n' "$_dotf_raw" | grep -oE '[0-9]+\.[0-9]+\.[0-9]+|dev' | head -n1 || :
}

# _dotf_sha256 <file>: print the file's sha256. macOS ships no sha256sum before
# 26 (and there it lives in /sbin, off the PATH an installer stream runs with);
# `shasum` is in /usr/bin on every release. This script runs from a raw stream
# with no utils.sh, so the helper is local. An unusable tool prints nothing and
# the caller's comparison fails, which is the safe direction.
_dotf_sha256() {
    if command -v sha256sum >/dev/null 2>&1; then
        sha256sum "$1" | awk '{print $1}'
    elif command -v shasum >/dev/null 2>&1; then
        shasum -a 256 "$1" | awk '{print $1}'
    fi
}

# _dotf_fetch <url> <sums_url> <artifact> <workdir>: download the artifact and
# checksums into workdir, verify sha256, extract `dotf`. A mismatch or missing
# entry aborts (return 1) and leaves nothing extracted.
_dotf_fetch() {
    url="$1"; sums_url="$2"; artifact="$3"; work="$4"
    if ! curl -fsSL "$url" -o "$work/$artifact"; then
        log_error "install_dotf: download failed: $url"; return 1
    fi
    if ! curl -fsSL "$sums_url" -o "$work/checksums.txt"; then
        log_error "install_dotf: checksums download failed: $sums_url"; return 1
    fi
    _dotf_expected="$(awk -v f="$artifact" '$2 == f {print $1}' "$work/checksums.txt")"
    if [ -z "$_dotf_expected" ]; then
        log_error "install_dotf: $artifact not listed in checksums.txt"; return 1
    fi
    _dotf_actual="$(_dotf_sha256 "$work/$artifact")"
    if [ "$_dotf_expected" != "$_dotf_actual" ]; then
        log_error "install_dotf: checksum mismatch for $artifact (want $_dotf_expected, got $_dotf_actual)"
        return 1
    fi
    if ! tar -xzf "$work/$artifact" -C "$work" dotf; then
        log_error "install_dotf: failed to extract dotf from $artifact"; return 1
    fi
}

# install_dotf [version] [dest_dir] [base_url]: idempotently install the pinned
# `dotf` release. No-op when the pinned version is already on PATH; converges on
# drift. Returns non-zero (no binary left in dest) on any download/verify error.
# On success DOTF_BIN names the binary it vetted: the one on PATH when it kept
# it, else the one it placed in dest_dir, which a fresh machine does not have
# on PATH yet.
install_dotf() {
    version="${1:-${DOTF_VERSION:-}}"
    dest="${2:-$HOME/.local/bin}"
    base="${3:-$DOTF_RELEASE_BASE}"

    if [ -z "$version" ]; then
        version="$(_dotf_latest_version)" || return 1
    fi

    _dotf_osname="$(_dotf_os "$(uname -s)")" || return 1
    _dotf_archname="$(_dotf_arch "$(uname -m)")" || return 1

    # Empty means "nothing recognisable installed", which includes no dotf at
    # all: the absence check lives inside the seam so that this function has
    # exactly ONE line that depends on the machine it runs on, and a test that
    # stubs it controls the whole question. It used to be two -- the seam plus
    # an `if command_exists dotf` wrapper around this block -- and the wrapper
    # alone was enough to make a stubbed test read the real environment again.
    _dotf_current="$(_dotf_current_version)"

    # `dev` is what a source build reports (cli/cmd/dotf/main.go). A source
    # build on PATH is deliberate -- a dev box building the tree, or CI
    # building the PR under test -- and the release installer must not
    # replace it: the release lags the tree, and testing it certified
    # nothing about the change (#1305: `dotf harness mirror` did not exist
    # in 0.51.0; and setup sources versions.conf, so a DOTF_VERSION=dev
    # env override cannot express this). Remove the binary to converge.
    #
    # This gate is FIRST, and that is why picking an impossible version could
    # not disable it: `dev` returns before the version comparison below is ever
    # reached. tests/install.bats assumed one gate and there were two.
    if [ "$_dotf_current" = "dev" ]; then
        log_info "dotf is a source build (dev); leaving it in place (remove it to converge to the $version release)"
        DOTF_BIN="$(command -v dotf)"
        return 0
    fi
    if [ "$_dotf_current" = "$version" ]; then
        log_info "dotf $version already installed; skipping"
        DOTF_BIN="$(command -v dotf)"
        return 0
    fi
    if [ -n "$_dotf_current" ]; then
        log_info "dotf $_dotf_current drifted from pinned $version; converging"
    fi

    _dotf_artifact="dotf_${version}_${_dotf_osname}_${_dotf_archname}.tar.gz"
    _dotf_tmp="$(mktemp -d)" || return 1

    _dotf_fetch \
        "${base}/v${version}/${_dotf_artifact}" \
        "${base}/v${version}/checksums.txt" \
        "$_dotf_artifact" "$_dotf_tmp"
    _dotf_rc=$?

    # Exec probe before anything is placed: a binary that verifies but cannot
    # run here (wrong arch, a broken build) must not replace a working one.
    if [ "$_dotf_rc" -eq 0 ] && ! "$_dotf_tmp/dotf" version >/dev/null 2>&1; then
        log_error "install_dotf: the downloaded dotf $version does not run on this machine"
        _dotf_rc=1
    fi

    if [ "$_dotf_rc" -eq 0 ]; then
        ensure_directory "$dest"
        # Stage beside the target, then rename into place. Writing *onto* a live
        # binary fails with ETXTBSY (BUG-037), which broke the upgrade path in
        # exactly the case dotf is in daily use — the long-lived
        # `dotf secrets run -- <agent>` wrappers hold it open. rename(2) has no
        # such restriction, is atomic, and leaves any already-running process on
        # its own inode. Staging inside $dest keeps it on one filesystem, which
        # is what makes the rename atomic rather than a copy.
        _dotf_staged="$dest/.dotf.new.$$"
        if cp "$_dotf_tmp/dotf" "$_dotf_staged" &&
            chmod 0755 "$_dotf_staged" &&
            mv -f "$_dotf_staged" "$dest/dotf"; then
            log_success "dotf $version installed to $dest/dotf"
            DOTF_BIN="$dest/dotf"
        else
            rm -f "$_dotf_staged"
            log_error "install_dotf: failed to place binary in $dest"
            _dotf_rc=1
        fi
    fi

    rm -rf "$_dotf_tmp"
    return "$_dotf_rc"
}

# Run only when executed directly, not when sourced (by setup or bats).
# `(return)` succeeds only in a sourced context, so this is robust where the
# BASH_SOURCE-vs-$0 comparison is not (some shells/harnesses align them).
if ! (return 0 2>/dev/null); then
    # Standalone: with no DOTF_VERSION exported, a checkout's versions.conf
    # (the SSOT) names the pin; a raw stream has none and installs the latest
    # release. setup-linux.sh sources versions.conf itself and never gets here.
    if [ -z "${DOTF_VERSION:-}" ] && [ -n "$_DOTF_SCRIPT_DIR" ] && [ -f "$_DOTF_SCRIPT_DIR/versions.conf" ]; then
        # shellcheck source=/dev/null
        . "$_DOTF_SCRIPT_DIR/versions.conf"
    fi
    install_dotf "" "${DOTF_BIN_DIR:-$HOME/.local/bin}" || exit 1
    exec "$DOTF_BIN" converge "$@"
fi
