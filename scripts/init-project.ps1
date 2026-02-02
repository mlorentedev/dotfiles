<#
.SYNOPSIS
    Initialize a new project with Dual AI Memory (Claude & Gemini)

.DESCRIPTION
    Creates project structure, copies AI configurations, deploys skills,
    and optionally initializes stack-specific tooling.

.PARAMETER ProjectName
    Name of the project directory to create. Use "." for current directory.

.PARAMETER Stack
    Technology stack to initialize: python, go, node, ts

.EXAMPLE
    .\init-project.ps1 my-project python

.EXAMPLE
    .\init-project.ps1 . node

.NOTES
    Requires: dotfiles setup completed (setup-windows.ps1)
    AI configs expected at: ~/.claude/ and ~/.gemini/
#>

param(
    [Parameter(Position=0)]
    [string]$ProjectName = ".",

    [Parameter(Position=1)]
    [ValidateSet("python", "go", "node", "ts", "none")]
    [string]$Stack = "python"
)

# ============================================================================
# CONFIGURATION
# ============================================================================

$ClaudeHome = "$env:USERPROFILE\.claude"
$GeminiHome = "$env:USERPROFILE\.gemini"

# ============================================================================
# HELPER FUNCTIONS
# ============================================================================

function Write-Info { param([string]$Message) Write-Host "[INFO] $Message" -ForegroundColor Blue }
function Write-Success { param([string]$Message) Write-Host "[SUCCESS] $Message" -ForegroundColor Green }
function Write-Warn { param([string]$Message) Write-Host "[WARNING] $Message" -ForegroundColor Yellow }
function Write-Err { param([string]$Message) Write-Host "[ERROR] $Message" -ForegroundColor Red }

# ============================================================================
# PROJECT DIRECTORY
# ============================================================================

if ($ProjectName -ne ".") {
    if (-not (Test-Path $ProjectName)) {
        New-Item -ItemType Directory -Path $ProjectName -Force | Out-Null
    }
    Set-Location $ProjectName
    Write-Info "Created project directory: $ProjectName"
}

$ProjectRoot = Get-Location

# ============================================================================
# BASE STRUCTURE
# ============================================================================

Write-Info "Creating project structure..."

$Directories = @("src", "tests", "docs", "scripts", "tasks", ".claude\skills")
foreach ($dir in $Directories) {
    if (-not (Test-Path $dir)) {
        New-Item -ItemType Directory -Path $dir -Force | Out-Null
    }
}

# ============================================================================
# AI MEMORY INJECTION
# ============================================================================

# Copy CLAUDE.md
if (Test-Path "$ClaudeHome\CLAUDE.md") {
    Copy-Item "$ClaudeHome\CLAUDE.md" "." -Force
    Write-Success "Injected CLAUDE.md"
} else {
    Write-Warn "CLAUDE.md not found in global config ($ClaudeHome)"
}

# Copy GEMINI.md
if (Test-Path "$GeminiHome\GEMINI.md") {
    Copy-Item "$GeminiHome\GEMINI.md" "." -Force
    Write-Success "Injected GEMINI.md"
} else {
    Write-Warn "GEMINI.md not found in global config ($GeminiHome)"
}

# Copy skills (Claude Code specific)
if (Test-Path "$ClaudeHome\skills") {
    $SkillDirs = Get-ChildItem "$ClaudeHome\skills" -Directory -ErrorAction SilentlyContinue
    foreach ($skillDir in $SkillDirs) {
        $targetDir = ".claude\skills\$($skillDir.Name)"
        if (-not (Test-Path $targetDir)) {
            New-Item -ItemType Directory -Path $targetDir -Force | Out-Null
        }
        Copy-Item "$($skillDir.FullName)\*" $targetDir -Recurse -Force -ErrorAction SilentlyContinue
    }
    Write-Success "Injected skills to .claude\skills\"
}

# ============================================================================
# TASK MANAGEMENT
# ============================================================================

$TodoContent = @"
# Project Tasks

## Pending

- [ ] Initial setup complete

## In Progress

## Done

"@

$LessonsContent = @"
# Lessons Learned

*Record mistakes and prevention rules here. Review at session start.*

## Patterns to Avoid

## Corrections Log

"@

Set-Content -Path "tasks\todo.md" -Value $TodoContent -Encoding UTF8
Set-Content -Path "tasks\lessons.md" -Value $LessonsContent -Encoding UTF8
Write-Success "Created tasks\todo.md and tasks\lessons.md"

# ============================================================================
# STACK INITIALIZATION
# ============================================================================

switch ($Stack) {
    "python" {
        Write-Info "Initializing Python project..."

        # Try poetry first
        $poetry = Get-Command poetry -ErrorAction SilentlyContinue
        $uv = Get-Command uv -ErrorAction SilentlyContinue

        if ($poetry) {
            & poetry init -n --name (Split-Path -Leaf $ProjectRoot) 2>$null
            & poetry add --group dev pytest pytest-cov mypy ruff 2>$null
        } elseif ($uv) {
            & uv init 2>$null
        } else {
            Write-Warn "Neither poetry nor uv found. Skipping Python setup."
        }
    }

    "go" {
        Write-Info "Initializing Go project..."

        $go = Get-Command go -ErrorAction SilentlyContinue
        if ($go) {
            $projectName = Split-Path -Leaf $ProjectRoot
            & go mod init $projectName 2>$null
        } else {
            Write-Warn "Go not found. Skipping Go setup."
        }

        $MakefileContent = @"
.PHONY: build test lint run

build:
	go build -o bin/app ./cmd/...

test:
	go test -v -race -cover ./...

lint:
	golangci-lint run

run:
	go run ./cmd/main.go
"@
        Set-Content -Path "Makefile" -Value $MakefileContent -Encoding UTF8
    }

    "node" {
        Write-Info "Initializing Node.js project..."

        $npm = Get-Command npm -ErrorAction SilentlyContinue
        if ($npm) {
            & npm init -y 2>$null
            & npm i -D typescript @types/node tsx vitest 2>$null
        } else {
            Write-Warn "npm not found. Skipping Node.js setup."
        }
    }

    "ts" {
        Write-Info "Initializing TypeScript project..."

        $npm = Get-Command npm -ErrorAction SilentlyContinue
        if ($npm) {
            & npm init -y 2>$null
            & npm i -D typescript @types/node tsx vitest 2>$null
        } else {
            Write-Warn "npm not found. Skipping TypeScript setup."
        }
    }

    "none" {
        Write-Info "Skipping stack-specific initialization."
    }
}

# ============================================================================
# GIT INITIALIZATION
# ============================================================================

if (-not (Test-Path ".git")) {
    $git = Get-Command git -ErrorAction SilentlyContinue
    if ($git) {
        & git init | Out-Null
        Write-Success "Initialized git repository"
    }
}

# ============================================================================
# GITIGNORE
# ============================================================================

if (-not (Test-Path ".gitignore")) {
    $GitignoreContent = @"
# Dependencies
node_modules/
__pycache__/
.venv/
vendor/

# Build
dist/
build/
bin/
*.egg-info/

# IDE
.idea/
.vscode/
*.swp

# Environment
.env
.env.local
*.secret

# OS
.DS_Store
Thumbs.db

# Test/Coverage
.coverage
htmlcov/
.pytest_cache/
coverage.out
"@
    Set-Content -Path ".gitignore" -Value $GitignoreContent -Encoding UTF8
    Write-Success "Created .gitignore"
}

# ============================================================================
# SUMMARY
# ============================================================================

Write-Host ""
Write-Success "Project initialized with Dual AI Configuration"
Write-Host ""
Write-Host "Structure:" -ForegroundColor Cyan
Write-Host "  CLAUDE.md / GEMINI.md      - AI Memory files (Dual-Core)"
Write-Host "  .claude\skills\            - Custom skills"
Write-Host "  tasks\todo.md              - Task tracking"
Write-Host "  tasks\lessons.md           - Learning log"
