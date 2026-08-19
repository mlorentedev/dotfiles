package secrets

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
	status string                     // unauthenticated | locked | unlocked
	items  map[string]json.RawMessage // id -> full item JSON
	names  map[string]string          // id -> name, for /list/object/items

	// folders backs /list/object/folders and POST /object/folder (id -> name),
	// the taxonomy half of the write path (BUG-084 / OPS-028).
	folders map[string]string

	// created records POST /object/item bodies in arrival order, so a test can
	// assert what CreateItem actually put on the wire, not merely that it 200'd.
	created []json.RawMessage

	// nextID is the id handed to the next created item/folder; "" -> "generated".
	nextID string

	// syncs counts POST /sync, so a test can assert a write made itself visible.
	syncs int

	// failSync makes POST /sync report failure, exercising the written-but-stale path.
	failSync bool

	unlockPassword string // "" -> any password succeeds
	failUnlock     bool
}

func (f *fakeBWServe) handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// A locked daemon refuses DATA endpoints — items, folders, sync — while
		// still answering /status, /unlock and /lock. The fake models that because
		// backend selection now probes a data endpoint rather than /status (BUG-082:
		// a /status call poisons item reads for ~0.5s), so "locked" has to be
		// observable the way the real daemon makes it observable.
		if f.status != "unlocked" && (strings.HasPrefix(r.URL.Path, "/object/") ||
			strings.HasPrefix(r.URL.Path, "/list/object/") || r.URL.Path == "/sync") {
			writeEnvelope(w, false, "Vault is locked.", nil)
			return
		}
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/status":
			// Real bw serve wraps this under "template" (captured live against
			// bw 2026.5.0, 2026-08-15) -- see bwServeStatusData's doc comment.
			// The fake matches reality, not the earlier wrong assumption, so a
			// regression here is caught by every test in this file that
			// depends on Status(), not just a single dedicated case.
			writeEnvelope(w, true, "", map[string]any{
				"object":   "template",
				"template": map[string]string{"status": f.status},
			})
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
		case r.Method == http.MethodPost && r.URL.Path == "/sync":
			f.syncs++
			if f.failSync {
				writeEnvelope(w, false, "Failed to sync.", nil)
				return
			}
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
			// Real bw serve wraps this under {"object":"list","data":[...]}
			// (captured live, 2026-08-15) -- same wrapping pattern as /status.
			writeEnvelope(w, true, "", map[string]any{"object": "list", "data": out})
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/object/item/"):
			id := strings.TrimPrefix(r.URL.Path, "/object/item/")
			item, ok := f.items[id]
			if !ok {
				writeEnvelope(w, false, "Not found.", nil)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprintf(w, `{"success":true,"data":%s}`, item)

		// --- write path (BUG-084) ---------------------------------------
		// bw serve takes a RAW JSON body on these, where the `bw` CLI needs
		// base64 on stdin. That asymmetry is the only real difference between
		// BWServeWriter and BWPut, so the fake reads raw JSON deliberately: a
		// regression to base64 fails to decode here.
		case r.Method == http.MethodPut && strings.HasPrefix(r.URL.Path, "/object/item/"):
			id := strings.TrimPrefix(r.URL.Path, "/object/item/")
			if _, ok := f.items[id]; !ok {
				writeEnvelope(w, false, "Not found.", nil)
				return
			}
			body, err := io.ReadAll(r.Body)
			if err != nil || !json.Valid(body) {
				writeEnvelope(w, false, "malformed request body", nil)
				return
			}
			f.items[id] = json.RawMessage(body)
			writeEnvelope(w, true, "", json.RawMessage(body))

		case r.Method == http.MethodPost && r.URL.Path == "/object/item":
			body, err := io.ReadAll(r.Body)
			if err != nil || !json.Valid(body) {
				writeEnvelope(w, false, "malformed request body", nil)
				return
			}
			id := f.newID()
			if f.items == nil {
				f.items = map[string]json.RawMessage{}
			}
			if f.names == nil {
				f.names = map[string]string{}
			}
			f.items[id] = json.RawMessage(body)
			var named struct {
				Name string `json:"name"`
			}
			_ = json.Unmarshal(body, &named)
			f.names[id] = named.Name
			f.created = append(f.created, json.RawMessage(body))
			writeEnvelope(w, true, "", json.RawMessage(body))

		case r.Method == http.MethodGet && r.URL.Path == "/list/object/folders":
			out := []map[string]string{}
			for id, name := range f.folders {
				out = append(out, map[string]string{"id": id, "name": name})
			}
			writeEnvelope(w, true, "", map[string]any{"object": "list", "data": out})

		case r.Method == http.MethodPost && r.URL.Path == "/object/folder":
			var body struct {
				Name string `json:"name"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			id := f.newID()
			if f.folders == nil {
				f.folders = map[string]string{}
			}
			f.folders[id] = body.Name
			writeEnvelope(w, true, "", map[string]string{"id": id, "name": body.Name})

		default:
			http.NotFound(w, r)
		}
	}
}

// newID hands out the id for a created item/folder, defaulting to a fixed value so
// assertions stay deterministic.
func (f *fakeBWServe) newID() string {
	if f.nextID != "" {
		return f.nextID
	}
	return "generated-id"
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

// TestBWServeClient_Status_RealCapturedResponseShape is a regression test for
// the bug the user's live smoke test caught on 2026-08-15: the raw response
// bytes captured directly from `curl http://127.0.0.1:8087/status` against a
// real bw 2026.5.0 daemon, wired through an httptest.Server verbatim (no
// fakeBWServe reshaping) so this test fails if the parser ever regresses to
// only handling the flat (wrong) shape, independent of the shared fake.
func TestBWServeClient_Status_RealCapturedResponseShape(t *testing.T) {
	const capturedResponse = `{"success":true,"data":{"object":"template","template":{"serverUrl":null,"lastSync":"2026-08-14T06:29:31.291Z","userEmail":"mlorente@duck.com","userId":"ce7b26cd-8cba-4a7a-87d1-b2a7005dffce","status":"locked"}}}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(capturedResponse))
	}))
	defer srv.Close()
	c := BWServeClient{BaseURL: srv.URL}

	st, err := c.Status()
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if st != "locked" {
		t.Fatalf("expected locked from the real captured response shape, got %q — the parser regressed to the flat-only assumption", st)
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

// TestBWServeReader_Field_RealCapturedListResponseShape is a regression test
// for the second live smoke-test bug (2026-08-15): the raw bytes captured
// via `curl http://127.0.0.1:8087/list/object/items?search=...` against a
// real bw 2026.5.0 daemon, wired through an httptest.Server verbatim. The
// token value is replaced with an obvious placeholder; only the envelope
// shape matters here, never the captured secret.
func TestBWServeReader_Field_RealCapturedListResponseShape(t *testing.T) {
	const listResponse = `{"success":true,"data":{"object":"list","data":[{"type":1,"name":"github-bitacora-pat","favorite":false,"reprompt":0,"id":"07f4de3b-d74a-48f7-a111-b4a6006a2f20","collectionIds":[],"object":"item","folderId":"5f1985f7-9d84-45c1-bd18-b4a60012a18f","fields":[{"type":1,"name":"api-token","value":"REDACTED-not-a-real-token"}],"login":{"uris":[],"fido2Credentials":[],"passwordRevisionDate":null},"passwordHistory":[],"creationDate":"2026-08-14T06:26:36.260Z","revisionDate":"2026-08-14T06:26:36.260Z","attachments":[]}]}}`
	const itemResponse = `{"success":true,"data":{"type":1,"name":"github-bitacora-pat","id":"07f4de3b-d74a-48f7-a111-b4a6006a2f20","fields":[{"type":1,"name":"api-token","value":"REDACTED-not-a-real-token"}]}}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasPrefix(r.URL.Path, "/list/object/items"):
			_, _ = w.Write([]byte(listResponse))
		case strings.HasPrefix(r.URL.Path, "/object/item/"):
			_, _ = w.Write([]byte(itemResponse))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	r := BWServeReader{Client: BWServeClient{BaseURL: srv.URL}}

	got, err := r.Field("github-bitacora-pat", "api-token")
	if err != nil {
		t.Fatalf("Field: %v — the list-endpoint parser regressed to the flat-array-only assumption", err)
	}
	if got != "REDACTED-not-a-real-token" {
		t.Fatalf("got %q, want the placeholder token", got)
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

// TestBWServeClient_UnparseableResponseNamesStatusAndSize is the guard for the
// diagnostic gap that let #988 stay uncharacterised: `bw serve` answers some
// /object/item GETs with an HTTP 500 carrying the plain text "Internal Server Error",
// and the old error reported only `invalid character 'I'` — true, and useless.
//
// The error must name the status code and the byte count, and must NOT echo the body:
// a 200 whose body merely failed to parse can still carry vault material.
func TestBWServeClient_UnparseableResponseNamesStatusAndSize(t *testing.T) {
	const body = "Internal Server Error"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	_, err := BWServeClient{BaseURL: srv.URL}.Status()
	if err == nil {
		t.Fatal("expected an error for a non-JSON response")
	}
	msg := err.Error()
	for _, want := range []string{"HTTP 500", "21 bytes"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("error must name %q so the failure is identifiable, got: %s", want, msg)
		}
	}
	if strings.Contains(msg, body) {
		t.Fatalf("error must NOT echo the response body (it may carry vault material): %s", msg)
	}
}
