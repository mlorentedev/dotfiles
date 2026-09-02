---
id: "HARNESS-106-skill-capability"
type: spec
status: implementing # draft | implementing | verifying | archived
created: "2026-09-01"
issue: "mlorentedev/dotfiles#1420"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# HARNESS-106-skill-capability

## Why

<!-- from issue #1420: HARNESS-106: no persona can consume the skills its own gate demands, because no capability maps to the skill-invocation tool -->

No persona can invoke a skill, so no persona can consume the forced skills its
own gate demands. `harness/capability-map.json` declares the neutral vocabulary
`[read, search, edit, shell, web]` and nothing maps to a skill-invocation
primitive, so all seven personas deploy without it — and claude's `tools:` is an
allow-list, which that map's own audited comment records: *"a tool not named is
unavailable to the agent."* Under `enforce: warn` this is silent noise; under
`enforce: block` it is a hard deadlock, because `dotf harness gate`'s escape
hatch (*"invoking a skill is never blocked: forbidding it would deadlock the
session"*) assumes an agent able to emit a call it was never granted.

## What

A persona that declares `skills:` can invoke a skill, on every harness where
that is expressible, and the gate leaves a durable record of every decision it
makes — so a `warn` canary accumulates evidence instead of emitting into a
stream nothing persists.

Both halves are one change because they are one defect seen from two sides:
without the capability there is no consumption, and without consumption or a
decision record there is nothing to observe. Measured 2026-09-01 — see
`docs/lessons/lesson-254-a-canary-that-emits-where-nothing-persists.md`.

Concretely, after this:

- Each persona record declaring `skills:` also declares a neutral
  skill-invocation capability, in the **vault SSOT**; the repo's capability map
  translates it per harness; a deploy propagates it. Nothing is edited in a
  generated file, and nothing is configured per session or per machine.
- `dotf harness gate` appends one record per decision — allow, warn, block, and
  `role did not resolve` — to a durable location, independent of stderr, of exit
  code, and of whether the acting persona holds the skill tool.
- `agent_type` becomes **measured** rather than inferred, because the decision
  record is the first channel that can report which persona the gate resolved.

## Out of scope

- **Promoting any skill to `enforce: block`.** This makes promotion *possible*
  and *observable*; the promotion is a separate decision on separate evidence.
- **Migrating the remaining 31 skills to a declared severity.** Gated on this:
  migrating first would add 31 more unsatisfiable declarations and 31 more
  unobservable warns.
- **The gate's user-level binding scope** (#1421) — it forks a `dotf` on every
  tool call in every repo on the machine. Real, filed, orthogonal.
- **`dotf pr triage-queue`'s timestamp comparison** (#1422).
- **Adding copilot to `agents.bind`.** It is absent from the manifest entirely,
  so it has no gate path to fix; deciding whether it gets one is its own work.

## Risks / open questions

- **A harness with no native skill primitive is the design decision here, not a
  footnote.** `claude` maps cleanly (`Skill`, an allow-list entry). `opencode`
  does not: its form is a `permission:` decision-map whose native keys are
  read/edit/glob/grep/list/bash/webfetch/websearch, and it has no skill concept.
  The map is documented to **fail loudly** on an unmapped capability and must
  never fall back to a permissive default (constraint C15), so silently omitting
  it is not available. It needs an explicit way to declare *"this harness has no
  equivalent"* that is distinguishable from *"nobody has mapped it yet"* — the
  same distinction the roster guard already enforces between an unreadable skill
  list and an empty one.

- **ADR-027 and #561 name the wrong event for opencode.** They call
  `tool.execute.before` opencode's gate. Measured against the installed
  `@opencode-ai/plugin` type definitions, it is
  `(input, output: {args}) => Promise<void>` and **cannot deny** — it can only
  mutate arguments. opencode's blocking primitive is `permission.ask`, whose
  `output.status` accepts `"deny"`. So opencode can hold the capability and
  still not be able to *enforce*; those are separate properties and this spec
  must not conflate them.

- **agy's payload field names are unverified.** agy authors its own payload and
  no agy payload has ever been captured. A mismatch degrades to "not
  understood", which allows and writes nothing — indistinguishable from working.
  The decision record is what makes agy verifiable at all; until it exists, no
  claim about agy can be made either way.

- **Where the record lives, and what reads it.** The consumption ledger under
  `~/.local/state/dotfiles/gate/` is per-scope state, not an audit trail, and
  reusing it would conflate two purposes. Whatever is chosen must survive the
  session that wrote it and be machine-readable.

- **Anything that reads the gate's state directory must be Go or `bash -c`,
  never an inline zsh loop.** Measured 2026-09-01: a verification loop over
  `NEW=~/...-*.json` printed "no file for my session id" with the file present,
  because zsh neither globs nor word-splits an unquoted scalar. Two rows of this
  project's own prohibited-pattern table, and both fail silently with an empty
  result that reads as a finding.

- **A named subagent's `agent_id` collides in the state path's readable prefix.**
  `StatePath` truncates the sanitised scope to 48 characters, of which a session
  UUID plus its hyphen consumes 37 — so 11 characters of the agent id survive,
  and role-based names share those routinely. The sha256 suffix keeps the file
  path correct, but any *reader* matching by prefix will treat two scopes as one.
  See `docs/lessons/lesson-255-truncation-not-hostile-input-made-the-digest-load-bearing.md`.

## Acceptance criteria

- [x] **AC1** — A neutral skill-invocation capability exists in the map's
      `vocabulary`, schema-valid, and `dotf harness resolve-capabilities
      --harness claude` emits the native tool for a persona declaring it.
- [x] **AC2** — A harness with no native equivalent is declared as such
      **explicitly**. Resolution neither silently omits the capability nor falls
      back to a permissive default, and "declared inapplicable" is
      distinguishable from "unmapped" in both the schema and the failure message.
- [x] **AC3** — Every persona record declaring `skills:` also declares the
      capability, asserted by a guard that fails red when one does not.
- [ ] **AC4** — A dispatched persona **can invoke a skill**, proven by a dispatch
      that writes a consumption record — never by a config file containing a key.
- [ ] **AC5** — `dotf harness gate` writes a durable record for every decision it
      takes, including `allow` and `role did not resolve`, reachable without
      `--debug` and without reading a transcript.
- [ ] **AC6** — A `warn` decision is **observable after the session ends**, from
      that record alone.
- [ ] **AC7** — `agent_type` is reported in the record, so a real dispatch shows
      which persona the gate resolved. This is the criterion that converts the
      standing inference into a measurement.
- [x] **AC8** — The change propagates by deploy: the vault SSOT plus the repo map
      are the only edited sources, no generated file is hand-edited, and a
      re-run reports `changed=0`.
- [x] **AC9** — Every new check has a red-direction test that fails when the
      thing it guards is broken.

## References

- Bitácora: `mlorentedev/dotfiles#1420`
- `HARNESS-045-hook-emission` (#561) — **its AC3 and AC4 are falsified by this
  spec's measurements** and must be corrected there rather than restated here.
  AC4 asserts a `warn` skill *"emits"*, through a channel that does not persist;
  AC3 asks that a `block` skill causes a call to be blocked, which a deadlocked
  persona satisfies — an acceptance criterion the broken state passes.
- `harness/capability-map.json` — the vocabulary, and its C15 fail-loud constraint
- `docs/lessons/lesson-254-a-canary-that-emits-where-nothing-persists.md`
- `docs/lessons/lesson-255-truncation-not-hostile-input-made-the-digest-load-bearing.md`
- Adjacent, deliberately separate: #1421 (hook binding scope), #1422 (triage
  queue), #1418 (setup/doctor repo-root mismatch)
