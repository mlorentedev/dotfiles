---
id: "CLI-069-voice-nan-whisper"
type: spec
status: draft # draft | implementing | verifying | archived
created: "2026-09-01"
issue: "mlorentedev/dotfiles#1426"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# CLI-069-voice-nan-whisper

> **Naming**: file lives at `<repo>/specs/CLI-069-voice-nan-whisper/proposal.md`. `CLI-069-voice-nan-whisper` is `AREA-NNN-slug`.

## Why

<!-- from issue #1426: CLI-069: Add dotf voice dictation via NaN Whisper API -->

Typing detailed prompts, architectural constraints, and task specifications into agent CLIs and terminals at 40–50 WPM creates a cognitive bottleneck and results in underspecified requests. Local STT setups require downloading multi-gigabyte Whisper weights and managing GPU/CUDA/Vulkan dependencies across multiple workstations (MSI, laptops, desktop). `CLI-069` implements a zero-local-dependency `dotf voice` dictation command that records push-to-talk audio clips and dispatches them to our existing `https://api.nan.builders/v1/audio/transcriptions` Whisper gateway, enabling instantaneous speech-to-text across all dotfiles-managed machines without local model bloat or battery drain.

## What

- **`dotf voice` CLI command in Go:** Implemented within `cli/internal/voice` and registered in `cli/cmd`.
- **Push-to-Talk interactive mode (`dotf voice`):** Captures microphone input (using standard system utilities like `sox`, `arecord`, or `ffmpeg`), sends the snippet as `multipart/form-data` to `https://api.nan.builders/v1/audio/transcriptions` with `model="whisper"`, and types the returned text into the active focused window (`wtype` / `ydotool` / `xdotool`) or falls back to clipboard.
- **Clipboard flag (`dotf voice --copy`):** Transcribes audio and copies text directly to the system clipboard without typing.
- **Direct file transcription mode (`dotf voice --file <path>`):** Transcribes an existing audio file and emits the transcript directly to `stdout` for scriptability.
- **Secure credential resolution:** Resolves `$NAN_API_KEY` through the dotfiles environment/secrets pipeline (`dotf secrets` / env contract), failing fast with actionable guidance if unset.
- **Safe lifecycle & cleanup:** All temporary recording files are strictly removed on exit or error.
- **Desktop keybinding integration:** Documents and provides standard window manager keybindings (e.g. Hyprland, i3, Gnome) for the global push-to-talk shortcut.

## Out of scope

- Running local Whisper neural network inference or downloading model weights to the client.
- Real-time continuous streaming over WebSockets (request-response audio clip batching is sufficient for prompt dictation).
- Text-to-Speech (TTS) audio synthesis (e.g. Kokoro) — handled in separate audio tools.

## Risks / open questions

- **Display Server & Typing Injection:** Wayland environments restrict arbitrary window typing compared to X11. RESOLVED: detect available injection tools in priority order (`wtype` -> `ydotool` -> `xdotool` -> clipboard paste via `wl-copy`/`xclip`), with `--copy` as explicit guarantee.
- **Microphone recording tool availability:** Linux machines may have `sox`, `ffmpeg`, or `arecord`. RESOLVED: implement a probe chain (`sox` -> `ffmpeg` -> `arecord`) and report an actionable error if none is found.
- **API Rate limits / Network timeout:** NaN gateway rate limits (100 RPM shared). RESOLVED: 15-second client timeout with clean error classification on 401, 429, or 5xx.

## Acceptance criteria

- [ ] AC1 — `dotf voice --file <audio-path>` transcribes a given audio file via `https://api.nan.builders/v1/audio/transcriptions` using `$NAN_API_KEY` and outputs the text to `stdout`.
- [ ] AC2 — `dotf voice` (interactive) records from the default microphone, sends the clip to the NaN Whisper endpoint, and types the transcribed text into the focused window.
- [ ] AC3 — `dotf voice --copy` copies the transcription directly to the clipboard without typing.
- [ ] AC4 — If `$NAN_API_KEY` is missing or invalid, the command exits with a non-zero code and a clear, actionable error message without leaking credentials.
- [ ] AC5 — Temporary recording files are guaranteed to be cleaned up on success, interruption (SIGINT), or error.
- [ ] AC6 — Unit tests verify multipart request serialization, auth header construction, HTTP error handling, and JSON response parsing.

## References

- Bitácora board: mlorentedev/dotfiles#1426.
- Knowledge pattern: [[pattern-nan-builders-gateway.md]] (`00_meta/patterns/pattern-nan-builders-gateway.md`).
- Layered Architecture: [[pattern-layered-architecture.md]].
