package secrets

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// TestBWServeCommand_BindsLocalhostOnly is AC5: the constructed command must
// never bind beyond 127.0.0.1, regardless of bin/port — no live process
// involved, just the args a real Start() would hand to exec.Command.
func TestBWServeCommand_BindsLocalhostOnly(t *testing.T) {
	cmd := bwServeCommand("bw", 8087)
	args := strings.Join(cmd.Args, " ")
	if !strings.Contains(args, "--hostname 127.0.0.1") {
		t.Fatalf("expected --hostname 127.0.0.1 in args, got: %q", args)
	}
	if strings.Contains(args, "--hostname all") || strings.Contains(args, "0.0.0.0") {
		t.Fatalf("command must never bind beyond loopback, got: %q", args)
	}
}

func TestBWServeCommand_DefaultsBinToBw(t *testing.T) {
	cmd := bwServeCommand("", 8087)
	if cmd.Args[0] != "bw" {
		t.Fatalf("expected bin to default to %q, got %q", "bw", cmd.Args[0])
	}
}

// fakeBWServe is a minimal httptest-backed double of bw serve's REST API,
// covering exactly the endpoints this package calls. Response shapes are
// copied from the OPS-021 spike's live probes against bw 2026.5.0.
type fakeBWServe struct {
	status string // unauthenticated | locked | unlocked
	items  map[string]json.RawMessage // id -> full item JSON
	names  map[string]string          // id -> name, for /list/object/items

	unlockPassword string // "" -> any password succeeds
	failUnlock     bool
}

func (f *fakeBWServe) handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/status":
			writeEnvelope(w, true, "", map[string]string{"status": f.status})
		case r.Method == http.MethodPost && r.URL.Path == "/unlock":
			var body struct {
				Password string `json:"password"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			if f.failUnlock || (f.unlockPassword != "" && body.Password != f.unlockPassword) {
				writeEnvelope(w, false, "Cryptography error, The decryption operation failed", nil)
				return
			}
			f.status = "unlocked"
			writeEnvelope(w, true, "", nil)
		case r.Method == http.MethodPost && r.URL.Path == "/lock":
			f.status = "locked"
			writeEnvelope(w, true, "", map[string]string{"title": "Your vault is locked."})
		case r.Method == http.MethodGet && r.URL.Path == "/list/object/items":
			search := r.URL.Query().Get("search")
			var out []map[string]string
			for id, name := range f.names {
				if search == "" || strings.Contains(name, search) {
					out = append(out, map[string]string{"id": id, "name": name})
				}
			}
			writeEnvelope(w, true, "", out)
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/object/item/"):
			id := strings.TrimPrefix(r.URL.Path, "/object/item/")
			item, ok := f.items[id]
			if !ok {
				writeEnvelope(w, false, "Not found.", nil)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprintf(w, `{"success":true,"data":%s}`, item)
		default:
			http.NotFound(w, r)
		}
	}
}

func writeEnvelope(w http.ResponseWriter, success bool, message string, data any) {
	w.Header().Set("Content-Type", "application/json")
	env := map[string]any{"success": success}
	if message != "" {
		env["message"] = message
	}
	if data != nil {
		env["data"] = data
	}
	_ = json.NewEncoder(w).Encode(env)
}

func TestBWServeClient_Status(t *testing.T) {
	f := &fakeBWServe{status: "locked"}
	srv := httptest.NewServer(f.handler())
	defer srv.Close()
	c := BWServeClient{BaseURL: srv.URL}

	st, err := c.Status()
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if st != "locked" {
		t.Fatalf("expected locked, got %q", st)
	}
}

func TestBWServeClient_Status_Unreachable(t *testing.T) {
	c := BWServeClient{BaseURL: "http://127.0.0.1:1", HTTPClient: &http.Client{Timeout: 200 * time.Millisecond}}
	_, err := c.Status()
	if err == nil {
		t.Fatal("expected an error against an unreachable daemon")
	}
	if !isErrUnreachable(err) {
		t.Fatalf("expected ErrBWServeUnreachable, got: %v", err)
	}
}

func TestBWServeClient_Unlock_WrongPassword_NeverLeaksIt(t *testing.T) {
	f := &fakeBWServe{status: "locked", unlockPassword: "the-real-password"}
	srv := httptest.NewServer(f.handler())
	defer srv.Close()
	c := BWServeClient{BaseURL: srv.URL}

	err := c.Unlock("a-wrong-guess")
	if err == nil {
		t.Fatal("expected an error for a wrong password")
	}
	if strings.Contains(err.Error(), "a-wrong-guess") {
		t.Fatalf("error must never contain the attempted password, got: %v", err)
	}
}

func TestBWServeClient_Unlock_CorrectPassword_Succeeds(t *testing.T) {
	f := &fakeBWServe{status: "locked", unlockPassword: "the-real-password"}
	srv := httptest.NewServer(f.handler())
	defer srv.Close()
	c := BWServeClient{BaseURL: srv.URL}

	if err := c.Unlock("the-real-password"); err != nil {
		t.Fatalf("Unlock: %v", err)
	}
	st, _ := c.Status()
	if st != "unlocked" {
		t.Fatalf("expected unlocked after a correct Unlock, got %q", st)
	}
}

func TestBWServeClient_Unlock_Idempotent(t *testing.T) {
	f := &fakeBWServe{status: "unlocked"} // already unlocked
	srv := httptest.NewServer(f.handler())
	defer srv.Close()
	c := BWServeClient{BaseURL: srv.URL}

	if err := c.Unlock("anything"); err != nil {
		t.Fatalf("Unlock against an already-unlocked daemon must succeed, got: %v", err)
	}
}

func TestBWServeClient_Lock(t *testing.T) {
	f := &fakeBWServe{status: "unlocked"}
	srv := httptest.NewServer(f.handler())
	defer srv.Close()
	c := BWServeClient{BaseURL: srv.URL}

	if err := c.Lock(); err != nil {
		t.Fatalf("Lock: %v", err)
	}
	st, _ := c.Status()
	if st != "locked" {
		t.Fatalf("expected locked after Lock, got %q", st)
	}
}

func TestBWServeReader_Field_MatchesBWGetShape(t *testing.T) {
	f := &fakeBWServe{
		status: "unlocked",
		names:  map[string]string{"item-1": "my-secret-item"},
		items: map[string]json.RawMessage{
			"item-1": json.RawMessage(`{"id":"item-1","name":"my-secret-item","login":{"username":"u","password":"p"},"notes":"n","fields":[{"name":"custom-field","value":"v"}]}`),
		},
	}
	srv := httptest.NewServer(f.handler())
	defer srv.Close()
	r := BWServeReader{Client: BWServeClient{BaseURL: srv.URL}}

	cases := map[string]string{"password": "p", "username": "u", "notes": "n", "custom-field": "v"}
	for field, want := range cases {
		got, err := r.Field("my-secret-item", field)
		if err != nil {
			t.Fatalf("Field(%q): %v", field, err)
		}
		if got != want {
			t.Fatalf("Field(%q) = %q, want %q", field, got, want)
		}
	}
}

func TestBWServeReader_Field_NotFound(t *testing.T) {
	f := &fakeBWServe{status: "unlocked", names: map[string]string{}}
	srv := httptest.NewServer(f.handler())
	defer srv.Close()
	r := BWServeReader{Client: BWServeClient{BaseURL: srv.URL}}

	_, err := r.Field("does-not-exist", "password")
	if !isErrItemNotFound(err) {
		t.Fatalf("expected ErrBWItemNotFound, got: %v", err)
	}
}

func TestBWServeReader_Field_AmbiguousNameErrors(t *testing.T) {
	// The search endpoint is fuzzy (substring), so two items whose names both
	// CONTAIN the search term but where only the search term itself is not an
	// exact match for either must not silently pick one.
	f := &fakeBWServe{
		status: "unlocked",
		names:  map[string]string{"a": "zoho", "b": "zoho"}, // two items, same exact name
		items: map[string]json.RawMessage{
			"a": json.RawMessage(`{"id":"a","name":"zoho","notes":"first"}`),
			"b": json.RawMessage(`{"id":"b","name":"zoho","notes":"second"}`),
		},
	}
	srv := httptest.NewServer(f.handler())
	defer srv.Close()
	r := BWServeReader{Client: BWServeClient{BaseURL: srv.URL}}

	_, err := r.Field("zoho", "notes")
	if err == nil || !strings.Contains(err.Error(), "More than one result") {
		t.Fatalf("expected an ambiguous-match error, got: %v", err)
	}
}

func TestBWServeReader_Field_FuzzySearchDoesNotMatchNonExactName(t *testing.T) {
	// A search for "my-item" also matches "my-item-backup" server-side (fuzzy
	// substring), but only the exact name must be accepted.
	f := &fakeBWServe{
		status: "unlocked",
		names:  map[string]string{"a": "my-item", "b": "my-item-backup"},
		items: map[string]json.RawMessage{
			"a": json.RawMessage(`{"id":"a","name":"my-item","notes":"exact"}`),
			"b": json.RawMessage(`{"id":"b","name":"my-item-backup","notes":"decoy"}`),
		},
	}
	srv := httptest.NewServer(f.handler())
	defer srv.Close()
	r := BWServeReader{Client: BWServeClient{BaseURL: srv.URL}}

	got, err := r.Field("my-item", "notes")
	if err != nil {
		t.Fatalf("Field: %v", err)
	}
	if got != "exact" {
		t.Fatalf("expected the exact-name match, got %q", got)
	}
}

func TestBWServeDaemon_Start_NoOpWhenAlreadyRunning(t *testing.T) {
	f := &fakeBWServe{status: "locked"}
	srv := httptest.NewServer(f.handler())
	defer srv.Close()

	d := &BWServeDaemon{
		Client: BWServeClient{BaseURL: srv.URL},
		newCmd: func(string, int) *exec.Cmd {
			t.Fatal("Start must not spawn a process when the daemon is already reachable")
			return nil
		},
	}
	if err := d.Start(time.Second); err != nil {
		t.Fatalf("Start: %v", err)
	}
}

func TestBWServeDaemon_Status_AbsentWhenUnreachable(t *testing.T) {
	d := &BWServeDaemon{
		Client: BWServeClient{BaseURL: "http://127.0.0.1:1", HTTPClient: &http.Client{Timeout: 200 * time.Millisecond}},
	}
	st, err := d.Status()
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if st != "absent" {
		t.Fatalf("expected absent, got %q", st)
	}
}

// fakeShellout is a minimal BWReader double standing in for BWGet, so
// BWFallbackReader's tests never need a real bw binary.
type fakeShellout struct {
	called bool
	value  string
	err    error
}

func (f *fakeShellout) Field(_, _ string) (string, error) {
	f.called = true
	return f.value, f.err
}

// TestBWFallbackReader_UsesServe_WhenUnlocked is AC2: an unlocked, reachable
// daemon is preferred over the shellout.
func TestBWFallbackReader_UsesServe_WhenUnlocked(t *testing.T) {
	f := &fakeBWServe{
		status: "unlocked",
		names:  map[string]string{"a": "my-item"},
		items:  map[string]json.RawMessage{"a": json.RawMessage(`{"id":"a","name":"my-item","notes":"from-serve"}`)},
	}
	srv := httptest.NewServer(f.handler())
	defer srv.Close()
	shellout := &fakeShellout{value: "from-shellout"}

	r := BWFallbackReader{
		Serve:    BWServeReader{Client: BWServeClient{BaseURL: srv.URL}},
		Shellout: shellout,
	}
	got, err := r.Field("my-item", "notes")
	if err != nil {
		t.Fatalf("Field: %v", err)
	}
	if got != "from-serve" {
		t.Fatalf("expected the serve path, got %q", got)
	}
	if shellout.called {
		t.Fatal("shellout must not be called when the daemon is unlocked")
	}
}

// TestBWFallbackReader_FallsBackToShellout_WhenLocked is AC3: a
// reachable-but-locked daemon still falls back — only "unlocked" opts in.
func TestBWFallbackReader_FallsBackToShellout_WhenLocked(t *testing.T) {
	f := &fakeBWServe{status: "locked"}
	srv := httptest.NewServer(f.handler())
	defer srv.Close()
	shellout := &fakeShellout{value: "from-shellout"}

	r := BWFallbackReader{
		Serve:    BWServeReader{Client: BWServeClient{BaseURL: srv.URL}},
		Shellout: shellout,
	}
	got, err := r.Field("my-item", "notes")
	if err != nil {
		t.Fatalf("Field: %v", err)
	}
	if got != "from-shellout" || !shellout.called {
		t.Fatalf("expected the shellout fallback, got %q (called=%v)", got, shellout.called)
	}
}

// TestBWFallbackReader_FallsBackToShellout_WhenAbsent is AC3: no daemon
// running at all — existing behavior, unchanged.
func TestBWFallbackReader_FallsBackToShellout_WhenAbsent(t *testing.T) {
	shellout := &fakeShellout{value: "from-shellout"}
	r := BWFallbackReader{
		Serve:    BWServeReader{Client: BWServeClient{BaseURL: "http://127.0.0.1:1", HTTPClient: &http.Client{Timeout: 200 * time.Millisecond}}},
		Shellout: shellout,
	}
	got, err := r.Field("my-item", "notes")
	if err != nil {
		t.Fatalf("Field: %v", err)
	}
	if got != "from-shellout" || !shellout.called {
		t.Fatalf("expected the shellout fallback, got %q (called=%v)", got, shellout.called)
	}
}

func isErrUnreachable(err error) bool {
	return errors.Is(err, ErrBWServeUnreachable)
}

func isErrItemNotFound(err error) bool {
	return errors.Is(err, ErrBWItemNotFound)
}
