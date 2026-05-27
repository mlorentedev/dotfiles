---
id: "AI-010-ollama-native"
type: spec
status: draft # draft | implementing | verifying | archived
created: "2026-05-27"
tags: [spec, proposal]
template_version: "1.0"
---

# AI-010-ollama-native

> **Naming**: file lives at `<repo>/specs/AI-010-ollama-native/proposal.md`. `AI-010-ollama-native` is `YYYY-MM-DD-<slug>` or `<TICKET-NN>`.

## Why

<!-- from 11-tasks.md: Native Windows Ollama installer (`winget install Ollama.Ollama` with `.exe` fallback) integrated into `setup-windows.ps1` (opt-in `-WithOllama` flag, ~5–40 GB disk). Detected in healthcheck section 6/10. Default model(s) decided in the proposal, not here. Independent. -->

`healthcheck.ps1` section 6/10 probes for Ollama on Windows, but `setup-windows.ps1` never installs it. Users see a healthcheck FAIL on Windows and don't have a one-command remedy from the dotfiles toolkit — they must manually find the Ollama Windows installer. Linux side already has opportunistic install hooks for AI tools (Antigravity in PR #121, OpenCode in PR #34); Ollama on Windows is the missing parity piece for local-LLM workflows.

## What

`setup-windows.ps1` learns to install Ollama via `winget install Ollama.Ollama`, with `.exe` fallback (download from `https://ollama.com/download/OllamaSetup.exe` and run silently) when winget is unavailable or fails. Behind a new `-WithOllama` switch (default OFF — Ollama install pulls 5-40 GB of models).

## Out of scope

- **Auto-pulling default models** (`ollama pull llama3.2`, etc.). Default model choice is the user's; this ticket installs the runtime only.
- **Linux Ollama install hook** — current Linux setup already supports Ollama via Antigravity / OpenCode integration; if Linux native install is also wanted, separate AI-XXX.
- **GPU detection** (CUDA / DirectML / ROCm probes). Ollama auto-detects at runtime; setup script does not pre-flight.
- **Service-mode configuration** (start Ollama as a Windows Service) — out of scope; user starts manually or via Task Scheduler.

## Risks / open questions

- **R1**: winget availability. Windows 10 pre-1809 lacks winget. Detect via `Get-Command winget -ErrorAction SilentlyContinue` and branch to `.exe` fallback.
- **R2**: Silent install of `.exe`. Ollama setup may not support `/S` flag; verify via `OllamaSetup.exe /?`. If interactive-only, log a warning and instruct user to run it manually.
- **R3**: Disk-space preflight. 5 GB minimum (runtime) + model size. Probe `Get-PSDrive C` free space before install; abort cleanly if insufficient.
- **R4**: Healthcheck integration. Section 6/10 should pass after `-WithOllama` install. Bats verifies the round-trip.
- **R5**: Default model decision. **Open question for proposal phase**: should `-WithOllama` also include a `-OllamaModel <name>` companion flag? Recommendation: no — keep the install ticket atomic; model-pull is a separate workflow.

## Acceptance criteria

- [ ] `setup-windows.ps1` accepts `-WithOllama` switch; default OFF.
- [ ] When flag set: `winget install Ollama.Ollama` runs; on failure, `.exe` fallback engages.
- [ ] After install: `ollama --version` resolves on PATH.
- [ ] Healthcheck section 6/10 PASSes after `-WithOllama` install.
- [ ] Disk-space preflight prevents install when < 5 GB free; logs WARN.
- [ ] Bats: structural assert flag exists; happy-path mocks winget; bad-path covers winget-absent fallback to `.exe`.
- [ ] README documents the opt-in flag + disk-size disclaimer.

## References

- Vault: `10_projects/dotfiles/11-tasks.md` → AI-010.
- Healthcheck probe site: `scripts/healthcheck.ps1` section 6.
- Companion (Linux opportunistic AI installs): PR #121 (agy auto-install), PR #34 (OpenCode bootstrap).
