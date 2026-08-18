---
id: lesson-073-onboarding-a-junior-remote-agent-verify-its-self-a
type: lesson
status: active
created: "2026-05-31"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 073: Onboarding a junior/remote agent: verify its self-authored docs, enforce boundaries mechanically

**Context:** HERMES-001 — integrating Hermes, a low-capability remote ops agent (Debian 13 on NaN infra, Telegram), into the dotfiles ecosystem via ai/hermes/setup.sh + a curated vault SSOT at 80_agents/hermes-nan/.
**Problem:** The agent's own self-authored vault docs had silently drifted from reality: validate.sh checked filenames (memory.md, skills.md) the folder no longer used (numbered 10-memory.md, 11-skills.md), a constitution AGENTS.md was referenced everywhere but did not exist, and the vault clone lived in ephemeral /tmp so a reboot lost the auto-push git hook. Separately, the write-zone boundary (commit only within 80_agents/) was instruction-only — a junior agent can ignore instructions, and prompt-injection or a bug could push anywhere.
**Solution:** Treat a junior/remote agent's own docs as CLAIMS to verify, not truth. Probe the real box before designing provisioning — config path (/hermes-home/config.yaml, not ~/.hermes/), the commit mechanism (git CLI vs MCP — determines whether git hooks fire), and credential handling (token was embedded in the remote URL) were ALL different from assumptions; one probe round saved a wrong design. Convert soft instruction-only boundaries into MECHANICAL guardrails: once the probe confirmed Hermes commits via git CLI, install local git hooks in its clone — pre-commit rejecting paths outside the write-zone + token-like content, pre-push rejecting non-fast-forward (force) pushes — each with a functional test. Hooks are local to the agent's clone (never tracked/cloned), so they harden one consumer without touching others.
**Tags:** `#hermes` `#agent-onboarding` `#guardrails` `#remote-agent` `#verify-before-trust` `#git-hooks`
