#!/usr/bin/env bash
# Convert an MSYS path (/c/...) to a native Windows path for pwsh. Git Bash
# auto-converts paths in plain arguments but NOT inside quoted -Command
# strings, and native pwsh resolves /c/... against the current drive root
# (C:\c\... -> not found). On Linux/macOS cygpath is absent and the path
# passes through unchanged.
_winpath() {
    if command -v cygpath >/dev/null 2>&1; then
        cygpath -w "$1"
    else
        printf '%s' "$1"
    fi
}
