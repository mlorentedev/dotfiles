package redact

import (
	"bytes"
	"strconv"
	"strings"
	"testing"
)

func TestWriter_RedactsInjectedSecrets(t *testing.T) {
	injected := []string{
		"OPENROUTER_API_KEY=mock-openrouter-test-token-val",
		"NAN_API_KEY=mock-nan-test-token-val",
		"SHORT=abc", // len < 6, not redacted
	}

	var buf bytes.Buffer
	rw := New(&buf, injected)

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
func TestWriter_ReleasesFrameWithNoSecretPrefixImmediately(t *testing.T) {
	injected := []string{"OPENROUTER_API_KEY=mock-openrouter-test-token-val"}

	var buf bytes.Buffer
	rw := New(&buf, injected)

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
func TestWriter_HoldsBackATrailingSecretPrefix(t *testing.T) {
	const secret = "mock-openrouter-test-token-val"
	injected := []string{"OPENROUTER_API_KEY=" + secret}

	var buf bytes.Buffer
	rw := New(&buf, injected)

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

// #1857: at end of stream a held prefix can no longer complete, so Flush
// redacts it only when it is credential material. One letter, or a public
// format prefix, is ordinary output: `results` used to print as
// `result[REDACTED:NAN_API_KEY]` for a key starting with `s`.
func TestWriter_FlushRedactsOnlyAMaterialPrefix(t *testing.T) {
	// Split so no source line reads as an sk- key to a secret scanner.
	const long = "sk-" + "mock-nan-test-token-value-0123456789" // 39 bytes: floor 6
	const short = "zq8w7v"                                      // 6 bytes: floor 3
	injected := []string{"NAN_API_KEY=" + long, "SHORT_KEY=" + short}
	cases := []struct{ out, want string }{
		{"results", "results"},
		{"status: pass", "status: pass"},
		{"key: sk-", "key: sk-"},
		{"key: sk-mo", "key: sk-mo"},
		{"key: sk-moc", "key: [REDACTED:NAN_API_KEY]"},
		{"key: " + long[:20], "key: [REDACTED:NAN_API_KEY]"},
		{"x zq", "x zq"},
		{"x zq8", "x [REDACTED:SHORT_KEY]"},
		{"x zq8w7", "x [REDACTED:SHORT_KEY]"},
	}
	for _, tc := range cases {
		var buf bytes.Buffer
		rw := New(&buf, injected)
		if _, err := rw.Write([]byte(tc.out)); err != nil {
			t.Fatalf("Write: %v", err)
		}
		if err := rw.Flush(); err != nil {
			t.Fatalf("Flush: %v", err)
		}
		if got := buf.String(); got != tc.want {
			t.Errorf("child printed %q: got %q, want %q", tc.out, got, tc.want)
		}
	}
}

// SEC-001 review round 1, F1: a secret delivered in writes shorter than itself
// must never reach the target. The reviewed commit gated its hold-back on
// len(data) >= maxSecretLen, so 1-3 byte writes went out verbatim. SEC-002's
// prefix-aware hold-back closed it before the retroactive review ran; this pins
// the case the review reproduced, at every small chunk size.
func TestWriter_SecretInTinyChunksNeverLeaks(t *testing.T) {
	const secret = "mock-openrouter-test-token-val"
	input := "key is " + secret + " and done\n"
	want := "key is [REDACTED:OPENROUTER_API_KEY] and done\n"
	for size := 1; size <= 3; size++ {
		t.Run(strconv.Itoa(size)+"-byte writes", func(t *testing.T) {
			var buf bytes.Buffer
			rw := New(&buf, []string{"OPENROUTER_API_KEY=" + secret})
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

// A tail that is a proper prefix of a secret must not be written out raw when
// the stream ends -- those are the leading bytes of a credential, and Flush is
// exactly the moment the transcript is closed.
func TestWriter_DoesNotFlushAPartialSecretPrefixRaw(t *testing.T) {
	const secret = "mock-openrouter-test-token-val"
	injected := []string{"OPENROUTER_API_KEY=" + secret}

	var buf bytes.Buffer
	rw := New(&buf, injected)

	// The child dies mid-key: a long proper prefix and then nothing.
	if _, err := rw.Write([]byte("key is mock-openrouter-test-toke")); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if err := rw.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}

	got := buf.String()
	if strings.Contains(got, "mock-open") {
		t.Errorf("Flush emitted credential material raw: %q", got)
	}
	if !strings.Contains(got, "[REDACTED:OPENROUTER_API_KEY]") {
		t.Errorf("the partial prefix was not reported as redacted material: %q", got)
	}
}

// If one injected value is a prefix of another, replacing the shorter first
// breaks the longer one's match and leaks its suffix. Order is free to fix.
func TestWriter_HandlesOneSecretBeingAPrefixOfAnother(t *testing.T) {
	injected := []string{
		"SHORTKEY=mock-prefix-value",
		"LONGKEY=mock-prefix-value-with-more",
	}

	var buf bytes.Buffer
	rw := New(&buf, injected)

	if _, err := rw.Write([]byte("using mock-prefix-value-with-more now\n")); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if err := rw.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}

	got := buf.String()
	if strings.Contains(got, "-with-more") {
		t.Errorf("the longer secret's suffix leaked because the shorter one matched first: %q", got)
	}
	if !strings.Contains(got, "[REDACTED:LONGKEY]") {
		t.Errorf("the longer secret was not redacted under its own name: %q", got)
	}
}
