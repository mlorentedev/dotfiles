---
id: lesson-107-wire-all-consumers-must-enumerate-the-non-shell-on
type: lesson
status: active
created: "2026-06-18"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 107: "Wire all consumers" must enumerate the non-shell ones — services and daemons never source a shell profile

**Context**: After ADR-025 wired the vault-path cascade into shells, hooks and the Go CLI, the Claude Code Hive still pointed at the old path.

**Problem**: `paths.{sh,ps1}` only reach interactive shells. The hive `serve` daemon (systemd `--user` on Linux, Scheduled Task on Windows) and the agy MCP registration never source a shell profile, so a shell-only mechanism silently missed them. `hive client` is a stateless proxy; the daemon reads `HIVE_VAULT_PATH` from its OWN process env at start — and `claude mcp add` has no `--env`, so the path can't flow through the MCP registration either.

**Solution**: Provision the *service* environment directly — `~/.config/environment.d/10-hive-vault.conf` (read by the systemd `--user` manager for all user services) on Linux, and a User-scope env var (Scheduled Tasks inherit it) on Windows — both from the same cascade via a new `dotf env path <KEY>`.

**Rule**: When a change claims to "wire all consumers" of a value, list them by *execution context*, not convenience: interactive shells, login shells, services (systemd/launchd/Scheduled Task), cron, MCP daemons, GUI/ADE processes. Each has its own env-provisioning mechanism; a shell-profile mechanism reaches none of the non-shell ones. A proxy daemon (client→server) holds its state in the *server* — set the server's env, not the client's.
