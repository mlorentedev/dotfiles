#!/usr/bin/env bats
# The repo's own .pre-commit-config.yaml. No installer: the global dispatcher
# (git-hooks/) hands every stage to `pre-commit hook-impl`, and `dotf tools
# install pre-commit` puts the binary on the machine.

# bats file_tags=os-sensitive

setup() {
    export DOTFILES_DIR="$BATS_TEST_DIRNAME/.."
    export SCRIPTS_DIR="$DOTFILES_DIR/scripts"
}

@test ".pre-commit-config.yaml declares sdd-spec-gate pre-push hook" {
    grep -q 'sdd-spec-gate' "$DOTFILES_DIR/.pre-commit-config.yaml"
    grep -A1 'sdd-spec-gate' "$DOTFILES_DIR/.pre-commit-config.yaml" | grep -q 'spec-gate'
    grep -B1 'pre-push' "$DOTFILES_DIR/.pre-commit-config.yaml" | grep -q 'sdd-spec-gate\|stages'
}

@test ".pre-commit-config.yaml declares check-bats-names at pre-commit stage [#925]" {
    # scripts/test.sh (the dotfiles-test hook) never ran bats or this script —
    # a non-ASCII @test name shipped past local pre-commit and was only
    # caught by a full adversarial-review round. This hook is the guard.
    grep -q 'check-bats-names' "$DOTFILES_DIR/.pre-commit-config.yaml"
    grep -A2 'id: check-bats-names' "$DOTFILES_DIR/.pre-commit-config.yaml" | grep -q 'check-bats-names.sh'
    grep -A5 'id: check-bats-names' "$DOTFILES_DIR/.pre-commit-config.yaml" | grep -q 'pre-commit'
}

@test "check-bats-names.sh exits 0 on this repo's own tests/ tree [#925]" {
    # The hook is only a guard if it actually passes on a clean tree — pins
    # that invariant so a future non-ASCII @test name is caught here, not
    # three rounds of review later.
    run "$SCRIPTS_DIR/check-bats-names.sh" "$DOTFILES_DIR/tests/"
    [ "$status" -eq 0 ]
}

# lint-powershell (#2678): the hook runs the pinned `dotf lint powershell`,
# which exits 2 where pwsh is absent, so the skip lives in the hook's entry.
# Read through pre-commit's own model (YAML, then shlex on `entry`), so the
# test runs the command pre-commit would run, not a copy of it.
lint_powershell_hook() {
    python3 - "$DOTFILES_DIR/.pre-commit-config.yaml" "$1" <<'PY'
import shlex, sys, yaml
hooks = [h for r in yaml.safe_load(open(sys.argv[1]))["repos"] for h in r["hooks"]]
h = next(h for h in hooks if h["id"] == "lint-powershell")
field = sys.argv[2]
if field == "argv":
    for a in shlex.split(h["entry"]):
        print(a)
else:
    print(h.get(field))
PY
}

# Runs the hook's entry with only $STUBS on PATH: a runner image that ships
# pwsh in /usr/bin (GitHub's ubuntu does) cannot leak into the no-pwsh case.
run_lint_powershell_hook() {
    local script
    script="$(lint_powershell_hook argv | sed -n 3p)"
    [[ "$(lint_powershell_hook argv | sed -n 1,2p | tr '\n' ' ')" == "sh -c " ]] || false
    run env PATH="$STUBS" /bin/sh -c "$script"
}

stub() {
    printf '#!/bin/sh\n%s\n' "$2" > "$STUBS/$1"
    chmod +x "$STUBS/$1"
}

@test "lint-powershell is a pre-commit hook on .ps1/.psm1 that takes no filenames [#2678]" {
    [[ "$(lint_powershell_hook language)" == "system" ]] || false
    [[ "$(lint_powershell_hook pass_filenames)" == "False" ]] || false
    [[ "$(lint_powershell_hook stages)" == "['pre-commit']" ]] || false
    files="$(lint_powershell_hook files)"
    for p in setup-windows.ps1 powershell/profile.ps1 powershell/Mod.psm1; do
        printf '%s\n' "$p" | grep -Eq "$files" || false
    done
    for p in setup-linux.sh docs/x.ps1.md; do
        ! printf '%s\n' "$p" | grep -Eq "$files" || false
    done
}

@test "lint-powershell skips with a message where pwsh is absent, and never runs dotf [#2678]" {
    STUBS="$(mktemp -d)"
    stub dotf 'echo called >"$0.log"; exit 2'
    run_lint_powershell_hook
    [[ "$status" -eq 0 ]] || false
    [[ "$output" == *pwsh* ]] || false
    [[ ! -e "$STUBS/dotf.log" ]] || false
}

@test "lint-powershell runs dotf lint powershell where pwsh is present and keeps its exit code [#2678]" {
    STUBS="$(mktemp -d)"
    stub pwsh 'exit 0'
    stub dotf 'echo "$*" >"$0.log"; exit 1'
    run_lint_powershell_hook
    [[ "$status" -eq 1 ]] || false
    [[ "$(cat "$STUBS/dotf.log")" == "lint powershell" ]] || false
}
