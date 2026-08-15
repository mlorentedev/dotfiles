package secrets

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// DefaultBWServePort is bw serve's own documented default port.
const DefaultBWServePort = 8087

// bwServeHostname is the ONLY hostname this package will ever bind bw serve
// to. Its REST API is unauthenticated by design (see BWGet's doc comment) —
// anything other than loopback exposes the whole unlocked vault to the local
// network.
const bwServeHostname = "127.0.0.1"

// bwServeCommand builds the command that starts bw serve bound to
// bwServeHostname:port. Split out as a pure function (no side effects) so the
// bind-safety invariant (AC5) is unit-testable without spawning a process —
// the actual process lifecycle is live-verified only, like the rest of this
// package's bw shell-outs (BWGet, BWPut).
func bwServeCommand(bin string, port int) *exec.Cmd {
	if bin == "" {
		bin = "bw"
	}
	cmd := exec.Command(bin, "serve", "--hostname", bwServeHostname, "--port", strconv.Itoa(port)) //nolint:gosec // bin/port are operator-controlled config, not external input
	// Detach from this process's session so the daemon survives the CLI
	// invocation exiting — the whole point is one unlock outliving any single
	// `dotf` call. Platform-specific (see bwserve_unix.go / bwserve_windows.go).
	cmd.SysProcAttr = bwServeDetachAttr()
	return cmd
}

// bwServeEnvelope is bw serve's response wrapper: every endpoint replies
// {"success": bool, "message"?: string, "data"?: ...}. Verified empirically
// against bw 2026.5.0 (OPS-021 spike, #675) — a wrong-password /unlock
// returns success:false with a real crypto-error message, not a bare HTTP
// error code, so the envelope is the only reliable success signal.
type bwServeEnvelope struct {
	Success bool            `json:"success"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

// bwServeStatusData is the payload bw serve's /status returns inside data.
//
// The real shape, captured live against bw 2026.5.0 (2026-08-15, first live
// unlock of the daemon this PR added — the OPS-021 spike's earlier probe of
// this same endpoint was apparently misread, since it never actually drove
// BWFallbackReader against a live daemon): the status fields are wrapped one
// level deeper than assumed, under "template":
//
//	{"success":true,"data":{"object":"template","template":{"status":"locked",...}}}
//
// Parsing only the flat shape silently returned an empty Status with a nil
// error (valid JSON, just no top-level "status" key) — BWFallbackReader's
// `st == "unlocked"` check then never matched, so it silently fell back to
// the CLI shellout on every call, defeating the daemon read path entirely
// while reporting no error at all. Handling both shapes here, preferring the
// nested one when present, is the fix — and the reason it stays defensive
// rather than dropping the flat fallback is that this response shape is
// undocumented behavior of a third-party CLI, not a contract this package
// controls.
type bwServeStatusData struct {
	Status   string `json:"status"` // unauthenticated | locked | unlocked
	Template *struct {
		Status string `json:"status"`
	} `json:"template"`
}

// status returns the actual status value, preferring the "template"-nested
// shape when present (see bwServeStatusData's doc comment).
func (d bwServeStatusData) status() string {
	if d.Template != nil && d.Template.Status != "" {
		return d.Template.Status
	}
	return d.Status
}

// ErrBWServeUnreachable marks a daemon that did not answer at all (not
// running, wrong port, still starting) — distinct from a reachable daemon
// that reported failure, which callers need to tell apart (Start()'s poll
// loop retries the former, never the latter).
var ErrBWServeUnreachable = errors.New("bw serve daemon unreachable")

// BWServeClient talks to an already-running bw serve daemon over its local
// REST API. Every method here is unit-tested against an httptest.Server
// fake — no real bw process, no unlocked vault — mirroring how fieldFromItem
// is unit-tested while BWGet's shell-out is live-verified only.
type BWServeClient struct {
	BaseURL string // e.g. "http://127.0.0.1:8087"; "" -> default host:port

	// HTTPClient is overridable in tests; nil -> a client with a short,
	// bounded timeout (a hung daemon must not hang the whole CLI).
	HTTPClient *http.Client
}

func (c BWServeClient) baseURL() string {
	if c.BaseURL != "" {
		return c.BaseURL
	}
	return fmt.Sprintf("http://%s:%d", bwServeHostname, DefaultBWServePort)
}

func (c BWServeClient) httpClient() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return &http.Client{Timeout: 5 * time.Second}
}

// call issues method to path (optionally with a JSON body), decodes bw
// serve's envelope, and returns its data on success or a descriptive error
// on failure. A transport-level failure (connection refused, timeout) is
// always ErrBWServeUnreachable — the one signal Start()'s poll loop needs to
// keep retrying instead of giving up.
func (c BWServeClient) call(method, path string, body any) (json.RawMessage, error) {
	var reqBody *bytes.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("encode request: %w", err)
		}
		reqBody = bytes.NewReader(b)
	} else {
		reqBody = bytes.NewReader(nil)
	}
	req, err := http.NewRequest(method, c.baseURL()+path, reqBody) //nolint:noctx // bounded by the client's own Timeout
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %s %s: %v", ErrBWServeUnreachable, method, path, err)
	}
	defer func() { _ = resp.Body.Close() }()
	var env bwServeEnvelope
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		return nil, fmt.Errorf("%s %s: bw serve returned no parseable envelope: %w", method, path, err)
	}
	if !env.Success {
		msg := env.Message
		if msg == "" {
			msg = "no message"
		}
		return nil, fmt.Errorf("%s %s: %s", method, path, msg)
	}
	return env.Data, nil
}

// Status reports the daemon's lock state. Returns ErrBWServeUnreachable
// (wrapped) when nothing is listening — the caller decides whether that
// means "not started" or "just crashed"; this method only reports what it
// observed.
func (c BWServeClient) Status() (string, error) {
	data, err := c.call(http.MethodGet, "/status", nil)
	if err != nil {
		return "", err
	}
	var st bwServeStatusData
	if err := json.Unmarshal(data, &st); err != nil {
		return "", fmt.Errorf("GET /status: unparseable data: %w", err)
	}
	return st.status(), nil
}

// Unlock POSTs password to the daemon's /unlock endpoint. The password is
// held only for the duration of this call — never logged, never returned in
// any error this method produces, never written to disk. Idempotent: calling
// Unlock while already unlocked succeeds (bw serve itself treats a redundant
// unlock as a success, not an error — verified in the OPS-021 spike).
func (c BWServeClient) Unlock(password string) error {
	_, err := c.call(http.MethodPost, "/unlock", map[string]string{"password": password})
	if err != nil {
		return fmt.Errorf("unlock: %w", scrubPassword(err, password))
	}
	return nil
}

// Lock POSTs to the daemon's /lock endpoint, re-locking the vault without
// stopping the daemon process itself (a lighter, faster-to-reverse operation
// than killing it — the next `dotf secrets unlock` only has to re-POST the
// password, not wait out a fresh Node cold start).
func (c BWServeClient) Lock() error {
	_, err := c.call(http.MethodPost, "/lock", nil)
	if err != nil {
		return fmt.Errorf("lock: %w", err)
	}
	return nil
}

// scrubPassword defends against a future bw serve response ever echoing the
// submitted password back in an error message (never observed in the OPS-021
// spike, but Unlock's contract — password never appears in an error — should
// hold even if that changes upstream).
func scrubPassword(err error, password string) error {
	if password == "" || err == nil {
		return err
	}
	msg := err.Error()
	if !strings.Contains(msg, password) {
		return err
	}
	return errors.New(strings.ReplaceAll(msg, password, "[redacted]"))
}

// bwServeListItem is the subset of a /list/object/items entry this package
// needs to find the exact item a registry entry names — the id/name pair, not
// the full item body (that comes from a second, targeted GET).
type bwServeListItem struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// BWServeReader is the serve-backed BWReader: a drop-in for BWGet behind the
// existing seam (resolve.go's bwResolver), selected automatically when a
// daemon is reachable and unlocked. It reuses fieldFromItem for extraction so
// the two backends can never diverge in what a "field" means for a given item
// shape.
type BWServeReader struct {
	Client BWServeClient
}

// Field looks up item by exact name or id match (mirroring `bw get item
// <item>`'s own disambiguation — see BWGet's doc comment) and extracts field
// from it via fieldFromItem, so a serve-backed read produces byte-identical
// results to the CLI-shellout read for the same vault state.
func (r BWServeReader) Field(item, field string) (string, error) {
	itemJSON, err := r.getItemJSON(item)
	if err != nil {
		return "", err
	}
	return fieldFromItem(itemJSON, field)
}

// getItemJSON resolves item (a name or an id) to its full JSON body. The
// serve API's search endpoint is fuzzy (substring match), so this always
// filters to an EXACT name or id match before accepting a result — a fuzzy
// hit would silently read the wrong secret.
//
// NOTE on the list-endpoint envelope shape: /status, /unlock and /lock were
// verified empirically against a live bw serve 2026.5.0 (OPS-021 spike,
// #675). /list/object/items was NOT — no unlocked vault was used in that
// spike. This assumes bw's envelope wraps the array directly under `data`
// (call() already unwraps one envelope level); the live verification task in
// tasks.md confirms or corrects this against a real daemon before archive.
func (r BWServeReader) getItemJSON(item string) ([]byte, error) {
	data, err := r.Client.call(http.MethodGet, "/list/object/items?search="+url.QueryEscape(item), nil)
	if err != nil {
		return nil, fmt.Errorf("bw serve list items %q: %w", item, err)
	}
	var items []bwServeListItem
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, fmt.Errorf("bw serve list items %q: unparseable data: %w", item, err)
	}
	var matches []bwServeListItem
	for _, it := range items {
		if it.Name == item || it.ID == item {
			matches = append(matches, it)
		}
	}
	switch len(matches) {
	case 0:
		return nil, fmt.Errorf("%w: bw serve item %q: not found", ErrBWItemNotFound, item)
	case 1:
		// fall through
	default:
		return nil, fmt.Errorf("bw serve item %q: More than one result was found", item)
	}
	itemData, err := r.Client.call(http.MethodGet, "/object/item/"+url.PathEscape(matches[0].ID), nil)
	if err != nil {
		return nil, fmt.Errorf("bw serve get item %q: %w", item, err)
	}
	return itemData, nil
}

// BWServeDaemon owns the lifecycle of a dotf-managed bw serve process: start
// it (detached, localhost-only), poll until reachable, and delegate lock
// state to BWServeClient. The process-spawn half is live-verified only (like
// BWGet/BWPut); the HTTP half it wraps is the fully unit-tested
// BWServeClient.
type BWServeDaemon struct {
	Bin  string // bw binary; "" -> "bw"
	Port int    // 0 -> DefaultBWServePort

	Client BWServeClient // talks to the daemon once it's up

	// newCmd is the process-construction seam, overridable in tests so
	// Start()'s retry/timeout behavior is testable without a real bw binary.
	// nil -> bwServeCommand.
	newCmd func(bin string, port int) *exec.Cmd
}

func (d *BWServeDaemon) port() int {
	if d.Port != 0 {
		return d.Port
	}
	return DefaultBWServePort
}

func (d *BWServeDaemon) command() *exec.Cmd {
	if d.newCmd != nil {
		return d.newCmd(d.Bin, d.port())
	}
	return bwServeCommand(d.Bin, d.port())
}

// Running reports whether a daemon is already reachable, regardless of lock
// state.
func (d *BWServeDaemon) Running() bool {
	_, err := d.Client.Status()
	return err == nil
}

// Start ensures a bw serve daemon is running and reachable, spawning one
// (detached) if none answers yet. It is idempotent: calling Start against an
// already-running daemon is a fast no-op (the first Status() check succeeds
// and it returns immediately without spawning a second process).
func (d *BWServeDaemon) Start(pollTimeout time.Duration) error {
	if d.Running() {
		return nil
	}
	cmd := d.command()
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start bw serve: %w", err)
	}
	deadline := time.Now().Add(pollTimeout)
	for time.Now().Before(deadline) {
		if d.Running() {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("bw serve did not become reachable within %s", pollTimeout)
}

// Unlock delegates to the client — see BWServeClient.Unlock's contract.
func (d *BWServeDaemon) Unlock(password string) error { return d.Client.Unlock(password) }

// Lock delegates to the client — see BWServeClient.Lock.
func (d *BWServeDaemon) Lock() error { return d.Client.Lock() }

// Status delegates to the client, mapping an unreachable daemon to the
// explicit "absent" state rather than surfacing a raw connection error —
// doctor and `dotf secrets unlock` both need a three-way state (absent /
// locked / unlocked), not a Go error to pattern-match.
func (d *BWServeDaemon) Status() (string, error) {
	st, err := d.Client.Status()
	if errors.Is(err, ErrBWServeUnreachable) {
		return "absent", nil
	}
	return st, err
}

// BWFallbackReader is the BWReader this package's callers should default to:
// it reads through a local bw serve daemon when one is reachable and
// unlocked, and falls back to the CLI shellout otherwise — additively, with
// no change to any consumer (AC2, AC3). The status check runs once per
// Field() call — ~9ms against a running daemon (OPS-021 spike measurement),
// effectively free against an absent one (a fast connection-refused).
type BWFallbackReader struct {
	Serve    BWServeReader
	Shellout BWReader // typically BWGet{}
}

func (r BWFallbackReader) Field(item, field string) (string, error) {
	if st, err := r.Serve.Client.Status(); err == nil && st == "unlocked" {
		return r.Serve.Field(item, field)
	}
	return r.Shellout.Field(item, field)
}
