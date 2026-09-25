package cmd

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/harness"
)

// TestHelperProcess is not a real test: when re-exec'd with GO_WANT_HELPER_PROCESS=1
// it plays the role of `run`'s child — printing the injected FOO value and exiting
// with EXIT_CODE — so runChild's env-injection and exit-code propagation are
// exercised cross-platform without depending on a system shell.
func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	_, _ = io.WriteString(os.Stdout, os.Getenv("FOO"))
	code, _ := strconv.Atoi(os.Getenv("EXIT_CODE"))
	os.Exit(code)
}

func helperArgv() []string { return []string{os.Args[0], "-test.run=TestHelperProcess"} }

func TestRunChild_PropagatesExitCodeAndInjectsEnv(t *testing.T) {
	var out bytes.Buffer
	environ := append(os.Environ(), "GO_WANT_HELPER_PROCESS=1", "FOO=secret-value", "EXIT_CODE=3")
	code, err := runChild(helperArgv(), environ, nil, &out, io.Discard)
	if err != nil {
		t.Fatalf("runChild: %v", err)
	}
	if code != 3 {
		t.Errorf("exit code = %d, want 3 (child status propagated)", code)
	}
	if out.String() != "secret-value" {
		t.Errorf("child stdout = %q, want the injected FOO value", out.String())
	}
}

func TestRunChild_ZeroExit(t *testing.T) {
	environ := append(os.Environ(), "GO_WANT_HELPER_PROCESS=1", "FOO=x", "EXIT_CODE=0")
	code, err := runChild(helperArgv(), environ, nil, io.Discard, io.Discard)
	if err != nil || code != 0 {
		t.Fatalf("want (0,nil), got (%d,%v)", code, err)
	}
}

func TestRunChild_LaunchFailureIsError(t *testing.T) {
	if _, err := runChild([]string{"definitely-no-such-binary-xyz123"}, os.Environ(), nil, io.Discard, io.Discard); err == nil {
		t.Fatal("expected a launch error for a missing binary")
	}
}

func TestAssertSafeChildCommand(t *testing.T) {
	cases := []struct {
		name    string
		argv    []string
		wantErr bool
	}{
		{"empty argv", []string{}, true},
		{"bare env", []string{"env"}, true},
		{"path to env", []string{"/usr/bin/env"}, true},
		{"bare printenv", []string{"printenv"}, true},
		{"bare export", []string{"export"}, true},
		{"shell -c env", []string{"sh", "-c", "env | grep SECRET"}, true},
		{"quoted env in sh -c", []string{"sh", "-c", "'env'"}, true},
		{"double quoted env in sh -c", []string{"sh", "-c", "\"env\""}, true},
		{"escaped env in sh -c", []string{"sh", "-c", "\\env"}, true},
		{"bundled flag bash -lc", []string{"bash", "-lc", "env"}, true},
		{"interleaved flag bash -i -c", []string{"bash", "-i", "-c", "env"}, true},
		{"long flag bash --norc -c", []string{"bash", "--norc", "-c", "env"}, true},
		{"bash -c set", []string{"bash", "-c", "set"}, true},
		{"bash -c declare -p", []string{"bash", "-c", "declare -p"}, true},
		{"bash -c printenv", []string{"bash", "-c", "printenv FOO"}, true},
		{"bash -c export", []string{"bash", "-c", "export -p"}, true},
		// SEC-001 review round 1, F2: a boundary-class regex missed an absolute
		// path (`/` was not a boundary) and a redirect with no space (`>` was not
		// one either). Whole-token matching closes both.
		{"absolute env in sh -c", []string{"sh", "-c", "/usr/bin/env"}, true},
		{"absolute printenv in bash -c", []string{"bash", "-c", "/usr/bin/printenv FOO"}, true},
		{"redirect with no space", []string{"sh", "-c", "env>x"}, true},
		{"input redirect with no space", []string{"sh", "-c", "env<x"}, true},
		{"relative path to env", []string{"sh", "-c", "./env"}, true},
		{"line continuation inside env", []string{"sh", "-c", "en\\\nv"}, true},
		{"quotes inside the word", []string{"sh", "-c", "'e'n\"v\""}, true},
		{"backslash inside the word", []string{"sh", "-c", "e\\nv"}, true},
		{"upper case, as a case-insensitive filesystem runs it", []string{"sh", "-c", "/usr/bin/ENV"}, true},
		{"allowed hyphenated word", []string{"sh", "-c", "run-env-check"}, false},
		{"allowed dotenv file", []string{"sh", "-c", "cat .env.example"}, false},
		// SEC-001 review round 2, F5: naming the file instead of the command.
		// A Windows executable suffix was kept, and a backslash path is not a path
		// to filepath.Base on Linux, so `env.exe` and `C:\...\env.exe` ran.
		{"windows executable suffix", []string{"env.exe"}, true},
		{"windows path to env.exe", []string{`C:\Program Files\Git\usr\bin\env.exe`}, true},
		{"upper-case windows suffix", []string{"PRINTENV.EXE"}, true},
		{"allowed executable that only starts with env", []string{"envoy.exe"}, false},
		// busybox is a multi-call binary: its first argument is the command it
		// runs, so `busybox env` is `env`, and `busybox sh -c` is a shell.
		{"busybox applet env", []string{"busybox", "env"}, true},
		{"busybox applet by path", []string{"/bin/busybox", "printenv"}, true},
		{"busybox shell snippet", []string{"busybox", "sh", "-c", "env"}, true},
		{"busybox ash snippet", []string{"busybox", "ash", "-c", "env"}, true},
		{"allowed busybox applet", []string{"busybox", "ls", "-la"}, false},
		{"allowed tool", []string{"goreleaser", "release"}, false},
		{"allowed python", []string{"python3", "script.py"}, false},
		{"allowed dotf review", []string{"dotf", "review"}, false},
		{"allowed echo env word", []string{"echo", "running in safe environment"}, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := assertSafeChildCommand(tc.argv)
			if (err != nil) != tc.wantErr {
				t.Errorf("assertSafeChildCommand(%v) err = %v, wantErr = %v", tc.argv, err, tc.wantErr)
			}
		})
	}
}

func TestRedactWriter_RedactsInjectedSecrets(t *testing.T) {
	injected := []string{
		"OPENROUTER_API_KEY=mock-openrouter-test-token-val",
		"NAN_API_KEY=mock-nan-test-token-val",
		"SHORT=abc", // len < 6, not redacted
	}

	var buf bytes.Buffer
	rw := newRedactWriter(&buf, injected)

	input := "Connecting with OPENROUTER_API_KEY=mock-openrouter-test-token-val and NAN_API_KEY=mock-nan-test-token-val, short is abc.\n"
	n, err := rw.Write([]byte(input))
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	if n != len(input) {
		t.Errorf("Write returned n = %d, want %d", n, len(input))
	}
	if err := rw.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}

	got := buf.String()
	want := "Connecting with OPENROUTER_API_KEY=[REDACTED:OPENROUTER_API_KEY] and NAN_API_KEY=[REDACTED:NAN_API_KEY], short is abc.\n"
	if got != want {
		t.Errorf("redacted output mismatch:\ngot:  %q\nwant: %q", got, want)
	}
}

// SEC-002. The hold-back must be PREFIX-AWARE, not a fixed window.
//
// The rule this replaced withheld maxSecretLen-1 bytes on every write no matter
// what they were, so with a 49-byte token the last 48 bytes of every frame
// stayed invisible until more output arrived. Against a pipe drained at Flush()
// that is unobservable; against a terminal it is the whole defect -- the tail of
// a TUI frame is its cursor positioning.
func TestRedactWriter_ReleasesFrameWithNoSecretPrefixImmediately(t *testing.T) {
	injected := []string{"OPENROUTER_API_KEY=mock-openrouter-test-token-val"}

	var buf bytes.Buffer
	rw := newRedactWriter(&buf, injected)

	// Ordinary terminal output. Nothing here is a prefix of the secret, so all
	// of it must reach the target on this Write -- WITHOUT a Flush.
	frame := "\x1b[2J\x1b[H hello from the tui \x1b[10;20H"
	if _, err := rw.Write([]byte(frame)); err != nil {
		t.Fatalf("Write: %v", err)
	}

	if got := buf.String(); got != frame {
		t.Errorf("frame was not released on write (this is the SEC-002 defect):\ngot:  %q\nwant: %q", got, frame)
	}
}

// The complement: a trailing run that IS a proper prefix of a secret must still
// be withheld, or the redaction is defeated by chunking alone.
func TestRedactWriter_HoldsBackATrailingSecretPrefix(t *testing.T) {
	const secret = "mock-openrouter-test-token-val"
	injected := []string{"OPENROUTER_API_KEY=" + secret}

	var buf bytes.Buffer
	rw := newRedactWriter(&buf, injected)

	// Ends mid-secret: "mock-open" is a proper prefix and must not be emitted.
	if _, err := rw.Write([]byte("key is mock-open")); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if got := buf.String(); got != "key is " {
		t.Errorf("the secret prefix leaked or too much was held:\ngot:  %q\nwant: %q", got, "key is ")
	}

	// Completing it must redact the whole value, never emit it.
	if _, err := rw.Write([]byte("router-test-token-val\n")); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if err := rw.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}

	got := buf.String()
	if strings.Contains(got, secret) {
		t.Error("the secret reached the target in full after being split across two writes")
	}
	if want := "key is [REDACTED:OPENROUTER_API_KEY]\n"; got != want {
		t.Errorf("split-write redaction mismatch:\ngot:  %q\nwant: %q", got, want)
	}
}

// SEC-001 review round 1, F1: a secret delivered in writes shorter than itself
// must never reach the target. The reviewed commit gated its hold-back on
// len(data) >= maxSecretLen, so 1-3 byte writes went out verbatim. SEC-002's
// prefix-aware hold-back closed it before the retroactive review ran; this pins
// the case the review reproduced, at every small chunk size.
func TestRedactWriter_SecretInTinyChunksNeverLeaks(t *testing.T) {
	const secret = "mock-openrouter-test-token-val"
	input := "key is " + secret + " and done\n"
	want := "key is [REDACTED:OPENROUTER_API_KEY] and done\n"
	for size := 1; size <= 3; size++ {
		t.Run(strconv.Itoa(size)+"-byte writes", func(t *testing.T) {
			var buf bytes.Buffer
			rw := newRedactWriter(&buf, []string{"OPENROUTER_API_KEY=" + secret})
			for i := 0; i < len(input); i += size {
				end := min(i+size, len(input))
				if _, err := rw.Write([]byte(input[i:end])); err != nil {
					t.Fatalf("Write: %v", err)
				}
				if strings.Contains(buf.String(), secret[:6]) {
					t.Fatalf("after byte %d the target already holds the secret's first 6 bytes: %q", end, buf.String())
				}
			}
			if err := rw.Flush(); err != nil {
				t.Fatalf("Flush: %v", err)
			}
			if got := buf.String(); got != want {
				t.Errorf("got %q, want %q", got, want)
			}
		})
	}
}

// SEC-001 AC9, measured 2026-09-23: Claude Code exports CLAUDECODE (and
// CLAUDE_CODE_ENTRYPOINT), never CLAUDE_CODE, so a refusal keyed on CLAUDE_CODE
// alone never fired in the harness it was written for (#1646). Each declared
// marker must be sufficient on its own, and an environment carrying none must
// not be read as an agent session.
func TestDetectAgentSession_EachMarkerIsSufficient(t *testing.T) {
	clear := func() {
		for _, m := range agentSessionMarkers {
			t.Setenv(m, "")
		}
	}
	clear()
	if detectAgentSession() {
		t.Fatal("no marker set, yet an agent session was detected")
	}
	for _, m := range agentSessionMarkers {
		clear()
		t.Setenv(m, "1")
		if !detectAgentSession() {
			t.Errorf("%s=1 alone was not detected as an agent session", m)
		}
	}
}

func TestDetectAgentSession_KnowsTheVariableClaudeCodeActuallySets(t *testing.T) {
	if !slices.Contains(agentSessionMarkers, "CLAUDECODE") {
		t.Fatal("CLAUDECODE is the variable Claude Code exports; without it the refusal never fires under Claude")
	}
}

// F8: the doctrine that tells agents about the refusal names the same markers
// the code reads. A marker added to one and not the other is how the ADR came to
// omit ANTIGRAVITY_CLI.
func TestAgentSessionMarkersAreDocumented(t *testing.T) {
	root := repoRootForTest(t)
	for _, doc := range []string{"AGENTS.md", "docs/adr/adr-028-secrets-two-tier-bitwarden-age.md"} {
		data, err := os.ReadFile(filepath.Join(root, doc))
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range agentSessionMarkers {
			if !strings.Contains(string(data), "`"+m+"`") {
				t.Errorf("%s does not name the agent-session marker `%s`", doc, m)
			}
		}
	}
}

// SEC-001 review round 2, F1: pi exports AI_AGENT and PI_CODING_AGENT, which
// the list did not know, so `secrets show` printed plaintext in a pi session.
// TestDetectAgentSession_EachMarkerIsSufficient loops over the list itself and
// cannot notice an absent member; this table holds the facts independently. Each
// harness harness/model-map.json declares must have a row, so a harness added
// there without its marker fails here instead of shipping a refusal that never
// fires in it.
func TestAgentSessionMarkers_CoverEveryHarness(t *testing.T) {
	known := map[string]struct {
		markers  []string
		evidence string
	}{
		"claude":   {[]string{"CLAUDECODE", "AI_AGENT"}, "measured in a live Claude Code 2.1 session, 2026-09-24"},
		"pi":       {[]string{"AI_AGENT", "PI_CODING_AGENT"}, "pi 0.87.1 sets both at its CLI and RPC entry points, and documents that children inherit them"},
		"opencode": {[]string{"OPENCODE"}, "measured live under `opencode run`, 2026-09-24"},
		"copilot":  {[]string{"COPILOT_CLI"}, "measured live under `copilot -p`, 2026-09-24"},
		"agy":      {[]string{"ANTIGRAVITY_AGENT"}, "measured live under `agy --print`, 2026-09-24"},
		"codex":    {[]string{"CODEX_THREAD_ID", "CODEX_SANDBOX"}, "std-env's agent-detection table; Codex is not installed where this was written"},
	}
	for h, k := range known {
		for _, m := range k.markers {
			if !slices.Contains(agentSessionMarkers, m) {
				t.Errorf("%s exports %s (%s), but agentSessionMarkers does not list it", h, m, k.evidence)
			}
		}
	}

	raw, err := os.ReadFile(filepath.Join(repoRootForTest(t), harness.ModelMapFile))
	if err != nil {
		t.Fatalf("read %s: %v", harness.ModelMapFile, err)
	}
	var doc struct {
		Harnesses map[string]json.RawMessage `json:"harnesses"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("parse %s: %v", harness.ModelMapFile, err)
	}
	if len(doc.Harnesses) == 0 {
		t.Fatalf("%s declares no harnesses; this test would pass vacuously", harness.ModelMapFile)
	}
	for h := range doc.Harnesses {
		if _, ok := known[h]; !ok {
			t.Errorf("%s declares harness %q, but no session marker is known for it: measure what it exports, then add it here and to agentSessionMarkers", harness.ModelMapFile, h)
		}
	}
}

// countingBW records how many secret values were read through it.
type countingBW struct {
	fakeBW
	reads *int
}

func (c countingBW) Field(item, field string) (string, error) {
	*c.reads++
	return c.fakeBW.Field(item, field)
}

// SEC-001 review round 2, F4: AC1 says a refused command exits "without
// decrypting or launching". The guard ran inside runChild, after every mapped
// secret had been resolved, so a refused `env` still decrypted the store.
func TestSecretsRun_RefusesBeforeResolvingSecrets(t *testing.T) {
	useTempRegistry(t, "version: 1\nsecrets:\n  - {id: bw-foo, plane: app, backend: bw, bw: {item: it, field: password}, expose: {env: FOO}}\n")
	reads := 0
	useBwReader(t, countingBW{fakeBW{"it/password": "synthetic-value"}, &reads})

	c := newSecretsRunCmd()
	c.SetArgs([]string{"--", "env"})
	c.SetOut(io.Discard)
	c.SetErr(io.Discard)
	err := c.Execute()
	if err == nil || !strings.Contains(err.Error(), "refusing") {
		t.Fatalf("secrets run -- env: got %v, want a refusal", err)
	}
	if reads != 0 {
		t.Errorf("the refused command read %d secret value(s) before being refused", reads)
	}

	// Positive control: the counter does count a resolution.
	reg, err := loadRegistry()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := resolveInjectedSecrets(reg, nil); err != nil {
		t.Fatal(err)
	}
	if reads == 0 {
		t.Fatal("countingBW recorded no read for a real resolution; the assertion above proves nothing")
	}
}
