---
tags: [spec, verification, templates]
created: "2026-09-01"
---

# Verification - CLI-069-voice-nan-whisper

## Evidence

Map every acceptance criterion from `proposal.md` to concrete proof (commit hash, test name, or observed behavior).

- [ ] AC1 — File transcription `dotf voice --file <audio>` prints transcript to stdout -> test `TestVoiceFileTranscription`
- [ ] AC2 — Interactive push-to-talk microphone capture & window typing injection -> test `TestVoiceInteractiveTyping`
- [ ] AC3 — Direct clipboard transcription via `dotf voice --copy` -> test `TestVoiceClipboardMode`
- [ ] AC4 — Missing/invalid `$NAN_API_KEY` credential check fails cleanly -> test `TestVoiceMissingCredentials`
- [ ] AC5 — Temporary audio file cleanup on exit / interrupt -> test `TestVoiceTempFileCleanup`
- [ ] AC6 — HTTP multipart client serialization & status code error handling -> test `TestNanWhisperClient_Transcribe`

## Test status

- Test suite: `go test -v ./cli/internal/voice/...`
- Build & Lint: `go vet ./...` and `golangci-lint run ./cli/internal/voice/...`
- Manual smoke test: Record voice sample via `dotf voice --copy` and verify clipboard contents match spoken prompt

## Decisions made during implementation

-

## Promotion candidates

- [ ] Lesson for the repo's `docs/lessons/`? Push-to-talk audio streaming with fallback display typing in Wayland/X11
- [ ] ADR-worthy decision for the repo's `docs/adr/`? Centralized remote STT gateway vs local Whisper models in developer tooling
- [ ] New pattern candidate for `00_meta/patterns/`? No (covered by `pattern-nan-builders-gateway.md`)

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/CLI-069-voice-nan-whisper/` -> `specs/archive/CLI-069-voice-nan-whisper/`
- [ ] Bitácora board ticket for this spec (#1426) closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)

