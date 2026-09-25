[CmdletBinding()]
param(
    [string]$PublicKeyPath = (Join-Path $PSScriptRoot '..\ssh\id_ed25519_ts_bridge_acemagic.pub'),
    [string]$AuthorizedKeysPath = "$env:ProgramData\ssh\administrators_authorized_keys",
    [switch]$SkipSystemConfiguration,
    [switch]$SkipAcl
)

$ErrorActionPreference = 'Stop'

function Assert-PublicKey {
    param([Parameter(Mandatory)][string]$PublicKey)

    $key = $PublicKey.Trim()
    if ($key -notmatch '^(ssh-ed25519|ssh-rsa|ecdsa-sha2-nistp\d+)\s+[A-Za-z0-9+/=]+(?:\s+.*)?$') {
        throw 'Input must be one OpenSSH public key, never private key material.'
    }
    $key
}

function Get-PublicKeyIdentity {
    param([Parameter(Mandatory)][string]$PublicKey)

    ((Assert-PublicKey -PublicKey $PublicKey) -split '\s+', 3)[0..1] -join ' '
}

function Get-PublicKeyComment {
    param([Parameter(Mandatory)][string]$PublicKey)

    $parts = (Assert-PublicKey -PublicKey $PublicKey) -split '\s+', 3
    if ($parts.Count -eq 3) { $parts[2] } else { '' }
}

function Update-AuthorizedKeyFile {
    param(
        [Parameter(Mandatory)][string]$Path,
        [Parameter(Mandatory)][string]$PublicKey,
        [scriptblock]$AclCommandRunner
    )

    $key = Assert-PublicKey -PublicKey $PublicKey
    $identity = Get-PublicKeyIdentity -PublicKey $key
    $comment = Get-PublicKeyComment -PublicKey $key
    $existing = if (Test-Path -LiteralPath $Path) {
        @(Get-Content -LiteralPath $Path | Where-Object { $_.Trim() })
    } else {
        @()
    }
    $preserved = [Collections.Generic.List[string]]::new()
    $managedPresent = $false
    foreach ($line in $existing) {
        try {
            if ($line -eq $key -and -not $managedPresent) {
                $preserved.Add($line)
                $managedPresent = $true
                continue
            }

            $sameIdentity = (Get-PublicKeyIdentity -PublicKey $line) -eq $identity
            $sameManagedComment = $comment -and
                (Get-PublicKeyComment -PublicKey $line) -eq $comment
            if (-not ($sameIdentity -or $sameManagedComment)) {
                $preserved.Add($line)
            }
        } catch {
            $preserved.Add($line)
        }
    }
    if (-not $managedPresent) {
        $preserved.Add($key)
    }
    $desired = @($preserved)
    if (($existing -join "`n") -eq ($desired -join "`n")) {
        return $false
    }

    $directory = Split-Path -Parent $Path
    New-Item -ItemType Directory -Path $directory -Force | Out-Null
    $temporary = Join-Path $directory ".$([IO.Path]::GetFileName($Path)).$PID.tmp"
    New-Item -ItemType File -Path $temporary -Force | Out-Null
    Set-AdministratorAuthorizedKeysAcl -Path $temporary -CommandRunner $AclCommandRunner
    Set-Content -LiteralPath $temporary -Value $desired -Encoding ascii
    Move-Item -LiteralPath $temporary -Destination $Path -Force
    $true
}

function Set-AdministratorAuthorizedKeysAcl {
    param(
        [Parameter(Mandatory)][string]$Path,
        [scriptblock]$CommandRunner
    )

    if ($CommandRunner) {
        Invoke-AclCommand -Runner $CommandRunner -Arguments @(
            $Path,
            '/inheritance:r',
            '/grant:r', '*S-1-5-32-544:F',
            '/grant:r', '*S-1-5-18:F'
        )
        return
    }

    $CommandRunner = {
        param([string[]]$Arguments)
        & icacls.exe @Arguments | Out-Null
        $LASTEXITCODE
    }

    $administrators = 'S-1-5-32-544'
    $system = 'S-1-5-18'
    $currentAcl = Get-Acl -LiteralPath $Path
    $currentOwner = ([Security.Principal.NTAccount]$currentAcl.Owner).
        Translate([Security.Principal.SecurityIdentifier]).Value
    if ($currentOwner -ne $administrators) {
        Invoke-AclCommand -Runner $CommandRunner -Arguments @(
            $Path, '/setowner', "*$administrators"
        )
    }

    Invoke-AclCommand -Runner $CommandRunner -Arguments @($Path, '/inheritance:r')
    $allowed = @($administrators, $system)
    $currentSids = @($currentAcl.Access | ForEach-Object {
        $_.IdentityReference.Translate([Security.Principal.SecurityIdentifier]).Value
    } | Sort-Object -Unique)
    foreach ($sid in $currentSids | Where-Object { $_ -notin $allowed }) {
        Invoke-AclCommand -Runner $CommandRunner -Arguments @($Path, '/remove:g', "*$sid")
        Invoke-AclCommand -Runner $CommandRunner -Arguments @($Path, '/remove:d', "*$sid")
    }
    Invoke-AclCommand -Runner $CommandRunner -Arguments @(
        $Path,
        '/grant:r', "*${administrators}:F",
        '/grant:r', "*${system}:F"
    )
}

function Invoke-AclCommand {
    param(
        [Parameter(Mandatory)][scriptblock]$Runner,
        [Parameter(Mandatory)][string[]]$Arguments
    )

    $exitCode = & $Runner $Arguments
    if ($exitCode -ne 0) {
        throw "ACL reconciliation failed for '$($Arguments[0])' (exit $exitCode)."
    }
}

function Enable-WindowsOpenSshServer {
    $capability = Get-WindowsCapability -Online -Name 'OpenSSH.Server~~~~0.0.1.0'
    if ($capability.State -ne 'Installed') {
        Add-WindowsCapability -Online -Name $capability.Name | Out-Null
    }

    Set-Service -Name sshd -StartupType Automatic
    if ((Get-Service -Name sshd).Status -ne 'Running') {
        Start-Service -Name sshd
    }

    $rule = Get-NetFirewallRule -Name 'OpenSSH-Server-In-TCP' -ErrorAction SilentlyContinue
    if ($null -eq $rule) {
        New-NetFirewallRule -Name 'OpenSSH-Server-In-TCP' -DisplayName 'OpenSSH Server (sshd)' `
            -Enabled True -Direction Inbound -Protocol TCP -Action Allow -LocalPort 22 | Out-Null
    } elseif ($rule.Enabled -ne 'True') {
        Enable-NetFirewallRule -Name 'OpenSSH-Server-In-TCP'
    }
}

function Invoke-OpenSshHostReconciliation {
    param(
        [Parameter(Mandatory)][string]$PublicKeyPath,
        [Parameter(Mandatory)][string]$AuthorizedKeysPath,
        [switch]$SkipSystemConfiguration,
        [switch]$SkipAcl
    )

    if (-not (Test-Path -LiteralPath $PublicKeyPath -PathType Leaf)) {
        throw "Public key not found: $PublicKeyPath"
    }
    if (-not $SkipSystemConfiguration) {
        Enable-WindowsOpenSshServer
    }

    $key = Get-Content -LiteralPath $PublicKeyPath -Raw
    $changed = Update-AuthorizedKeyFile -Path $AuthorizedKeysPath -PublicKey $key
    if (-not $SkipAcl) {
        Set-AdministratorAuthorizedKeysAcl -Path $AuthorizedKeysPath
    }
    Write-Host "OpenSSH host reconciled (authorized key changed: $changed)"
}

if ($MyInvocation.InvocationName -ne '.') {
    Invoke-OpenSshHostReconciliation -PublicKeyPath $PublicKeyPath `
        -AuthorizedKeysPath $AuthorizedKeysPath `
        -SkipSystemConfiguration:$SkipSystemConfiguration -SkipAcl:$SkipAcl
}
