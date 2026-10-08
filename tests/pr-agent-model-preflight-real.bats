#!/usr/bin/env bats
# The real-dependency sibling of pr-agent-model-preflight.bats (BUG-055 rule,
# tests/stub-real-pairing.bats). That suite stubs `curl`, which proves what the
# script ASKS curl to do. This one runs the real curl against a local HTTP
# server and proves what curl actually does with it: that it reads the
# Authorization header from a config on stdin (`-K -`), that `-w '%{http_code}'`
# yields the status, and that `-m` turns a server that never answers into the
# "no answer" path. A misspelt config directive passes every stub test and
# sends no key at all.
#
# No network: the server listens on 127.0.0.1 on a port the kernel picks.

# bats file_tags=os-sensitive

bats_require_minimum_version 1.5.0

setup() {
    command -v curl >/dev/null || skip "curl not available"
    command -v python3 >/dev/null || skip "python3 not available"
    PREFLIGHT="$BATS_TEST_DIRNAME/../scripts/pr-agent-model-preflight.sh"
    OUT="$BATS_TEST_TMPDIR/output"
    : > "$OUT"
    AUTH_LOG="$BATS_TEST_TMPDIR/auth.log"
    PORT_FILE="$BATS_TEST_TMPDIR/port"

    # alive answers 200, dead answers 401, hang holds the connection open, stall
    # sends a 200 status line and never the body.
    cat > "$BATS_TEST_TMPDIR/server.py" <<'PY'
import http.server, json, socketserver, sys, time
auth_log, port_file = sys.argv[1], sys.argv[2]
class H(http.server.BaseHTTPRequestHandler):
    def do_POST(self):
        body = json.loads(self.rfile.read(int(self.headers["Content-Length"])))
        with open(auth_log, "a") as f:
            f.write("%s %s\n" % (body["model"], self.headers.get("Authorization", "<none>")))
        if body["model"] == "hang":
            time.sleep(30)
            return
        if body["model"] == "stall":
            # Status and headers now, the body never.
            self.send_response(200)
            self.send_header("Content-Length", "100")
            self.end_headers()
            self.wfile.flush()
            time.sleep(30)
            return
        code = 200 if body["model"] == "alive" else 401
        self.send_response(code)
        self.send_header("Content-Length", "2")
        self.end_headers()
        self.wfile.write(b"{}")
    def log_message(self, *a):
        pass
class S(http.server.ThreadingHTTPServer):
    # HTTPServer.server_bind resolves socket.getfqdn(host) only to fill
    # server_name. On a macOS runner that reverse lookup outlived the 5 s
    # start-up wait below, so every test failed in setup (run 37722585322).
    # Nothing here reads server_name.
    def server_bind(self):
        socketserver.TCPServer.server_bind(self)
        self.server_name, self.server_port = self.server_address[:2]
s = S(("127.0.0.1", 0), H)
with open(port_file, "w") as f:
    f.write(str(s.server_address[1]))
s.serve_forever()
PY
    python3 "$BATS_TEST_TMPDIR/server.py" "$AUTH_LOG" "$PORT_FILE" &
    SERVER_PID=$!
    for _ in $(seq 50); do
        [ -s "$PORT_FILE" ] && break
        sleep 0.1
    done
    [ -s "$PORT_FILE" ] || { echo "the local server did not start" >&2; return 1; }
    NAN_API_BASE="http://127.0.0.1:$(cat "$PORT_FILE")"
    export NAN_API_BASE
    export NAN_API_KEY="sk-test-not-a-real-key"
    export PREFLIGHT_TIMEOUT=2
    unset GITHUB_STEP_SUMMARY
}

teardown() {
    if [ -n "${SERVER_PID:-}" ]; then
        kill "$SERVER_PID" 2>/dev/null || true
    fi
}

@test "preflight-real: curl sends the key read from stdin, and the status comes back" {
    run "$PREFLIGHT" --model openai/dead --fallbacks '["openai/alive"]' --output "$OUT"
    [ "$status" -eq 0 ]
    grep -qxF 'model=openai/alive' "$OUT"
    [[ "$output" == *"openai/dead answered HTTP 401"* ]]
    # Every probe carried the key, so `-K -` parsed the config line.
    [ "$(grep -c "Bearer $NAN_API_KEY\$" "$AUTH_LOG")" -eq 2 ]
}

@test "preflight-real: a server that never answers ends at the timeout, as no answer" {
    run "$PREFLIGHT" --model openai/hang --fallbacks '["openai/alive"]' --output "$OUT"
    [ "$status" -eq 0 ]
    grep -qxF 'model=openai/alive' "$OUT"
    [[ "$output" == *"openai/hang gave no answer within 2s"* ]]
}

@test "preflight-real: a 200 status line whose body never arrives is no answer" {
    run "$PREFLIGHT" --model openai/stall --fallbacks '["openai/alive"]' --output "$OUT"
    [ "$status" -eq 0 ]
    grep -qxF 'model=openai/alive' "$OUT"
    [[ "$output" == *"openai/stall gave no answer within 2s"* ]]
}
