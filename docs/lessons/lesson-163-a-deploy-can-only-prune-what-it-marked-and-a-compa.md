---
id: lesson-163-a-deploy-can-only-prune-what-it-marked-and-a-compa
type: lesson
status: active
created: "2026-08-07"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 163: A deploy can only prune what it marked, and a compatibility fence set by agent identity points the wrong way

**Context**: Auditing the 36-skill cross-agent library for consistency. `targets[]` in a skill's frontmatter is documented as the *compatibility truth*: absent means every agent, present means only the listed ones. Seven skills declared it; the deploy honours it and prunes an output once its skill drops that agent.

**Problem**: Two defects that look unrelated and are not. First, the fence was backwards. The six skills fenced to `[claude]` were fenced in May by agent identity, and five of them depend on nothing Claude-specific -- one patches a vault file, two are judgment procedures, one teaches the very authoring format the pipeline distributes to everyone. Meanwhile the genuinely coupled skills were unfenced: six that call MCP tools shipped everywhere, though MCP registration is a client-side fact and not a property of the skill at all. Second, five skill directories had sat in `~/.gemini/skills` since `2026-05-27`, two of them fenced to another agent, surviving every re-deploy for over two months. `deploy_prune()` was working exactly as designed: it only removes outputs carrying the `generated: true` marker, because that marker is the only proof the engine wrote a file rather than a human. Anything written by the pre-provenance deploy is therefore unprunable *and* unreported -- and the residue that persists is precisely the fenced kind, since a still-targeted skill gets overwritten by the next render and heals itself.

**Solution**: Fence only on a primitive no other harness can be given -- the fence dropped from six skills to two (both bound to the agent-local auto-memory store), and the rule plus a rationale table now live in `pattern-cross-agent-skill-pipeline.md` rather than in anyone's memory. For the residue: warn when an unmarked entry *shadows* a record name, so a third-party skill that owns its own name stays silent while a stale copy of ours gets reported; never delete unmarked files. Two of the four smoke tests that broke used the just-unfenced skill as their "Claude-only" exemplar, and a fifth hard-coded the expected command count -- both repointed at a skill fenced for a durable reason, and the count derived from the records instead.

**Rule**: Ask what a skill would actually fail on, not which agent it was written for. "Uses a tool only one client has registered" is an installation gap that belongs to the installer; only a dependency on a store or mechanism that cannot be handed to another harness justifies an exclusion. And any deploy that proves ownership with a marker has a permanent blind spot for everything written before the marker existed: it must report what it cannot prune, or the residue rots silently -- exactly the shape as a hard-coded count in a test, which is also a claim that stops being true without anyone being told.
