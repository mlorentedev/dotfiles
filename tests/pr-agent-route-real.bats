#!/usr/bin/env bats
# The real-dependency sibling of pr-agent-route.bats (BUG-055 rule,
# tests/stub-real-pairing.bats). That suite stubs `curl`, which proves what the
# script ASKS curl to do. This one runs the real curl against a local HTTP
# server and proves what curl does with it: that it reads the x-api-key header
# from a config on stdin (`-K -`), that it reaches `/v1/messages` with the
# version header, that `-w '%{http_code}'` yields the status, and that `-m`
# turns a server that never answers into the "no answer" path.
#
# No network: the server listens on 127.0.0.1 on a port the kernel picks.

# bats file_tags=os-sensitive

bats_require_minimum_version 1.5.0

setup() {
    command -v curl >/dev/null || skip "curl not available"
    command -v python3 >/dev/null || skip "python3 not available"
    ROUTE="$BATS_TEST_DIRNAME/../scripts/pr-agent-route.sh"
    OUT="$BATS_TEST_TMPDIR/output"
    : > "$OUT"
    REQ_LOG="$BATS_TEST_TMPDIR/requests.log"
    PORT_FILE="$BATS_TEST_TMPDIR/port"
    MODE_FILE="$BATS_TEST_TMPDIR/mode"
    printf 'alive' > "$MODE_FILE"

    # The server answers by the mode in MODE_FILE, not by the model: the probe
    # asks for the real allowlisted id, so no test-only id ever needs to pass
    # the allowlist. alive answers 200, dead answers 401, hang holds the
    # connection open, stall sends a 200 status line and never the body.
    cat > "$BATS_TEST_TMPDIR/server.py" <<'PY'
import http.server, json, socketserver, sys, time
req_log, port_file, mode_file = sys.argv[1], sys.argv[2], sys.argv[3]
class H(http.server.BaseHTTPRequestHandler):
    def do_POST(self):
        body = json.loads(self.rfile.read(int(self.headers["Content-Length"])))
        with open(mode_file) as f:
            mode = f.read().strip()
        with open(req_log, "a") as f:
            f.write("%s %s %s %s\n" % (self.path, body["model"],
                    self.headers.get("x-api-key", "<none>"),
                    self.headers.get("anthropic-version", "<none>")))
        if mode == "hang":
            time.sleep(30)
            return
        if mode == "stall":
            self.send_response(200)
            self.send_header("Content-Length", "100")
            self.end_headers()
            self.wfile.flush()
            time.sleep(30)
            return
        code = 200 if mode == "alive" else 401
        self.send_response(code)
        self.send_header("Content-Length", "2")
        self.end_headers()
        self.wfile.write(b"{}")
    def log_message(self, *a):
        pass
class S(http.server.ThreadingHTTPServer):
    # Skip the reverse lookup HTTPServer.server_bind does for server_name; on a
    # macOS runner it outlived the start-up wait (run 37722585322).
    def server_bind(self):
        socketserver.TCPServer.server_bind(self)
        self.server_name, self.server_port = self.server_address[:2]
s = S(("127.0.0.1", 0), H)
with open(port_file, "w") as f:
    f.write(str(s.server_address[1]))
s.serve_forever()
PY
    python3 "$BATS_TEST_TMPDIR/server.py" "$REQ_LOG" "$PORT_FILE" "$MODE_FILE" &
    SERVER_PID=$!
    for _ in $(seq 50); do
        [ -s "$PORT_FILE" ] && break
        sleep 0.1
    done
    [ -s "$PORT_FILE" ] || { echo "the local server did not start" >&2; return 1; }
    ANTHROPIC_API_BASE="http://127.0.0.1:$(cat "$PORT_FILE")"
    export ANTHROPIC_API_BASE
    export PR_AGENT_ANTHROPIC_API_KEY="sk-ant-test-not-a-real-key"
    export ROUTE_PROBE_TIMEOUT=2
    export NAN_OUTCOME=success NAN_MODEL=openai/mimo-v2.6-flash NAN_FALLBACKS='[]'
    export POOL_FILE="$BATS_TEST_TMPDIR/pool.json"
    cat > "$POOL_FILE" <<'JSON'
{"pool": [
 {"id": "nan/mimo-v2.6-flash", "pr_agent": {"model": "openai/mimo-v2.6-flash", "weight": 1}},
 {"id": "anthropic-review/claude-haiku-5-5", "pr_agent": {"model": "anthropic/claude-haiku-5-5", "weight": 1}}
]}
JSON
    unset PR_AGENT_PROVIDER GITHUB_STEP_SUMMARY PR_CHANGED_LINES PR_LABELS
}

teardown() {
    if [ -n "${SERVER_PID:-}" ]; then
        kill "$SERVER_PID" 2>/dev/null || true
    fi
}

out() { sed -n "s/^$1=//p" "$OUT"; }

@test "route-real: curl sends the key from stdin and the version header to /v1/messages" {
    PR_AGENT_DRAW=1 run "$ROUTE" --output "$OUT"
    [ "$status" -eq 0 ]
    [ "$(out reviewer)" = "anthropic/claude-haiku-5-5" ]
    [ "$(cat "$REQ_LOG")" = "/v1/messages claude-haiku-5-5 $PR_AGENT_ANTHROPIC_API_KEY 2023-06-01" ]
}

@test "route-real: a refusal comes back as its status" {
    printf 'dead' > "$MODE_FILE"
    PR_AGENT_DRAW=0 run "$ROUTE" --output "$OUT"
    [ "$status" -eq 0 ]
    [ "$(out second)" = "none" ]
    [[ "$(out note)" == *"answered HTTP 401"* ]] || false
}

@test "route-real: a server that never answers ends at the timeout, as no answer" {
    printf 'hang' > "$MODE_FILE"
    PR_AGENT_DRAW=0 run "$ROUTE" --output "$OUT"
    [ "$status" -eq 0 ]
    [ "$(out second)" = "none" ]
    [[ "$(out note)" == *"no answer within 2s"* ]] || false
}

@test "route-real: a 200 status line whose body never arrives is no answer" {
    printf 'stall' > "$MODE_FILE"
    PR_AGENT_DRAW=0 run "$ROUTE" --output "$OUT"
    [ "$status" -eq 0 ]
    [ "$(out second)" = "none" ]
    [[ "$(out note)" == *"no answer within 2s"* ]] || false
}
