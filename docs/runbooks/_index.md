# Dotfiles — Runbooks Index

Operational guides and procedures for managing the dotfiles environment.

| Runbook | Description | Status |
|---|---|---|
| [guide-new-machine.md](guide-new-machine.md) | A machine from zero: converge, `dotf doctor --scope machine`, then restoring the identity (age key, Bitwarden, GitHub, vault) | Active |
| [guide-secrets-governance.md](guide-secrets-governance.md) | Secrets lifecycle: first machine, add, converge, curate, rotate, retire, backup, offline copy, recover, quarterly check | Active |
| [guide-opencode-go-setup.md](guide-opencode-go-setup.md) | OpenCode setup, NaN provider, models & coexistence rules | Active |
| [guide-agent-provisioning.md](guide-agent-provisioning.md) | What each AI agent gets (instructions, skills, configs, hooks, MCP, plugins) and which file declares it | Active |
| [guide-cross-agent-memory.md](guide-cross-agent-memory.md) | Cross-agent session memory bridge (handoff threads, vault-linked auto-memory) | Active |
| [guide-antigravity-cli-migration.md](guide-antigravity-cli-migration.md) | Antigravity CLI (`agy`) setup, permissions & model configuration | Active |
| [guide-pr-agent-reviewer.md](guide-pr-agent-reviewer.md) | PR-Agent automated code review workflow | Active |
| [guide-knowledge-distillation.md](guide-knowledge-distillation.md) | Knowledge crystallization, observation promotion & vault syncing | Active |
| [guide-tmux.md](guide-tmux.md) | Tmux terminal multiplexer workflows, keybindings & clipboard bridge | Active |
| [guide-self-deploy-timer.md](guide-self-deploy-timer.md) | Self-deploy background timer and autodeploy configuration | Active |
| [release-dotf.md](release-dotf.md) | Releasing and installing `dotf`: mirror, deploy, install, verify by effect | Active |
| [guide-git-config.md](guide-git-config.md) | Git config ownership (`~/.gitconfig` vs the deployed include) and the GitHub credential helper that works without a shell | Active |
| [tool-installation.md](tool-installation.md) | How tools reach a machine: one channel per class, the dotf catalog, and adding a tool | Active |
