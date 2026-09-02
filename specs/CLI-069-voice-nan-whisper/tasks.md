---
tags: [spec, tasks, templates]
created: "2026-09-01"
---

# Tasks - CLI-069-voice-nan-whisper

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.
>
> **Inline markers**:
> - `[P]` — safe to run in parallel (independent task).
> - `[AC<n>]` — satisfies acceptance criterion #`<n>` from `proposal.md`.

## Setup

- [ ] Branch created from main: `feat/CLI-069-voice-nan-whisper`
- [ ] `proposal.md` is complete and acceptance criteria are testable
- [ ] No open questions left in `proposal.md` "Risks / open questions"

## Implementation

- [ ] [P] [AC6] Write failing unit tests for NaN Whisper HTTP client in `cli/internal/voice/client_test.go` (multipart encoding, Authorization header, response parsing, error status handling)
- [ ] [AC6] Implement `NanWhisperClient` in `cli/internal/voice/client.go` with 15s timeout and structured error handling
- [ ] [P] [AC4] Write tests for missing or invalid `$NAN_API_KEY` credential resolution
- [ ] [AC4] Implement credential check in `cli/internal/voice` ensuring clean exit and actionable advice without credential leakage
- [ ] [P] [AC1] Write failing CLI tests for `dotf voice --file <path>`
- [ ] [AC1] Implement `--file` flag in `cli/internal/voice/voice.go` and register `voice` command in `cli/cmd/voice.go`
- [ ] [P] [AC5] Write tests for temporary audio file lifecycle and signal-interruption cleanup
- [ ] [AC5] Implement audio recording probe and wrapper (`sox` / `ffmpeg` / `arecord`) with guaranteed `defer os.Remove(...)` cleanup
- [ ] [P] [AC2] [AC3] Implement push-to-talk interactive recording and window typing synthesizer (`wtype` / `ydotool` / `xdotool` / clipboard fallback)
- [ ] [AC3] Implement `--copy` flag to copy transcription directly to clipboard without typing
- [ ] Document desktop environment shortcut integration in dotfiles window manager configs

## Closing

- [ ] Every acceptance criterion from `proposal.md` is covered by at least one test
- [ ] Every acceptance criterion has a matching entry in `features.json` with a non-vacuous verification command
- [ ] `go test ./cli/internal/voice/...` and `go vet ./...` pass with zero warnings
- [ ] No unrelated changes in the diff (no scope creep)
- [ ] `verification.md` filled in with evidence
- [ ] PR opened referencing this spec folder

## Machine-readable features

This spec emits a sibling `features.json` following [[pattern-feature-list-as-primitive]]. The JSON is the harness-facing contract: each acceptance criterion maps to ≥1 feature with `id`, `behavior`, `verification` (executable command), `state` (lifecycle), and `evidence` (harness-captured output).

