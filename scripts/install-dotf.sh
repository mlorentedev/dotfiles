#!/usr/bin/env bash
# install-dotf.sh — fetch + verify + install the `dotf` CLI release binary.
#
# The ADR-020 bootstrap step (the shell layer "detect OS/arch, fetch binary,
# PATH"): download the pinned `dotf` release from GitHub, verify its sha256
# against the release checksums.txt, and install it to ~/.local/bin.
#
# Sourced by setup-linux.sh; also runnable standalone to (re)install/upgrade:
#     ./scripts/install-dotf.sh [version] [dest_dir] [base_url]
#
# DOTF_VERSION is the pinned version (versions.conf SSOT). The functions take
# the version/dest/base_url as args so bats can drive them against a file://
# fixture with no network. Cross-shell: bash + zsh safe.

# Resolve this script's directory once — used to find utils.sh and, when run
# standalone, versions.conf (the DOTF_VERSION SSOT).
_DOTF_SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]:-$0}")" && pwd)"

# Load logging + helpers if the caller (setup) has not already sourced utils.sh.
if ! command -v log_info >/dev/null 2>&1; then
    # shellcheck source=/dev/null
    . "$_DOTF_SCRIPT_DIR/utils.sh"
fi

# Release location; overridable (tests pass a file:// base).
DOTF_RELEASE_BASE="${DOTF_RELEASE_BASE:-https://github.com/mlorentedev/dotfiles/releases/download}"

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
    _dotf_actual="$(sha256sum "$work/$artifact" | awk '{print $1}')"
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
install_dotf() {
    version="${1:-${DOTF_VERSION:-}}"
    dest="${2:-$HOME/.local/bin}"
    base="${3:-$DOTF_RELEASE_BASE}"

    if [ -z "$version" ]; then
        log_error "install_dotf: no version given (set DOTF_VERSION in versions.conf)"
        return 1
    fi

    _dotf_osname="$(_dotf_os "$(uname -s)")" || return 1
    _dotf_archname="$(_dotf_arch "$(uname -m)")" || return 1

    if command_exists dotf; then
        _dotf_current="$(dotf version 2>/dev/null | awk '{print $NF}')"
        if [ "$_dotf_current" = "$version" ]; then
            log_info "dotf $version already installed; skipping"
            return 0
        fi
        [ -n "$_dotf_current" ] && log_info "dotf $_dotf_current drifted from pinned $version; converging"
    fi

    _dotf_artifact="dotf_${version}_${_dotf_osname}_${_dotf_archname}.tar.gz"
    _dotf_tmp="$(mktemp -d)" || return 1

    _dotf_fetch \
        "${base}/v${version}/${_dotf_artifact}" \
        "${base}/v${version}/checksums.txt" \
        "$_dotf_artifact" "$_dotf_tmp"
    _dotf_rc=$?

    if [ "$_dotf_rc" -eq 0 ]; then
        ensure_directory "$dest"
        if cp "$_dotf_tmp/dotf" "$dest/dotf" && chmod 0755 "$dest/dotf"; then
            log_success "dotf $version installed to $dest/dotf"
        else
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
    # Standalone: with no version arg and none exported, load the pinned
    # DOTF_VERSION from versions.conf (the SSOT). setup-linux.sh sources
    # versions.conf before us, so this only fires on a direct ./install-dotf.sh run.
    if [ -z "${1:-}" ] && [ -z "${DOTF_VERSION:-}" ] && [ -f "$_DOTF_SCRIPT_DIR/../versions.conf" ]; then
        # shellcheck source=/dev/null
        . "$_DOTF_SCRIPT_DIR/../versions.conf"
    fi
    install_dotf "$@"
fi
