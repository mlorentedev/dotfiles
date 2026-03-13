<#
.SYNOPSIS
    Decrypts age-encrypted secrets and exports as environment variables

.DESCRIPTION
    Reads env-mapping.conf and decrypts *.secret.age files using age.
    Supports both environment variable secrets (VAR=filename) and
    file secrets (@VAR=filename>dest_path).

    Source this script in your PowerShell profile to load secrets at startup:
      . "$env:USERPROFILE\.dotfiles\scripts\load-secrets.ps1"

.NOTES
    Requires: age (https://github.com/FiloSottile/age)
    Key file: ~/.config/age/key.txt
    Mapping: $DOTFILES_DIR\sensitive\env-mapping.conf
#>

Set-StrictMode -Version Latest

# Configuration
$script:SecretsDir = if ($env:DOTFILES_DIR) { "$env:DOTFILES_DIR\sensitive" } else { "$env:USERPROFILE\.dotfiles\sensitive" }
$script:MappingFile = Join-Path $script:SecretsDir 'env-mapping.conf'
$script:KeyPath = if ($env:AGE_KEY_PATH) { $env:AGE_KEY_PATH } else { "$env:USERPROFILE\.config\age\key.txt" }
$script:SecretsLoaded = 0

function Test-IsFileSecret {
    param([string]$VarName)
    return $VarName.StartsWith('@')
}

function Expand-TildePath {
    param([string]$Path)
    if ($Path.StartsWith('~/') -or $Path.StartsWith('~\')) {
        return Join-Path $env:USERPROFILE $Path.Substring(2)
    }
    if ($Path -eq '~') { return $env:USERPROFILE }
    return $Path
}

function Invoke-AgeDecrypt {
    param([string]$EncryptedFile)

    if (-not (Test-Path $EncryptedFile)) { return $null }

    try {
        $result = & age --decrypt --identity $script:KeyPath $EncryptedFile 2>$null
        if ($LASTEXITCODE -eq 0) { return $result }
    } catch {
        # Decryption failed
    }
    return $null
}

function Import-EnvSecret {
    param(
        [string]$VarName,
        [string]$FileName
    )

    $encryptedFile = Join-Path $script:SecretsDir "$FileName.secret.age"
    if (-not (Test-Path $encryptedFile)) { return }

    $value = Invoke-AgeDecrypt -EncryptedFile $encryptedFile
    if ($value) {
        # Trim trailing newlines
        $cleanValue = ($value -join '').TrimEnd("`r", "`n")
        [Environment]::SetEnvironmentVariable($VarName, $cleanValue, 'Process')
        $script:SecretsLoaded++
    }
}

function Import-FileSecret {
    param(
        [string]$RawVar,
        [string]$RawValue
    )

    # Strip @ prefix for env var name
    $varName = $RawVar.TrimStart('@')

    # Parse filename>dest_path
    $parts = $RawValue -split '>', 2
    $fileName = $parts[0]
    $destPath = Expand-TildePath $parts[1]

    $encryptedFile = Join-Path $script:SecretsDir "$fileName.secret.age"
    if (-not (Test-Path $encryptedFile)) { return }

    # Cache: skip if dest is newer than encrypted source
    if ((Test-Path $destPath) -and ((Get-Item $destPath).LastWriteTime -gt (Get-Item $encryptedFile).LastWriteTime)) {
        [Environment]::SetEnvironmentVariable($varName, $destPath, 'Process')
        $script:SecretsLoaded++
        return
    }

    # Ensure parent directory exists
    $destDir = Split-Path -Parent $destPath
    if (-not (Test-Path $destDir)) {
        New-Item -ItemType Directory -Path $destDir -Force | Out-Null
    }

    # Decrypt to destination
    try {
        & age --decrypt --identity $script:KeyPath --output $destPath $encryptedFile 2>$null
        if ($LASTEXITCODE -eq 0) {
            [Environment]::SetEnvironmentVariable($varName, $destPath, 'Process')
            $script:SecretsLoaded++
        } else {
            # Remove potentially corrupt partial file
            Remove-Item -Path $destPath -Force -ErrorAction SilentlyContinue
        }
    } catch {
        Remove-Item -Path $destPath -Force -ErrorAction SilentlyContinue
    }
}

function Import-AllSecrets {
    if (-not (Test-Path $script:MappingFile)) { return }
    if (-not (Test-Path $script:KeyPath)) { return }
    if (-not (Get-Command age -ErrorAction SilentlyContinue)) { return }

    $script:SecretsLoaded = 0

    foreach ($line in Get-Content $script:MappingFile) {
        $trimmed = $line.Trim()

        # Skip comments and empty lines
        if (-not $trimmed -or $trimmed.StartsWith('#')) { continue }
        if ($trimmed -notmatch '=') { continue }

        $eqIndex = $trimmed.IndexOf('=')
        $varName = $trimmed.Substring(0, $eqIndex).Trim()
        $fileName = $trimmed.Substring($eqIndex + 1).Trim()

        if (Test-IsFileSecret $varName) {
            Import-FileSecret -RawVar $varName -RawValue $fileName
        } else {
            Import-EnvSecret -VarName $varName -FileName $fileName
        }
    }
}

function Invoke-SecretsRefresh {
    <#
    .SYNOPSIS
        Force reload all secrets from encrypted files
    #>

    # Unset existing vars
    if (Test-Path $script:MappingFile) {
        foreach ($line in Get-Content $script:MappingFile) {
            $trimmed = $line.Trim()
            if (-not $trimmed -or $trimmed.StartsWith('#')) { continue }
            if ($trimmed -notmatch '=') { continue }

            $eqIndex = $trimmed.IndexOf('=')
            $varName = $trimmed.Substring(0, $eqIndex).Trim()

            if (Test-IsFileSecret $varName) {
                $cleanVar = $varName.TrimStart('@')
                $rawValue = $trimmed.Substring($eqIndex + 1).Trim()
                $parts = $rawValue -split '>', 2
                $destPath = Expand-TildePath $parts[1]
                Remove-Item -Path $destPath -Force -ErrorAction SilentlyContinue
                [Environment]::SetEnvironmentVariable($cleanVar, $null, 'Process')
            } else {
                [Environment]::SetEnvironmentVariable($varName, $null, 'Process')
            }
        }
    }

    Import-AllSecrets
    Write-Host "Secrets refreshed: $($script:SecretsLoaded) variables loaded"
}

function Show-SecretsList {
    <#
    .SYNOPSIS
        Show all mapped secrets and their load status
    #>

    if (-not (Test-Path $script:MappingFile)) {
        Write-Host "Mapping file not found: $($script:MappingFile)"
        return
    }

    Write-Host "Secret mappings ($($script:MappingFile)):"
    Write-Host ""
    Write-Host "Environment Variables:"

    $hasFileSecrets = $false

    foreach ($line in Get-Content $script:MappingFile) {
        $trimmed = $line.Trim()
        if (-not $trimmed -or $trimmed.StartsWith('#')) { continue }
        if ($trimmed -notmatch '=') { continue }

        $eqIndex = $trimmed.IndexOf('=')
        $varName = $trimmed.Substring(0, $eqIndex).Trim()
        $fileName = $trimmed.Substring($eqIndex + 1).Trim()

        if (Test-IsFileSecret $varName) {
            $hasFileSecrets = $true
            continue
        }

        $encryptedFile = Join-Path $script:SecretsDir "$fileName.secret.age"
        if (-not (Test-Path $encryptedFile)) {
            Write-Host "  x $varName (missing: $fileName.secret.age)"
        } elseif ([Environment]::GetEnvironmentVariable($varName, 'Process')) {
            Write-Host "  * $varName"
        } else {
            Write-Host "  o $varName (not loaded)"
        }
    }

    if ($hasFileSecrets) {
        Write-Host ""
        Write-Host "File Secrets:"
        foreach ($line in Get-Content $script:MappingFile) {
            $trimmed = $line.Trim()
            if (-not $trimmed -or $trimmed.StartsWith('#')) { continue }
            if ($trimmed -notmatch '=') { continue }

            $eqIndex = $trimmed.IndexOf('=')
            $varName = $trimmed.Substring(0, $eqIndex).Trim()
            $fileName = $trimmed.Substring($eqIndex + 1).Trim()

            if (-not (Test-IsFileSecret $varName)) { continue }

            $cleanVar = $varName.TrimStart('@')
            $parts = $fileName -split '>', 2
            $fsFileName = $parts[0]
            $destPath = Expand-TildePath $parts[1]
            $encryptedFile = Join-Path $script:SecretsDir "$fsFileName.secret.age"

            if (-not (Test-Path $encryptedFile)) {
                Write-Host "  x $cleanVar -> $destPath (missing: $fsFileName.secret.age)"
            } elseif (Test-Path $destPath) {
                Write-Host "  * $cleanVar -> $destPath (deployed)"
            } else {
                Write-Host "  o $cleanVar -> $destPath (not deployed)"
            }
        }
    }
}

# Export functions as aliases for discoverability
Set-Alias -Name secrets_refresh -Value Invoke-SecretsRefresh -Scope Global
Set-Alias -Name secrets_list -Value Show-SecretsList -Scope Global

# Auto-load secrets when this file is sourced (dot-sourced)
Import-AllSecrets
