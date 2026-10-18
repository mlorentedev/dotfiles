#!/usr/bin/env bats
# The real-dependency layer for tests/precommit-config.bats (BUG-055 pairing).
# That suite stubs `pwsh` and `dotf` to pin the lint-powershell hook's guard: it
# skips where pwsh is absent and hands over to `dotf lint powershell` where it
# is present. A stub accepts any invocation, so only the real tools can say that
# pre-commit parses the entry, that the real dotf runs the real pwsh through it,
# and that a failing lint fails the hook.
#
# The hook is copied out of the repo's own config, so this runs the entry the
# repo ships. dotf is built from this tree (CI exports DOTF_BIN, and a build that
# fails is a failure: tests/lib/dotf-bin.bash). GitHub's ubuntu runners ship
# pwsh but not PSScriptAnalyzer, so CI sees the "analyzer not at its pin" exit 2
# and a box with the analyzer sees the probe's finding (exit 1). Both must fail
# the hook, and neither may take the no-pwsh skip. Where pre-commit is absent,
# or pwsh is outside a job that promises it (tests/lib/pwsh.bash), the test
# skips: that box is the stubbed suite's case.

load 'lib/dotf-bin'
load 'lib/pwsh'

setup() {
    command -v pre-commit >/dev/null 2>&1 || skip "pre-commit not installed"
    # A box without pwsh is the stubbed suite's case: the hook skips there.
    require_pwsh
    REPO="$BATS_TEST_DIRNAME/.."
    dotf_bin_resolve "${BATS_FILE_TMPDIR:-/tmp/bats_pcr_$$}/bin/dotf"
    WORK="$(mktemp -d)"
    FIXTURE="$WORK/repo"
    mkdir -p "$FIXTURE" "$WORK/bin"
    # The entry calls `dotf` by name; link the tree's build under that name.
    ln -s "$DOTF_BIN" "$WORK/bin/dotf"
    git -C "$FIXTURE" init -q
    git -C "$FIXTURE" config user.email t@t.io
    git -C "$FIXTURE" config user.name tester
    cp "$REPO/versions.conf" "$REPO/.PSScriptAnalyzerSettings.psd1" "$FIXTURE/"
    python3 - "$REPO/.pre-commit-config.yaml" "$FIXTURE/.pre-commit-config.yaml" <<'PY'
import sys, yaml
hooks = [h for r in yaml.safe_load(open(sys.argv[1]))["repos"] for h in r["hooks"]]
hook = next(h for h in hooks if h["id"] == "lint-powershell")
yaml.safe_dump({"repos": [{"repo": "local", "hooks": [hook]}]}, open(sys.argv[2], "w"))
PY
}

teardown() { [ -z "${WORK:-}" ] || rm -rf "$WORK"; }

@test "real pre-commit runs the real dotf lint powershell through the hook, and a failing lint fails it [#2678]" {
    printf '%s\n' 'Invoke-Expression "Get-Date"' > "$FIXTURE/probe.ps1"
    git -C "$FIXTURE" add -A
    cd "$FIXTURE"
    run env PATH="$WORK/bin:$PATH" pre-commit run lint-powershell --all-files
    [[ "$status" -ne 0 ]] || false
    [[ "$output" != *"skipping lint-powershell"* ]] || false
    [[ "$output" == *PSAvoidUsingInvokeExpression* || "$output" == *"is not installed: run 'dotf converge --only pwsh-modules'"* ]] || false
}
