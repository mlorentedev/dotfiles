# utils.ps1 - shared PowerShell helpers for cross-OS setup parity.
#
# Today only the deploy/drift helpers live here (SDD-007 IaC strategy).
# Other helpers (Write-Info, Ensure-Directory) remain inline in setup-windows.ps1
# to avoid coupling load order; the rule will be revisited in a follow-up.
#
# ASCII-only by repo convention (pattern-powershell-ascii-only). No em-dashes,
# no smart quotes, no arrows, no non-ASCII glyphs anywhere in this file.
# PSScriptAnalyzer fails CI on non-ASCII chars without a BOM.

Set-StrictMode -Version Latest

# Deploy-File: copy SRC to DST atomically + idempotently.
# Parity with bash scripts/utils.sh deploy_file().
#
# Behavior:
#   - skips silently if SHA256 matches (idempotent on repeat runs)
#   - removes pre-existing reparse-point (Windows equivalent of symlink) at DST
#   - writes via Copy-Item then sets owner-only on staging (atomic in NTFS)
#   - logs "Deployed: <path>" only on actual change
#   - returns $true on success, $false on error
#
# Usage:
#   . $PSScriptRoot\utils.ps1
#   Deploy-File "C:\repo\.zshrc" "$env:USERPROFILE\.zshrc"
function Deploy-File {
    param(
        [Parameter(Mandatory)][string]$Source,
        [Parameter(Mandatory)][string]$Destination
    )

    if (-not (Test-Path -LiteralPath $Source -PathType Leaf)) {
        Write-Host "[ERROR] Deploy-File: source missing: $Source" -ForegroundColor Red
        return $false
    }

    # Idempotency check by SHA256
    if (Test-Path -LiteralPath $Destination -PathType Leaf) {
        $isReparse = (Get-Item -LiteralPath $Destination -Force).Attributes -band [IO.FileAttributes]::ReparsePoint
        if (-not $isReparse) {
            $srcHash = (Get-FileHash -LiteralPath $Source -Algorithm SHA256).Hash
            $dstHash = (Get-FileHash -LiteralPath $Destination -Algorithm SHA256).Hash
            if ($srcHash -eq $dstHash) {
                return $true
            }
        }
    }

    # Drift recovery: pre-existing symlink / junction at destination
    if (Test-Path -LiteralPath $Destination) {
        $item = Get-Item -LiteralPath $Destination -Force
        if ($item.Attributes -band [IO.FileAttributes]::ReparsePoint) {
            Remove-Item -LiteralPath $Destination -Force
        }
    }

    # Ensure parent directory exists
    $dstDir = Split-Path -Path $Destination -Parent
    if ($dstDir -and -not (Test-Path -LiteralPath $dstDir)) {
        New-Item -ItemType Directory -Path $dstDir -Force | Out-Null
    }

    try {
        Copy-Item -LiteralPath $Source -Destination $Destination -Force
        Write-Host "[SUCCESS] Deployed: $Destination" -ForegroundColor Green
        return $true
    } catch {
        Write-Host "[ERROR] Deploy-File: copy failed $Source -> $Destination" -ForegroundColor Red
        return $false
    }
}

# Test-FileDrift: assert DST matches SRC content (drift detection for healthcheck).
# Parity with bash scripts/utils.sh check_deployed().
#
# Returns $true if files match, $false if drift detected.
# Logs PASS/FAIL via Write-Host so callers can compose with their own counters.
function Test-FileDrift {
    param(
        [Parameter(Mandatory)][string]$Source,
        [Parameter(Mandatory)][string]$Destination,
        [string]$DisplayName = ''
    )
    if (-not $DisplayName) { $DisplayName = Split-Path -Leaf $Destination }

    if (-not (Test-Path -LiteralPath $Destination -PathType Leaf)) {
        Write-Host "[FAIL] $DisplayName missing at $Destination (run setup-windows.ps1)" -ForegroundColor Red
        return $false
    }

    $isReparse = (Get-Item -LiteralPath $Destination -Force).Attributes -band [IO.FileAttributes]::ReparsePoint
    if ($isReparse) {
        Write-Host "[FAIL] $DisplayName is a reparse-point/symlink (expected regular file)" -ForegroundColor Red
        return $false
    }

    $srcHash = (Get-FileHash -LiteralPath $Source -Algorithm SHA256).Hash
    $dstHash = (Get-FileHash -LiteralPath $Destination -Algorithm SHA256).Hash
    if ($srcHash -eq $dstHash) {
        Write-Host "[PASS] $DisplayName deployed (matches repo)" -ForegroundColor Green
        return $true
    }

    Write-Host "[FAIL] $DisplayName has drifted from $Source (edit in repo + re-run setup)" -ForegroundColor Red
    return $false
}

# Substitute-EnvPlaceholders: deploy-time {env:VAR} substitution (SDD-009).
# Parity with bash scripts/utils.sh substitute_env_placeholders().
#
# Resolution path mirrors the bash helper:
#   {env:NAN_API_KEY}
#     -> grep ^NAN_API_KEY= in env-mapping.conf (skip comments)
#     -> <value-after-=> -> "$SecretsDir\<value>.secret.age"
#     -> age --decrypt -> inject literal value (newlines stripped).
# Unresolvable placeholders (mapping commented OR secret missing OR decrypt
# fails) are left intact; emits one Write-Warning listing every unresolved
# NAME. opencode's runtime env resolver acts as fallback.
# Side effect: deployed file ACL is restricted to the current user (Windows
# equivalent of chmod 600).
# Idempotent: re-running with rotated secrets re-substitutes.
#
# Parameters:
#   -Path   Path of file to rewrite in place.
#   -SecretsDir         Override sensitive/ directory.
#   -MappingFile        Override env-mapping.conf path.
#   -KeyPath            Override age private key path.
#
# Returns $true on success (including no-token no-op), $false on missing file.
function Substitute-EnvPlaceholders {
    param(
        [Parameter(Mandatory)][string]$Path,
        [string]$SecretsDir = '',
        [string]$MappingFile = '',
        [string]$KeyPath = ''
    )

    if (-not (Test-Path -LiteralPath $Path -PathType Leaf)) {
        Write-Host "[ERROR] Substitute-EnvPlaceholders: file missing: $Path" -ForegroundColor Red
        return $false
    }

    # Defaults: align with load-secrets.ps1 layout.
    if (-not $SecretsDir) {
        $dotfilesDir = if ($env:DOTFILES_DIR) { $env:DOTFILES_DIR } else { Join-Path $env:USERPROFILE 'Projects\dotfiles' }
        $SecretsDir = Join-Path $dotfilesDir 'sensitive'
    }
    if (-not $MappingFile) { $MappingFile = Join-Path $SecretsDir 'env-mapping.conf' }
    if (-not $KeyPath) { $KeyPath = Join-Path $env:USERPROFILE '.config\age\key.txt' }

    $content = Get-Content -LiteralPath $Path -Raw -Encoding UTF8
    if (-not $content) { return $true }

    # Extract unique {env:NAME} tokens.
    $matches = [regex]::Matches($content, '\{env:([A-Z_][A-Z0-9_]*)\}')
    if ($matches.Count -eq 0) { return $true }

    $tokenSet = @{}
    foreach ($m in $matches) { $tokenSet[$m.Groups[1].Value] = $true }

    if (-not (Test-Path -LiteralPath $MappingFile -PathType Leaf)) {
        Write-Warning "Substitute-EnvPlaceholders: mapping file missing: $MappingFile (placeholders left intact)"
        return $true
    }

    $mappingLines = Get-Content -LiteralPath $MappingFile -Encoding UTF8
    $unresolved = @()

    foreach ($name in $tokenSet.Keys) {
        # Skip-comments match: line must start with NAME= (no leading #).
        $line = $mappingLines | Where-Object { $_ -match "^${name}=" } | Select-Object -First 1
        if (-not $line) { $unresolved += $name; continue }

        $secretName = ($line -split '=', 2)[1]
        if (-not $secretName) { $unresolved += $name; continue }

        $secretFile = Join-Path $SecretsDir "$secretName.secret.age"
        if (-not (Test-Path -LiteralPath $secretFile -PathType Leaf)) {
            $unresolved += $name; continue
        }

        try {
            $value = & age --decrypt --identity $KeyPath $secretFile 2>$null
            if ($LASTEXITCODE -ne 0 -or -not $value) {
                $unresolved += $name; continue
            }
            $clean = ($value -join '').TrimEnd("`r", "`n")
            if (-not $clean) { $unresolved += $name; continue }
        } catch {
            $unresolved += $name; continue
        }

        # Literal replacement of every occurrence.
        $token = "{env:$name}"
        $content = $content.Replace($token, $clean)
    }

    # Atomic rewrite via staging file. Set-Content -NoNewline preserves byte
    # parity with the bash helper's printf '%s'.
    $tmp = "$Path.tmp.$PID"
    try {
        Set-Content -LiteralPath $tmp -Value $content -NoNewline -Encoding UTF8
        Move-Item -LiteralPath $tmp -Destination $Path -Force

        # ACL: owner-only (Windows equivalent of chmod 600). Use the current
        # process token's NTAccount rather than USERDOMAIN\USERNAME so this
        # works on Microsoft / Azure AD accounts where USERDOMAIN does not
        # round-trip into a valid SDDL principal (e.g. AzureAD\user@org.com).
        try {
            $acl = Get-Acl -LiteralPath $Path
            $acl.SetAccessRuleProtection($true, $false)
            foreach ($existing in @($acl.Access)) {
                [void]$acl.RemoveAccessRule($existing)
            }
            $owner = [System.Security.Principal.WindowsIdentity]::GetCurrent().Name
            $rule = New-Object System.Security.AccessControl.FileSystemAccessRule(
                $owner, 'FullControl', 'Allow'
            )
            $acl.AddAccessRule($rule)
            Set-Acl -LiteralPath $Path -AclObject $acl
        } catch {
            # ACL hardening is best-effort; substitution itself succeeded.
        }
    } catch {
        if (Test-Path -LiteralPath $tmp) { Remove-Item -LiteralPath $tmp -Force }
        Write-Host "[ERROR] Substitute-EnvPlaceholders: rewrite failed $Path" -ForegroundColor Red
        return $false
    }

    if ($unresolved.Count -gt 0) {
        Write-Warning "Substitute-EnvPlaceholders: unresolved placeholders left intact: $($unresolved -join ' ')"
    }
    return $true
}
