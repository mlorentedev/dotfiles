package cmd

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/secrets"
)

// probeRegistry: one bw-backed secret (probeable) and one age-backed (not), the
// two shapes the command must treat differently.
const probeRegistry = `
version: 1
secrets:
  - id: NAN_API_KEY
    plane: app
    backend: bw
    bw: { item: nan-api-key, field: api-key, folder: apps }
    expose: { env: NAN_API_KEY }
  - id: SSH_KEY
    plane: app
    backend: age
    age: ssh
    expose: { env: SSH_KEY }
`

func runProbe(t *testing.T, args ...string) (string, error) {
	t.Helper()
	useTempRegistry(t, probeRegistry)
	c := newSecretsProbeCmd()
	var out bytes.Buffer
	c.SetOut(&out)
	c.SetErr(&out)
	c.SetArgs(args)
	err := c.Execute()
	return out.String(), err
}

// A count of zero ran the loop zero times and exited 0 — success reported for
// work never done, which is the exact class this command was built to close,
// reproduced inside it. It must refuse instead.
func TestProbeCmd_NonPositiveCountIsRefused(t *testing.T) {
	for _, n := range []string{"0", "-5"} {
		out, err := runProbe(t, "NAN_API_KEY", "--count", n)
		if err == nil {
			t.Errorf("--count %s must be refused, not silently do nothing (output: %q)", n, out)
			continue
		}
		if !strings.Contains(err.Error(), "at least 1") {
			t.Errorf("--count %s: the error must name the constraint, got: %v", n, err)
		}
	}
}

// Probing a secret that does not resolve through the daemon has nothing to
// probe. Saying so beats reporting on a request that was never going to happen.
func TestProbeCmd_NonBWSecretIsRefused(t *testing.T) {
	_, err := runProbe(t, "SSH_KEY")
	if err == nil {
		t.Fatal("an age-backed secret has no daemon request to probe; want a refusal")
	}
	if !strings.Contains(err.Error(), "nothing to probe") {
		t.Errorf("the refusal must say why, got: %v", err)
	}
}

func TestProbeCmd_UnknownSecretIsRefused(t *testing.T) {
	_, err := runProbe(t, "NO_SUCH_SECRET")
	if err == nil {
		t.Fatal("an id absent from the registry must be refused")
	}
	if !strings.Contains(err.Error(), "not in the registry") {
		t.Errorf("the refusal must name the cause, got: %v", err)
	}
}

// Finding 5 from the adversarial review: every cmd-level probe test asserted a
// refusal, so a regression in the WIRING — flags not reaching ShapeProbe, the
// success predicate inverted, the distribution miscounted — would pass unit
// tests while the command misbehaved.
func TestProbeCmd_ReportsDistributionAndNeverPrintsAValue(t *testing.T) {
	const secret = "SUPER-SECRET-CANARY-VALUE"
	var itemHits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/list/object/items") {
			_, _ = w.Write([]byte(`{"success":true,"data":{"data":[{"id":"abc","name":"nan-api-key"}]}}`))
			return
		}
		itemHits++
		// Alternate success and failure so the distribution has something to say
		// and --raw has a non-2xx body to show.
		if itemHits%2 == 0 {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("Internal Server Error"))
			return
		}
		_, _ = w.Write([]byte(`{"success":true,"data":{"fields":[{"name":"api-key","value":"` + secret + `"}]}}`))
	}))
	defer srv.Close()

	orig := probeClient
	probeClient = func() secrets.BWServeClient { return secrets.BWServeClient{BaseURL: srv.URL} }
	t.Cleanup(func() { probeClient = orig })

	out, err := runProbe(t, "NAN_API_KEY", "--raw", "--count", "4")
	if err != nil {
		t.Fatalf("probe failed: %v\n%s", err, out)
	}
	if strings.Contains(out, secret) {
		t.Fatalf("the value reached command output:\n%s", out)
	}
	for _, want := range []string{"HTTP 200", "HTTP 500", "4 probes", "not deterministic"} {
		if !strings.Contains(out, want) {
			t.Errorf("want %q in the distribution report, got:\n%s", want, out)
		}
	}
	// --raw earns its keep here: the failing attempts' bodies are shown.
	if !strings.Contains(out, "Internal Server Error") {
		t.Errorf("--raw must surface the non-2xx body under --count, got:\n%s", out)
	}
}
