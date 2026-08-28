package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/mlorentedev/dotfiles/cli/internal/secrets"
)

// fakeBWServeHandler mirrors the internal/secrets package's fake — kept
// local (not exported from there) since only this package's command tests
// need an httptest double of bw serve's REST API.
func fakeBWServeHandler(t *testing.T, status *string, unlockPassword string) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/status":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"success": true,
				"data":    map[string]string{"status": *status},
			})
		case r.Method == http.MethodPost && r.URL.Path == "/unlock":
			var body struct {
				Password string `json:"password"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			if unlockPassword != "" && body.Password != unlockPassword {
				_ = json.NewEncoder(w).Encode(map[string]any{"success": false, "message": "Cryptography error, The decryption operation failed"})
				return
			}
			*status = "unlocked"
			_ = json.NewEncoder(w).Encode(map[string]any{"success": true})
		case r.Method == http.MethodPost && r.URL.Path == "/lock":
			*status = "locked"
			_ = json.NewEncoder(w).Encode(map[string]any{"success": true})
		default:
			http.NotFound(w, r)
		}
	}
}

// fakeDaemonPID is the pid the fake daemon's trace records, so a test can
// assert unlock/lock print it (#1315) without a process behind it.
const fakeDaemonPID = 4242

// withFakeDaemon points the lifecycle seam at an httptest bw serve and gives
// it a state dir carrying a pid file, the trace a daemon started by dotf
// leaves behind. It returns the state so tests can assert on its paths.
func withFakeDaemon(t *testing.T, status *string, unlockPassword string) secrets.BWServeState {
	t.Helper()
	srv := httptest.NewServer(fakeBWServeHandler(t, status, unlockPassword))
	t.Cleanup(srv.Close)
	state := secrets.NewBWServeState(t.TempDir())
	if err := state.WritePID(fakeDaemonPID); err != nil {
		t.Fatal(err)
	}
	old := bwDaemonAddr
	bwDaemonAddr = func() *secrets.BWServeDaemon {
		return &secrets.BWServeDaemon{
			Client: secrets.BWServeClient{BaseURL: srv.URL, HTTPClient: &http.Client{Timeout: 2 * time.Second}},
			State:  state,
		}
	}
	t.Cleanup(func() { bwDaemonAddr = old })
	return state
}

// assertTrace is AC3 of CLI-057: a confirmation line names the daemon's pid
// and its log path, so the operator learns where to look before it is gone.
func assertTrace(t *testing.T, out string, state secrets.BWServeState) {
	t.Helper()
	if !strings.Contains(out, "pid 4242") || !strings.Contains(out, state.LogPath()) {
		t.Fatalf("expected the pid and the log path in the confirmation, got: %q", out)
	}
}

func withFakePassword(t *testing.T, pw string, err error) {
	t.Helper()
	old := readPassword
	readPassword = func() ([]byte, error) { return []byte(pw), err }
	t.Cleanup(func() { readPassword = old })
}

func runUnlock(t *testing.T) (string, string, error) {
	t.Helper()
	cmd := newSecretsUnlockCmd()
	var out, errOut bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)
	err := cmd.Execute()
	return out.String(), errOut.String(), err
}

func runLock(t *testing.T) (string, error) {
	t.Helper()
	cmd := newSecretsLockCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&bytes.Buffer{})
	err := cmd.Execute()
	return out.String(), err
}

// TestSecretsUnlock_Succeeds_PasswordNeverInOutput is AC1: a correct unlock
// succeeds, and the password never appears in stdout or stderr.
func TestSecretsUnlock_Succeeds_PasswordNeverInOutput(t *testing.T) {
	status := "locked"
	state := withFakeDaemon(t, &status, "correct-horse-battery-staple")
	withFakePassword(t, "correct-horse-battery-staple", nil)

	out, errOut, err := runUnlock(t)
	if err != nil {
		t.Fatalf("unlock: %v", err)
	}
	if !strings.Contains(out, "unlocked") {
		t.Fatalf("expected confirmation in stdout, got: %q", out)
	}
	assertTrace(t, out, state)
	if strings.Contains(out, "correct-horse-battery-staple") || strings.Contains(errOut, "correct-horse-battery-staple") {
		t.Fatal("password must never appear in command output")
	}
}

// TestSecretsUnlock_WrongPassword_ErrorsWithoutLeakingIt is AC1's failure
// path: the same guarantee holds on error.
func TestSecretsUnlock_WrongPassword_ErrorsWithoutLeakingIt(t *testing.T) {
	status := "locked"
	withFakeDaemon(t, &status, "the-real-password")
	withFakePassword(t, "a-wrong-guess", nil)

	_, errOut, err := runUnlock(t)
	if err == nil {
		t.Fatal("expected an error for a wrong password")
	}
	if strings.Contains(err.Error(), "a-wrong-guess") || strings.Contains(errOut, "a-wrong-guess") {
		t.Fatalf("password must never appear in the error, got err=%v errOut=%q", err, errOut)
	}
}

// TestSecretsUnlock_Idempotent is AC6: unlocking twice while already
// unlocked never prompts and never errors.
func TestSecretsUnlock_Idempotent(t *testing.T) {
	status := "unlocked"
	withFakeDaemon(t, &status, "")
	old := readPassword
	readPassword = func() ([]byte, error) {
		t.Fatal("must not prompt when already unlocked")
		return nil, nil
	}
	t.Cleanup(func() { readPassword = old })

	out, _, err := runUnlock(t)
	if err != nil {
		t.Fatalf("unlock: %v", err)
	}
	if !strings.Contains(out, "already unlocked") {
		t.Fatalf("expected an idempotent no-op message, got: %q", out)
	}
}

// TestSecretsLock is AC6: lock re-locks a running, unlocked daemon.
func TestSecretsLock(t *testing.T) {
	status := "unlocked"
	state := withFakeDaemon(t, &status, "")

	out, err := runLock(t)
	if err != nil {
		t.Fatalf("lock: %v", err)
	}
	if !strings.Contains(out, "locked") {
		t.Fatalf("expected confirmation, got: %q", out)
	}
	assertTrace(t, out, state)
	if status != "locked" {
		t.Fatalf("expected the daemon to report locked, got %q", status)
	}
}

// TestSecretsLock_NoDaemon is AC6's no-op path: nothing to lock, no error.
func TestSecretsLock_NoDaemon(t *testing.T) {
	old := bwDaemonAddr
	bwDaemonAddr = func() *secrets.BWServeDaemon {
		return &secrets.BWServeDaemon{
			Client: secrets.BWServeClient{BaseURL: "http://127.0.0.1:1", HTTPClient: &http.Client{Timeout: 200 * time.Millisecond}},
		}
	}
	t.Cleanup(func() { bwDaemonAddr = old })

	out, err := runLock(t)
	if err != nil {
		t.Fatalf("lock: %v", err)
	}
	if !strings.Contains(out, "no daemon running") {
		t.Fatalf("expected a no-op message, got: %q", out)
	}
}

// TestSecretsUnlock_NoPIDFileIsSaidNotGuessed is AC3's honest branch: a daemon
// dotf did not start (a hand-run `bw serve`, or one started before #1315) has
// no pid file, and the confirmation says "pid unknown" rather than inventing
// one — while still naming the log path, which is where a future start writes.
func TestSecretsUnlock_NoPIDFileIsSaidNotGuessed(t *testing.T) {
	status := "unlocked"
	state := withFakeDaemon(t, &status, "")
	if err := os.Remove(state.PIDPath()); err != nil {
		t.Fatal(err)
	}

	out, _, err := runUnlock(t)
	if err != nil {
		t.Fatalf("unlock: %v", err)
	}
	if !strings.Contains(out, "pid unknown") || !strings.Contains(out, state.LogPath()) {
		t.Fatalf("expected 'pid unknown' and the log path, got: %q", out)
	}
	if strings.Contains(out, "pid 4242") {
		t.Fatalf("a removed pid file must not be reported as a pid, got: %q", out)
	}
}
