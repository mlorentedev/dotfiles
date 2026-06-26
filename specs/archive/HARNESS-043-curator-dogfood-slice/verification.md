---
tags: [spec, verification]
created: "2026-06-25"
---

# Verification - HARNESS-043-curator-dogfood-slice

## Evidence

Decision: **presence = uniform marked-region injection** across claude/opencode/pi/copilot (agy out). Action/dispatch (plugin primitives) → H-045.

- [x] `curator` neutral definition exists + validates, `targets: [claude, opencode, pi, copilot]` → `00_meta/agents/definitions/curator/AGENT.md`; bats `agents: --check fails when a record is missing a required key (kind)`; `--check` prints `[check] OK -> agent curator`.
- [x] `agents` manifest block (`deploy[]` + `presence[]`) + render to `~/.claude/agents/curator.md` with provenance → bats `agents: --refresh writes a verbatim AGENT.md record`, `agents: --deploy renders agent-md (... neutral keys dropped)`.
- [x] Offline `--check` clean (no vault) → bats `agents: --check validates the record renders offline (no vault)`.
- [x] Forced skills present by code in every harness → bats `agents: --deploy injects a presence region (forced skills) into every harness instructions file` (claude CLAUDE.md, opencode/pi AGENTS.md, copilot instructions).
- [x] Coexistence + idempotence → bats `agents: presence injection is idempotent and leaves the patterns region intact` + `... appends a fresh region when the file has no presence markers`.
- [x] Per-harness `targets[]` honored → bats `agents: presence respects per-agent targets[] (a persona only appears for harnesses it targets)`.
- [x] Sandbox-`$HOME` deploy proof (real script, isolated repo+HOME) → all 4 instructions files get `presence=1 generated=1 skills=yes`, patterns region + user content intact, idempotent on re-deploy.

## Test status

- Agent subset: `bats -f 'agents:' tests/compile-harness.bats` → **9/9 pass**.
- Full suite: `bats tests/compile-harness.bats` → recorded at commit time (pre-existing patterns/skill tests must stay green; the AGENT-PRESENCE namespace is disjoint from the GENERATED region, so no interaction).
- Known pre-existing issue (NOT this slice): one ENGINE-002 test (`AC6 ... drift gate`) has an em-dash in its `@test` name that bats 1.13.0 fails to register, so the runner reports "executed N-1 of N" with exit 0 (silent skip). Filed as a separate bitácora ticket; untouched by this diff.

## Decisions made during implementation

- **Injection over hook (agnosticism + no OS axis):** every daily harness already loads a managed instructions file; presence ≡ injecting the forced-skills directive there. One uniform mechanism replaces the claude-only `SessionStart` shell-command hook — which also **eliminates** the Windows≠POSIX command-form debt the first build surfaced (text injection is cross-OS).
- **Plugins are the Action level, not Presence:** a `chat.system.transform` / `session_start` plugin that only adds text is equivalent to instructions-injection. Plugins earn their place only when they gate (Action) or activate per-agent (dispatch) → H-045/H-046.
- **Distinct marker namespace:** `<!-- BEGIN HARNESS AGENT-PRESENCE … -->` coexists with the patterns `GENERATED` region in the same file; the injector replaces only its own region (or appends if absent) and never touches the patterns region — asserted by test.
- **Global injection accepted:** presence is per-session, not per-active-persona (same as the old hook). Per-agent activation deferred to the roster fan-out (H-046).

## Promotion candidates

- [x] Lesson for `docs/lessons.md`? **yes** — "presence determinism is cheapest as marked-region injection into each harness's always-loaded instructions file; the provider plugin primitives (`SessionStart`/`chat.system.transform`/`session_start`) are the Action level, not Presence. Injection is uniform across harnesses and cross-OS, sidestepping the per-OS command-form problem a hook carries." Promote at archive.
- [ ] ADR-worthy? no — ADR-027 governs; this refines the Presence-level mechanism (injection) and defers plugins to Action. Worth a one-line note in ADR-027 / the pattern.
- [x] Pattern update? **yes** — `pattern-cross-agent-agent-pipeline.md` presence table → injection; plugins → Action level.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved to `specs/archive/HARNESS-043-curator-dogfood-slice/`
- [ ] Bitácora #559 ticked with PR link
- [ ] Promotions above executed (lesson + pattern update)
