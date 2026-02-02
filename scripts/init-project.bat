@echo off
setlocal enabledelayedexpansion

:: init-project.bat
:: Initialize a new project with Claude Code configuration
:: Usage: init-project.bat [project-name] [stack]
:: Example: init-project.bat my-project python

set PROJECT_NAME=%~1
set STACK=%~2
set CLAUDE_HOME=%USERPROFILE%\.claude

:: Defaults
if "%PROJECT_NAME%"=="" set PROJECT_NAME=.
if "%STACK%"=="" set STACK=python

:: Create project directory if not current
if not "%PROJECT_NAME%"=="." (
    if not exist "%PROJECT_NAME%" mkdir "%PROJECT_NAME%"
    cd "%PROJECT_NAME%"
    echo [INFO] Created project directory: %PROJECT_NAME%
)

:: Create base structure
echo [INFO] Creating project structure...
if not exist "src" mkdir src
if not exist "tests" mkdir tests
if not exist "docs" mkdir docs
if not exist "scripts" mkdir scripts
if not exist "tasks" mkdir tasks
if not exist ".claude\skills" mkdir ".claude\skills"

:: Copy CLAUDE.md
if exist "%CLAUDE_HOME%\CLAUDE.md" (
    copy /Y "%CLAUDE_HOME%\CLAUDE.md" "." >nul
    echo [SUCCESS] Copied CLAUDE.md
)

:: Copy skills
if exist "%CLAUDE_HOME%\skills" (
    for /D %%s in ("%CLAUDE_HOME%\skills\*") do (
        set SKILL_NAME=%%~nxs
        if not exist ".claude\skills\!SKILL_NAME!" mkdir ".claude\skills\!SKILL_NAME!"
        xcopy /E /I /Y "%%s\*" ".claude\skills\!SKILL_NAME!\" >nul 2>&1
    )
    echo [SUCCESS] Copied skills to .claude\skills\
)

:: Create tasks/todo.md
(
echo # Project Tasks
echo.
echo ## Pending
echo.
echo - [ ] Initial setup complete
echo.
echo ## In Progress
echo.
echo ## Done
echo.
) > tasks\todo.md
echo [SUCCESS] Created tasks\todo.md

:: Create tasks/lessons.md
(
echo # Lessons Learned
echo.
echo *Record mistakes and prevention rules here. Review at session start.*
echo.
echo ## Patterns to Avoid
echo.
echo ## Corrections Log
echo.
) > tasks\lessons.md
echo [SUCCESS] Created tasks\lessons.md

:: Stack-specific initialization
if "%STACK%"=="python" (
    echo [INFO] Initializing Python project...
    where poetry >nul 2>&1
    if !errorlevel! equ 0 (
        poetry init -n --name "%PROJECT_NAME%" 2>nul
        poetry add --group dev typer rich pydantic pytest pytest-cov mypy ruff 2>nul
    ) else (
        where uv >nul 2>&1
        if !errorlevel! equ 0 (
            uv init 2>nul
        )
    )
)

if "%STACK%"=="go" (
    echo [INFO] Initializing Go project...
    where go >nul 2>&1
    if !errorlevel! equ 0 (
        go mod init %PROJECT_NAME% 2>nul
    )
    (
echo .PHONY: build test lint run
echo.
echo build:
echo 	go build -o bin/app ./cmd/...
echo.
echo test:
echo 	go test -v -race -cover ./...
echo.
echo lint:
echo 	golangci-lint run
echo.
echo run:
echo 	go run ./cmd/main.go
    ) > Makefile
)

if "%STACK%"=="node" (
    echo [INFO] Initializing Node/TypeScript project...
    where npm >nul 2>&1
    if !errorlevel! equ 0 (
        npm init -y 2>nul
        npm i -D typescript @types/node tsx vitest 2>nul
    )
)

if "%STACK%"=="ts" (
    echo [INFO] Initializing Node/TypeScript project...
    where npm >nul 2>&1
    if !errorlevel! equ 0 (
        npm init -y 2>nul
        npm i -D typescript @types/node tsx vitest 2>nul
    )
)

:: Git initialization
if not exist ".git" (
    where git >nul 2>&1
    if !errorlevel! equ 0 (
        git init >nul
        echo [SUCCESS] Initialized git repository
    )
)

:: Create .gitignore if not exists
if not exist ".gitignore" (
    (
echo # Dependencies
echo node_modules/
echo __pycache__/
echo .venv/
echo vendor/
echo.
echo # Build
echo dist/
echo build/
echo bin/
echo *.egg-info/
echo.
echo # IDE
echo .idea/
echo .vscode/
echo *.swp
echo.
echo # Environment
echo .env
echo .env.local
echo *.secret
echo.
echo # OS
echo .DS_Store
echo Thumbs.db
echo.
echo # Test/Coverage
echo .coverage
echo htmlcov/
echo .pytest_cache/
echo coverage.out
    ) > .gitignore
    echo [SUCCESS] Created .gitignore
)

echo.
echo [SUCCESS] Project initialized with Claude Code configuration
echo.
echo Structure:
echo   CLAUDE.md              - Project instructions
echo   .claude\skills\        - Custom skills
echo   tasks\todo.md          - Task tracking
echo   tasks\lessons.md       - Learning log

endlocal
