---
tags: [spec, tasks, templates]
created: "2026-08-27"
---

# Tasks - HARNESS-045-hook-emission

## Done this sitting

- [x] [AC8] Persona loader with declared per-skill severity, on a real YAML
      parser rather than a third hand-rolled frontmatter reader
- [x] [AC8] `Decide()` — the whole policy as a pure function over a
      harness-neutral `ToolCall`, testable with no harness installed
- [x] [AC3] [AC4] `dotf harness gate` — exits 0 or 2 and nothing else, so the
      harnesses' contradictory hook-error semantics never apply
- [x] [AC5] [AC6] `MergeHooks` — find-by-marker emission that coexists with a
      live third-party writer, verified against the REAL deployed settings file
- [x] [AC1] `agents.bind` declared in `harness/manifest.json`, every event name
      measured against the installed harness

- [x] [AC1] Wire `bind` into setup, replacing `merge_claude_settings`'s
      positional jq paths. **Done 2026-08-31.** Both scripts call
      `dotf harness bind` behind a `harness --help` capability probe; both merge
      functions lost their hook parameters and assignments; the template lost its
      `hooks` block. Measured against a COPY of the deployed files, not a
      fixture: `~/.claude/settings.json` SessionStart stayed at 2 entries in 2
      groups (ours adopted and marked, Orca's untouched), SessionEnd 1→1 adopted,
      PreToolUse 1→2 (the gate appended as a NEW group), every other event
      unchanged; `~/.gemini/settings.json` BeforeTool 1→2 with no top-level key
      lost. A second run reported `hooks already current` on both — changed=0.

      **One defect found and fixed before the cutover was safe: on Windows this
      would have DUPLICATED the session hook.** Adoption is by exact command
      equality, and `setup-windows.ps1` deployed `"…\dotf.exe" mem session-start`
      — quoted, `.exe` — while `resolveDotfPath` produced neither. Fixed by
      `hookBinaryToken(path, goos)` (goos is a PARAMETER, so the Windows branch
      is exercised from Linux) plus the suffix in `resolveDotfPath`. Both pinned
      by test and mutation-checked. Caught statically, so no Windows sitting was
      needed to find it — but the first real Windows run is still the only proof.

      **A correction to what this file said.** The red list named "the doctor's
      PAT checks" and five bats files. Measured: `checks_pat.go` is GitHub token
      expiry and touches none of this, no doctor check asserts the hook shape at
      all, and `session-start-config.bats` only pins `session-start-config.json`'s
      schema. FOUR bats files went red, and the fourth was not on the list:
      `guard-no-session-handoff.bats`, which asserted the SessionEnd command as a
      literal in each setup script. Each was rewritten to follow the SSOT to the
      manifest rather than deleted.

      **The task was mis-stated, and measurement changed its shape.** It read
      as tidying: swap positional jq paths for a marker-based merge. Replayed on
      2026-08-27 against a copy of the DEPLOYED `~/.claude/settings.json`, the jq
      assignment took SessionStart from 2 groups to 1 — deleting Orca's live
      hook. `setup-windows.ps1` carried the identical assignment for both events.
      So it was never tidying; it removed a defect that fired on the next setup
      run on either OS.

      **The emitter (`dotf harness bind`, built 2026-08-27):** manifest-driven
      emission over `agents.bind`, which gained `emit_hooks` (id/event/command
      per target); atomic temp+rename; writes only when changed; refuses an
      unparseable settings file rather than bootstrapping over someone's edit;
      skips `emit: false` out loud and `requires_command` when absent.
      Mutation-checked: neutering `sameCommand` puts two copies of the mem hook
      in SessionStart.

## Done: the enabler AC3/AC4 were waiting on (2026-08-31)

- [x] **The gate learns its persona from the payload.** MEASURED: the whole chain
      — bind, hook, gate, `Decide` — was live and enforced NOTHING. The manifest
      emits `harness gate --harness claude` with no `--role`; `loadGatePersona`
      returned nil on an empty role; `Decide` answered *"no persona in scope"*.
      Every tool call was allowed whatever any skill declared, on all 35 skills.

      A static `--role` in the manifest could not fix it: one hook serves the
      whole harness, so it would pin every session to one persona. Claude
      documents `agent_type`/`agent_id` on every hook event fired inside a
      subagent and neither on a main-thread call, so **the harness already knew
      what the gate was missing** — no session-state mechanism, nothing to clean
      up on a crash, no SubagentStart/Stop lifecycle. `--role` survives as an
      override.

      **Consumption is now scoped to the acting persona, not the session.** A
      subagent reuses the parent's session id, so keying by session alone let one
      persona's skill runs satisfy another's gate.

      Verified end to end against a record declaring severity: main-thread call
      allows; dispatched reviewer blocks with exit 2 **naming the skill to
      invoke**; invoking it allows the next call while still warning on the
      unconsumed `enforce: warn` skill; a second agent in the same session does
      not inherit that consumption.

      **DOCUMENTED, NOT MEASURED on this box:** the payload field names. The gate
      is live in no deployed settings file here. The design makes that
      acceptable — a wrong name yields no persona, hence Allow, the pre-existing
      behaviour — so a guess costs enforcement, never a blocked session. Confirm
      by observing `[gate] warn` on a real dispatch **before** promoting any skill
      to `enforce: block`.

- [x] **`dotf doctor` reports unmigrated skills.** The check existed only as a
      comment: `Decide`'s EnforceUnset branch claimed "surfaced by `dotf doctor`",
      `UnmigratedSkills()` was written and unit-tested, and **nothing in
      production called it**. It now answers **35 of 35**. WARN, never FAIL — an
      unmigrated skill is a deliberate state, and failing the health command over
      planned work trains the reader to ignore the line.

- [x] **A role that does not resolve is said out loud.** `--role reviewr` exited 0
      in silence, so a typo or a renamed record disabled enforcement while every
      session reported health. Allowing is the right decision; silence was not.

- [x] **Fixed a cross-cutting regression found on the way** (`d4ea0f5`, #1404,
      merged the same day): `rootCmd.SetErr(io.Discard)` discarded every
      deliberate stderr write in the CLI — 16 call sites across 9 command files,
      all four `dotf secrets` subcommands among them. The gate blocked with exit 2
      and NO reason. Fixed via Cobra's `SilenceErrors`, which is the mechanism for
      suppressing only the automatic wrapper; two guards added.

## Still not done, and deliberately so

- [~] **Migrate the 35 skills to declared severity.** CANARY LANDED, 31 REMAIN.
      `reviewer` migrated vault-side (`c9b3181c` on knowledge master), all four
      skills at `enforce: warn`, and `compile-harness.sh --refresh` carried it
      into `harness/agents/reviewer/AGENT.md`. Refresh touched that record and
      nothing else — the blast radius was the whole question, and it was measured
      rather than assumed. `dotf doctor` moved from **35 of 35** unmigrated to
      **31 of 35**, so the migration now has a progress meter instead of a plan.

      **`check-roster-consistency.py` was updated in the same movement**, and the
      shape is the one its own docstring prescribed: it now calls
      `dotf harness resolve-skills` instead of growing the repository's third
      hand-rolled frontmatter reader. Loud failure relocated, not weakened — a
      non-zero exit, a missing binary, or a timeout all raise `UnreadableSkills`;
      nothing degrades to `[]`. **Proof the update was necessary and not
      cosmetic:** the pre-change guard, run from `git show HEAD:` against the
      migrated vault, exits 1 naming the record. AC7's loud failure did exactly
      what it was built for — it refused instead of passing silently.

      Severity decided by the owner, not inferred: all four at `warn` now, and
      **only `verification-before-completion`** is a candidate for `block` later.
      The other three degrade a review by their absence; they do not invalidate
      one.

      Canary measured end-to-end against the migrated record, six cases:

      | Case | Payload | Result |
      |---|---|---|
      | A | `agent_type: reviewer`, Bash | 4 `[gate] warn` lines, **exit 0** |
      | B | no `agent_type` (main thread) | silent, exit 0 |
      | C | `agent_type: builder` (unmigrated) | silent, exit 0 — `EnforceUnset` acts on nothing, as designed |
      | D | `Skill` tool, `skill: audit` | consumption recorded, exit 0 |
      | E | same `agent_id`, next Bash | 3 warns; `audit` gone |
      | F | **same role, different `agent_id`** | all 4 warns again |

      F is the reviewer finding from #1410 verified at runtime rather than only
      in a unit test: a second dispatch of one role does not inherit the first's
      consumption. It confirms the ledger separates by invocation **given
      distinct ids** — it does not confirm the harness sends distinct ids, which
      is still the open measurement.

- [ ] **Confirm `agent_type` / `agent_id` against a REAL dispatch.** The canary
      above was driven by hand-written payloads, which proves the gate's logic
      and NOT the field names. Until `[gate] warn` lines are observed in a
      genuine subagent dispatch, `agent_type` remains documented-not-measured,
      and the same observation must check that two dispatches of one role carry
      different `agent_id`s. **Precondition for promoting anything to `block`.**
      Nothing before this point is enforcement; it is instrumentation.

- [ ] **Migrate the remaining 31 skills** across `architect` (3), `builder` (9),
      `curator` (8), `planner` (6), `shipper` (4) and `hermes-nan` (1). Blocked
      on nothing technical — the path is proven — but each role's severities are
      an owner decision, and the canary's evidence should be in hand first.


- [ ] [AC2] Presence emission. Only the Action half is built.
- [ ] **The pi and opencode TS wrappers.** Declared in the manifest with
      `emit: false` so the gap is visible rather than remembered. They need two
      generated-code templates plus their deploy, which is a sitting of its own —
      splitting it was a decision, not an omission.
- [x] [AC7] The block-style guards. **Corrected against measurement, 2026-08-27:
      THREE parsers read `skills:` and TWO break silently**, not four and three.
      The count carried forward was reconstruction; each line below was executed
      against a record in the mapping form rather than read.
      - `scripts/compile-harness.sh:1016` — `skill_field` returns EMPTY, so the
        presence block renders `MUST consume: none`. The worst of the three: it
        is the presence mechanism itself, and it disarms every harness at once.
        **MEASURED.**
      - `specs/HARNESS-046/check-roster-consistency.py:64` — falls back to `[]`
        and compares "no skills" against the roster. It is that spec's declared
        verification (`features.json:26`), so it runs. **MEASURED.**
      - `cli/internal/doctor/checks_agent_tiers.go` `readAgentFrontmatter` —
        **NOT a breakage.** It does skip indented lines, but its only consumers
        are `fm["model"]` and `fm["targets"]`; it never reads `skills`. A latent
        trap with no live effect, so no guard was built for it. Building one
        would be a guard over a non-consumer.

      **Shape decided this sitting: delegate, do not copy.** Copying
      `agent_capability_line`'s refuse would have made the migration a flag-day —
      the renderer would reject every migrated record permanently, and that
      renderer feeds the presence block on all four harnesses. Instead
      `build_agent_presence` calls the new `dotf harness resolve-skills`, which
      reads both forms through `LoadPersona`'s real YAML parser. The copied
      refuse survives where it is still needed: **inside the degrade path**, for
      a dotf that is absent or predates the subcommand, where the awk fallback
      would otherwise render `none`.

      Contract: ids only, no `--harness` flag. Severity is `dotf harness gate`'s
      input, not the presence text's — and the doctrine payload has no room for
      it, being capped at 12000 characters for `.gemini/GEMINI.md` (breached
      twice already, most recently by the roster going from one persona to seven).

      Verified byte-identical to the awk read across all 7 shipped records, and
      the stub now advertises `resolve-skills`, so **every pre-existing presence
      test in `compile-harness.bats` exercises the delegated path** — their
      unchanged expectations are the equivalence proof.

- [ ] [AC7] **Not met, and it cannot be met by writing the schema.** AC7 asks for
      a loud failure "in the schema validation". Measured: `compile-harness.sh`'s
      `validate_skill_frontmatter` evaluates `jq -r '.required[]'` and nothing
      else — its own comment says "do not read a passing --check as type
      validation" — and `skills` is not in `required`. So no type constraint in
      `agent-frontmatter.schema.json` is enforced by anything. The subschema was
      written anyway, as the SSOT for the shape that `parseSkills` implements,
      and it is labelled unenforced in its own description. **Ticketed as
      HARNESS-089 (#1318)** rather than absorbed here — the hole is wider than
      this field and wider than this schema, so fixing it inside an AC7 sitting
      would have been the wrong size. Enforcement today lives in `parseSkills`
      plus the two guards above.

- [ ] Migrate the 35 skills across 7 personas to declared severity. Unblocked by
      the guards above. **The records under `harness/agents/` are GENERATED**
      (`generated_from: 00_meta/agents/definitions/...`): the migration edits the
      vault definitions and re-runs `compile-harness.sh --refresh`, vault
      committed direct to master. Editing the records in place is clobbered by
      the next refresh. Per-skill severity is a design decision for the owner,
      not 35 silent judgement calls.

## Closing

- [x] `go test ./...` — 18/18 packages
- [x] `go build`, `go vet`, and `GOOS=windows go vet` all pass
- [x] `golangci-lint run` on the pinned 2.12.2 — 0 issues
- [x] `bats tests/compile-harness.bats` — 72/72, exit 0 (the manifest edit adds a
      key without disturbing the render)
- [x] The gate driven end to end as a binary, on all four harnesses
- [x] Emission verified against the real `~/.claude/settings.json`
- [ ] PR opened referencing this spec folder
- [ ] Independent adversarial review before archive — the implementing session
      cannot be the reviewer

## Machine-readable features

`features.json` is the harness-facing contract. The agent may not write
`"state": "passing"`; only the harness may, after running `verification` and
capturing exit code 0.
