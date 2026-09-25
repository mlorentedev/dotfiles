# Pester 5 contract for OPS-048 Windows SSH key recovery.

$script:onWindows = $env:OS -eq 'Windows_NT'

Describe 'Windows SSH key recovery' -Skip:(-not $script:onWindows) {
    BeforeAll {
        $script:Repo = Join-Path $PSScriptRoot '..'
        $script:ClientScript = Join-Path $script:Repo 'scripts\windows-ssh-client-key.ps1'
        $script:HostScript = Join-Path $script:Repo 'scripts\windows-openssh-authorize.ps1'
        $script:PublicKey = Join-Path $script:Repo 'ssh\id_ed25519_ts_bridge_acemagic.pub'
        $script:Registry = Join-Path $script:Repo 'secrets\registry.yaml'
        $script:SshConfig = Join-Path $script:Repo 'ssh\config'
        $script:Runbook = Join-Path $script:Repo 'docs\runbooks\windows-ssh-key-recovery.md'
        $script:Features = Join-Path $script:Repo 'specs\OPS-048-windows-ssh-key-recovery\features.json'
        $script:Sandbox = Join-Path ([IO.Path]::GetTempPath()) "dotfiles-ops048-$PID"
        New-Item -ItemType Directory -Path $script:Sandbox -Force | Out-Null
    }

    AfterAll {
        if ($script:Sandbox -and (Test-Path $script:Sandbox)) {
            Remove-Item $script:Sandbox -Recurse -Force
        }
    }

    Context 'client key guard' {
        It 'ships the client guard and committed public key' {
            $script:ClientScript | Should -Exist
            $script:PublicKey | Should -Exist
        }

        It 'accepts a matching private/public key pair and rejects a mismatch' {
            . $script:ClientScript
            $keyA = Join-Path $script:Sandbox 'key-a'
            $keyB = Join-Path $script:Sandbox 'key-b'
            & ssh-keygen -q -t ed25519 -N '' -f $keyA | Out-Null
            & ssh-keygen -q -t ed25519 -N '' -f $keyB | Out-Null

            { Assert-SshKeyPair -PrivateKeyPath $keyA -PublicKeyPath "$keyA.pub" } |
                Should -Not -Throw
            { Assert-SshKeyPair -PrivateKeyPath $keyA -PublicKeyPath "$keyB.pub" } |
                Should -Throw '*fingerprint mismatch*'
        }

        It 'rejects a private key that requires an interactive passphrase' {
            . $script:ClientScript
            $key = Join-Path $script:Sandbox 'encrypted-key'
            & ssh-keygen -q -t ed25519 -N 'test-passphrase' -f $key | Out-Null

            { Assert-NonInteractivePrivateKey -PrivateKeyPath $key } |
                Should -Throw '*requires a passphrase*'
        }

        It 'reconciles a protected owner and SYSTEM ACL' {
            . $script:ClientScript
            $calls = [Collections.Generic.List[object]]::new()
            $runner = {
                param([string[]]$Arguments)
                $calls.Add($Arguments)
                0
            }

            Set-PrivateKeyAcl -Path 'C:\keys\dedicated' -OwnerSid 'S-1-5-21-42' `
                -CommandRunner $runner

            $flat = ($calls | ForEach-Object { $_ -join ' ' }) -join "`n"
            $flat | Should -Match '/inheritance:r'
            $flat | Should -Match '\*S-1-5-21-42:F'
            $flat | Should -Match '\*S-1-5-18:F'
        }

        It 'removes every unrelated ACL entry from the private key' {
            . $script:ClientScript
            $key = Join-Path $script:Sandbox 'acl-key'
            Set-Content -LiteralPath $key -Value 'synthetic-private-key' -Encoding ascii
            & icacls.exe $key /grant '*S-1-5-32-544:F' | Out-Null
            $ownerSid = [Security.Principal.WindowsIdentity]::GetCurrent().User.Value

            Set-PrivateKeyAcl -Path $key -OwnerSid $ownerSid

            $acl = Get-Acl -LiteralPath $key
            $acl.AreAccessRulesProtected | Should -BeTrue
            $sids = @($acl.Access | ForEach-Object {
                $_.IdentityReference.Translate([Security.Principal.SecurityIdentifier]).Value
            } | Sort-Object -Unique)
            $sids | Should -Be @($ownerSid, 'S-1-5-18' | Sort-Object)
        }

        It 'repairs the private key ACL before loading the key' {
            . $script:ClientScript
            $calls = [Collections.Generic.List[string]]::new()
            $key = Join-Path $script:Sandbox 'reconcile-order'
            Set-Content -LiteralPath $key -Value 'synthetic-private-key' -Encoding ascii

            Mock Set-PrivateKeyAcl { $calls.Add('acl') }
            Mock Assert-SshKeyPair { $calls.Add('pair'); 'SHA256:test' }
            Mock Assert-NonInteractivePrivateKey { $calls.Add('interactive') }

            Invoke-SshClientKeyReconciliation -PrivateKeyPath $key -PublicKeyPath "$key.pub"

            $calls | Should -Be @('acl', 'pair', 'interactive')
        }
    }

    Context 'OpenSSH host reconciliation' {
        It 'ships the host reconciler' {
            $script:HostScript | Should -Exist
        }

        It 'adds the dedicated key once and preserves unrelated keys' {
            . $script:HostScript
            $authorized = Join-Path $script:Sandbox 'administrators_authorized_keys'
            $other = 'ssh-ed25519 AAAAC3NzaOtherKey operator@example'
            Set-Content -LiteralPath $authorized -Value $other -Encoding ascii
            $dedicated = (Get-Content -LiteralPath $script:PublicKey -Raw).Trim()

            Update-AuthorizedKeyFile -Path $authorized -PublicKey $dedicated | Should -BeTrue
            Update-AuthorizedKeyFile -Path $authorized -PublicKey $dedicated | Should -BeFalse

            $lines = @(Get-Content -LiteralPath $authorized)
            $lines | Should -Contain $other
            @($lines | Where-Object { $_ -eq $dedicated }).Count | Should -Be 1
        }

        It 'replaces the previous managed identity during rotation' {
            . $script:HostScript
            $authorized = Join-Path $script:Sandbox 'rotated_authorized_keys'
            $previous = 'ssh-ed25519 AAAAC3NzaPrevious ts-bridge-acemagic-admin'
            $other = 'ssh-ed25519 AAAAC3NzaOtherKey operator@example'
            Set-Content -LiteralPath $authorized -Value @($previous, $other) -Encoding ascii
            $replacement = (Get-Content -LiteralPath $script:PublicKey -Raw).Trim()

            Update-AuthorizedKeyFile -Path $authorized -PublicKey $replacement | Should -BeTrue

            $lines = @(Get-Content -LiteralPath $authorized)
            $lines | Should -Not -Contain $previous
            $lines | Should -Contain $other
            $lines | Should -Contain $replacement
        }

        It 'rejects private material instead of writing it as an authorized key' {
            . $script:HostScript
            { Assert-PublicKey -PublicKey '-----BEGIN OPENSSH PRIVATE KEY-----' } |
                Should -Throw '*public key*'
        }

        It 'reconciles the administrator key store ACL' {
            . $script:HostScript
            $calls = [Collections.Generic.List[object]]::new()
            $runner = {
                param([string[]]$Arguments)
                $calls.Add($Arguments)
                0
            }

            Set-AdministratorAuthorizedKeysAcl -Path 'C:\ProgramData\ssh\administrators_authorized_keys' `
                -CommandRunner $runner

            $flat = ($calls | ForEach-Object { $_ -join ' ' }) -join "`n"
            $flat | Should -Match '/inheritance:r'
            $flat | Should -Match '\*S-1-5-32-544:F'
            $flat | Should -Match '\*S-1-5-18:F'
        }
    }

    Context 'declarative recovery surfaces' {
        It 'declares the Bitwarden-backed private key file without private material' {
            $registry = Get-Content -LiteralPath $script:Registry -Raw
            $registry | Should -Match '(?m)^\s*- id: ACEMAGIC_OFFICE_SSH_KEY$'
            $registry | Should -Match 'item: acemagic-office-ssh'
            $registry | Should -Match 'field: notes'
            $registry | Should -Match 'path: "~/.ssh/id_ed25519_ts_bridge_acemagic"'
            $registry | Should -Not -Match 'BEGIN OPENSSH PRIVATE KEY'
        }

        It 'defines stable mesh and LAN aliases with the dedicated identity' {
            $config = Get-Content -LiteralPath $script:SshConfig -Raw
            $config | Should -Match '(?ms)^Host acemagic-office\s+.*?HostName acemagic-office\b'
            $config | Should -Match '(?ms)^Host acemagic-office-lan\s+.*?HostName 192\.168\.137\.245\b'
            @([regex]::Matches($config, 'IdentityFile ~/.ssh/id_ed25519_ts_bridge_acemagic')).Count |
                Should -Be 2
            @([regex]::Matches($config, '(?m)^\s+IdentitiesOnly yes$')).Count | Should -Be 2
        }

        It 'documents the complete lifecycle and the trust boundary' {
            $script:Runbook | Should -Exist
            $runbook = Get-Content -LiteralPath $script:Runbook -Raw
            foreach ($heading in @(
                'First trust bootstrap',
                'Normal reconciliation',
                'Clean-machine recovery',
                'Rotation and revocation',
                'Disaster recovery',
                'Verification'
            )) {
                $runbook | Should -Match ([regex]::Escape($heading))
            }
            $runbook | Should -Match 'already authenticated administrative channel'
            $runbook | Should -Match 'never.*private key.*command argument'
            $runbook | Should -Match 'PreferredAuthentications=publickey'
            $runbook | Should -Match 'BatchMode=yes'
            $runbook | Should -Match 'server host-key fingerprint'
            $runbook | Should -Match 'Update\s+the expected public fingerprint'
        }

        It 'requires Windows Pester evidence for every acceptance criterion' {
            $features = Get-Content -LiteralPath $script:Features -Raw | ConvertFrom-Json

            foreach ($feature in $features) {
                $feature.verification | Should -Match '\$IsWindows'
                $feature.verification | Should -Match 'PassedCount'
                $feature.verification | Should -Match 'throw'
            }
        }
    }
}
