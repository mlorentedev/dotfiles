@echo off
setlocal enabledelayedexpansion

:: windows-setup.bat
:: One-time setup: copies Claude Code skills and config to Windows home
:: Run from the dotfiles repository root

set CLAUDE_HOME=%USERPROFILE%\.claude
set SCRIPT_DIR=%~dp0
set REPO_DIR=%SCRIPT_DIR%..

echo [INFO] Setting up Claude Code for Windows...

:: Create directories
if not exist "%CLAUDE_HOME%" mkdir "%CLAUDE_HOME%"
if not exist "%CLAUDE_HOME%\skills" mkdir "%CLAUDE_HOME%\skills"

:: Copy CLAUDE.md
if exist "%REPO_DIR%\ai\claude\CLAUDE.md" (
    copy /Y "%REPO_DIR%\ai\claude\CLAUDE.md" "%CLAUDE_HOME%\" >nul
    echo [SUCCESS] Copied CLAUDE.md
) else (
    echo [WARNING] CLAUDE.md not found
)

:: Copy init-project.bat
if exist "%REPO_DIR%\scripts\init-project.bat" (
    copy /Y "%REPO_DIR%\scripts\init-project.bat" "%CLAUDE_HOME%\" >nul
    echo [SUCCESS] Copied init-project.bat
)

:: Copy skills (each skill directory)
for /D %%s in ("%REPO_DIR%\ai\skills\*") do (
    set SKILL_NAME=%%~nxs
    if not exist "%CLAUDE_HOME%\skills\!SKILL_NAME!" mkdir "%CLAUDE_HOME%\skills\!SKILL_NAME!"
    xcopy /E /I /Y "%%s\*" "%CLAUDE_HOME%\skills\!SKILL_NAME!\" >nul 2>&1
)
echo [SUCCESS] Copied skills to %CLAUDE_HOME%\skills\

echo.
echo [SUCCESS] Claude Code configured at %CLAUDE_HOME%
echo.
echo To initialize a new project, run:
echo   %CLAUDE_HOME%\init-project.bat my-project python
echo.
echo Or add %CLAUDE_HOME% to your PATH and run:
echo   init-project.bat my-project python

endlocal
