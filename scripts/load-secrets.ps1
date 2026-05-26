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

# Helper: read a SecureString value from the user and return as plain string.
# Plain text is held in memory only for the duration of one encrypt call and
# is never written to disk in plaintext (age reads from stdin).
function Read-SecretValue {
    param([string]$Prompt)
    $secure = Read-Host -Prompt $Prompt -AsSecureString
    $bstr = [Runtime.InteropServices.Marshal]::SecureStringToBSTR($secure)
    try {
        return [Runtime.InteropServices.Marshal]::PtrToStringAuto($bstr)
    } finally {
        [Runtime.InteropServices.Marshal]::ZeroFreeBSTR($bstr)
    }
}

# Helper: encrypt $Value to $OutputFile using the public key parsed from KeyPath.
# Mirrors age_encrypt() in scripts/utils.sh.
function Invoke-AgeEncrypt {
    param(
        [Parameter(Mandatory)][string]$OutputFile,
        [Parameter(Mandatory)][string]$Value
    )

    if (-not (Test-Path $script:KeyPath)) {
        Write-Host "Error: Key file not found at $script:KeyPath" -ForegroundColor Red
        return $false
    }
    if (-not (Get-Command age -ErrorAction SilentlyContinue)) {
        Write-Host "Error: age not installed" -ForegroundColor Red
        return $false
    }

    # Parse public key (age1...) from identity file
    $pubkey = (Select-String -Path $script:KeyPath -Pattern 'age1[0-9a-z]+').Matches.Value | Select-Object -First 1
    if (-not $pubkey) {
        Write-Host "Error: could not parse public key from $script:KeyPath" -ForegroundColor Red
        return $false
    }

    $tmpOut = "$OutputFile.tmp.$PID"
    try {
        $Value | & age -r $pubkey -o $tmpOut 2>$null
        if ($LASTEXITCODE -eq 0 -and (Test-Path $tmpOut)) {
            Move-Item -LiteralPath $tmpOut -Destination $OutputFile -Force
            return $true
        }
    } catch {
        # fall through
    }
    Remove-Item -LiteralPath $tmpOut -Force -ErrorAction SilentlyContinue
    return $false
}

# Add a new secret: encrypt value, append mapping, export into current shell.
# Mirrors secrets_add() in scripts/load-secrets.sh.
# Usage: Add-Secret VAR_NAME filename     (or alias `secrets_add`)
function Add-Secret {
    param(
        [Parameter(Mandatory)][string]$VarName,
        [Parameter(Mandatory)][string]$FileName
    )

    if (-not (Test-Path $script:MappingFile)) {
        Write-Host "Error: Mapping file not found at $script:MappingFile" -ForegroundColor Red
        return
    }

    # Idempotency: bail if mapping already exists (point to rotate instead)
    $existing = Select-String -Path $script:MappingFile -Pattern "^${VarName}=" -SimpleMatch
    if ($existing) {
        Write-Host "Error: Mapping already exists for $VarName" -ForegroundColor Red
        Write-Host "Use 'secrets_rotate $VarName' to update the value"
        return
    }

    $encryptedFile = Join-Path $script:SecretsDir "$FileName.secret.age"
    if (Test-Path $encryptedFile) {
        Write-Host "Error: Encrypted file already exists: $encryptedFile" -ForegroundColor Red
        return
    }

    $value = Read-SecretValue -Prompt "Enter value for $VarName"
    if (-not $value) {
        Write-Host "Error: Value cannot be empty" -ForegroundColor Red
        return
    }

    if (-not (Invoke-AgeEncrypt -OutputFile $encryptedFile -Value $value)) {
        Write-Host "Error: Encryption failed" -ForegroundColor Red
        return
    }

    Add-Content -LiteralPath $script:MappingFile -Value "${VarName}=${FileName}"
    [Environment]::SetEnvironmentVariable($VarName, $value, 'Process')

    Write-Host "Secret added successfully:" -ForegroundColor Green
    Write-Host "  Variable: $VarName (exported in current shell)"
    Write-Host "  File: ${FileName}.secret.age"
}

# Rotate an existing secret: backup, re-encrypt new value, restore on failure.
# Mirrors secrets_rotate() in scripts/load-secrets.sh.
# Usage: Update-Secret VAR_NAME     (or alias `secrets_rotate`)
function Update-Secret {
    param([Parameter(Mandatory)][string]$VarName)

    if (-not (Test-Path $script:MappingFile)) {
        Write-Host "Error: Mapping file not found at $script:MappingFile" -ForegroundColor Red
        return
    }

    $match = Select-String -Path $script:MappingFile -Pattern "^${VarName}=" | Select-Object -First 1
    if (-not $match) {
        Write-Host "Error: No mapping found for $VarName" -ForegroundColor Red
        Write-Host "Use 'secrets_add $VarName <filename>' to create a new secret"
        return
    }

    $fileName = ($match.Line -split '=', 2)[1].Trim()
    $encryptedFile = Join-Path $script:SecretsDir "$fileName.secret.age"
    $backupFile = "$encryptedFile.bak"

    if (Test-Path $encryptedFile) {
        Copy-Item -LiteralPath $encryptedFile -Destination $backupFile -Force
    }

    Write-Host "Rotating secret: $VarName"
    $value = Read-SecretValue -Prompt "Enter new value"

    if (-not $value) {
        Write-Host "Error: Value cannot be empty" -ForegroundColor Red
        Remove-Item -LiteralPath $backupFile -Force -ErrorAction SilentlyContinue
        return
    }

    if (-not (Invoke-AgeEncrypt -OutputFile $encryptedFile -Value $value)) {
        # Restore backup
        if (Test-Path $backupFile) {
            Move-Item -LiteralPath $backupFile -Destination $encryptedFile -Force
        }
        Write-Host "Error: Encryption failed, restored backup" -ForegroundColor Red
        return
    }

    Remove-Item -LiteralPath $backupFile -Force -ErrorAction SilentlyContinue
    [Environment]::SetEnvironmentVariable($VarName, $value, 'Process')

    Write-Host "Secret rotated successfully: $VarName (env updated in current shell)" -ForegroundColor Green
}

# Export functions as aliases for discoverability + cross-shell parity with .sh
Set-Alias -Name secrets_refresh -Value Invoke-SecretsRefresh -Scope Global
Set-Alias -Name secrets_list    -Value Show-SecretsList     -Scope Global
Set-Alias -Name secrets_add     -Value Add-Secret           -Scope Global
Set-Alias -Name secrets_rotate  -Value Update-Secret        -Scope Global

# Auto-load secrets when this file is sourced (dot-sourced)
Import-AllSecrets
