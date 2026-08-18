---
id: lesson-087-non-streaming-chat-endpoints-behind-a-gateway-drop
type: lesson
status: active
created: "2026-06-12"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 087: Non-streaming chat endpoints behind a gateway drop long generations — a client timeout cannot fix a server-side cut

**Context**: CLI-003 `dot review` QA: live review of a real 12KB staged diff through the NaN gateway (`deepseek-v4-flash`, non-streaming chat completions).

**Problem**: The gateway closes long non-streaming responses mid-generation. Reproduced at the 120s client timeout and again at 300s (TCP read died at ~168s) — the cut is provider-side, so no client-side `--timeout` value can help. Hello-world smoke tests never trigger it; the failure only appears at realistic payload sizes.

**Solution**: Kept the 120s default instead of chasing the timeout; documented `--provider openrouter` as the escape hatch for large diffs (same 12KB diff reviewed in ~10s) and recorded the limitation in `cli/README.md`.

**Rule**: QA API integrations with realistic payload sizes, not hello-world ones. When a remote endpoint drops long responses, change the route (streaming, different provider) — not the client timeout.
