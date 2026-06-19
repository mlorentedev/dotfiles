# Pester 5 behavioral tests for scripts/orca-hook-tune.ps1 (DX-006).
#
# The script only manipulates files, so these run on any OS with pwsh: every
# case writes throwaway sample hooks under a per-test temp dir via the script's
# -HookConfig / -HookScript seams, so a real Orca install is never touched.

Describe 'orca-hook-tune.ps1 (DX-006 Orca hook fix)' {

    BeforeAll {
        $script:Target = Join-Path $PSScriptRoot '..\scripts\orca-hook-tune.ps1'

        # Orca's generated artifacts, reproduced exactly enough to exercise the patch.
        $script:OrcaJson = @'
{
  "version": 1,
  "hooks": {
    "PreToolUse": [
      { "type": "command", "powershell": "stub", "timeoutSec": 5 }
    ],
    "SessionStart": [
      { "type": "command", "powershell": "stub", "timeoutSec": 5 }
    ]
  }
}
'@
        $script:OrcaHook = @'
Write-Output '{}'
if (-not $env:ORCA_AGENT_HOOK_PORT) { exit 0 }
try {
  $body = @{ a = 1 } | ConvertTo-Json -Depth 100
  Invoke-WebRequest -UseBasicParsing -Method Post -Uri ('http://127.0.0.1:' + $env:ORCA_AGENT_HOOK_PORT + '/hook/copilot') -Headers @{ 'Content-Type'='application/json'; 'X-Orca-Agent-Hook-Token'=$env:ORCA_AGENT_HOOK_TOKEN } -Body $body -TimeoutSec 2 | Out-Null
} catch {}
exit 0
'@
    }

    BeforeEach {
        $script:Dir  = Join-Path ([System.IO.Path]::GetTempPath()) "orca-hook-test-$PID-$([guid]::NewGuid().ToString('N').Substring(0,8))"
        New-Item -ItemType Directory -Path $script:Dir -Force | Out-Null
        $script:Cfg  = Join-Path $script:Dir 'orca.json'
        $script:Scr  = Join-Path $script:Dir 'copilot-hook.ps1'
    }

    AfterEach {
        if ($script:Dir -and (Test-Path $script:Dir)) { Remove-Item $script:Dir -Recurse -Force }
    }

    It 'skips cleanly when neither file exists' {
        $out = & $script:Target -HookConfig $script:Cfg -HookScript $script:Scr *>&1 | Out-String
        $LASTEXITCODE | Should -Be 0
        $out | Should -Match 'nothing to do'
    }

    It 'bumps orca.json timeout and rewrites copilot-hook.ps1 to HttpWebRequest' {
        Set-Content -LiteralPath $script:Cfg -Value $script:OrcaJson -Encoding UTF8
        Set-Content -LiteralPath $script:Scr -Value $script:OrcaHook -Encoding UTF8

        & $script:Target -HookConfig $script:Cfg -HookScript $script:Scr *>&1 | Out-Null
        $LASTEXITCODE | Should -Be 0

        $cfg = Get-Content -LiteralPath $script:Cfg -Raw
        $cfg | Should -Not -Match '"timeoutSec"\s*:\s*5\b'
        $cfg | Should -Match '"timeoutSec": 30'

        $scr = Get-Content -LiteralPath $script:Scr -Raw
        $scr | Should -Not -Match 'Invoke-WebRequest'
        $scr | Should -Match 'HttpWebRequest'
        # surrounding logic preserved
        $scr | Should -Match "if \(-not \`$env:ORCA_AGENT_HOOK_PORT\)"
    }

    It 'is idempotent: a second run changes nothing' {
        Set-Content -LiteralPath $script:Cfg -Value $script:OrcaJson -Encoding UTF8
        Set-Content -LiteralPath $script:Scr -Value $script:OrcaHook -Encoding UTF8

        & $script:Target -HookConfig $script:Cfg -HookScript $script:Scr *>&1 | Out-Null
        $out = & $script:Target -HookConfig $script:Cfg -HookScript $script:Scr *>&1 | Out-String
        $out | Should -Match 'already tuned'
        $out | Should -Not -Match 'bumped'
    }

    It 'leaves an already-generous timeout untouched' {
        Set-Content -LiteralPath $script:Cfg -Value ($script:OrcaJson -replace '"timeoutSec": 5', '"timeoutSec": 30') -Encoding UTF8
        $out = & $script:Target -HookConfig $script:Cfg -HookScript $script:Scr *>&1 | Out-String
        $out | Should -Match 'already >= 30'
    }

    It 'writes a timestamped backup before changing a file' {
        Set-Content -LiteralPath $script:Cfg -Value $script:OrcaJson -Encoding UTF8
        & $script:Target -HookConfig $script:Cfg -HookScript $script:Scr *>&1 | Out-Null
        (Get-ChildItem -Path $script:Dir -Filter 'orca.json.bak.*') | Should -Not -BeNullOrEmpty
    }

    It 'Check mode exits 1 on drift and 0 once clean' {
        Set-Content -LiteralPath $script:Cfg -Value $script:OrcaJson -Encoding UTF8
        Set-Content -LiteralPath $script:Scr -Value $script:OrcaHook -Encoding UTF8

        & $script:Target -Check -HookConfig $script:Cfg -HookScript $script:Scr *>&1 | Out-Null
        $LASTEXITCODE | Should -Be 1

        & $script:Target -HookConfig $script:Cfg -HookScript $script:Scr *>&1 | Out-Null
        & $script:Target -Check -HookConfig $script:Cfg -HookScript $script:Scr *>&1 | Out-Null
        $LASTEXITCODE | Should -Be 0
    }
}
