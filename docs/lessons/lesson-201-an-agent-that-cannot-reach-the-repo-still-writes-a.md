---
id: lesson-201-an-agent-that-cannot-reach-the-repo-still-writes-a
type: lesson
status: active
created: "2026-08-14"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 201: An agent that cannot reach the repo still writes a confident review

**Context**: HARNESS-071 (#955) added a reviewer pool and `dotf spec review`, with `agy/gemini-3.1-pro-high` as the non-Anthropic fallback beside `nan/deepseek-v4-flash`. The spec's acceptance criterion demanded each configured arm produce a *real review*, on the grounds that a fallback never observed working is decoration (#898).

**Problem**: the Gemini arm failed three times, and **every failure presented as success**. (1) `agy --print` consumes a value, so `agy --print --model X … "<prompt>"` made `--print` swallow `--model`: the model went unset, the prompt was orphaned, and agy replied with a session greeting at exit 0. (2) agy prompts for approval on every tool call, and a detached run has no human to answer — every call was auto-*denied*, and the run stopped after 14 seconds reporting `{"status":"SUCCESS","response":""}`. (3) Worst: agy runs its shell commands in its **own install directory**, not the caller's — `pwd` answered `~/.gemini/antigravity-cli` and `git rev-parse HEAD` failed with "not a git repository". That run wrote a *well-formed* `review.md`: correct frontmatter, correct spec id, correct self-reported reviewer, verdict PASS. It would have passed the archive gate. And it had not executed a single test.

**Solution**: `--dangerously-skip-permissions` (a detached reviewer cannot answer prompts), `--sandbox` (bounds what that approval reaches — verified to cost nothing: git, `go test` and file writes all work under it), and `--add-dir <repoRoot>`, isolated as the fix for the reach problem: with it alone `pwd` resolves to the repo, git resolves the right sha, and the suite runs from inside the reviewer. `pi` needs none of the three, so the two arms are pinned apart by tests in both directions rather than "unified".

**Rule**: when delegating work to another agent, verify it can *reach and act on* the target before trusting anything it returns — a well-formed artifact is evidence about the agent's formatting, not about its access. The tell was in the output all along: an all-A rubric, one speculative finding restating someone else's, nothing independently verified. Treat a suspiciously agreeable review as a **capability** symptom first and a judgement symptom second. And prove reach with the cheapest possible probe (`pwd`, `git rev-parse`, one real command) before spending twenty minutes on a review whose conclusions you cannot use.

**Tags**: `ai-tooling`, `verification`, `adversarial-review`, `delegation`
