---
id: lesson-164-the-platform-s-documented-cap-decides-the-render-a
type: lesson
status: active
created: "2026-08-08"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 164: The platform's documented cap decides the render, and a shared file is injected into, never written

**Context**: Closing the last two holes in cross-agent doctrine coverage. Four harnesses already carried the enforced rules and the presence block; Antigravity and codex carried nothing. The obvious fix was to deploy the same `AGENTS.md` everyone else gets.

**Problem**: The obvious fix would have failed silently in two different ways, and neither agent was installed on the machine doing the work, so neither failure would have been observed. Antigravity caps **each rules file at 12000 characters** and `AGENTS.md` is 21851 -- the file would have been rejected or truncated with no error surfaced to the deploy. Codex is worse: it accepts files until the combined global-plus-project chain reaches `project_doc_max_bytes` (32 KiB by default) and then stops, so a 22 KB global copy would have quietly consumed the budget the *repository's own* `AGENTS.md` needed -- the most specific file in the chain, and the one whose loss is least visible. Separately, `~/.gemini/GEMINI.md` is written by both Antigravity and the Gemini CLI (google-gemini/gemini-cli#16058, closed as not planned), so any deploy that owns the file rather than a region inside it destroys the other tool's configuration. And codex reads `AGENTS.override.md` *in preference to* `AGENTS.md`, so with an override present the deploy reports success while changing nothing the agent ever reads.

**Solution**: Send the compact payload -- enforced rules plus presence, ~2 KB -- through the same marked-region mechanism, and record the limit, the measured size and the source URL next to each manifest row so the decision is re-derivable rather than trusted. Inject: replace our own region or append a fresh one, never rewrite the file. Two warnings cover what could otherwise pass unnoticed: a file over its platform's documented cap (checked after injection, against the *whole* file, since the platform counts bytes we did not write), and a shadow file that wins at read time. No skills row was added for codex, because no primary source documents its skill-discovery path and inferring one is exactly the guesswork the ticket existed to remove.

**Rule**: Before deploying an instruction file to an agent you cannot run, read that agent's own docs for the size limit and the file-precedence order -- both are load-bearing and neither is guessable, and a limit is usually enforced by truncation rather than by an error. When a target file is shared with another tool, the pipeline owns a region, not the file. And when you cannot smoke-test, say so in the PR and name the specific assumptions instead of letting green CI imply a coverage it never had.
