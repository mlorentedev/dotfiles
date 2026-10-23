# Pester 5: behaviour of the pi and opencode wrappers in powershell/profile.ps1
# (#2909). They put secrets into this session for as long as the TUI runs, so
# what they leave behind is the contract: the app is launched with the secrets,
# the injected values are gone afterwards, and a value the session held before
# survives. The tests observe only that, the environment before and after and
# what the app saw, so they still hold once the wrappers become a
# `dotf secrets run` call over a pseudo-console (SEC-009, #2296).
#
# Pester, not bats, because Windows is where these wrappers run and CI runs
# every *.Tests.ps1 there. Each case runs in a child `pwsh -NoProfile`, so the
# environment it changes is never this process's. The wrapper functions come
# from the real profile through the parser, so nothing else in it executes.
# `dotf` is a function stub returning fixed values; the app is a script on PATH
# that records the environment it was started with.

BeforeAll {
    $script:ProfilePath = (Resolve-Path (Join-Path $PSScriptRoot '..\powershell\profile.ps1')).Path

    $script:Runner = @'
param(
    [Parameter(Mandatory)][string]$ProfilePath,
    [Parameter(Mandatory)][string]$Bin,
    [Parameter(Mandatory)][string]$Report,
    [Parameter(Mandatory)][string]$App,
    [string]$Prelude = ''
)
$ErrorActionPreference = 'Stop'
$ast = [System.Management.Automation.Language.Parser]::ParseFile($ProfilePath, [ref]$null, [ref]$null)
$ast.FindAll({ $args[0] -is [System.Management.Automation.Language.FunctionDefinitionAst] }, $true) |
    Where-Object { $_.Name -in 'Invoke-WithSessionSecrets', 'pi', 'opencode' } |
    ForEach-Object { . ([scriptblock]::Create($_.Extent.Text)) }
function dotf { [Convert]::ToBase64String([Text.Encoding]::UTF8.GetBytes('nan-injected::::or-injected::::oai-injected')) }

$env:REPORT = $Report
if ($IsWindows -or $env:OS -eq 'Windows_NT') {
    Set-Content -Path (Join-Path $Bin "$App.cmd") -Value "@echo off`r`n> ""%REPORT%"" echo NAN=%NAN_API_KEY% OR=%OPENROUTER_API_KEY% OAI=%OPENAI_API_KEY% ARGS=%*"
} else {
    $p = Join-Path $Bin $App
    Set-Content -Path $p -Value "#!/bin/sh`nprintf 'NAN=%s OR=%s OAI=%s ARGS=%s\n' ""`$NAN_API_KEY"" ""`$OPENROUTER_API_KEY"" ""`$OPENAI_API_KEY"" ""`$*"" > ""`$REPORT"""
    chmod +x $p
}
$env:PATH = $Bin + [IO.Path]::PathSeparator + $env:PATH
if ($Prelude) { . ([scriptblock]::Create($Prelude)) }
try { & $App --flag value } catch { "CAUGHT: $_" }
foreach ($n in 'NAN_API_KEY', 'OPENROUTER_API_KEY', 'OPENAI_API_KEY') {
    $v = [Environment]::GetEnvironmentVariable($n)
    if ($null -eq $v) { "$n=ABSENT" } else { "$n=$v" }
}
'@

    # Runs <App> through its wrapper after <Prelude>, and returns what the app
    # recorded and the NAME=<value|ABSENT> lines the session held afterwards.
    function Invoke-Wrapper {
        param([string]$App, [string]$Prelude)
        $dir = Join-Path ([IO.Path]::GetTempPath()) ('wrapper-' + [IO.Path]::GetRandomFileName())
        $bin = Join-Path $dir 'bin'
        New-Item -ItemType Directory -Path $bin -Force | Out-Null
        $runner = Join-Path $dir 'run.ps1'
        $report = Join-Path $dir 'report.txt'
        Set-Content -LiteralPath $runner -Value $script:Runner -Encoding UTF8
        try {
            $out = & (Get-Process -Id $PID).Path -NoProfile -NonInteractive -File $runner `
                -ProfilePath $script:ProfilePath -Bin $bin -Report $report -App $App -Prelude $Prelude 2>&1
            [pscustomobject]@{
                ExitCode = $LASTEXITCODE
                Output   = ($out | Out-String)
                Report   = if (Test-Path $report) { (Get-Content -Raw $report).Trim() } else { '' }
            }
        } finally {
            Remove-Item -LiteralPath $dir -Recurse -Force -ErrorAction SilentlyContinue
        }
    }
}

Describe 'powershell/profile.ps1 secret wrappers leave the session as they found it' {
    It 'pi: launched with the secrets, and a value the session held before survives' {
        $r = Invoke-Wrapper -App pi -Prelude "`$env:NAN_API_KEY = 'PRE-EXISTING'; Remove-Item Env:OPENROUTER_API_KEY -ErrorAction SilentlyContinue"
        $r.ExitCode | Should -Be 0 -Because $r.Output
        $r.Report | Should -Match 'NAN=nan-injected OR=or-injected .*ARGS=--flag value'
        $r.Output | Should -Match 'NAN_API_KEY=PRE-EXISTING'
        $r.Output | Should -Match 'OPENROUTER_API_KEY=ABSENT'
    }

    It 'opencode: every injected secret is gone afterwards, and pre-existing ones survive' {
        $r = Invoke-Wrapper -App opencode -Prelude "`$env:OPENAI_API_KEY = 'MINE'; Remove-Item Env:NAN_API_KEY, Env:OPENROUTER_API_KEY -ErrorAction SilentlyContinue"
        $r.ExitCode | Should -Be 0 -Because $r.Output
        $r.Report | Should -Match 'NAN=nan-injected OR=or-injected OAI=oai-injected ARGS=--flag value'
        $r.Output | Should -Match 'NAN_API_KEY=ABSENT'
        $r.Output | Should -Match 'OPENROUTER_API_KEY=ABSENT'
        $r.Output | Should -Match 'OPENAI_API_KEY=MINE'
    }

    It 'an app that fails to start still restores the session' {
        $r = Invoke-Wrapper -App pi -Prelude "`$env:NAN_API_KEY = 'PRE-EXISTING'; function global:Get-Command { throw 'app lookup failed' }"
        $r.ExitCode | Should -Be 0 -Because $r.Output
        $r.Output | Should -Match 'CAUGHT: app lookup failed'
        $r.Output | Should -Match 'NAN_API_KEY=PRE-EXISTING'
        $r.Output | Should -Match 'OPENROUTER_API_KEY=ABSENT'
    }
}
