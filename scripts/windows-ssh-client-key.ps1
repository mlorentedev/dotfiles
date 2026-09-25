[CmdletBinding()]
param(
    [string]$PrivateKeyPath = "$HOME\.ssh\id_ed25519_ts_bridge_acemagic",
    [string]$PublicKeyPath = (Join-Path $PSScriptRoot '..\ssh\id_ed25519_ts_bridge_acemagic.pub'),
    [switch]$SkipAcl
)

$ErrorActionPreference = 'Stop'

function Get-SshKeyFingerprint {
    param([Parameter(Mandatory)][string]$Path)

    $output = & ssh-keygen -lf $Path 2>&1
    if ($LASTEXITCODE -ne 0) {
        throw "Unable to read SSH key fingerprint for '$Path': $output"
    }
    if ($output -notmatch '^\d+\s+(SHA256:[^\s]+)\s+') {
        throw "Unexpected ssh-keygen fingerprint output for '$Path'."
    }
    $Matches[1]
}

function Assert-SshKeyPair {
    param(
        [Parameter(Mandatory)][string]$PrivateKeyPath,
        [Parameter(Mandatory)][string]$PublicKeyPath
    )

    if (-not (Test-Path -LiteralPath $PrivateKeyPath -PathType Leaf)) {
        throw "Private key not found: $PrivateKeyPath"
    }
    if (-not (Test-Path -LiteralPath $PublicKeyPath -PathType Leaf)) {
        throw "Public key not found: $PublicKeyPath"
    }

    $privateFingerprint = Get-SshKeyFingerprint -Path $PrivateKeyPath
    $publicFingerprint = Get-SshKeyFingerprint -Path $PublicKeyPath
    if ($privateFingerprint -ne $publicFingerprint) {
        throw "SSH key fingerprint mismatch: private=$privateFingerprint public=$publicFingerprint"
    }
    $privateFingerprint
}

function Assert-NonInteractivePrivateKey {
    param([Parameter(Mandatory)][string]$PrivateKeyPath)

    $output = & ssh-keygen -y -P '' -f $PrivateKeyPath 2>&1
    if ($LASTEXITCODE -eq 0) {
        return
    }
    if ($output -match 'incorrect passphrase') {
        throw "Private key '$PrivateKeyPath' requires a passphrase and cannot be used non-interactively."
    }
    throw "Unable to validate non-interactive private key '$PrivateKeyPath': $output"
}

function Set-PrivateKeyAcl {
    param(
        [Parameter(Mandatory)][string]$Path,
        [Parameter(Mandatory)][string]$OwnerSid,
        [scriptblock]$CommandRunner
    )

    if ($CommandRunner) {
        Invoke-AclCommand -Runner $CommandRunner -Arguments @(
            $Path,
            '/inheritance:r',
            '/grant:r', "*${OwnerSid}:F",
            '/grant:r', '*S-1-5-18:F'
        )
        return
    }

    $CommandRunner = {
        param([string[]]$Arguments)
        & icacls.exe @Arguments | Out-Null
        $LASTEXITCODE
    }

    $currentAcl = Get-Acl -LiteralPath $Path
    $currentOwner = ([Security.Principal.NTAccount]$currentAcl.Owner).
        Translate([Security.Principal.SecurityIdentifier]).Value
    if ($currentOwner -ne $OwnerSid) {
        Invoke-AclCommand -Runner $CommandRunner -Arguments @($Path, '/setowner', "*$OwnerSid")
    }

    Invoke-AclCommand -Runner $CommandRunner -Arguments @($Path, '/inheritance:r')
    $allowed = @($OwnerSid, 'S-1-5-18')
    $currentSids = @($currentAcl.Access | ForEach-Object {
        $_.IdentityReference.Translate([Security.Principal.SecurityIdentifier]).Value
    } | Sort-Object -Unique)
    foreach ($sid in $currentSids | Where-Object { $_ -notin $allowed }) {
        Invoke-AclCommand -Runner $CommandRunner -Arguments @($Path, '/remove:g', "*$sid")
        Invoke-AclCommand -Runner $CommandRunner -Arguments @($Path, '/remove:d', "*$sid")
    }
    Invoke-AclCommand -Runner $CommandRunner -Arguments @(
        $Path,
        '/grant:r', "*${OwnerSid}:F",
        '/grant:r', '*S-1-5-18:F'
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

function Invoke-SshClientKeyReconciliation {
    param(
        [Parameter(Mandatory)][string]$PrivateKeyPath,
        [Parameter(Mandatory)][string]$PublicKeyPath,
        [switch]$SkipAcl
    )

    if (-not (Test-Path -LiteralPath $PrivateKeyPath -PathType Leaf)) {
        throw "Private key not found: $PrivateKeyPath"
    }
    if (-not $SkipAcl) {
        $ownerSid = [Security.Principal.WindowsIdentity]::GetCurrent().User.Value
        Set-PrivateKeyAcl -Path $PrivateKeyPath -OwnerSid $ownerSid
    }
    $fingerprint = Assert-SshKeyPair -PrivateKeyPath $PrivateKeyPath -PublicKeyPath $PublicKeyPath
    Assert-NonInteractivePrivateKey -PrivateKeyPath $PrivateKeyPath
    Write-Host "SSH client key reconciled: $fingerprint"
}

if ($MyInvocation.InvocationName -ne '.') {
    Invoke-SshClientKeyReconciliation -PrivateKeyPath $PrivateKeyPath `
        -PublicKeyPath $PublicKeyPath -SkipAcl:$SkipAcl
}
