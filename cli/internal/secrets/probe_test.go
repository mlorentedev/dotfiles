package secrets

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// sentinel stands in for a credential. Every test here asks the same question in
// a different shape: can this value reach output? The answer must be no in all of
// them, which is why they are written before the code that formats anything.
const sentinel = "SUPER-SECRET-CANARY-VALUE"

func itemBody(value string) string {
	return `{"success":true,"data":{"object":"item","name":"nan-api-key",` +
		`"fields":[{"type":1,"name":"api-key","value":"` + value + `"}],` +
		`"login":{"password":"` + value + `"}}}`
}

// The whole ticket in one assertion. A 200 from /object/item/<id> IS the
// credential, so shaping one must not put it in the report -- in any mode.
func TestShapeProbe_NeverEmitsAValue(t *testing.T) {
	for _, raw := range []bool{false, true} {
		res := ProbeResult{
			Status: http.StatusOK, ContentType: "application/json",
			Size: len(itemBody(sentinel)), Body: []byte(itemBody(sentinel)),
		}

		got := ShapeProbe(res, raw).String()

		if strings.Contains(got, sentinel) {
			t.Fatalf("raw=%v: the value reached output:\n%s", raw, got)
		}
	}
}

// --raw is the flag that could quietly undo this feature, so the 2xx bound is
// pinned separately rather than folded into the loop above: a 200 body is never
// diagnostic material, it is only ever a credential.
func TestShapeProbe_RawNeverShowsA2xxBody(t *testing.T) {
	res := ProbeResult{
		Status: http.StatusOK, ContentType: "application/json",
		Size: len(itemBody(sentinel)), Body: []byte(itemBody(sentinel)),
	}

	got := ShapeProbe(res, true).String()

	if strings.Contains(got, "body:") {
		t.Errorf("a 2xx body must never be echoed, even with --raw:\n%s", got)
	}
}

// The other half: a non-2xx body is the actual artifact being hunted -- the
// `Internal Server Error` page that made #988 unreadable for two sessions -- and
// it cannot be a credential, so --raw shows it.
func TestShapeProbe_RawShowsNon2xxBody(t *testing.T) {
	res := ProbeResult{
		Status: http.StatusInternalServerError, ContentType: "text/plain",
		Size: 21, Body: []byte("Internal Server Error"),
	}

	withRaw := ShapeProbe(res, true).String()
	without := ShapeProbe(res, false).String()

	if !strings.Contains(withRaw, "Internal Server Error") {
		t.Errorf("--raw must show a non-2xx body; that is the artifact:\n%s", withRaw)
	}
	if strings.Contains(without, "Internal Server Error") {
		t.Errorf("without --raw the body stays out of the report:\n%s", without)
	}
}

// A non-2xx body is unbounded attacker-adjacent input; showing it must not mean
// pasting an HTML error page into a terminal.
func TestShapeProbe_RawBodyIsCapped(t *testing.T) {
	res := ProbeResult{
		Status: http.StatusBadGateway, ContentType: "text/html",
		Size: 10000, Body: []byte(strings.Repeat("x", 10000)),
	}

	got := ShapeProbe(res, true).String()

	if len(got) > rawBodyCap+512 {
		t.Errorf("raw body was not capped: report is %d bytes", len(got))
	}
	if !strings.Contains(got, "truncated") {
		t.Errorf("a truncated body must say so, else it reads as the whole reply:\n%s", got)
	}
}

// Fingerprints are what make a probe useful without being dangerous: they answer
// "did this change?" -- the question a liveness check cannot answer, since an
// unrevoked old credential authenticates exactly as well as a new one.
func TestShapeProbe_FingerprintsDistinguishValues(t *testing.T) {
	shape := func(v string) string {
		body := itemBody(v)
		return ShapeProbe(ProbeResult{Status: 200, Size: len(body), Body: []byte(body)}, false).String()
	}

	same, other := shape(sentinel), shape(sentinel+"-rotated")

	if same != shape(sentinel) {
		t.Error("identical values must fingerprint identically, or nothing can be compared")
	}
	if same == other {
		t.Error("different values must fingerprint differently, or rotation cannot be proven")
	}
}

// Field names and lengths are schema, not content, and they are what tells an
// operator whether the mapping is wrong versus the value being absent.
func TestShapeProbe_ReportsShapeNotContent(t *testing.T) {
	body := itemBody(sentinel)

	got := ShapeProbe(ProbeResult{Status: 200, Size: len(body), Body: []byte(body)}, false).String()

	for _, want := range []string{"api-key", "len=25", "HTTP 200"} {
		if !strings.Contains(got, want) {
			t.Errorf("want %q in the report, got:\n%s", want, got)
		}
	}
}

// The case that started all of this: a reply that is not JSON at all. The report
// has to say so plainly, because "invalid character 'I'" is what two sessions
// spent a day failing to interpret.
func TestShapeProbe_NonJSONIsStatedNotGuessed(t *testing.T) {
	res := ProbeResult{
		Status: http.StatusInternalServerError, ContentType: "text/plain",
		Size: 21, Body: []byte("Internal Server Error"),
	}

	got := ShapeProbe(res, false).String()

	if !strings.Contains(got, "not JSON") {
		t.Errorf("a non-JSON reply must be named as such:\n%s", got)
	}
	if !strings.Contains(got, "HTTP 500") {
		t.Errorf("the status is the fact that was missing all along:\n%s", got)
	}
}

// An empty value is a real state (a cleared field) and must not read as a normal
// one. Fingerprint already renders it "(empty)"; the report must not undo that.
func TestShapeProbe_EmptyValueIsLegible(t *testing.T) {
	body := itemBody("")

	got := ShapeProbe(ProbeResult{Status: 200, Size: len(body), Body: []byte(body)}, false).String()

	if !strings.Contains(got, "(empty)") {
		t.Errorf("an empty value must be legible as empty, not as just another value:\n%s", got)
	}
}

// Finding 3 from the adversarial review: the field-label path handed a
// non-string `value` straight back into the walk, where it matched no case and
// vanished. No leak, but the field became invisible in a report whose whole job
// is to describe the payload's shape.
func TestShapeProbe_NonStringValuesStayVisible(t *testing.T) {
	body := `{"success":true,"data":{"fields":[` +
		`{"name":"enabled","value":true},` +
		`{"name":"retries","value":3},` +
		`{"name":"cleared","value":null}]}}`

	got := ShapeProbe(ProbeResult{Status: 200, Size: len(body), Body: []byte(body)}, false).String()

	for _, want := range []string{"enabled", "retries", "cleared", "(bool)", "(float64)", "(null)"} {
		if !strings.Contains(got, want) {
			t.Errorf("want %q in the report, got:\n%s", want, got)
		}
	}
	// The type is reported; the value never is. Scoped to the values block --
	// "true" also appears in the envelope's own success flag, which is
	// daemon-authored control data rather than payload content.
	for _, line := range strings.Split(got, "\n") {
		if !strings.Contains(line, "data.fields") {
			continue
		}
		if strings.Contains(line, "true") || strings.Contains(line, " 3 ") {
			t.Errorf("a non-string value must be described by type, never printed: %s", line)
		}
	}
}

// Finding 2: ProbeItemID holds a credential-bearing search response in memory.
// It is safe by scoping today; this pins that only the id ever leaves, so a
// future edit that renders or wraps the body fails here rather than in a log.
func TestProbeItemID_ReturnsOnlyTheIDNeverBodyBytes(t *testing.T) {
	const secret = "SUPER-SECRET-CANARY-VALUE"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"success":true,"data":{"data":[{"id":"abc-123","name":"nan-api-key",` +
			`"fields":[{"name":"api-key","value":"` + secret + `"}]}]}}`))
	}))
	defer srv.Close()

	id, err := (BWServeClient{BaseURL: srv.URL}).ProbeItemID("nan-api-key")
	if err != nil {
		t.Fatal(err)
	}
	if id != "abc-123" {
		t.Errorf("want the resolved id, got %q", id)
	}
	if strings.Contains(id, secret) {
		t.Error("the item body escaped through the returned id")
	}
	// The ambiguity and not-found paths must not carry it either.
	_, err = (BWServeClient{BaseURL: srv.URL}).ProbeItemID("no-such-item")
	if err != nil && strings.Contains(err.Error(), secret) {
		t.Errorf("the not-found error carried body bytes: %v", err)
	}
}
