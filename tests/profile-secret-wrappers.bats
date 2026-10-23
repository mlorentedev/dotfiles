#!/usr/bin/env bats
# Behaviour of the PowerShell pi and opencode wrappers in powershell/profile.ps1
# (#2909). They put secrets into this session for as long as the TUI runs, so
# what they leave behind is the contract: the app is launched with the
# secrets, the injected values are gone afterwards, and a value the session
# held before survives. The tests observe only that, the environment before and
# after and what the app saw, so they still hold once the wrappers become a
# `dotf secrets run` call over a pseudo-console (SEC-009, #2296).
#
# The wrapper functions are taken from the real profile through the parser's
# AST, not by running the profile, so nothing else in it executes. `dotf` is a
# function stub returning fixed values; the app is a script on PATH that records
# the environment it was started with.

load 'winpath'
load 'lib/pwsh'

setup() {
    require_pwsh
    PROFILE_SCRIPT="$BATS_TEST_DIRNAME/../powershell/profile.ps1"
    BIN="$BATS_TEST_TMPDIR/bin"
    mkdir -p "$BIN"
    REPORT="$BATS_TEST_TMPDIR/report.txt"
}

# _run_wrapper <app> <powershell lines run before the wrapper>
# Defines the wrappers from the profile, stubs dotf, puts a recording <app> on
# PATH, runs the prelude, then `<app> --flag value`, and prints the environment
# afterwards as NAME=<value|ABSENT> lines.
_run_wrapper() {
    local app="$1" prelude="$2" script="$BATS_TEST_TMPDIR/run.ps1"
    cat > "$script" <<EOF
\$ErrorActionPreference = 'Stop'
\$ast = [System.Management.Automation.Language.Parser]::ParseFile('$(_winpath "$PROFILE_SCRIPT")', [ref]\$null, [ref]\$null)
\$defs = \$ast.FindAll({ param(\$n) \$n -is [System.Management.Automation.Language.FunctionDefinitionAst] }, \$true) |
    Where-Object { \$_.Name -in 'Invoke-WithSessionSecrets', 'pi', 'opencode' }
foreach (\$d in \$defs) { . ([scriptblock]::Create(\$d.Extent.Text)) }
function dotf { [Convert]::ToBase64String([Text.Encoding]::UTF8.GetBytes('nan-injected::::or-injected::::oai-injected')) }

\$bin = '$(_winpath "$BIN")'
\$report = '$(_winpath "$REPORT")'
\$env:REPORT = \$report
if (\$IsWindows) {
    Set-Content -Path (Join-Path \$bin '$app.cmd') -Value "@echo off\`r\`n> ""%REPORT%"" echo NAN=%NAN_API_KEY% OR=%OPENROUTER_API_KEY% OAI=%OPENAI_API_KEY% ARGS=%*"
} else {
    \$p = Join-Path \$bin '$app'
    Set-Content -Path \$p -Value "#!/bin/sh\`nprintf 'NAN=%s OR=%s OAI=%s ARGS=%s\\\\n' ""\`\$NAN_API_KEY"" ""\`\$OPENROUTER_API_KEY"" ""\`\$OPENAI_API_KEY"" ""\`\$*"" > ""\`\$REPORT"""
    chmod +x \$p
}
\$env:PATH = \$bin + [IO.Path]::PathSeparator + \$env:PATH
$prelude
try { $app --flag value } catch { "CAUGHT: \$_" }
foreach (\$n in 'NAN_API_KEY', 'OPENROUTER_API_KEY', 'OPENAI_API_KEY') {
    \$v = [Environment]::GetEnvironmentVariable(\$n)
    if (\$null -eq \$v) { "\$n=ABSENT" } else { "\$n=\$v" }
}
EOF
    run pwsh -NoProfile -NonInteractive -File "$(_winpath "$script")"
}

@test "pi: launched with the secrets, and a value the session held before survives" {
    _run_wrapper pi "\$env:NAN_API_KEY = 'PRE-EXISTING'; Remove-Item Env:\\OPENROUTER_API_KEY -ErrorAction SilentlyContinue"
    [ "$status" -eq 0 ] || { echo "$output"; return 1; }
    grep -q 'NAN=nan-injected OR=or-injected .*ARGS=--flag value' "$REPORT" \
        || { echo "the app did not see the injected secrets:"; cat "$REPORT"; return 1; }
    [[ "$output" == *"NAN_API_KEY=PRE-EXISTING"* ]] || { echo "a pre-existing NAN_API_KEY was not restored:"; echo "$output"; return 1; }
    [[ "$output" == *"OPENROUTER_API_KEY=ABSENT"* ]] || { echo "an injected secret outlived the app:"; echo "$output"; return 1; }
}

@test "opencode: every injected secret is gone afterwards, and pre-existing ones survive" {
    _run_wrapper opencode "\$env:OPENAI_API_KEY = 'MINE'; Remove-Item Env:\\NAN_API_KEY, Env:\\OPENROUTER_API_KEY -ErrorAction SilentlyContinue"
    [ "$status" -eq 0 ] || { echo "$output"; return 1; }
    grep -q 'NAN=nan-injected OR=or-injected OAI=oai-injected ARGS=--flag value' "$REPORT" \
        || { echo "the app did not see the injected secrets:"; cat "$REPORT"; return 1; }
    for want in NAN_API_KEY=ABSENT OPENROUTER_API_KEY=ABSENT OPENAI_API_KEY=MINE; do
        [[ "$output" == *"$want"* ]] || { echo "want $want after the app exits:"; echo "$output"; return 1; }
    done
}

@test "an app that fails to start still restores the session" {
    _run_wrapper pi "\$env:NAN_API_KEY = 'PRE-EXISTING'; function global:Get-Command { throw 'app lookup failed' }"
    [ "$status" -eq 0 ] || { echo "$output"; return 1; }
    [[ "$output" == *"CAUGHT: app lookup failed"* ]] || { echo "the failure did not reach the caller:"; echo "$output"; return 1; }
    for want in NAN_API_KEY=PRE-EXISTING OPENROUTER_API_KEY=ABSENT; do
        [[ "$output" == *"$want"* ]] || { echo "want $want after the failure:"; echo "$output"; return 1; }
    done
}
