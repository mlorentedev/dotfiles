#!/usr/bin/env bash
# AI-046 AC6: a switch from deepseek-v4-flash to qwen3.6 in a session whose
# history, reasoning included, exceeds qwen3.6's 262K window completes without a
# context overflow.
#
# The history is synthetic: assistant turns attributed to nan/deepseek-v4-flash,
# each carrying a large `thinking` block and a short answer, sized so the
# answers alone fit qwen3.6 and answers plus reasoning do not. pi-ai replays a
# cross-model thinking block as plain text (the package's
# src/cross-model-thinking-guard.ts), so without the guard the request overflows.
#
# Two arms, one variable (NAN_THINKING_GUARD): guard on is the criterion, guard
# off is the control that proves the history really overflows. pi's automatic
# compaction is disabled in the isolated agent dir, or a compact-and-retry
# could make both arms succeed and the run would say nothing about the guard.
#
# Only qwen3.6 is called, and it is unmetered (harness/nan-quotas.json). The
# key comes from the caller's environment and is never printed:
#
#   dotf secrets run --only NAN_API_KEY -- bash specs/AI-046-pi-nan-provider/measure-ac6.sh
#
# Knobs: PI_BIN (default ~/.local/bin/pi), AC6_THINK_CHARS, AC6_TEXT_CHARS,
# AC6_TURNS. A small run (AC6_THINK_CHARS=20000 AC6_TEXT_CHARS=4000) checks the
# session format before spending the full request.
set -eu

REPO="$(cd "$(dirname "${BASH_SOURCE[0]:-$0}")/../.." && pwd)"
PI_BIN="${PI_BIN:-$HOME/.local/bin/pi}"
THINK_CHARS="${AC6_THINK_CHARS:-2000000}"
TEXT_CHARS="${AC6_TEXT_CHARS:-300000}"
TURNS="${AC6_TURNS:-30}"

# Preflight: every tool this script relies on, before it reports anything.
for tool in jq python3 "$PI_BIN"; do
    command -v "$tool" >/dev/null 2>&1 || { echo "missing: $tool" >&2; exit 2; }
done
[ -n "${NAN_API_KEY:-}" ] || { echo "NAN_API_KEY is not set; run through dotf secrets run" >&2; exit 2; }

WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT
AGENT="$WORK/agent"
mkdir -p "$AGENT"
cp "$REPO/ai/pi/nan-provider.json" "$AGENT/nan-provider.json"
printf '%s\n' '{"compaction":{"enabled":false}}' >"$AGENT/settings.json"

SOURCE="$(jq -r '.packages[].source | select(test("/pi-nan-provider@"))' "$REPO/ai/pi/packages.json")"
[ -n "$SOURCE" ] || { echo "no pi-nan-provider entry in ai/pi/packages.json" >&2; exit 2; }
PI_CODING_AGENT_DIR="$AGENT" "$PI_BIN" install "$SOURCE" >"$WORK/install.log" 2>&1 \
    || { cat "$WORK/install.log" >&2; exit 2; }

python3 - "$WORK/history.jsonl" "$THINK_CHARS" "$TEXT_CHARS" "$TURNS" "$PWD" <<'PY'
import json, random, sys, uuid
from datetime import datetime, timedelta, timezone

out, think_chars, text_chars, turns, cwd = sys.argv[1], int(sys.argv[2]), int(sys.argv[3]), int(sys.argv[4]), sys.argv[5]
rng = random.Random(46)  # deterministic: the same history every run
vocab = ("state service request model window context token deploy config review "
         "branch commit merge cluster node volume secret policy metric latency "
         "budget quota replay reasoning answer summary module parser handler "
         "cache index schema record lesson runbook ticket gate check test").split()

def prose(n):
    words, size = [], 0
    while size < n:
        w = rng.choice(vocab)
        words.append(w)
        size += len(w) + 1
    return " ".join(words)

t0 = datetime(2026, 9, 27, tzinfo=timezone.utc)
clock = iter(t0 + timedelta(seconds=i) for i in range(10_000))
def stamp():
    t = next(clock)
    return t.isoformat().replace("+00:00", "Z"), int(t.timestamp() * 1000)

lines, parent = [], None
iso, _ = stamp()
lines.append({"type": "session", "version": 3, "id": str(uuid.UUID(int=rng.getrandbits(128))), "timestamp": iso, "cwd": cwd})

def entry(kind, **fields):
    global parent
    iso, _ = stamp()
    eid = f"{rng.getrandbits(32):08x}"
    lines.append({"type": kind, "id": eid, "parentId": parent, "timestamp": iso, **fields})
    parent = eid

entry("model_change", provider="nan", modelId="deepseek-v4-flash")
per_think, per_text = think_chars // turns, text_chars // (2 * turns)
for i in range(turns):
    _, ms = stamp()
    entry("message", message={"role": "user", "content": f"Question {i}: " + prose(per_text), "timestamp": ms})
    _, ms = stamp()
    entry("message", message={
        "role": "assistant",
        "content": [
            {"type": "thinking", "thinking": prose(per_think), "thinkingSignature": "reasoning_content"},
            {"type": "text", "text": f"Answer {i}: " + prose(per_text)},
        ],
        "api": "openai-completions", "provider": "nan", "model": "deepseek-v4-flash",
        "usage": {"input": 0, "output": 0, "cacheRead": 0, "cacheWrite": 0, "totalTokens": 0,
                  "cost": {"input": 0, "output": 0, "cacheRead": 0, "cacheWrite": 0, "total": 0}},
        "stopReason": "stop", "timestamp": ms,
    })

with open(out, "w") as f:
    for line in lines:
        f.write(json.dumps(line) + "\n")
PY

printf 'history: %s turns, %s thinking chars, %s answer chars\n' "$TURNS" "$THINK_CHARS" "$TEXT_CHARS"

run_arm() {
    arm="$1" guard="$2"
    session="$WORK/$arm.jsonl"
    cp "$WORK/history.jsonl" "$session"
    rc=0
    # stdin from /dev/null: in print mode pi reads a non-TTY stdin as more
    # prompt, so an inherited open pipe or socket makes it wait forever.
    NAN_THINKING_GUARD="$guard" PI_CODING_AGENT_DIR="$AGENT" "$PI_BIN" \
        --session "$session" --model nan/qwen3.6 -p "Reply with exactly the word PONG" \
        </dev/null >"$WORK/$arm.out" 2>"$WORK/$arm.err" || rc=$?
    # The last assistant entry is the qwen3.6 reply this run appended.
    jq -rs --arg arm "$arm" --arg rc "$rc" '
        [.[] | select(.type == "message" and .message.role == "assistant")] | last | .message
        | "\($arm): exit=\($rc) model=\(.provider)/\(.model) stopReason=\(.stopReason) input=\(.usage.input) cacheRead=\(.usage.cacheRead) error=\((.errorMessage // "none")[0:160])"
    ' "$session" | tee "$WORK/$arm.result"
}

# The verdict reads the model and stop reason, never the exit status alone: a
# run that failed before calling qwen3.6 leaves the synthetic deepseek turn as
# the last assistant entry, and must not count as either outcome.
verdict=0
run_arm guard-on 1
grep -q ' model=nan/qwen3.6 stopReason=stop ' "$WORK/guard-on.result" \
    || { echo "FAIL: with the guard on, qwen3.6 did not answer" >&2; verdict=1; }
run_arm guard-off 0
grep -q ' model=nan/qwen3.6 stopReason=error .*maximum context length' "$WORK/guard-off.result" \
    || { echo "FAIL: the control did not overflow, so this history does not test the guard" >&2; verdict=1; }
[ "$verdict" -eq 0 ] && echo "AC6: PASS"
exit "$verdict"
