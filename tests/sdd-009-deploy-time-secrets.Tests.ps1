# Pester tests for SDD-009: Substitute-EnvPlaceholders in scripts/utils.ps1.
#
# Cross-OS parity with tests/sdd-009-deploy-time-secrets.bats. Run on Windows
# (or via pwsh on WSL with age + age-keygen installed). WIN-004 will pick this
# up in CI on the windows-latest runner.
#
# Requires: Pester 5+, age + age-keygen on PATH.
#
# ASCII-only by repo convention (pattern-powershell-ascii-only).

BeforeAll {
    $script:DotfilesDir = Split-Path -Parent $PSScriptRoot
    $script:ScriptsDir = Join-Path $script:DotfilesDir 'scripts'
    . (Join-Path $script:ScriptsDir 'utils.ps1')

    $script:AgeMissing = (-not (Get-Command age -ErrorAction SilentlyContinue)) -or
                        (-not (Get-Command age-keygen -ErrorAction SilentlyContinue))
}

Describe 'Substitute-EnvPlaceholders' -Skip:$script:AgeMissing {

    BeforeEach {
        $script:FixDir = Join-Path ([System.IO.Path]::GetTempPath()) "sdd009_${PID}_$(Get-Random)"
        New-Item -ItemType Directory -Path $script:FixDir -Force | Out-Null
        $script:SecretsDir = Join-Path $script:FixDir 'sensitive'
        New-Item -ItemType Directory -Path $script:SecretsDir -Force | Out-Null

        $script:KeyPath = Join-Path $script:FixDir 'key.txt'
        & age-keygen -o $script:KeyPath 2>$null
        $pubkey = (Select-String -Path $script:KeyPath -Pattern 'age1[0-9a-z]+').Matches.Value | Select-Object -First 1

        # Encrypt the NAN test value.
        $secretFile = Join-Path $script:SecretsDir 'nan.api-key.secret.age'
        'FIXTURE-NAN-VALUE-ONE' | & age -r $pubkey -o $secretFile

        $script:MappingFile = Join-Path $script:SecretsDir 'env-mapping.conf'
        @(
            'NAN_API_KEY=nan.api-key'
            '# OLLAMA_API_KEY=ollama.api-key'
        ) | Set-Content -LiteralPath $script:MappingFile -Encoding UTF8

        $script:TargetFile = Join-Path $script:FixDir 'opencode.jsonc'
        @(
            '{'
            '  "provider": {'
            '    "nan":    { "options": { "apiKey": "{env:NAN_API_KEY}" } },'
            '    "ollama": { "options": { "apiKey": "{env:OLLAMA_API_KEY}" } }'
            '  }'
            '}'
        ) | Set-Content -LiteralPath $script:TargetFile -Encoding UTF8
    }

    AfterEach {
        if (Test-Path -LiteralPath $script:FixDir) {
            Remove-Item -LiteralPath $script:FixDir -Recurse -Force
        }
    }

    It 'is exported from utils.ps1' {
        Get-Command Substitute-EnvPlaceholders -ErrorAction SilentlyContinue | Should -Not -BeNullOrEmpty
    }

    It 'substitutes resolvable {env:NAN_API_KEY} with decrypted value' {
        $result = Substitute-EnvPlaceholders -Path $script:TargetFile `
            -SecretsDir $script:SecretsDir `
            -MappingFile $script:MappingFile `
            -KeyPath $script:KeyPath
        $result | Should -BeTrue
        (Get-Content -LiteralPath $script:TargetFile -Raw) | Should -Match '"apiKey": "FIXTURE-NAN-VALUE-ONE"'
    }

    It 'leaves unresolvable {env:OLLAMA_API_KEY} placeholder intact' {
        Substitute-EnvPlaceholders -Path $script:TargetFile `
            -SecretsDir $script:SecretsDir `
            -MappingFile $script:MappingFile `
            -KeyPath $script:KeyPath | Out-Null
        $content = Get-Content -LiteralPath $script:TargetFile -Raw
        $content | Should -Match '"apiKey": "FIXTURE-NAN-VALUE-ONE"'
        $content | Should -Match '\{env:OLLAMA_API_KEY\}'
    }

    It 'emits a warning naming the unresolved placeholder' {
        $warn = Substitute-EnvPlaceholders -Path $script:TargetFile `
            -SecretsDir $script:SecretsDir `
            -MappingFile $script:MappingFile `
            -KeyPath $script:KeyPath 3>&1
        ($warn | Out-String) | Should -Match 'OLLAMA_API_KEY'
    }

    It 'is idempotent under secret rotation' {
        # First pass.
        Substitute-EnvPlaceholders -Path $script:TargetFile `
            -SecretsDir $script:SecretsDir `
            -MappingFile $script:MappingFile `
            -KeyPath $script:KeyPath | Out-Null
        (Get-Content -LiteralPath $script:TargetFile -Raw) | Should -Match 'FIXTURE-NAN-VALUE-ONE'

        # Rotate the secret.
        $pubkey = (Select-String -Path $script:KeyPath -Pattern 'age1[0-9a-z]+').Matches.Value | Select-Object -First 1
        $secretFile = Join-Path $script:SecretsDir 'nan.api-key.secret.age'
        'FIXTURE-NAN-VALUE-ROTATED' | & age -r $pubkey -o $secretFile

        # Restore source-style placeholders to simulate re-deploy.
        @(
            '{'
            '  "provider": {'
            '    "nan":    { "options": { "apiKey": "{env:NAN_API_KEY}" } },'
            '    "ollama": { "options": { "apiKey": "{env:OLLAMA_API_KEY}" } }'
            '  }'
            '}'
        ) | Set-Content -LiteralPath $script:TargetFile -Encoding UTF8

        Substitute-EnvPlaceholders -Path $script:TargetFile `
            -SecretsDir $script:SecretsDir `
            -MappingFile $script:MappingFile `
            -KeyPath $script:KeyPath | Out-Null
        $content = Get-Content -LiteralPath $script:TargetFile -Raw
        $content | Should -Match 'FIXTURE-NAN-VALUE-ROTATED'
        $content | Should -Not -Match 'FIXTURE-NAN-VALUE-ONE'
    }

    It 'is a no-op when file contains zero {env:} tokens' {
        '{ "model": "gpt-4", "temperature": 0.7 }' | Set-Content -LiteralPath $script:TargetFile -Encoding UTF8
        $beforeHash = (Get-FileHash -LiteralPath $script:TargetFile -Algorithm SHA256).Hash
        Substitute-EnvPlaceholders -Path $script:TargetFile `
            -SecretsDir $script:SecretsDir `
            -MappingFile $script:MappingFile `
            -KeyPath $script:KeyPath | Out-Null
        $afterHash = (Get-FileHash -LiteralPath $script:TargetFile -Algorithm SHA256).Hash
        $afterHash | Should -Be $beforeHash
    }

    It 'fails gracefully on missing target file' {
        $result = Substitute-EnvPlaceholders -Path (Join-Path $script:FixDir 'nope.jsonc') `
            -SecretsDir $script:SecretsDir `
            -MappingFile $script:MappingFile `
            -KeyPath $script:KeyPath
        $result | Should -BeFalse
    }

    It 'produces byte-equivalent output to the bash helper (cross-OS parity)' {
        # Same fixture as the bats test: NAN substituted, OLLAMA intact.
        Substitute-EnvPlaceholders -Path $script:TargetFile `
            -SecretsDir $script:SecretsDir `
            -MappingFile $script:MappingFile `
            -KeyPath $script:KeyPath | Out-Null
        $content = Get-Content -LiteralPath $script:TargetFile -Raw
        $content | Should -Match '"apiKey": "FIXTURE-NAN-VALUE-ONE"'
        $content | Should -Match '"apiKey": "\{env:OLLAMA_API_KEY\}"'
    }
}
