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

## Not done, and deliberately so

- [ ] [AC1] Wire `bind` into setup, replacing `merge_claude_settings`'s
      positional jq paths. The merge is written and tested; nothing writes to a
      live file yet.
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
