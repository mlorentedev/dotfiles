---
id: lesson-311
type: lesson
status: active
created: "2026-09-27"
owner: manu
tags: [lesson, pi, stdin, background, measurement]
---

# 311 — pi's print mode waits on an inherited stdin

## What happened

AI-046's AC6 measurement (`specs/AI-046-pi-nan-provider/measure-ac6.sh`) runs `pi --session <file> --model nan/qwen3.6 -p "<prompt>"` twice. Run in the foreground from the agent's shell, both arms finished in 13 seconds. Run again as a background job, the first arm sat for ten minutes. It had no TCP connection open and used no CPU.

`/proc/<pid>/fd/0` was a socket. In print mode pi reads a piped stdin and prepends it to the first prompt (pi `docs/cli.md`), so it waits for end-of-file. A foreground shell hands pi a stdin that closes. A background runner hands it an open socket or pipe that never does. The same command, in the same script, behaves differently depending on who launched it.

The script's verdict caught it. The last assistant entry in the session was still the synthetic deepseek turn, not a qwen3.6 reply, so the arm was reported as failed, not as a pass or a hang.

## The rule

Any non-interactive `pi -p` call from a script, a CI job or a background task gets `</dev/null`, unless the script really is piping input to it. Do not rely on the caller's stdin being closed: a runner's stdin is whatever the runner inherited.

When a scripted run stalls with no network activity and no CPU, read `/proc/<pid>/fd/0` before anything else.

Refs: AI-046 (#1764), `specs/AI-046-pi-nan-provider/verification.md` (AC6).
