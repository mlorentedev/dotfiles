#!/usr/bin/env bash
#
# Run the NON-LINUX Gate f implementation on this Linux machine.
#
#   bash specs/BUG-093-gate-f-fail-closed/flip-proxy.sh
#
# AC7 used to name `GOOS=windows go vet`, which COMPILES sweep_proc_other.go and
# never runs it -- which is how round 2 shipped a red `test (windows-latest)`
# past a fully green local loop. This executes that path instead.
#
# FOUR files are flipped, not two. Round 5 caught verification.md describing a
# two-file flip: with only the implementations swapped, the still-`//go:build
# linux` sweep_proc_linux_test.go is selected and the package fails to compile
# (`undefined: isNumericPID`). The conclusion was right and the command was
# wrong, which is worse than either -- it reads as reproducible and is not.

set -u

HERE="$(cd "$(dirname "${BASH_SOURCE[0]:-$0}")" && pwd)"
PKG="$(cd "$HERE/../../cli/internal/worktree" && pwd)"
CLI="$(cd "$HERE/../../cli" && pwd)"

for _cmd in sed cp mv go; do
    command -v "$_cmd" >/dev/null 2>&1 || { printf 'PREFLIGHT FAILED: %s not on PATH\n' "$_cmd" >&2; exit 2; }
done

# One name per positional parameter, NOT a space-joined string. `for f in $FILES`
# is the repo's own prohibited pattern: zsh does not word-split an unquoted
# parameter, so the loop runs ONCE with all four names glued together, every
# `cp` fails, and no .bak is written -- while the seds below still flip the tags
# because they name files literally. Measured here: under zsh the first version
# left the working tree flipped and restored nothing, and it reported success
# either way, because a flipped tree is what it is meant to be testing.
set -- sweep_proc_linux.go sweep_proc_linux_test.go sweep_proc_other.go sweep_proc_other_test.go
cd "$PKG" || exit 1
for f in "$@"; do cp "$f" "$f.bak" || { printf 'could not back up %s\n' "$f" >&2; exit 2; }; done
# shellcheck disable=SC2317  # invoked by the trap below, not inline
restore() {
    cd "$PKG" || return 1
    for f in "$@"; do
        [ -f "$f.bak" ] && mv "$f.bak" "$f"
    done
}
trap 'restore sweep_proc_linux.go sweep_proc_linux_test.go sweep_proc_other.go sweep_proc_other_test.go' EXIT

sed -i 's|^//go:build linux$|//go:build proxy_excluded|' sweep_proc_linux.go sweep_proc_linux_test.go
sed -i 's|^//go:build !linux$|//go:build linux|' sweep_proc_other.go sweep_proc_other_test.go

printf 'tags after flip: %s | %s\n' \
    "$(head -1 sweep_proc_linux.go)" "$(head -1 sweep_proc_other.go)"

out="$(cd "$CLI" && go test ./internal/worktree/ -count=1 2>&1)"
rc=$?
printf '%s\n' "$out" | tail -5
if [ "$rc" -eq 0 ]; then
    printf 'RESULT: the Windows leg would be GREEN\n'
else
    printf 'RESULT: the Windows leg would be RED\n'
fi

# Post-condition. A proxy that leaves the tags flipped poisons every later run:
# the linux implementation is then excluded from the build, so mutations to it
# cannot fail and the mutation harness reports them as SURVIVED. That is exactly
# what happened before this check existed, and the tree looked fine at a glance.
restore sweep_proc_linux.go sweep_proc_linux_test.go sweep_proc_other.go sweep_proc_other_test.go
if [ "$(head -1 sweep_proc_linux.go)" != "//go:build linux" ] ||
   [ "$(head -1 sweep_proc_other.go)" != "//go:build !linux" ]; then
    printf 'RESTORE FAILED: build tags are still flipped. Run: git checkout -- %s\n' "$PKG" >&2
    exit 3
fi
exit "$rc"
