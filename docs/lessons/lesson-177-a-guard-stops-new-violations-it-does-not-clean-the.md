---
id: lesson-177-a-guard-stops-new-violations-it-does-not-clean-the
type: lesson
status: active
created: "2026-08-09"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 177: A guard stops new violations; it does not clean the stock, and the rule then reads as if it did

**Context**: #857 reported that crystallize corrupts a `MEMORY.md` whose body sits inside a YAML block scalar. The plan of record was to teach the Go port (#490) to edit that shape, deriving the indent rather than assuming it. Before writing any code, two things were measured: what a `yaml.v3` roundtrip actually does to those files, and where the shape came from.

**Problem**: The shape was already forbidden. `00_meta/templates/agent-memory.md` has said *"Plain-markdown auto-memory — never a `content: |` YAML block"* since 2026-06-20. But all 17 affected files were wrapped in a single vault commit on **2026-05-26** — three and a half weeks *earlier*. Someone hit this, diagnosed it, and wrote the rule into the template that scaffolds new files. Nothing migrated the files already broken. The rule then read, to every later reader, as if the problem were handled: the template is the SSOT, the SSOT forbids the shape, therefore the shape does not exist. It existed in 17 files for two and a half months, and the plan was about to add permanent CLI support for it.

**Solution**: Reframe from "build the capability" to "eliminate the invalid state" — migrate the files (#864), keep #862's refusal guard permanently, drop YAML scope from #490 entirely. Two measurements forced it. First, no `yaml.v3` roundtrip can edit these files losslessly: a literal block scalar cannot carry trailing whitespace round-trippably, so a conforming emitter drops it — and two trailing spaces are the markdown hard break the handoff convention needs (measured: pollex 5 hard-break lines → 0, hive 4 → 0, kubelab 4 → 0). Second, and worse for the original plan, de-indenting *correctly* still would not have fixed the file: the wrapper put the first body line at indent 4 and every later line at 6, so the YAML-correct block indent is 4 and the markers land at column 2 — where crystallize, which anchors every marker at column 0, still cannot see them.

**Rule**: When you write a rule in response to an incident, ask in the same breath how many existing artifacts already violate it, and migrate them or file the ticket in that session. A guard is half of incident→guard — and it is the half that reads as complete, because the rule describes the desired state as if asserting it achieved it. The tell is a rule whose commit date is *later* than the damage it describes: that ordering means the stock was never swept. Separately: before designing around a library, measure what it does to your real data. The plan here rested on "parse → mutate → re-dump" being lossless, and the fifteen minutes that disproved it also revealed the shape should never have been supported at all.
