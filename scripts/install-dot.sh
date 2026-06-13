#!/usr/bin/env bash
# install-dot.sh — fetch + verify + install the `dot` CLI release binary.
#
# The ADR-020 bootstrap step (the shell layer "detect OS/arch, fetch binary,
# PATH"): download the pinned `dot` release from GitHub, verify its sha256
# against the release checksums.txt, and install it to ~/.local/bin.
#
# Sourced by setup-linux.sh; also runnable standalone to (re)install/upgrade:
#     ./scripts/install-dot.sh [version] [dest_dir] [base_url]
#
# DOT_VERSION is the pinned version (versions.conf SSOT). The functions take
# the version/dest/base_url as args so bats can drive them against a file://
# fixture with no network. Cross-shell: bash + zsh safe.

# Load logging + helpers if the caller (setup) has not already sourced utils.sh.
if ! command -v log_info >/dev/null 2>&1; then
    _DOT_SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]:-$0}")" && pwd)"
    # shellcheck source=/dev/null
    . "$_DOT_SCRIPT_DIR/utils.sh"
fi

# Release location; overridable (tests pass a file:// base).
DOT_RELEASE_BASE="${DOT_RELEASE_BASE:-https://github.com/mlorentedev/dotfiles/releases/download}"

# _dot_arch <uname-m>: map host machine to the goreleaser arch token.
_dot_arch() {
    case "$1" in
        x86_64 | amd64) printf 'amd64' ;;
        aarch64 | arm64) printf 'arm64' ;;
        *) printf 'unsupported arch: %s\n' "$1" >&2; return 1 ;;
    esac
}

# _dot_os <uname-s>: map host OS to the goreleaser os token.
_dot_os() {
    case "$1" in
        Linux) printf 'linux' ;;
        Darwin) printf 'darwin' ;;
        *) printf 'unsupported os: %s\n' "$1" >&2; return 1 ;;
    esac
}

# _dot_fetch <url> <sums_url> <artifact> <workdir>: download the artifact and
# checksums into workdir, verify sha256, extract `dot`. A mismatch or missing
# entry aborts (return 1) and leaves nothing extracted.
_dot_fetch() {
    url="$1"; sums_url="$2"; artifact="$3"; work="$4"
    if ! curl -fsSL "$url" -o "$work/$artifact"; then
        log_error "install_dot: download failed: $url"; return 1
    fi
    if ! curl -fsSL "$sums_url" -o "$work/checksums.txt"; then
        log_error "install_dot: checksums download failed: $sums_url"; return 1
    fi
    _dot_expected="$(awk -v f="$artifact" '$2 == f {print $1}' "$work/checksums.txt")"
    if [ -z "$_dot_expected" ]; then
        log_error "install_dot: $artifact not listed in checksums.txt"; return 1
    fi
    _dot_actual="$(sha256sum "$work/$artifact" | awk '{print $1}')"
    if [ "$_dot_expected" != "$_dot_actual" ]; then
        log_error "install_dot: checksum mismatch for $artifact (want $_dot_expected, got $_dot_actual)"
        return 1
    fi
    if ! tar -xzf "$work/$artifact" -C "$work" dot; then
        log_error "install_dot: failed to extract dot from $artifact"; return 1
    fi
}

# install_dot [version] [dest_dir] [base_url]: idempotently install the pinned
# `dot` release. No-op when the pinned version is already on PATH; converges on
# drift. Returns non-zero (no binary left in dest) on any download/verify error.
install_dot() {
    version="${1:-${DOT_VERSION:-}}"
    dest="${2:-$HOME/.local/bin}"
    base="${3:-$DOT_RELEASE_BASE}"

    if [ -z "$version" ]; then
        log_error "install_dot: no version given (set DOT_VERSION in versions.conf)"
        return 1
    fi

    _dot_osname="$(_dot_os "$(uname -s)")" || return 1
    _dot_archname="$(_dot_arch "$(uname -m)")" || return 1

    if command_exists dot; then
        _dot_current="$(dot version 2>/dev/null | awk '{print $NF}')"
        if [ "$_dot_current" = "$version" ]; then
            log_info "dot $version already installed; skipping"
            return 0
        fi
        [ -n "$_dot_current" ] && log_info "dot $_dot_current drifted from pinned $version; converging"
    fi

    _dot_artifact="dot_${version}_${_dot_osname}_${_dot_archname}.tar.gz"
    _dot_tmp="$(mktemp -d)" || return 1

    _dot_fetch \
        "${base}/v${version}/${_dot_artifact}" \
        "${base}/v${version}/checksums.txt" \
        "$_dot_artifact" "$_dot_tmp"
    _dot_rc=$?

    if [ "$_dot_rc" -eq 0 ]; then
        ensure_directory "$dest"
        if cp "$_dot_tmp/dot" "$dest/dot" && chmod 0755 "$dest/dot"; then
            log_success "dot $version installed to $dest/dot"
        else
            log_error "install_dot: failed to place binary in $dest"
            _dot_rc=1
        fi
    fi

    rm -rf "$_dot_tmp"
    return "$_dot_rc"
}

# Run only when executed directly, not when sourced (by setup or bats).
# `(return)` succeeds only in a sourced context, so this is robust where the
# BASH_SOURCE-vs-$0 comparison is not (some shells/harnesses align them).
if ! (return 0 2>/dev/null); then
    install_dot "$@"
fi
