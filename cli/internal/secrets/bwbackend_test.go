package secrets

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// countingHandler wraps the fake and tallies requests to path, so a test can assert
// how many times the daemon was consulted — the observable difference between a pinned
// backend and a per-call one.
func countingHandler(n *int, path string, f *fakeBWServe) http.HandlerFunc {
	inner := f.handler()
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == path {
			*n++
		}
		inner(w, r)
	}
}

// TestSelectBWBackend_ReadAndWriteAlwaysAgree is AC5, and it is the whole point of
// BUG-084 (#993).
//
// The bug was not "the write path is broken". It was "the read path moved to the daemon
// and the write path did not", i.e. two halves of one command authenticating through
// different subjects. Fixing only the write implementation would leave that shape intact
// for the next path to move alone. This asserts the shape instead: for EVERY daemon
// state, the reader and the writer must come from the same side.
//
// If someone later adds a third backend and wires only one half to it, this fails.
func TestSelectBWBackend_ReadAndWriteAlwaysAgree(t *testing.T) {
	tests := []struct {
		name       string
		status     string // fake daemon status; "" -> daemon unreachable
		unreachable bool
		wantDaemon bool
	}{
		{name: "unlocked daemon is used for both halves", status: "unlocked", wantDaemon: true},
		{name: "locked daemon falls back on both halves", status: "locked", wantDaemon: false},
		{name: "unauthenticated daemon falls back on both halves", status: "unauthenticated", wantDaemon: false},
		{name: "no daemon falls back on both halves", unreachable: true, wantDaemon: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var c BWServeClient
			if tt.unreachable {
				// A closed server's URL: connection refused, the real "no daemon" signal.
				srv := httptest.NewServer((&fakeBWServe{}).handler())
				c = BWServeClient{BaseURL: srv.URL}
				srv.Close()
			} else {
				srv := httptest.NewServer((&fakeBWServe{status: tt.status}).handler())
				defer srv.Close()
				c = BWServeClient{BaseURL: srv.URL}
			}

			b := SelectBWBackend(c)

			if b.Daemon != tt.wantDaemon {
				t.Fatalf("Daemon = %v, want %v", b.Daemon, tt.wantDaemon)
			}
			_, readerIsDaemon := b.Reader.(BWServeReader)
			_, writerIsDaemon := b.Writer.(BWServeWriter)
			_, syncerIsDaemon := b.Syncer.(BWServeClient)

			// All three, not just read/write: sync is the third path, and a third
			// path that moves alone reproduces this bug. `rotate` writes, syncs,
			// then reads back to prove the write took — sync the wrong cache and
			// that proof is worthless.
			if readerIsDaemon != writerIsDaemon || readerIsDaemon != syncerIsDaemon {
				t.Fatalf("SPLIT BRAIN: reader daemon=%v, writer daemon=%v, syncer daemon=%v — this is exactly BUG-084",
					readerIsDaemon, writerIsDaemon, syncerIsDaemon)
			}
			if readerIsDaemon != tt.wantDaemon {
				t.Fatalf("backend halves = daemon:%v, want daemon:%v", readerIsDaemon, tt.wantDaemon)
			}
			if b.Reader == nil || b.Writer == nil || b.Syncer == nil {
				t.Fatal("a selected backend must have all three halves non-nil")
			}
		})
	}
}

// TestSelectBWBackend_ProbesOnce guards the pinning decision itself: selection must
// consult the daemon exactly once, so a daemon that changes state cannot split a
// command across two subjects. A per-call design would probe on every operation.
func TestSelectBWBackend_ProbesOnce(t *testing.T) {
	f := &fakeBWServe{status: "unlocked"}
	var probes int
	srv := httptest.NewServer(countingHandler(&probes, "/list/object/folders", f))
	defer srv.Close()

	b := SelectBWBackend(BWServeClient{BaseURL: srv.URL})
	if probes != 1 {
		t.Fatalf("readability probes during selection = %d, want exactly 1", probes)
	}

	// Using the pinned backend must not re-probe: the decision is already made.
	_, _ = b.Reader.Field("nope", "password")
	if probes != 1 {
		t.Fatalf("readability probes after use = %d, want 1 — the backend must be pinned, not re-chosen", probes)
	}
}

// TestSelectBWBackend_ShelloutNamesTheRealRemediation is AC3. A locked-vault failure
// must point at `dotf secrets unlock`, never leave the operator with bw's bare
// "Vault is locked." — which sends them to `bw unlock`, a command that prints a session
// key to a shell nobody reads and does not fix this failure. Four separate operators'
// confusions in one day traced to that message (see #993's "Related").
func TestSelectBWBackend_ShelloutNamesTheRealRemediation(t *testing.T) {
	lockErr := errors.New("Vault is locked.")

	tests := []struct {
		name        string
		daemonState string
		wantPhrase  string
	}{
		{
			name:        "no daemon",
			daemonState: "no bw serve daemon is running",
			wantPhrase:  "no bw serve daemon is running",
		},
		{
			name:        "daemon up but locked",
			daemonState: "the bw serve daemon is running but locked",
			wantPhrase:  "running but locked",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := lockHint(lockErr, tt.daemonState)
			if !errors.Is(got, ErrBWVaultLocked) {
				t.Fatalf("locked error must wrap ErrBWVaultLocked, got %v", got)
			}
			msg := got.Error()
			if !strings.Contains(msg, "dotf secrets unlock") {
				t.Fatalf("error must name `dotf secrets unlock`, got: %s", msg)
			}
			if !strings.Contains(msg, tt.wantPhrase) {
				t.Fatalf("error must describe the observed daemon state %q, got: %s", tt.wantPhrase, msg)
			}
			if !errors.Is(got, lockErr) {
				t.Fatalf("the original bw error must stay in the chain, got: %s", msg)
			}
		})
	}
}

// TestLockHint_LeavesUnrelatedErrorsAlone: the decorator exists to clarify ONE
// confusion. An unrelated failure must pass through untouched, or every error in the
// write path starts advertising an unlock the operator does not need.
func TestLockHint_LeavesUnrelatedErrorsAlone(t *testing.T) {
	orig := errors.New("More than one result was found")
	got := lockHint(orig, "no bw serve daemon is running")
	if got != orig { //nolint:errorlint // identity is the assertion: untouched, not merely equivalent
		t.Fatalf("unrelated error was rewritten: %v", got)
	}
	if lockHint(nil, "whatever") != nil {
		t.Fatal("a nil error must stay nil")
	}
}

// TestSelectBWBackend_ShelloutHalvesBothDecorate: the hint must be on the read half
// too. `set` reads before it writes, so a locked vault surfaces on the READ first —
// decorating only the writer would leave the common path reporting the bare message.
func TestSelectBWBackend_ShelloutHalvesBothDecorate(t *testing.T) {
	srv := httptest.NewServer((&fakeBWServe{status: "locked"}).handler())
	defer srv.Close()

	b := SelectBWBackend(BWServeClient{BaseURL: srv.URL})
	if _, ok := b.Reader.(lockHintReader); !ok {
		t.Fatalf("shellout reader must carry the lock hint, got %T", b.Reader)
	}
	if _, ok := b.Writer.(lockHintWriter); !ok {
		t.Fatalf("shellout writer must carry the lock hint, got %T", b.Writer)
	}
}

// TestSelectBWBackend_NeverCallsStatus is the guard for BUG-082 (#988), and it is the
// whole fix expressed as an assertion.
//
// `GET /status` poisons this daemon: for ~0.5s afterwards every `GET /object/item/<id>`
// returns HTTP 500 (measured 0/10 failures before a status call, 10/10 immediately
// after; upstream bitwarden/clients#20951). The old selection probed `/status` and then
// immediately read an item — so the probe broke the very call it was authorising, and
// `dotf secrets run` failed most of the time.
//
// Selection must therefore never touch `/status`. This asserts the mechanism, not the
// symptom, because the symptom needs a live daemon and half a second of timing.
func TestSelectBWBackend_NeverCallsStatus(t *testing.T) {
	for _, st := range []string{"unlocked", "locked", "unauthenticated"} {
		t.Run(st, func(t *testing.T) {
			var statusCalls int
			f := &fakeBWServe{status: st}
			srv := httptest.NewServer(countingHandler(&statusCalls, "/status", f))
			defer srv.Close()

			b := SelectBWBackend(BWServeClient{BaseURL: srv.URL})

			if statusCalls != 0 {
				t.Fatalf("selection called GET /status %d times — that poisons item reads for ~0.5s (BUG-082)", statusCalls)
			}
			// And it still reaches the right decision without it.
			if wantDaemon := st == "unlocked"; b.Daemon != wantDaemon {
				t.Fatalf("status=%s: Daemon = %v, want %v", st, b.Daemon, wantDaemon)
			}
		})
	}
}
