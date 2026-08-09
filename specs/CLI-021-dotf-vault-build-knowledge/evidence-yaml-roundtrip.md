---
id: "CLI-021-evidence-yaml-roundtrip"
type: spec
status: active
tags: [spec, evidence, yaml, crystallize]
created: "2026-08-09"
---

# Evidence — why the wrapped shape is edited surgically, not round-tripped

## Question

`MEMORY.md` for 17 of this machine's projects stores its body inside a YAML
literal block scalar (`content: |`). #857 requires crystallize to edit that shape
instead of refusing it. Two implementations were on the table:

1. **Whole-document roundtrip** — `yaml.Unmarshal` the document, mutate the
   `content` field, `yaml.Marshal` it back.
2. **Structural surgery** — preserve the document bytes, locate the block-scalar
   opener, derive its indent, de-indent the body, apply the existing insertion
   logic, re-indent, splice back.

Option 1 is the more obvious reading of "parse → mutate → re-dump" and was
selected first. It was measured before implementation.

## Method

A throwaway probe round-tripped each real wrapped `MEMORY.md` through
`go.yaml.in/yaml/v3 v3.0.5` — both via `yaml.Node` (style-preserving) and via
`map[string]any` — and compared the emitted bytes with the input. No mutation was
applied, so any difference is pure roundtrip loss.

## Result — the roundtrip is lossy on every file measured

| File | in → out | `---` kept | `\|` style kept | hard-break lines | blank-but-indented lines |
|---|---|---|---|---|---|
| pollex | 10055 → 9769 (−286) | no | **no** | 5 → **0** | 13 → **0** |
| hive | 5294 → 5249 (−45) | no | **no** | 4 → **0** | 0 → 0 |
| kubelab | 10897 → 10659 (−238) | no | **no** | 4 → **0** | 0 → 0 |
| garsync | 1026 → 977 (−49) | no | **no** | 0 → 0 | 3 → **0** |

Via `map[string]any` the loss is total: the body is re-emitted as a single
double-quoted line with `\n` escapes, so the file stops being readable as
markdown at all.

## Why it is lossy — and why no emitter flag fixes it

A YAML literal block scalar cannot carry **trailing whitespace** round-trippably:
on a re-parse those spaces are indistinguishable from block indentation, so a
conforming emitter must either switch to a quoted style or drop them. yaml.v3
does both — it abandons `|` and strips the spaces.

That is fatal here rather than cosmetic, because **two trailing spaces are the
markdown hard break** the handoff convention mandates so the `Last task` /
`Decisions` / `Open threads` / `Next action` fields render as separate lines.
Stripping them silently reflows every handoff block into one run-on paragraph —
the exact rendering defect the convention exists to prevent.

The whitespace that is *semantic* in Markdown is precisely the whitespace YAML
does not preserve.

## Decision — neither option; the shape itself is invalid state

Structural surgery (option 2) was proposed on this evidence and **overruled**:
the measurement reframed the question from *how* to edit the wrapped shape to
*whether it should exist at all*. Three facts say it should not:

1. The vault's own scaffolding SSOT forbids it. `00_meta/templates/agent-memory.md`
   carries, since `de0d5773` (2026-06-20): *"Plain-markdown auto-memory — never a
   `content: |` YAML block."*
2. Nothing we ship emits it. `cli/internal/vault/templates/vault-memory.md`
   writes plain markdown; so does every twin.
3. It was a single accident, not a convention. All 17 files were wrapped in one
   bulk edit — vault commit `1c216229`, `2026-05-26 21:17:41` — and nothing has
   wrapped a file since.

So the rule was written **three and a half weeks after** the damage, and the
already-wrapped files were never migrated. Teaching the CLI to edit a shape the
vault declares invalid would entrench an accident that already has a ruling
against it.

**Direction:** migrate the 17 files back to plain markdown once, and keep #862's
refusal guard permanently as defence-in-depth. #490 then ports the shell as it
stands, with no YAML scope at all.

`yaml.v3` still has a place, but only as a **test-only oracle** for the migration:
assert each migrated file's recovered body is byte-identical to the block's
decoded content.

## Secondary finding — pollex's block indent is 4, not 6

Prior notes recorded "pollex 6 spaces, hive 4". Measured, **all 17 wrapped files
open their block at indent 4**. pollex's *first* body line sits at 4 and every
later line at 6, so the block indent is 4 and those extra two spaces are
**content**, not indentation — a pre-existing malformation in that one file.

This is the concrete reason the implementation must derive the indent the way
YAML defines it (first non-empty line) rather than sampling a convenient line:
the two readings disagree on a real file that is in scope.
