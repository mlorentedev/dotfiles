---
id: lesson-026-config-deployment-guards-vs-tool-installation-guar
type: lesson
status: active
created: "2026-03-25"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 026: Config deployment guards vs tool installation guards

**Context**: Made Gemini config deployment conditional on `command -v gemini`. CI integration tests failed because the Docker container doesn't have gemini installed.

**Problem**: Conflated two concerns: (1) installing a tool's dependencies (needs the tool ecosystem present) and (2) deploying config files (just copying markdowns). The guard prevented harmless config from being deployed, breaking CI and also preventing pre-deployment on machines where the tool will be installed later.

**Solution**: Removed the CLI guard from config deployment. Config files are always deployed. Guards remain only around actual tool installation commands (e.g., `gh extension install github/gh-copilot`).

**Rule**: Separate "deploy config" from "install tool". Config file deployment (copying markdown, YAML, JSON) is always safe and should run unconditionally. Only guard commands that install binaries, extensions, or packages. A machine without the CLI benefits from pre-deployed config — it's ready when the tool arrives.
