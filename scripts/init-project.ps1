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
    [string]$Stack = "python",

    [switch]$WorkSdk,

    [string]$Family = "",

    [string]$Component = ""
)

# ============================================================================
# CONFIGURATION
# ============================================================================

$ClaudeHome = "$env:USERPROFILE\.claude"
$GeminiHome = "$env:USERPROFILE\.gemini"
$Today = Get-Date -Format 'yyyy-MM-dd'

# ============================================================================
# --WorkSdk MODE: vault-only entry for nested work SDK repos
# ============================================================================

if ($WorkSdk) {
    if (-not $Family -or -not $Component) {
        Write-Err "Usage: init-project.ps1 -WorkSdk -Family <family-slug> -Component <component>"
        exit 1
    }

    $KnowledgeHome = "$env:USERPROFILE\Projects\knowledge"
    $SdkDir = "$KnowledgeHome\50_work\45-development\$Family\$Component"

    Write-Info "Creating work SDK vault entry: 50_work/45-development/$Family/$Component"

    foreach ($subDir in @('30-architecture', '40-runbooks', '50-troubleshooting', 'memory')) {
        New-Item -ItemType Directory -Path "$SdkDir\$subDir" -Force | Out-Null
    }

    Set-Content -Path "$SdkDir\00-context.md" -Encoding UTF8 -Value @"
---
id: "$Family-$Component"
type: project
status: active
owner: manu
source_path:
  code: "`${PROJECTS_PATH}/<ProductFamily>/<component>"
  onedrive: "`${ONEDRIVE_PATH}/Products/<ProductFamily>"
stack: []
tags: [work, sdk, $Family]
created: "$Today"
---

# ${Component}: Work SDK Context

> **Goal:** [Describe this repo's purpose within the $Family product family]

## Technical Stack
- **Language:**
- **Key Tools:**

## Repos in this Family
See [[../00-context|$Family family context]] for all repos.

## Critical Links
- **Real repo path:** Fill in source_path above with actual `$env:USERPROFILE\Projects\...` path

## Knowledge Structure
- [[30-architecture]]: ADRs and Technical Decisions
- [[40-runbooks]]: Operational Guides
- [[50-troubleshooting]]: Known Issues
- [[memory/MEMORY]]: Session memory
- [[90-lessons]]: Lessons Learned
"@

    Set-Content -Path "$SdkDir\memory\MEMORY.md" -Encoding UTF8 -Value @"
# $Component - Work SDK Session Memory

## Session Handoff
> Updated: $Today
**Last task:** Vault entry initialized
**Decisions:** None
**Open threads:** Fill source_path in 00-context.md with real repo path
**Next action:** Open Claude in real repo -- session hook creates junction automatically

## User Preferences
- Follow global CLAUDE.md + workflow-protocol.md rules

## Last Crystallized: $Today
"@

    Set-Content -Path "$SdkDir\90-lessons.md" -Encoding UTF8 -Value @"
---
id: "$Family-$Component-lessons"
type: lesson
status: active
owner: manu
tags: [work, sdk, $Family]
---
# Lessons Learned - $Component

*Record non-trivial bugs and decisions here. Promoted to 00_meta/patterns/ if recurring.*
"@

    # Create parent family context if it doesn't exist
    $FamilyContext = "$KnowledgeHome\50_work\45-development\$Family\00-context.md"
    if (-not (Test-Path $FamilyContext)) {
        Set-Content -Path $FamilyContext -Encoding UTF8 -Value @"
---
id: "$Family"
type: project
status: active
owner: manu
tags: [work, sdk, product-family]
created: "$Today"
---

# ${Family}: Product Family Context

> Umbrella context for all repos in the $Family product family.

## Repos

| Component | Vault Path | Real Path |
|-----------|-----------|-----------|
| $Component | [[$Component/00-context]] | Fill in source_path |

## Work Context
- Product knowledge: [[../../20-products/$Family/00-overview|product overview]] (if exists)
"@
        Write-Success "Created family context: $FamilyContext"
    }

    Write-Success "Work SDK vault entry created: $SdkDir"
    Write-Host ""
    Write-Host "Next steps:" -ForegroundColor Cyan
    Write-Host "  1. Fill in source_path in $SdkDir\00-context.md"
    Write-Host "  2. Open Claude in the real repo - session hook creates junction automatically"
    exit 0
}

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

$Directories = @("src", "tests", "scripts", ".claude\skills")
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
# KNOWLEDGE VAULT INTEGRATION
# ============================================================================

$KnowledgeHome = "$env:USERPROFILE\Projects\knowledge"
$ProjectBaseName = Split-Path -Leaf $ProjectRoot
$ProjectKbDir = "$KnowledgeHome\10_projects\$ProjectBaseName"

Write-Info "Initializing Knowledge Vault structure..."

foreach ($subDir in @('30-architecture', '40-runbooks', '50-troubleshooting', 'memory')) {
    New-Item -ItemType Directory -Path "$ProjectKbDir\$subDir" -Force | Out-Null
}

# 00-context.md
if (-not (Test-Path "$ProjectKbDir\00-context.md")) {
    Set-Content -Path "$ProjectKbDir\00-context.md" -Encoding UTF8 -Value @"
---
id: "$ProjectBaseName"
type: project
status: active
owner: manu
repo_url: ""
stack: [$Stack]
tags: []
created: "$Today"
---

# ${ProjectBaseName}: Project Context

> **Goal:** [Concise, single-sentence project description]

## Technical Stack
- **Language:** $Stack
- **Frameworks:**
- **Infrastructure:**
- **Key Tools:**

## Critical Links
- **Repository:** ~/Projects/$ProjectBaseName
- **Production:**

## Knowledge Structure
- [[10-roadmap]]: Planning & Strategy
- [[11-tasks]]: Active Backlog
- [[30-architecture]]: ADRs and Technical Decisions
- [[40-runbooks]]: Operational Guides
- [[50-troubleshooting]]: Known Issues
- [[90-lessons]]: Lessons Learned

## Development Flow
1. Feature branch from ``main``
2. PR with tests
3. Merge -> deploy
"@
}

# 10-roadmap.md
if (-not (Test-Path "$ProjectKbDir\10-roadmap.md")) {
    Set-Content -Path "$ProjectKbDir\10-roadmap.md" -Encoding UTF8 -Value @"
---
id: "$ProjectBaseName-roadmap"
type: roadmap
status: active
owner: manu
created: "$Today"
---

# ${ProjectBaseName}: Roadmap and Vision

> **Vision:** [One-sentence vision]

## Strategic Themes

### 1. [Theme Name]
* **Goal:**
* **Status:** Not started
* **Next:**
"@
}

# 11-tasks.md
if (-not (Test-Path "$ProjectKbDir\11-tasks.md")) {
    Set-Content -Path "$ProjectKbDir\11-tasks.md" -Encoding UTF8 -Value @"
---
id: "$ProjectBaseName-tasks"
type: project
status: active
owner: manu
tags: []
---
# ${ProjectBaseName}: Active Backlog

Progress: [..........] 0%

## Pending

- [ ] Fill in 00-context.md with project details

## In Progress

## Done
"@
}

# 90-lessons.md
if (-not (Test-Path "$ProjectKbDir\90-lessons.md")) {
    Set-Content -Path "$ProjectKbDir\90-lessons.md" -Encoding UTF8 -Value @"
---
id: "$ProjectBaseName-lessons"
type: lesson
status: active
owner: manu
tags: []
---
# Lessons Learned - $ProjectBaseName

*Non-trivial bugs and decisions. Promoted to 00_meta/patterns/ if recurring across projects.*
"@
}

# memory/MEMORY.md
if (-not (Test-Path "$ProjectKbDir\memory\MEMORY.md")) {
    Set-Content -Path "$ProjectKbDir\memory\MEMORY.md" -Encoding UTF8 -Value @"
# $ProjectBaseName

## Session Handoff
> Updated: $Today
**Last task:** Project initialized via init-project.ps1
**Decisions:** None
**Open threads:** Fill 00-context.md with actual project details
**Next action:** Run /crystallize after first sprint

## User Preferences
- Follow global CLAUDE.md + workflow-protocol.md rules

## Architecture Map
- See [[00-context]] for stack and entry points

## Last Crystallized: $Today
"@
}

# Create memory junction immediately - don't wait for first Claude session
$EncodedCwd = $ProjectRoot.Path.Replace('\', '-').Replace(':', '').Replace('/', '-')
$ClaudeProjectDir = "$env:USERPROFILE\.claude\projects\$EncodedCwd"
if (-not (Test-Path $ClaudeProjectDir)) {
    New-Item -ItemType Directory -Path $ClaudeProjectDir -Force | Out-Null
}
$JunctionPath = "$ClaudeProjectDir\memory"
if (-not (Test-Path $JunctionPath)) {
    try {
        New-Item -ItemType Junction -Path $JunctionPath -Target "$ProjectKbDir\memory" -Force | Out-Null
        Write-Success "Created memory junction -> vault"
    } catch {
        Write-Warn "Could not create memory junction (non-fatal): $_"
    }
}

Write-Success "Created Knowledge Vault entry in 10_projects\$ProjectBaseName"

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
            & poetry add --group dev typer rich pydantic pytest pytest-cov mypy ruff 2>$null
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

# Activate pre-commit hooks (gitleaks) if pre-commit is available
$precommit = Get-Command pre-commit -ErrorAction SilentlyContinue
if ($precommit -and (Test-Path ".pre-commit-config.yaml")) {
    & pre-commit install 2>&1 | Out-Null
    Write-Success "Activated pre-commit hooks"
}

# ============================================================================
# PRE-COMMIT CONFIG (gitleaks for agent-touched repos)
# ============================================================================

if (-not (Test-Path ".pre-commit-config.yaml")) {
    $PreCommitContent = @"
# Threat model: code repo. Agents may paste secrets into test fixtures, env
# samples, or doc snippets. gitleaks blocks them before they reach git history.
#
# Setup (one-time):
#   pre-commit install
#
# Bypass (only with explicit user approval):
#   `$env:SKIP='gitleaks'; git commit ...

repos:
  - repo: https://github.com/gitleaks/gitleaks
    rev: v8.30.1
    hooks:
      - id: gitleaks
"@
    Set-Content -Path ".pre-commit-config.yaml" -Value $PreCommitContent -Encoding UTF8
    Write-Success "Created .pre-commit-config.yaml (gitleaks v8.30.1)"
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
Write-Host "  Knowledge Vault updated    - 11-tasks.md & 90-lessons.md in knowledge/10_projects/"
