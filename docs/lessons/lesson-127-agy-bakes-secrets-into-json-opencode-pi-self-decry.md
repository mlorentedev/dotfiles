---
id: lesson-127-agy-bakes-secrets-into-json-opencode-pi-self-decry
type: lesson
status: active
created: "2026-06-25"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 127: agy bakes secrets into JSON; opencode/pi self-decrypt (they ignore ambient env)

**Context**: Migrating setup off the load-secrets eager-source (which populated $NAN_API_KEY/$OPENROUTER_API_KEY in the setup process env for deploy-time config materialization).

**Problem**: Assumed both opencode and agy consumed the eager-loaded ambient env. They don't. `substitute_env_placeholders` (utils.sh) / `Substitute-EnvPlaceholders` (utils.ps1) resolve {env:VAR} by reading env-mapping.conf and age-decrypting the .secret.age file DIRECTLY -- they ignore the ambient env. Only the agy MCP block reads $env:OPENROUTER_API_KEY, because agy does NOT expand env vars inside JSON, so the key must be baked into mcp_config.json at deploy. So the eager NAN_API_KEY fetch was dead code, and env-mapping.conf can't be deleted while the substitute functions still read it.

**Solution**: B3 fetches only OPENROUTER_API_KEY via `dotf secrets show` for agy; dropped the dead NAN fetch. Left env-mapping.conf; tracked the substitute-functions -> registry migration (a future `dotf secrets render`) as the last step before deleting env-mapping.conf (#587).

**Rule**: Trace each secret consumer to its ACTUAL resolution path before migrating it. Two configs using {env:VAR} can resolve via completely different mechanisms (self-decrypt vs deploy-time bake vs runtime). Read the substitution function; don't assume ambient env.
