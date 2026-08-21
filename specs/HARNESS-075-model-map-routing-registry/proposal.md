---
id: "HARNESS-075-model-map-routing-registry"
type: spec
status: implementing # draft | implementing | verifying | archived
created: "2026-08-21"
issue: "mlorentedev/dotfiles#1124"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal, harness, orchestration, routing, registry]
template_version: "1.0"
---

# HARNESS-075-model-map-routing-registry

## Why

<!-- from issue #1124: HARNESS: build model-map.json — the declarative routing layer ADR-032 specified and nobody built -->

ADR-032 specified `harness/model-map.json` on 2026-08-09 and it was never built, so the policy
layer of the orchestration front is a document rather than a file: nothing declares which pools
exist, what concurrency they allow, how tiers resolve per harness, or what a dispatch falls back to.
ADR-035 (2026-08-21) closed the four questions that were blocking the build — who owns the filename,
what shape it takes, whether its budget is enforced, and whether a forge concept may enter it. This
spec builds what those two ADRs now fully describe.

## What

`harness/model-map.json` exists and is consumed. Concretely, after this PR:

- The repository declares its inference pools, their concurrency budgets and interactive reserves,
  which harness reaches which pool, how a neutral tier resolves to a model id per harness, and the
  ordered fallback chain per tier.
- A Go loader reads it, and the two consumer classes are distinguishable at the API: `tiers`
  resolves at compile time for the render engine, `chains` resolves at run time for a dispatcher.
- `harness/model-map.schema.json` constrains it — the **first JSON Schema over a registry in this
  repository**.
- `dotf doctor` reports on it — the **first doctor check over a registry**. A map that is absent,
  unparseable or schema-invalid is reported **loudly**, and never as "no pools" or "no limits".

## Out of scope

Drawn deliberately, because ADR-035 decides the *shape* and this spec must not quietly become the
whole orchestration front.

- **The level-2 budget semaphore.** ADR-035 splits enforcement into declaration (level 1) and
  enforcement where `dotf` is the launcher (level 2). **Level 2 requires a dispatcher, and there is
  none** — `dotf agent` is `unknown command`. Declaring a budget nothing decrements is exactly what
  ADR-035 calls advisory, so this spec ships **level 1 only** and says so, rather than implying an
  enforcement that does not exist.
- **`harness/capability-map.json`** — re-scoped to #560 by ADR-035.
- **Any dispatcher, spawn path or `dotf agent` noun.** The map is read; nothing yet routes on it at
  run time beyond the loader's API.
- **Retrofitting a schema or doctor check onto the other four registries.** ADR-035 records that
  this file sets a precedent the others do not meet, and that whether they follow is a separate
  decision. `triggers.json`'s missing guard is filed as #1137.
- **Any forge concept.** Decided by ADR-035 and enforced here by the schema: there is no field for
  one.

## Risks / open questions

No unresolved open questions — ADR-035 closed the four that were blocking. The risks below are
known and mitigated in the acceptance criteria rather than left floating.

- **The reference schema in ADR-032 §3 carries two defects** and copying it verbatim would build
  them in: `harnesses.codex.pools` names a `codex` pool that `pools` never declares, and
  `pools.openrouter` describes a provider deleted upstream. AC3 and AC4 turn both into assertions.
- **A doctor check that only reports "file missing" satisfies the letter of C15 and not its
  intent.** The failure mode this repository keeps hitting is a positive-looking signal, so the
  check must distinguish *absent*, *unparseable* and *schema-invalid*, and must never render any of
  them as an empty-but-valid map. AC6 pins this.
- **AC2 and AC6 need schema validation, and `cli` had three direct dependencies.** The first
  decision (2026-08-21) was **native Go, no new dependency** (C7 plus the module's evident
  three-dependency discipline), with the validator reading `model-map.schema.json` as data so the
  schema file stayed the single source of truth and only its interpreter was hand-rolled.

  **That decision was taken with a falsifiable threshold, and the threshold fired.** Declared before
  the evidence arrived: *any NEW interpreter-semantics finding in adversarial review round 4 means
  the no-dependency choice is empirically a defect factory.* Round 4 delivered **two** — malformed
  `properties` elements and non-string `required` members skipped silently (the class round 3's
  Blocker was meant to seal, one level deeper), and `minimum` unimplemented so negative budgets
  validated.

  **Family tally across four rounds: eight**, every one of them something a conforming
  implementation gets right by construction — `minLength`, `pattern`, `minimum`, member types,
  `properties` element types, `enum` deep equality, malformed-schema detection, and a non-conforming
  `minLength` that trimmed whitespace where draft 2020-12 counts code points untrimmed. That last
  one meant **the schema's two consumers could disagree about the same document**: an editor using a
  real validator accepted `"  "` where `dotf` rejected it.

  **Revised 2026-08-21: `santhosh-tekuri/jsonschema/v6` interprets the standard keywords.** What did
  not change: the schema FILE remains the SSOT, and both custom cross-block rules stay, because no
  schema language expresses them. What changed is only who interprets standard keywords — and the
  keyword allow-list died with the interpreter it guarded, since draft 2020-12 treats unknown
  keywords as annotations, which is exactly the tolerance `x-poolReferencesResolve` and
  `x-tiersHaveChains` need.

  **Measured dependency cost, reported rather than assumed:** direct requires 3 → 4; one new module
  links into the binary alongside it, `golang.org/x/text`, an official Go sub-repository;
  `dlclark/regexp2` appears in the module graph but does not link. Not "zero transitive", not a tree.

  **The proof the swap loses nothing:** every named regression test from rounds 1–4 passes against
  the library-backed validator. Three needed changes and each is recorded in the code rather than
  quietly applied — one test was DELETED because its premise (an unimplemented construct to be loud
  about) no longer exists, one had its inline schema corrected to express non-blank the standard way
  rather than relying on the old trimming, and one had its assertion moved from the old error
  wording to the property. **A fourth failure was a real regression this swap introduced and is
  fixed:** the library does not police our own `x-` keywords, so a bare bool assertion re-opened the
  round-2 defect. They are type-checked explicitly now.

- **The neighbouring registry embeds a build-time copy of itself, and `model-map.json` must not.**
  `triggers.go` carries `//go:embed triggers.json` and falls back to it when the repo file is not
  found — and `harness/triggers.json` and `cli/internal/harness/triggers.json` are byte-identical
  duplicates with **no sync mechanism** (filed on #1137). For a routing map that fallback is
  disqualifying: an absent file would resolve to a build-time default and the doctor check would
  report it healthy, which is precisely what C15 forbids. **Decision: `model-map.json` is never
  embedded and has no fallback.** Where the file cannot be read, the loader errors. This is a
  deliberate divergence from the neighbouring idiom, recorded so it reads as a decision rather than
  an inconsistency.

- **Two consumer classes over one file is the drift risk `version` exists for.** The loader must
  expose them separately so a compile-time consumer cannot silently depend on a run-time field.

## Acceptance criteria

- [ ] **AC1** — `harness/model-map.json` exists, parses, and carries all seven declared blocks:
      `$comment`, `version`, `pools`, `harnesses`, `tiers`, `chains`, `services`. Both registry
      idioms are present (`$comment` for the rationale, `version` for the two consumer classes),
      per ADR-035 Decision 2.
- [ ] **AC2** — `harness/model-map.schema.json` exists as the declarative contract, and the shipped
      map validates against **that file** — read at validation time, never re-expressed as Go
      literals. Standard draft-2020-12 keywords are interpreted by a conforming library
      (`santhosh-tekuri/jsonschema/v6`), so the subset the file may use is the whole draft rather
      than an allow-list Go has to keep up with.

      *Amended 2026-08-21, after round 5.* As first written this criterion bound the validator to
      be native with no schema-engine dependency, and bound an unimplemented construct to be a loud
      error. The Risks section above records why the first half was abandoned — the threshold
      declared before round 4 fired — but the criterion itself was left standing, so the spec
      asserted a property the shipped code contradicts. The second half survives the pivot in the
      only place it still means anything: draft 2020-12 makes an unknown keyword an annotation,
      which is the tolerance the two `x-` cross-block rules depend on to coexist with the library,
      and is therefore also what would make a MISSPELLED rule name invisible. So the `x-` namespace
      is this repo's own and is validated as a closed set — an `x-` keyword the validator does not
      implement is a loud error. Standard keywords are the library's to interpret.
- [ ] **AC3** — The schema **rejects** a `harnesses.<h>.pools[]` entry naming a pool absent from
      `pools`. Proven by a fixture that fails validation, not by inspection.
- [ ] **AC4** — The built map declares no `openrouter` pool, and no harness references one.
- [ ] **AC5** — A Go loader in `cli/internal/harness/` reads the map and exposes the two consumer
      classes distinctly: tier resolution (compile time) and chain resolution (run time).
- [ ] **AC6** — `dotf doctor` reports the map's state, and distinguishes **absent**, **unparseable**
      and **schema-invalid** as three loud outcomes. **No failure mode renders as an empty or
      permissive map** (C15). Proven by running doctor against each of the three broken states.
- [ ] **AC7** — Every pool declares its budget fields (`concurrency`, and `reserve_interactive` and
      `shared_with` where they apply), and the loader exposes them — **level 1 only**. Nothing in
      this PR decrements a counter, and the help text and any doctor output say so rather than
      implying enforcement.
- [ ] **AC8** — Guards ship in this PR (C10): bats coverage for the file and the doctor check, Go
      tests for the loader and the schema rejection.
- [ ] **AC9** — The full local loop is green: `go build ./... && go vet ./... && go test ./...`,
      `GOOS=windows go vet ./...`, `golangci-lint run` on the pinned version, and the bats suite.

## References

- Bitácora board: `mlorentedev/dotfiles#1124` (see the `issue:` frontmatter field)
- **`docs/adr/adr-035-model-map-routing-registry.md`** — decides shape, budget split and forge
  boundary. **Lands with PR #1136; this spec must not open its PR before that merges.**
- `docs/adr/adr-032-cross-harness-agent-orchestration.md` §3 — the reference schema, with the two
  defects AC3/AC4 assert against
- Constraints **C10** (guard in the same PR) and **C15** (an unreadable map fails loud) —
  `10_projects/dotfiles/session-protocol.md`
- `harness/reviewer-pool.json`, `harness/review-attestation.json`, `harness/triggers.json`,
  `harness/manifest.json` — the four existing registries whose idioms ADR-035 measured
- Related: #560 (re-scoped to `capability-map.json`), #1137 (`triggers.json` has no guard)
