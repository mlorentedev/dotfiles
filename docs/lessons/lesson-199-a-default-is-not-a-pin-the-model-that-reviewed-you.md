---
id: lesson-199-a-default-is-not-a-pin-the-model-that-reviewed-you
type: lesson
status: active
created: "2026-08-13"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 199: A default is not a pin: the model that reviewed your code may not be the one you think

**Context**: HARNESS-071 (#955). After making "adversarial reviews never run on an Anthropic model" a standing rule, the first non-Anthropic review was launched as bare `pi -p "<prompt>"`. It ran on `nan/deepseek-v4-flash`, which was the intended model, and `review.md`'s frontmatter recorded it — so the run looked like a successful pin.

**Problem**: nothing had pinned anything. `pi` resolved that model from `~/.pi/agent/settings.json`, a per-machine file that is not versioned and not part of the repo, while `pi --help` documents its own default provider as **google**. On any other machine — a fresh box, a teammate's laptop, CI — the identical command would have reviewed on Gemini, silently and with a frontmatter that honestly recorded whatever ran. The rule would have held by luck on one machine and been invisible everywhere else. The same trap sits one level up: "run it with `agy`" sounds like a provider guarantee, but `agy models` lists `claude-opus-4-6-thinking` and `claude-sonnet-4-6` beside the Gemini family, so pinning the *tool* constrains nothing about the *provider*.

**Solution**: the pool file carries `provider` and `model` as explicit fields rather than having them parsed out of the entry's id, and the launcher pins them explicitly on every invocation — per runner, not uniformly: `pi` takes `--provider` and `--model`, while `agy` has no `--provider` and selects the family through `--model` alone; an entry that lacks what its runner needs is an error, never a fallback to the runner's default. The identity that matters is enforced where it is durable — a gate on the artifact (`dotf spec archive` refuses a `review.md` whose `reviewer:` is outside the pool), not on the invocation, so it holds regardless of who launched the review or how. A mutation that drops `--provider` from the built argv turns a named test red.

**Rule**: when a policy names a model, a provider or a version, pass it explicitly at every invocation and assert it in a test — a value that arrives from a tool's own configuration is a coincidence that survives exactly as long as that machine does. Verify the tool's documented default before trusting an observed one: "it picked the right thing when I ran it" is evidence about your config, not about the command. And enforce the policy on the artifact the process produces, not on the command that produces it; the artifact is what a later reader, a CI job, or another agent can actually check.

**Tags**: `ai-tooling`, `verification`, `configuration`, `determinism`
