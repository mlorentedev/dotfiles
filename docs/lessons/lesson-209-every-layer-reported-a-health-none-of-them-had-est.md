---
id: lesson-209-every-layer-reported-a-health-none-of-them-had-est
type: lesson
status: active
created: "2026-08-16"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 209: Every layer reported a health none of them had established

**Context**: `pi` returned `401 Invalid API key` on every call against the `nan` provider, while the same key sent by hand returned 200. BUG-081b (#987).

**Problem**: `ai/pi/models.json` carried `{env:NAN_API_KEY}` — **dotf's** placeholder syntax. pi's resolver interpolates `$VAR` and `${VAR}` only, so a string with no `$` is a literal: pi sent the 17-character placeholder itself as the bearer token. Three separate layers each asserted something none had checked. `setup-linux.sh` logged that placeholders were *"resolved at runtime"*; nothing resolves that syntax. pi's own preflight `isConfigValueConfigured("{env:NAN_API_KEY}")` returned `true`, because a string with no `$` has no variables that could be missing. And the server can only answer 401, which is indistinguishable from a bad credential — so a session spent most of its length inspecting Bitwarden, the registry and the injection path, all of them healthy. A fourth layer joined later: `tests/pi-config.bats` asserted the config *used* `{env:NAN_API_KEY}`, so it passed for exactly as long as the bug existed.

**Solution**: the config declares `${NAN_API_KEY}` and pi resolves it from the environment `dotf secrets run` already injects (ADR-034). The credential stops being written to disk at all, which also retires the second copy that made rotation a two-store operation. The bats assertion was **inverted rather than deleted** — from "uses this placeholder" to "carries no placeholder the consumer cannot resolve" — turning the test that protected the bug into the guard against its regression. A `dotf doctor` check fails on either defect in the deployed copy, naming the provider and never the value.

**Rule**: when a config names a variable, the syntax belongs to **whoever resolves it**, not to whoever writes the file — and that ownership is worth verifying against the consumer's own resolver rather than inferring from a placeholder that looks conventional. More generally: a check that asserts a **symptom** ("uses placeholder X") passes while the bug lives; a check that asserts the **contract** ("carries nothing the consumer cannot resolve") fails. When several independent layers all report healthy and the system is broken, suspect that none of them measured the thing — an "unresolved" state that no layer models will be reported as fine by all of them.

**Tags**: `secrets`, `config`, `pi`, `diagnostics`, `testing`, `guard-design`
