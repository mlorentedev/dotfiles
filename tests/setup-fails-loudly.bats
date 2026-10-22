#!/usr/bin/env bats
# #2013 W4 (F-048): a failure in the steps setup cannot finish without makes
# setup exit non-zero. They warned and went on, so a setup that deployed no
# agent config or no skill still ended in success and read as a converged box.
# dotf's install is the preflight: without it no later step can run.

setup() {
    REPO="$(cd "$BATS_TEST_DIRNAME/.." && pwd)"
    SH="$REPO/setup-linux.sh"
    PS1="$REPO/setup-windows.ps1"
}

@test "setup-linux.sh stops when dotf cannot be installed" {
    grep -qE '^    install_dotf \|\| \{ log_error "[^"]+"; exit 1; \}$' "$SH"
    run grep -nE 'install_dotf \|\| log_warning|install\.sh not found; skipping' "$SH"
    [ "$status" -eq 1 ] || { echo "$output"; false; }
}

@test "setup-linux.sh stops when dotf deploy fails" {
    grep -qE '^    "\$_dotf" deploy \|\| \{ log_error "[^"]+"; exit 1; \}$' "$SH"
    run grep -nE 'skipping agent config deploy' "$SH"
    [ "$status" -eq 1 ] || { echo "$output"; false; }
}

@test "setup-linux.sh stops when dotf harness deploy fails" {
    block="$(awk '/^if dotf harness deploy --repo-root "\$CURRENT_DIR"/{f=1} f{print} f&&/^fi$/{exit}' "$SH")"
    [ -n "$block" ] || false
    grep -qE '^    exit 1$' <<<"$block"
    run grep -E 'log_warning' <<<"$block"
    [ "$status" -eq 1 ] || { echo "$block"; false; }
}

@test "setup-windows.ps1 stops when dotf harness deploy fails" {
    block="$(sed 's/\r$//' "$PS1" | awk '/^& dotf harness deploy --repo-root \$DotfilesDir$/{f=1} f{print} f&&/^}$/{exit}')"
    [ -n "$block" ] || false
    grep -qE '^    exit 1' <<<"$block"
    run grep -E 'Write-Warn' <<<"$block"
    [ "$status" -eq 1 ] || { echo "$block"; false; }
}

@test "setup-windows.ps1 stops when dotf cannot be installed" {
    block="$(awk '/if \(-not \(Install-Dotf\)\) \{/{f=1} f{print} f&&/^    \}/{exit}' "$PS1")"
    [ -n "$block" ] || false
    grep -qE '^        exit 1' <<<"$block"
    run grep -nE 'installation failed \(continuing|install\.ps1 not found; skipping' "$PS1"
    [ "$status" -eq 1 ] || { echo "$output"; false; }
}

@test "setup-windows.ps1 stops when dotf deploy fails" {
    block="$(awk '/^    & dotf deploy/{f=1} f{print} f&&/^    \}/{exit}' "$PS1")"
    [ -n "$block" ] || false
    grep -qE '^        exit 1' <<<"$block"
    run grep -nE 'skipping agent config deploy' "$PS1"
    [ "$status" -eq 1 ] || { echo "$output"; false; }
}
